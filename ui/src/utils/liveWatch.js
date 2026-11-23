import { initialState, isControlMessage, nextState, parseFrame } from './liveView'

// The browser's end of one live view: opens the watch socket, turns what
// arrives into page state, and keeps sending nothing but control messages the
// till accepts. Reconnects after a drop (kashi restarting, a network blip)
// unless kashi ended the view on purpose. The WebSocket and timers can be
// swapped so tests drive it without a network.

export const RETRY_MS = [1000, 2000, 4000, 8000]

export function createWatch ({ url, onState, onFrame, WebSocketImpl = globalThis.WebSocket, setTimer = setTimeout, clearTimer = clearTimeout }) {
  let state = initialState()
  let socket = null
  let closed = false
  let retries = 0
  let timer = null

  function setState (next) {
    if (next !== state) {
      state = next
      onState?.(state)
    }
  }

  function final () {
    return state.status === 'ended' || state.status === 'error'
  }

  function open () {
    socket = new WebSocketImpl(url)
    const mine = socket
    mine.addEventListener('open', () => {
      retries = 0
    })
    mine.addEventListener('message', event => {
      if (mine !== socket || typeof event.data !== 'string') {
        return
      }
      const frame = parseFrame(event.data)
      if (frame.kind === 'ignored') {
        return
      }
      setState(nextState(state, frame))
      if (frame.kind === 'frame') {
        onFrame?.(frame)
      }
    })
    mine.addEventListener('close', () => {
      if (mine !== socket || closed || final()) {
        return
      }
      if (retries >= RETRY_MS.length) {
        setState({ ...state, status: 'error', message: 'Lost the connection to kashi.' })
        return
      }
      setState({ ...state, status: 'connecting' })
      timer = setTimer(open, RETRY_MS[retries++])
    })
  }

  open()

  return {
    get state () { return state },
    /** Sends a control message if this view may control; returns whether it went out. */
    send (message) {
      if (!state.canControl || !isControlMessage(message) || !socket || socket.readyState !== 1) {
        return false
      }
      socket.send(JSON.stringify(message))
      return true
    },
    close () {
      closed = true
      clearTimer(timer)
      const s = socket
      socket = null
      s?.close()
    },
  }
}
