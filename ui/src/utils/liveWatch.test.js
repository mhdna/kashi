import { describe, expect, it } from 'vitest'
import { clockSkew, liveStartTime, PLAY_BEHIND_MS, shiftEvent } from './liveView'
import { createWatch, RETRY_MS } from './liveWatch'

class FakeSocket {
  static all = []
  constructor (url) {
    this.url = url
    this.readyState = 0
    this.sent = []
    this.handlers = {}
    FakeSocket.all.push(this)
  }
  addEventListener (name, fn) { (this.handlers[name] ||= []).push(fn) }
  send (data) { this.sent.push(JSON.parse(data)) }
  close () { this.readyState = 3 }
  emit (name, data) { for (const fn of this.handlers[name] || []) {fn(data)} }
  open () { this.readyState = 1; this.emit('open') }
  receive (obj) { this.emit('message', { data: typeof obj === 'string' ? obj : JSON.stringify(obj) }) }
  drop () { this.readyState = 3; this.emit('close') }
}

function rig () {
  FakeSocket.all = []
  const timers = []
  const states = []
  const frames = []
  const watch = createWatch({
    url: 'ws://k/live/watch/10?token=t',
    onState: s => states.push(s),
    onFrame: f => frames.push(f),
    WebSocketImpl: FakeSocket,
    setTimer: (fn, ms) => { timers.push({ fn, ms }); return timers.length },
    clearTimer: () => {},
  })
  return { watch, timers, states, frames, socket: () => FakeSocket.all.at(-1) }
}

describe('createWatch', () => {
  it('opens the url and walks connecting, waiting, live', () => {
    const r = rig()
    expect(r.socket().url).toBe('ws://k/live/watch/10?token=t')
    r.socket().open()
    r.socket().receive({ t: 'hello', control: true })
    expect(r.watch.state).toMatchObject({ status: 'waiting', canControl: true })
    r.socket().receive({ full: true, e: [{ type: 4 }], a: [] })
    expect(r.watch.state.status).toBe('live')
    expect(r.frames).toHaveLength(1)
    expect(r.frames[0].full).toBe(true)
  })

  it('ignores unreadable frames and binary data', () => {
    const r = rig()
    r.socket().receive('garbage')
    r.socket().receive({ t: 'weird' })
    r.socket().emit('message', { data: new ArrayBuffer(2) })
    expect(r.states).toHaveLength(0)
    expect(r.frames).toHaveLength(0)
  })

  it('sends control only when the view may control and the message is one the till takes', () => {
    const r = rig()
    r.socket().open()
    r.socket().receive({ t: 'hello', control: false })
    expect(r.watch.send({ t: 'click', id: 1 })).toBe(false)
    r.socket().receive({ t: 'hello', control: true })
    expect(r.watch.send({ t: 'click', id: 1 })).toBe(true)
    expect(r.watch.send({ t: 'snapshot' })).toBe(false)
    expect(r.watch.send(null)).toBe(false)
    expect(r.socket().sent).toEqual([{ t: 'click', id: 1 }])
    // Control taken away mid-session.
    r.socket().receive({ t: 'control', on: false })
    expect(r.watch.send({ t: 'click', id: 2 })).toBe(false)
  })

  it('does not send on a socket that is not open', () => {
    const r = rig()
    r.socket().receive({ t: 'hello', control: true })
    expect(r.watch.send({ t: 'click', id: 1 })).toBe(false)
  })

  it('reconnects after a drop with growing waits, then gives up with a message', () => {
    const r = rig()
    r.socket().open()
    r.socket().receive({ t: 'hello', control: false })
    for (const RETRY_M of RETRY_MS) {
      r.socket().drop()
      expect(r.watch.state.status).toBe('connecting')
      expect(r.timers.at(-1).ms).toBe(RETRY_M)
      r.timers.at(-1).fn()
    }
    r.socket().drop()
    expect(r.watch.state).toMatchObject({ status: 'error', message: 'Lost the connection to kashi.' })
    expect(FakeSocket.all).toHaveLength(RETRY_MS.length + 1)
  })

  it('a successful open resets the retries', () => {
    const r = rig()
    r.socket().drop()
    r.timers.at(-1).fn()
    r.socket().open()
    r.socket().drop()
    expect(r.timers.at(-1).ms).toBe(RETRY_MS[0])
  })

  it('does not reconnect after kashi ended or refused the view', () => {
    for (const final of [{ t: 'ended' }, { t: 'error', msg: 'too many tills' }]) {
      const r = rig()
      r.socket().open()
      r.socket().receive(final)
      r.socket().drop()
      expect(r.timers).toHaveLength(0)
      expect(FakeSocket.all).toHaveLength(1)
    }
  })

  it('does not reconnect after close(), and a late message from the old socket is ignored', () => {
    const r = rig()
    const old = r.socket()
    r.watch.close()
    old.drop()
    old.receive({ t: 'hello', control: true })
    expect(r.timers).toHaveLength(0)
    expect(r.states).toHaveLength(0)
  })
})

describe('clock skew', () => {
  it('moves events onto this clock', () => {
    const skew = clockSkew(10_000, 4000)
    expect(skew).toBe(6000)
    expect(shiftEvent({ type: 3, timestamp: 4500 }, skew)).toEqual({ type: 3, timestamp: 10_500 })
    expect(shiftEvent({ type: 3 }, 5).timestamp).toBe(5)
    expect(clockSkew(10, undefined)).toBe(0)
  })
})

describe('live clock start', () => {
  it('plays only a little behind', () => {
    expect(PLAY_BEHIND_MS).toBeLessThanOrEqual(50)
  })

  it('starts from the newest event, never later than now', () => {
    const skew = 1000
    expect(liveStartTime([{ timestamp: 100 }, { timestamp: 400 }], skew, 5000)).toBe(1400 - PLAY_BEHIND_MS)
    expect(liveStartTime([{ timestamp: 9000 }], skew, 5000)).toBe(5000 - PLAY_BEHIND_MS)
  })

  it('ignores events without a timestamp and falls back to now', () => {
    expect(liveStartTime([{}, { timestamp: 300 }], 0, 5000)).toBe(300 - PLAY_BEHIND_MS)
    expect(liveStartTime([{}], 0, 5000)).toBe(5000 - PLAY_BEHIND_MS)
    expect(liveStartTime([], 0, 5000)).toBe(5000 - PLAY_BEHIND_MS)
  })
})
