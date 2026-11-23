import { describe, expect, it } from 'vitest'
import fixture from '@/test/fixtures/live_view_protocol.json'
import {
  clickMessage, fitScale, initialState, inputMessage, isControlMessage, keyMessage, MAX_ACTIONS,
  nextState, parseFrame, pointOnTill, pushActions, screenSize, watchUrl,
} from './liveView'

// The fixture is written by a Go test (api/live_view_test.go) from what the
// server really sends: if the server's shapes change, these fail.
describe('what the server sends (Go fixture)', () => {
  it('reads hello with the control right', () => {
    expect(parseFrame(JSON.stringify(fixture.hello))).toEqual({ kind: 'hello', control: true })
  })
  it('reads a relayed till frame', () => {
    const f = parseFrame(JSON.stringify(fixture.frame))
    expect(f.kind).toBe('frame')
    expect(f.full).toBe(true)
    expect(f.events).toHaveLength(1)
    expect(f.actions[0].k).toBe('page')
  })
  it('reads waiting, ended, off and control off', () => {
    expect(parseFrame(JSON.stringify(fixture.waiting)).kind).toBe('waiting')
    expect(parseFrame(JSON.stringify(fixture.ended)).kind).toBe('ended')
    expect(parseFrame(JSON.stringify(fixture.off))).toEqual({ kind: 'error', message: 'Live view was turned off' })
    expect(parseFrame(JSON.stringify(fixture.controlOff))).toEqual({ kind: 'control', on: false })
  })
  it('the sessions message lists who watches which register', () => {
    expect(fixture.sessions.type).toBe('live_view_sessions')
    expect(fixture.sessions.live_sessions[0]).toEqual({ register_id: 10, branch_id: 1, watchers: ['Admin'] })
  })
  it('every control message it accepts is one the UI builds', () => {
    for (const c of fixture.controls) {
      expect(isControlMessage(c)).toBe(true)
    }
    expect(fixture.controls).toContainEqual(clickMessage(7))
    expect(fixture.controls).toContainEqual(inputMessage(7, 'abc'))
    expect(fixture.controls).toContainEqual(keyMessage({ key: 'Enter' }))
  })
  it('the till start and snapshot messages carry what the till reads', () => {
    expect(fixture.start).toEqual({ type: 'live_view_start', control: true })
    expect(fixture.snapshot).toEqual({ t: 'snapshot' })
  })
})

describe('parseFrame', () => {
  it('ignores junk', () => {
    for (const bad of ['', 'nope', 'null', '5', '"x"', '{}', '{"t":"zzz"}', '{"e":5}']) {
      expect(parseFrame(bad).kind).toBe('ignored')
    }
  })
  it('defaults missing actions and a missing full flag', () => {
    expect(parseFrame('{"e":[1]}')).toEqual({ kind: 'frame', full: false, events: [1], actions: [] })
  })
  it('a hello without control means no control', () => {
    expect(parseFrame('{"t":"hello"}')).toEqual({ kind: 'hello', control: false })
  })
  it('an error without a message still says something', () => {
    expect(parseFrame('{"t":"error"}').message).toBeTruthy()
  })
})

describe('nextState', () => {
  const run = frames => frames.reduce((s, f) => nextState(s, parseFrame(JSON.stringify(f))), initialState())

  it('goes connecting, waiting, live', () => {
    expect(initialState().status).toBe('connecting')
    expect(run([{ t: 'hello', control: true }])).toMatchObject({ status: 'waiting', canControl: true })
    expect(run([{ t: 'hello', control: true }, { e: [1], full: true }]).status).toBe('live')
  })
  it('goes back to waiting when the till drops and live again after', () => {
    const s = run([{ t: 'hello' }, { e: [1], full: true }, { t: 'waiting' }])
    expect(s.status).toBe('waiting')
    expect(nextState(s, { kind: 'frame', events: [], actions: [] }).status).toBe('live')
  })
  it('keeps the control right through waiting and live', () => {
    expect(run([{ t: 'hello', control: true }, { e: [1] }, { t: 'waiting' }]).canControl).toBe(true)
  })
  it('loses control when kashi says so, and nothing else changes', () => {
    const s = run([{ t: 'hello', control: true }, { e: [1] }, { t: 'control', on: false }])
    expect(s).toMatchObject({ status: 'live', canControl: false })
  })
  it('ended and error are final messages', () => {
    expect(run([{ t: 'hello' }, { t: 'ended' }])).toMatchObject({ status: 'ended' })
    expect(run([{ t: 'error', msg: 'too many tills' }])).toMatchObject({ status: 'error', message: 'too many tills' })
  })
  it('ignored frames change nothing', () => {
    const s = initialState()
    expect(nextState(s, { kind: 'ignored' })).toBe(s)
  })
})

describe('pushActions', () => {
  it('appends in order and returns the same list for nothing new', () => {
    const list = [{ l: 'a' }]
    expect(pushActions(list, [])).toBe(list)
    expect(pushActions(list, [{ l: 'b' }, { l: 'c' }]).map(a => a.l)).toEqual(['a', 'b', 'c'])
  })
  it('keeps only the latest MAX_ACTIONS, whatever the batch sizes', () => {
    let list = []
    let n = 0
    for (let i = 0; i < 100; i++) {
      const batch = Array.from({ length: (i * 7) % 23 }, () => ({ l: String(n++) }))
      list = pushActions(list, batch)
      expect(list.length).toBeLessThanOrEqual(MAX_ACTIONS)
      if (list.length > 0) {
        expect(list.at(-1).l).toBe(String(n - 1))
      }
    }
    expect(list.map(a => Number(a.l))).toEqual(Array.from({ length: list.length }, (_, i) => n - list.length + i))
  })
})

describe('screen size and scale', () => {
  it('finds the Meta event', () => {
    expect(screenSize([{ type: 2 }, { type: 4, data: { width: 1366, height: 768 } }])).toEqual({ width: 1366, height: 768 })
  })
  it('is null without a usable Meta event', () => {
    expect(screenSize([])).toBeNull()
    expect(screenSize([{ type: 4, data: { width: 0, height: 5 } }, null])).toBeNull()
  })
  it('fits the width and never scales up', () => {
    expect(fitScale(683, 1366)).toBe(0.5)
    expect(fitScale(2000, 1366)).toBe(1)
    expect(fitScale(0, 1366)).toBe(1)
    expect(fitScale(500, 0)).toBe(1)
  })
  it('maps a click on the scaled replay back to the till', () => {
    const rect = { left: 100, top: 50 }
    expect(pointOnTill(100, 50, rect, 0.5)).toEqual({ x: 0, y: 0 })
    expect(pointOnTill(400, 250, rect, 0.5)).toEqual({ x: 600, y: 400 })
    expect(pointOnTill(400, 250, rect, 1)).toEqual({ x: 300, y: 200 })
    expect(pointOnTill(400, 250, rect, 0)).toEqual({ x: 300, y: 200 })
  })
})

describe('control messages', () => {
  it('click needs a real node id', () => {
    expect(clickMessage(3)).toEqual({ t: 'click', id: 3 })
    for (const bad of [0, -1, 1.5, '3', null, undefined, Number.NaN]) {
      expect(clickMessage(bad)).toBeNull()
    }
  })
  it('input needs an id and a string, and is bounded', () => {
    expect(inputMessage(3, 'x')).toEqual({ t: 'input', id: 3, v: 'x' })
    expect(inputMessage(3, '')).toEqual({ t: 'input', id: 3, v: '' })
    expect(inputMessage(0, 'x')).toBeNull()
    expect(inputMessage(3, 5)).toBeNull()
    expect(inputMessage(3, 'a'.repeat(5000)).v).toHaveLength(2000)
  })
  it('passes named and printable keys, drops shortcuts and composition', () => {
    for (const key of ['Enter', 'Escape', 'Tab', 'Backspace', 'ArrowDown', 'a', 'Z', '7', ' ', '€', '😀']) {
      expect(keyMessage({ key })).toEqual({ t: 'key', key })
    }
    for (const key of ['Shift', 'Control', 'F5', 'Dead', 'Process', 'CapsLock', '']) {
      expect(keyMessage({ key })).toBeNull()
    }
    expect(keyMessage({ key: 'c', ctrlKey: true })).toBeNull()
    expect(keyMessage({ key: 'c', metaKey: true })).toBeNull()
    expect(keyMessage({ key: 'a', altKey: true })).toBeNull()
    expect(keyMessage({ key: 'a', isComposing: true })).toBeNull()
    expect(keyMessage(null)).toBeNull()
  })
  it('everything the UI builds is accepted by the server filter', () => {
    const built = [clickMessage(1), inputMessage(1, 'x'), keyMessage({ key: 'Enter' })]
    for (const m of built) {
      expect(isControlMessage(m)).toBe(true)
    }
    expect(isControlMessage({ t: 'snapshot' })).toBe(false)
    expect(isControlMessage(null)).toBe(false)
  })
})

describe('watchUrl', () => {
  it('builds the socket address and escapes the token', () => {
    expect(watchUrl('wss://x.test', 10, 'a b+c')).toBe('wss://x.test/live/watch/10?token=a%20b%2Bc')
  })
})
