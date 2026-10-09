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
    <img
      :alt="alt"
      class="zoom-stage__img"
      draggable="false"
      :src="src"
      :style="{ transform: `translate(${tx}px, ${ty}px) scale(${scale})` }"
    >

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
          @click="emit('rotate', -90)"
        />
        <v-btn
          aria-label="Rotate right"
          density="comfortable"
          :disabled="busy"
          icon="mdi-rotate-right"
          size="small"
          variant="text"
          @click="emit('rotate', 90)"
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
  import { ref, watch } from 'vue'

  // A photo viewer for dialogs: the image is fitted (contain) into a
  // fixed-height stage, and can be zoomed with the mouse wheel (towards the
  // cursor), the +/- buttons (towards the centre) or a double-click, then
  // dragged around while zoomed in.
  //
  // With `rotatable` it also offers rotate left/right buttons, and with
  // `replaceable` a button to pick another file. It changes nothing itself:
  // it emits `rotate` (-90 or 90) or `replace` (the picked File), the parent
  // saves that (usePhotoActions) and passes the re-fetched photo back as a
  // new `src`, holding `busy` while that's in flight.
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

  // A new photo starts fitted.
  watch(() => props.src, reset)

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

.zoom-stage__img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  transform-origin: 0 0;
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

.zoom-controls__level {
  min-width: 3.5em;
  text-align: center;
  font-variant-numeric: tabular-nums;
}
</style>
