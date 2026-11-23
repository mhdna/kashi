// Live view of a till: the plain logic behind ui/src/pages/live-view.vue, kept
// out of the component so it can be tested exhaustively. The server side is
// api/live_view.go and the till's is kashi-pos frontend/src/lib/liveView.ts;
// the message shapes are written to src/test/fixtures/live_view_protocol.json
// by a Go test and checked in liveView.test.js.

export const MAX_ACTIONS = 200

/** The watch socket's URL (the browser can't set headers, so the token rides in the query). */
export function watchUrl (wsBase, registerId, token) {
  return `${wsBase}/live/watch/${encodeURIComponent(registerId)}?token=${encodeURIComponent(token)}`
}

/**
 * Reads one text frame from kashi. kind is 'frame' for what the till
 * recorded ({ e: rrweb events, a: actions, full: true when it holds a full
 * snapshot }), or the control message's own `t`: hello, waiting, ended,
 * error, control. Anything unreadable is 'ignored'.
 */
export function parseFrame (text) {
  let m
  try {
    m = JSON.parse(text)
  } catch {
    return { kind: 'ignored' }
  }
  if (!m || typeof m !== 'object') {
    return { kind: 'ignored' }
  }
  if (typeof m.t === 'string') {
    switch (m.t) {
      case 'hello': { return { kind: 'hello', control: m.control === true }
      }
      case 'waiting': { return { kind: 'waiting' }
      }
      case 'ended': { return { kind: 'ended' }
      }
      case 'error': { return { kind: 'error', message: String(m.msg ?? 'Live view failed') }
      }
      case 'control': { return { kind: 'control', on: m.on === true }
      }
      default: { return { kind: 'ignored' }
      }
    }
  }
  if (Array.isArray(m.e)) {
    return {
      kind: 'frame',
      full: m.full === true,
      events: m.e,
      actions: Array.isArray(m.a) ? m.a : [],
    }
  }
  return { kind: 'ignored' }
}

export const initialState = () => ({ status: 'connecting', canControl: false, message: '' })

/** What the page shows after one parsed frame. status: connecting, waiting, live, ended, error. */
export function nextState (state, frame) {
  switch (frame.kind) {
    case 'hello': { return { status: 'waiting', canControl: frame.control, message: '' }
    }
    case 'waiting': { return { ...state, status: 'waiting' }
    }
    case 'frame': { return state.status === 'live' ? state : { ...state, status: 'live' }
    }
    case 'ended': { return { ...state, status: 'ended', message: 'The till stopped sharing its screen.' }
    }
    case 'error': { return { ...state, status: 'error', message: frame.message }
    }
    case 'control': { return frame.on ? state : { ...state, canControl: false }
    }
    default: { return state
    }
  }
}

/** Appends new actions to the activity list, newest last, keeping the last MAX_ACTIONS. */
export function pushActions (list, actions) {
  if (actions.length === 0) {
    return list
  }
  const next = list.concat(actions)
  return next.length > MAX_ACTIONS ? next.slice(next.length - MAX_ACTIONS) : next
}

/** The till's screen size from the Meta event (rrweb type 4) of a frame, if it has one. */
export function screenSize (events) {
  for (const e of events) {
    if (e?.type === 4 && e.data?.width > 0 && e.data?.height > 0) {
      return { width: e.data.width, height: e.data.height }
    }
  }
  return null
}

/** Scale that fits the till's screen into a box of the given width, never above 1. */
export function fitScale (boxWidth, screenWidth) {
  if (!(boxWidth > 0) || !(screenWidth > 0)) {
    return 1
  }
  return Math.min(1, boxWidth / screenWidth)
}

/** A point on the scaled replay, in the till's own pixels. */
export function pointOnTill (clientX, clientY, rect, scale) {
  const s = scale > 0 ? scale : 1
  return { x: Math.round((clientX - rect.left) / s), y: Math.round((clientY - rect.top) / s) }
}

// Keys the till understands besides a single printable character.
const NAMED_KEYS = new Set(['Enter', 'Escape', 'Tab', 'Backspace', 'Delete', 'ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'])

const validId = id => Number.isInteger(id) && id > 0

export function clickMessage (id) {
  return validId(id) ? { t: 'click', id } : null
}

export function inputMessage (id, value) {
  return validId(id) && typeof value === 'string' ? { t: 'input', id, v: value.slice(0, 2000) } : null
}

/** A key press as the till takes it, or null for modified, unknown or composed keys. */
export function keyMessage (event) {
  if (!event || event.ctrlKey || event.metaKey || event.altKey || event.isComposing) {
    return null
  }
  const key = event.key
  if (NAMED_KEYS.has(key) || (typeof key === 'string' && [...key].length === 1)) {
    return { t: 'key', key }
  }
  return null
}

/** Whether kashi lets a message from a watcher through to the till (mirrors validLiveControl in Go). */
export function isControlMessage (m) {
  return !!m && ['click', 'input', 'key'].includes(m.t)
}

// The till's clock and this one differ: events are moved onto this clock and
// played a little behind it so batches arrive in time.
export const PLAY_BEHIND_MS = 50

/**
 * Where the replayer's live clock starts: the newest event of the first batch
 * (never later than now), minus the small delay. A late batch is then already
 * in the past and plays at once instead of waiting.
 */
export function liveStartTime (events, skew, now) {
  const stamps = events.map(e => e?.timestamp).filter(t => Number.isFinite(t))
  const newest = stamps.length > 0 ? Math.max(...stamps) + skew : now
  return Math.min(newest, now) - PLAY_BEHIND_MS
}

export function clockSkew (now, firstTimestamp) {
  return Number.isFinite(firstTimestamp) ? now - firstTimestamp : 0
}

export function shiftEvent (event, skew) {
  return { ...event, timestamp: (event.timestamp ?? 0) + skew }
}

export const TYPED_MAX = 40

/** The characters shown in the "typing" overlay after a key: printable keys add, Backspace removes, other keys leave it. */
export function typedAfter (text, key) {
  if (key === 'Backspace') {
    return [...text].slice(0, -1).join('')
  }
  if (typeof key === 'string' && [...key].length === 1) {
    return [...text, key].slice(-TYPED_MAX).join('')
  }
  return text
}

/** Where to draw the click mark: the click position inside the box that holds the screen. */
export function markAt (clientX, clientY, rect) {
  return { x: Math.round(clientX - (rect?.left ?? 0)), y: Math.round(clientY - (rect?.top ?? 0)) }
}
