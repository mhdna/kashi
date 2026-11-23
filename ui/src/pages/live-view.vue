<template>
  <div class="page-root">
    <v-card class="live-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-btn aria-label="Back to branches" icon="mdi-arrow-left" to="/branches" variant="text" />
        <v-icon icon="mdi-monitor" />
        <span data-test="live-title">Live view: {{ title }}</span>
        <v-spacer />
        <v-switch
          v-if="state.canControl && state.status === 'live'"
          v-model="controlOn"
          color="warning"
          data-test="take-control"
          density="compact"
          hide-details
          inset
          label="Take control"
        />
      </v-card-title>
      <v-divider />

      <div class="pa-4">
        <v-alert
          v-if="state.status === 'ended' || state.status === 'error'"
          data-test="live-message"
          :type="state.status === 'error' ? 'error' : 'info'"
          variant="tonal"
        >
          {{ state.message }}
        </v-alert>
        <div
          v-else-if="state.status !== 'live'"
          class="text-medium-emphasis d-flex align-center ga-3"
          data-test="live-waiting"
        >
          <v-progress-circular indeterminate size="20" width="2" />
          {{ state.status === 'connecting' ? 'Connecting to kashi...' : 'Waiting for the till to share its screen...' }}
        </div>

        <div v-show="state.status === 'live'" ref="box" class="live-box" :style="{ height: stageHeight + 'px' }">
          <div ref="stage" class="live-stage" :style="{ transform: `scale(${scale})`, width: size.width + 'px', height: size.height + 'px' }" />
          <div
            v-if="controlOn && state.canControl"
            class="live-overlay"
            data-test="control-overlay"
            tabindex="0"
            @click="onClick"
            @keydown="onKey"
            @paste="onPaste"
          />
        </div>

        <div v-if="actions.length > 0" class="mt-4" data-test="live-actions">
          <div class="text-subtitle-2 mb-1">What they did</div>
          <div v-for="(a, i) in recent" :key="i" class="text-body-2 text-medium-emphasis">
            {{ a.l }}
          </div>
        </div>
      </div>
    </v-card>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
  import { useRoute } from 'vue-router'
  import { WS_BASE } from '@/config'
  import { useAuthStore } from '@/stores/auth'
  import {
    clickMessage, clockSkew, fitScale, initialState, inputMessage, keyMessage, liveStartTime,
    pointOnTill, pushActions, screenSize, shiftEvent, watchUrl,
  } from '@/utils/liveView'
  import { createWatch } from '@/utils/liveWatch'

  // The till's screen rebuilt here with rrweb, never a video: the till sends
  // its page structure once, then changes. rrweb is loaded only on this page.
  const route = useRoute()
  const auth = useAuthStore()
  const registerId = route.params.registerId
  const title = computed(() => route.query.name || `Till ${registerId}`)

  const state = ref(initialState())
  const actions = ref([])
  const recent = computed(() => actions.value.slice(-15).toReversed())
  const controlOn = ref(false)
  const box = ref(null)
  const stage = ref(null)
  const size = ref({ width: 1366, height: 768 })
  const scale = ref(1)
  const stageHeight = ref(0)

  let rrweb = null
  let replayer = null
  let skew = 0
  let queue = Promise.resolve()
  let watch = null
  let lastClicked = 0
  let observer = null

  function fit () {
    if (!box.value) {
      return
    }
    scale.value = fitScale(box.value.clientWidth, size.value.width)
    stageHeight.value = Math.round(size.value.height * scale.value)
  }

  async function loadRrweb () {
    if (!rrweb) {
      rrweb = await import('rrweb')
      await import('rrweb/dist/style.css')
    }
  }

  function startReplayer (events) {
    replayer?.destroy()
    stage.value.innerHTML = ''
    const next = new rrweb.Replayer([], {
      root: stage.value,
      liveMode: true,
      mouseTail: false,
      showWarning: false,
      UNSAFE_replayCanvas: false,
    })
    next.on('resize', d => {
      size.value = { width: d.width, height: d.height }
      fit()
    })
    skew = clockSkew(Date.now(), events[0].timestamp)
    next.startLive(liveStartTime(events, skew, Date.now()))
    replayer = next
  }

  async function handle (frame) {
    actions.value = pushActions(actions.value, frame.actions)
    if (frame.events.length === 0) {
      return
    }
    await loadRrweb()
    await nextTick()
    // Only a full snapshot can start (or restart) the picture.
    if (frame.full) {
      startReplayer(frame.events)
    } else if (!replayer) {
      return
    }
    const meta = screenSize(frame.events)
    if (meta) {
      size.value = meta
    }
    for (const e of frame.events) {
      replayer.addEvent(shiftEvent(e, skew))
    }
    fit()
  }

  // Frames are handled one at a time, in order, though loading rrweb is async.
  function onFrame (frame) {
    queue = queue.then(() => handle(frame)).catch(error => console.error('live view', error))
  }

  function tillElementAt (event) {
    const rect = stage.value.getBoundingClientRect()
    const { x, y } = pointOnTill(event.clientX, event.clientY, rect, scale.value)
    const doc = stage.value.querySelector('iframe')?.contentDocument
    return doc?.elementFromPoint(x, y) ?? null
  }

  function onClick (event) {
    const element = tillElementAt(event)
    const id = element ? replayer?.getMirror().getId(element) : -1
    const message = clickMessage(id)
    if (message && watch.send(message)) {
      lastClicked = id
    }
  }

  function onKey (event) {
    const message = keyMessage(event)
    if (message) {
      event.preventDefault()
      watch.send(message)
    }
  }

  function onPaste (event) {
    const text = event.clipboardData?.getData('text') ?? ''
    const node = lastClicked ? replayer?.getMirror().getNode(lastClicked) : null
    const message = inputMessage(lastClicked, `${node?.value ?? ''}${text}`)
    if (message) {
      event.preventDefault()
      watch.send(message)
    }
  }

  onMounted(() => {
    watch = createWatch({
      url: watchUrl(WS_BASE, registerId, auth.token),
      onState: s => {
        state.value = s
        if (!s.canControl) {
          controlOn.value = false
        }
      },
      onFrame,
    })
    if (typeof ResizeObserver !== 'undefined' && box.value) {
      observer = new ResizeObserver(fit)
      observer.observe(box.value)
    }
  })

  onBeforeUnmount(() => {
    watch?.close()
    observer?.disconnect()
    replayer?.destroy()
    replayer = null
  })

  // Exposed for the tests, which drive the page without a real replayer.
  defineExpose({ state, size, scale })
</script>

<style scoped>
.live-box {
  position: relative;
  width: 100%;
  overflow: hidden;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 8px;
}
.live-stage {
  transform-origin: top left;
}
.live-overlay {
  position: absolute;
  inset: 0;
  cursor: pointer;
  outline: 2px solid rgb(var(--v-theme-warning));
  outline-offset: -2px;
}
</style>
