import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { vuetify } from '@/test/vuetify'

class FakeSocket {
  static all = []
  constructor (url) {
    this.url = url
    this.readyState = 1
    this.sent = []
    this.handlers = {}
    FakeSocket.all.push(this)
  }
  addEventListener (name, fn) { (this.handlers[name] ||= []).push(fn) }
  send (data) { this.sent.push(JSON.parse(data)) }
  close () { this.readyState = 3 }
  receive (obj) { for (const fn of this.handlers.message || []) {fn({ data: JSON.stringify(obj) })} }
}

const replayers = []
class FakeReplayer {
  constructor (events, config) {
    this.config = config
    this.added = []
    this.destroyed = false
    this.mirror = { getId: el => el?.rrId ?? -1, getNode: id => ({ value: 'ab' + id }) }
    replayers.push(this)
  }
  on () {}
  startLive (t) { this.baseline = t }
  addEvent (e) { this.added.push(e) }
  getMirror () { return this.mirror }
  destroy () { this.destroyed = true }
}
vi.mock('rrweb', () => ({ Replayer: FakeReplayer }))
vi.mock('rrweb/dist/style.css', () => ({}))
vi.mock('@/config', () => ({ WS_BASE: 'wss://k.test/api' }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ token: 'tok' }) }))
vi.mock('vue-router', () => ({ useRoute: () => ({ params: { registerId: '10' }, query: { name: 'Centro CT1' } }) }))

const { default: LiveView } = await import('./live-view.vue')

let wrapper = null
const socket = () => FakeSocket.all.at(-1)
beforeEach(() => {
  FakeSocket.all = []
  replayers.length = 0
  vi.stubGlobal('WebSocket', FakeSocket)
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  vi.unstubAllGlobals()
})

function show () {
  wrapper = mount(LiveView, {
    global: { plugins: [vuetify()], stubs: { RouterLink: { template: '<a><slot /></a>' } } },
    attachTo: document.body,
  })
  return wrapper
}
const meta = { type: 4, timestamp: 1000, data: { width: 1366, height: 768 } }
const snap = { type: 2, timestamp: 1001, data: {} }
const full = (extra = {}) => ({ full: true, e: [meta, snap], a: [], ...extra })

describe('live-view page', () => {
  it('opens the watch socket for the register with the token and names the till', () => {
    show()
    expect(socket().url).toBe('wss://k.test/api/live/watch/10?token=tok')
    expect(wrapper.find('[data-test="live-title"]').text()).toBe('Live view: Centro CT1')
    expect(wrapper.find('[data-test="live-waiting"]').text()).toContain('Connecting')
  })

  it('waits for the till, then shows the screen once frames arrive', async () => {
    show()
    socket().receive({ t: 'hello', control: false })
    await flushPromises()
    expect(wrapper.find('[data-test="live-waiting"]').text()).toContain('Waiting for the till')
    expect(replayers).toHaveLength(0)

    socket().receive(full())
    await flushPromises()
    expect(wrapper.find('[data-test="live-waiting"]').exists()).toBe(false)
    expect(replayers).toHaveLength(1)
    expect(replayers[0].config).toMatchObject({ liveMode: true, UNSAFE_replayCanvas: false })
    expect(replayers[0].added.map(e => e.type)).toEqual([4, 2])
    expect(wrapper.vm.size).toEqual({ width: 1366, height: 768 })
  })

  it('a new full snapshot replaces the replayer, increments feed the current one, increments before any snapshot are ignored', async () => {
    show()
    socket().receive({ t: 'hello' })
    socket().receive({ e: [{ type: 3, timestamp: 5 }] })
    await flushPromises()
    expect(replayers).toHaveLength(0)

    socket().receive(full())
    socket().receive({ e: [{ type: 3, timestamp: 1100 }] })
    await flushPromises()
    expect(replayers[0].added).toHaveLength(3)

    socket().receive(full())
    await flushPromises()
    expect(replayers).toHaveLength(2)
    expect(replayers[0].destroyed).toBe(true)
    expect(replayers[1].added).toHaveLength(2)
  })

  it('keeps frames in order', async () => {
    show()
    socket().receive({ t: 'hello' })
    socket().receive(full())
    for (let i = 0; i < 10; i++) {
      socket().receive({ e: [{ type: 3, timestamp: 2000 + i, data: { n: i } }] })
    }
    await flushPromises()
    expect(replayers[0].added.slice(2).map(e => e.data.n)).toEqual([0, 1, 2, 3, 4, 5, 6, 7, 8, 9])
  })

  it('lists what they did, newest first, and keeps at most 15 shown', async () => {
    show()
    socket().receive({ t: 'hello' })
    socket().receive(full({ a: Array.from({ length: 20 }, (_, i) => ({ k: 'click', l: `Clicked ${i}`, at: i })) }))
    await flushPromises()
    const lines = wrapper.findAll('[data-test="live-actions"] .text-body-2').map(w => w.text())
    expect(lines).toHaveLength(15)
    expect(lines[0]).toBe('Clicked 19')
  })

  it('says so when the till stops, and when kashi refuses', async () => {
    show()
    socket().receive({ t: 'hello' })
    socket().receive({ t: 'ended' })
    await flushPromises()
    expect(wrapper.find('[data-test="live-message"]').text()).toContain('stopped sharing')
    wrapper.unmount()

    show()
    socket().receive({ t: 'error', msg: 'too many tills are being watched' })
    await flushPromises()
    expect(wrapper.find('[data-test="live-message"]').text()).toBe('too many tills are being watched')
  })

  it('offers control only when the server says this viewer may', async () => {
    show()
    socket().receive({ t: 'hello', control: false })
    socket().receive(full())
    await flushPromises()
    expect(wrapper.find('[data-test="take-control"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="control-overlay"]').exists()).toBe(false)
    wrapper.unmount()

    show()
    socket().receive({ t: 'hello', control: true })
    socket().receive(full())
    await flushPromises()
    expect(wrapper.find('[data-test="take-control"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="control-overlay"]').exists()).toBe(false) // not until switched on
  })

  describe('controlling', () => {
    async function controlled () {
      show()
      socket().receive({ t: 'hello', control: true })
      socket().receive(full())
      await flushPromises()
      await wrapper.find('[data-test="take-control"] input').setValue(true)
      await flushPromises()
    }

    it('a click on the screen is sent as a click on the till element under it', async () => {
      await controlled()
      const overlay = wrapper.find('[data-test="control-overlay"]')
      expect(overlay.exists()).toBe(true)
      const stage = wrapper.find('.live-stage').element
      stage.getBoundingClientRect = () => ({ left: 10, top: 20 })
      const iframe = document.createElement('iframe')
      stage.append(iframe)
      const target = { rrId: 42 }
      let asked = null
      Object.defineProperty(iframe, 'contentDocument', { value: { elementFromPoint: (x, y) => { asked = { x, y }; return target } } })
      await overlay.trigger('click', { clientX: 110, clientY: 120 })
      expect(asked).toEqual({ x: 100 / wrapper.vm.scale, y: 100 / wrapper.vm.scale })
      expect(socket().sent).toEqual([{ t: 'click', id: 42, ts: expect.any(Number) }])
    })

    it('a click on something that is not on the till sends nothing', async () => {
      await controlled()
      const stage = wrapper.find('.live-stage').element
      stage.getBoundingClientRect = () => ({ left: 0, top: 0 })
      const iframe = document.createElement('iframe')
      stage.append(iframe)
      Object.defineProperty(iframe, 'contentDocument', { value: { elementFromPoint: () => ({}) } }) // no node id
      await wrapper.find('[data-test="control-overlay"]').trigger('click', { clientX: 5, clientY: 5 })
      expect(socket().sent).toEqual([])
    })

    it('keys go through, shortcuts do not', async () => {
      await controlled()
      const overlay = wrapper.find('[data-test="control-overlay"]')
      await overlay.trigger('keydown', { key: 'Enter' })
      await overlay.trigger('keydown', { key: '5' })
      await overlay.trigger('keydown', { key: 'v', ctrlKey: true })
      await overlay.trigger('keydown', { key: 'Shift' })
      expect(socket().sent).toEqual([{ t: 'key', key: 'Enter', ts: expect.any(Number) }, { t: 'key', key: '5', ts: expect.any(Number) }])
    })

    it('shows a click mark and the typed characters at once, and clears them on the till\'s next frame', async () => {
      await controlled()
      const overlay = wrapper.find('[data-test="control-overlay"]')
      const stage = wrapper.find('.live-stage').element
      stage.getBoundingClientRect = () => ({ left: 0, top: 0 })
      const iframe = document.createElement('iframe')
      stage.append(iframe)
      Object.defineProperty(iframe, 'contentDocument', { value: { elementFromPoint: () => ({ rrId: 7 }) } })
      expect(wrapper.find('[data-test="click-mark"]').exists()).toBe(false)

      await overlay.trigger('click', { clientX: 30, clientY: 40 })
      expect(wrapper.find('[data-test="click-mark"]').attributes('style')).toContain('left: 30px')
      await overlay.trigger('keydown', { key: '1' })
      await overlay.trigger('keydown', { key: '2' })
      await overlay.trigger('keydown', { key: '3' })
      await overlay.trigger('keydown', { key: 'Backspace' })
      expect(wrapper.find('[data-test="typed-overlay"]').text()).toBe('12')

      socket().receive({ e: [{ type: 3, timestamp: 1002, data: {} }], a: [] })
      await flushPromises()
      expect(wrapper.find('[data-test="click-mark"]').exists()).toBe(false)
      expect(wrapper.find('[data-test="typed-overlay"]').exists()).toBe(false)
    })

    it('stamps controls with its clock and shows the delay from the till\'s ack', async () => {
      vi.useFakeTimers({ toFake: ['Date'] })
      try {
        vi.setSystemTime(50_000)
        await controlled()
        expect(wrapper.find('[data-test="live-delay"]').exists()).toBe(false)
        await wrapper.find('[data-test="control-overlay"]').trigger('keydown', { key: '5' })
        expect(socket().sent).toEqual([{ t: 'key', key: '5', ts: 50_000 }])

        vi.setSystemTime(50_180)
        socket().receive({ e: [], a: [], ack: 50_000 })
        await flushPromises()
        expect(wrapper.find('[data-test="live-delay"]').text()).toBe('Delay: 180 ms')

        // A frame without an ack keeps the last figure; a newer ack replaces it.
        socket().receive({ e: [], a: [] })
        await flushPromises()
        expect(wrapper.find('[data-test="live-delay"]').text()).toBe('Delay: 180 ms')
        vi.setSystemTime(50_400)
        socket().receive({ e: [], a: [], ack: 50_350 })
        await flushPromises()
        expect(wrapper.find('[data-test="live-delay"]').text()).toBe('Delay: 50 ms')
      } finally {
        vi.useRealTimers()
      }
    })

    it('a click on something that is not on the till leaves no mark', async () => {
      await controlled()
      const stage = wrapper.find('.live-stage').element
      stage.getBoundingClientRect = () => ({ left: 0, top: 0 })
      const iframe = document.createElement('iframe')
      stage.append(iframe)
      Object.defineProperty(iframe, 'contentDocument', { value: { elementFromPoint: () => ({}) } })
      await wrapper.find('[data-test="control-overlay"]').trigger('click', { clientX: 5, clientY: 5 })
      expect(wrapper.find('[data-test="click-mark"]').exists()).toBe(false)
    })

    it('control going away takes the overlay and the switch away at once', async () => {
      await controlled()
      socket().receive({ t: 'control', on: false })
      await flushPromises()
      expect(wrapper.find('[data-test="control-overlay"]').exists()).toBe(false)
      expect(wrapper.find('[data-test="take-control"]').exists()).toBe(false)
      expect(socket().sent).toEqual([])
    })
  })

  it('closes the socket and the replayer when the page is left', async () => {
    show()
    socket().receive({ t: 'hello' })
    socket().receive(full())
    await flushPromises()
    const s = socket()
    wrapper.unmount()
    wrapper = null
    expect(s.readyState).toBe(3)
    expect(replayers[0].destroyed).toBe(true)
  })
})
