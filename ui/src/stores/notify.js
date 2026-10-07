import { defineStore } from 'pinia'

let seq = 0
// id -> pending auto-dismiss timer, so a repeat can restart it.
const timers = new Map()

export const useNotifyStore = defineStore('notify', {
  state: () => ({
    items: [],
  }),
  actions: {
    push (message, type = 'info', timeout = 4000) {
      // Several requests failing together (e.g. every widget on a page when
      // the server goes down) all report the same text -- show it once and
      // just keep that one on screen a little longer.
      const existing = this.items.find(it => it.message === message && it.type === type)
      const id = existing ? existing.id : ++seq
      if (!existing) {
        this.items.push({ id, message, type, timeout })
      }
      clearTimeout(timers.get(id))
      timers.delete(id)
      if (timeout > 0) {
        timers.set(id, setTimeout(() => this.remove(id), timeout))
      }
      return id
    },
    remove (id) {
      clearTimeout(timers.get(id))
      timers.delete(id)
      const i = this.items.findIndex(it => it.id === id)
      if (i !== -1) this.items.splice(i, 1)
    },
  },
})
