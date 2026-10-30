<template>
  <div
    ref="stage"
    class="zoom-stage"
    :class="{ 'zoom-stage--zoomed': scale > 1, 'zoom-stage--dragging': dragging }"
    :style="{ height }"
    @dblclick="onDoubleClick"
    @pointercancel="endDrag"
    @pointerdown="startDrag"
    @pointermove="onDrag"
    @pointerup="endDrag"
    @wheel.prevent="onWheel"
  >
    <div class="zoom-stage__frame" :style="{ transform: `translate(${tx}px, ${ty}px) scale(${scale})` }">
      <img
        :alt="alt"
        class="zoom-stage__img"
        draggable="false"
        :src="src"
        :style="turnStyle"
      >
    </div>

    <!-- A rotate or replace in flight: a percentage while a replacement
         uploads, then a spinner while the server converts it. -->
    <div v-if="busy" class="zoom-busy" @dblclick.stop @pointerdown.stop>
      <v-progress-circular
        color="primary"
        :indeterminate="progress == null || progress >= 1"
        :model-value="(progress ?? 0) * 100"
        size="56"
        width="5"
      >
        <span v-if="progress != null && progress < 1" class="text-caption">{{ Math.round(progress * 100) }}%</span>
      </v-progress-circular>
      <div class="text-caption mt-2">
        {{ progress == null ? 'Saving…' : progress < 1 ? 'Uploading…' : 'Converting…' }}
      </div>
    </div>

    <!-- An unsaved turn: Undo and Save get their own bar at the top. In the
         toolbar they pushed the rotate buttons aside, so a second click on
         "rotate right" landed on Save. -->
    <div v-if="rotatable && pendingTurn" class="zoom-pending" @dblclick.stop @pointerdown.stop>
      <span class="text-caption">Turned {{ pendingTurn === 270 ? -90 : pendingTurn }}°</span>
      <v-btn
        density="comfortable"
        :disabled="busy"
        prepend-icon="mdi-close"
        rounded="pill"
        size="small"
        text="Undo"
        variant="text"
        @click="pendingTurn = 0"
      />
      <v-btn
        color="primary"
        density="comfortable"
        :disabled="busy"
        prepend-icon="mdi-check"
        rounded="pill"
        size="small"
        text="Save"
        variant="flat"
        @click="saveTurn"
      />
    </div>

    <div class="zoom-controls" @dblclick.stop @pointerdown.stop>
      <template v-if="replaceable">
        <v-btn
          aria-label="Replace photo"
          density="comfortable"
          :disabled="busy"
          icon="mdi-image-edit-outline"
          size="small"
          title="Replace photo"
          variant="text"
          @click="fileInput?.click()"
        />
        <input
          ref="fileInput"
          accept="image/jpeg,image/png,image/webp,image/heic"
          hidden
          type="file"
          @change="onFilePicked"
        >
        <v-divider v-if="!rotatable" class="mx-1" vertical />
      </template>
      <template v-if="rotatable">
        <v-btn
          aria-label="Rotate left"
          density="comfortable"
          :disabled="busy"
          icon="mdi-rotate-left"
          size="small"
          variant="text"
          @click="turn(-90)"
        />
        <v-btn
          aria-label="Rotate right"
          density="comfortable"
          :disabled="busy"
          icon="mdi-rotate-right"
          size="small"
          variant="text"
          @click="turn(90)"
        />
        <v-divider class="mx-1" vertical />
      </template>
      <v-btn
        aria-label="Zoom out"
        density="comfortable"
        :disabled="scale <= MIN_SCALE"
        icon="mdi-magnify-minus-outline"
        size="small"
        variant="text"
        @click="zoomBy(1 / BUTTON_STEP)"
      />
      <span class="zoom-controls__level text-caption">{{ Math.round(scale * 100) }}%</span>
      <v-btn
        aria-label="Zoom in"
        density="comfortable"
        :disabled="scale >= MAX_SCALE"
        icon="mdi-magnify-plus-outline"
        size="small"
        variant="text"
        @click="zoomBy(BUTTON_STEP)"
      />
      <v-btn
        aria-label="Fit to screen"
        density="comfortable"
        :disabled="scale === 1"
        icon="mdi-fit-to-screen-outline"
        size="small"
        variant="text"
        @click="reset"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

  // A photo viewer for dialogs: the image is fitted (contain) into a
  // fixed-height stage, and can be zoomed with the mouse wheel (towards the
  // cursor), the +/- buttons (towards the centre) or a double-click, then
  // dragged around while zoomed in.
  //
  // With `rotatable` it also offers rotate left/right buttons, and with
  // `replaceable` a button to pick another file. The rotate buttons only turn
  // the photo on screen, instantly; Save then emits `rotate` once with the
  // whole turn (-90, 90 or 180). It saves nothing itself: on `rotate` or
  // `replace` (the picked File) the parent saves that (usePhotoActions) and
  // passes the re-fetched photo back as a new `src`, holding `busy` while
  // that's in flight.
  const props = defineProps({
    src: {
      type: String,
      required: true,
    },
    alt: {
      type: String,
      default: '',
    },
    height: {
      type: String,
      default: 'min(70vh, 640px)',
    },
    rotatable: {
      type: Boolean,
      default: false,
    },
    replaceable: {
      type: Boolean,
      default: false,
    },
    busy: {
      type: Boolean,
      default: false,
    },
    // 0..1 of a replacement upload sent, null when none is under way.
    progress: {
      type: Number,
      default: null,
    },
  })

  const emit = defineEmits<{ rotate: [delta: number], replace: [file: File] }>()

  const fileInput = ref<HTMLInputElement | null>(null)
  function onFilePicked () {
    const file = fileInput.value?.files?.[0]
    // Cleared so picking the same file again still fires change.
    if (fileInput.value) fileInput.value.value = ''
    if (file) emit('replace', file)
  }

  const MIN_SCALE = 1
  const MAX_SCALE = 8
  const BUTTON_STEP = 1.5
  const DOUBLE_CLICK_SCALE = 2.5

  const stage = ref<HTMLElement | null>(null)
  const scale = ref(1)
  // Translation of the image's top-left corner, in stage pixels. The
  // transform origin is the top-left corner, so a stage point p shows the
  // image point (p - t) / scale.
  const tx = ref(0)
  const ty = ref(0)

  function reset () {
    scale.value = 1
    tx.value = 0
    ty.value = 0
  }

  // The turn shown on screen but not saved yet: 0, 90, 180 or 270 clockwise.
  const pendingTurn = ref(0)
  function turn (delta: number) {
    pendingTurn.value = (pendingTurn.value + delta + 360) % 360
  }
  function saveTurn () {
    // The server takes -90, 90 or 180 on top of the saved turn.
    emit('rotate', pendingTurn.value === 270 ? -90 : pendingTurn.value)
  }

  // A sideways turn swaps the image's box, so it is laid out at the stage's
  // height by its width, centred, and turned back into the stage.
  const stageWidth = ref(0)
  const stageHeight = ref(0)
  let observer: ResizeObserver | null = null
  onMounted(() => {
    observer = new ResizeObserver(() => {
      stageWidth.value = stage.value?.clientWidth ?? 0
      stageHeight.value = stage.value?.clientHeight ?? 0
    })
    observer.observe(stage.value!)
  })
  onBeforeUnmount(() => observer?.disconnect())
  const turnStyle = computed(() => {
    if (!pendingTurn.value) return {}
    if (pendingTurn.value === 180) return { transform: 'rotate(180deg)' }
    const w = stageWidth.value
    const h = stageHeight.value
    return {
      position: 'absolute' as const,
      width: `${h}px`,
      height: `${w}px`,
      left: `${(w - h) / 2}px`,
      top: `${(h - w) / 2}px`,
      transform: `rotate(${pendingTurn.value}deg)`,
    }
  })

  // A new photo (the saved turn coming back, or another photo) starts fitted
  // and unturned.
  watch(() => props.src, () => {
    reset()
    pendingTurn.value = 0
  })

  // Keep the scaled image covering the stage, so it can't be dragged out of
  // view: t stays within [size - size * scale, 0] on each axis.
  function clamp () {
    const { clientWidth: w, clientHeight: h } = stage.value!
    tx.value = Math.min(0, Math.max(w - w * scale.value, tx.value))
    ty.value = Math.min(0, Math.max(h - h * scale.value, ty.value))
  }

  // Zoom to `next`, keeping the image point under stage point (cx, cy) fixed.
  function zoomAt (next: number, cx: number, cy: number) {
    const s = Math.min(MAX_SCALE, Math.max(MIN_SCALE, next))
    const ratio = s / scale.value
    tx.value = cx - (cx - tx.value) * ratio
    ty.value = cy - (cy - ty.value) * ratio
    scale.value = s
    clamp()
  }

  function stagePoint (e: MouseEvent): [number, number] {
    const rect = stage.value!.getBoundingClientRect()
    return [e.clientX - rect.left, e.clientY - rect.top]
  }

  function zoomBy (factor: number) {
    const { clientWidth: w, clientHeight: h } = stage.value!
    zoomAt(scale.value * factor, w / 2, h / 2)
  }

  function onWheel (e: WheelEvent) {
    // Trackpads send many small deltas, mouse wheels a few large ones; an
    // exponential keeps both smooth and symmetric in/out.
    zoomAt(scale.value * Math.exp(-e.deltaY * 0.0015), ...stagePoint(e))
  }

  function onDoubleClick (e: MouseEvent) {
    if (scale.value > 1) {
      reset()
    } else {
      zoomAt(DOUBLE_CLICK_SCALE, ...stagePoint(e))
    }
  }

  const dragging = ref(false)
  let last: [number, number] | null = null

  function startDrag (e: PointerEvent) {
    if (scale.value <= 1 || e.button !== 0) {
      return
    }
    dragging.value = true
    last = [e.clientX, e.clientY]
    stage.value!.setPointerCapture(e.pointerId)
  }

  function onDrag (e: PointerEvent) {
    if (!dragging.value || !last) {
      return
    }
    tx.value += e.clientX - last[0]
    ty.value += e.clientY - last[1]
    last = [e.clientX, e.clientY]
    clamp()
  }

  function endDrag () {
    dragging.value = false
    last = null
  }
</script>

<style scoped>
.zoom-stage {
  position: relative;
  width: 100%;
  overflow: hidden;
  touch-action: none;
  user-select: none;
}

.zoom-stage--zoomed {
  cursor: grab;
}

.zoom-stage--dragging {
  cursor: grabbing;
}

.zoom-stage__frame {
  position: relative;
  width: 100%;
  height: 100%;
  transform-origin: 0 0;
}

.zoom-stage__img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.zoom-busy {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: rgba(var(--v-theme-surface), 0.7);
  cursor: progress;
}

.zoom-controls {
  position: absolute;
  right: 8px;
  bottom: 8px;
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 2px 4px;
  border-radius: 999px;
  background: rgba(var(--v-theme-surface), 0.85);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.25);
  cursor: default;
}

.zoom-pending {
  position: absolute;
  top: 8px;
  left: 50%;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 4px 2px 12px;
  border-radius: 999px;
  background: rgba(var(--v-theme-surface), 0.92);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.25);
  transform: translateX(-50%);
  cursor: default;
}

.zoom-controls__level {
  min-width: 3.5em;
  text-align: center;
  font-variant-numeric: tabular-nums;
}
</style>
