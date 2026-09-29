import { defineStore } from 'pinia'

let seq = 0

export const useNotifyStore = defineStore('notify', {
  state: () => ({
    items: [],
  }),
  actions: {
    push (message, type = 'info', timeout = 4000) {
      const id = ++seq
      this.items.push({ id, message, type, timeout })
      if (timeout > 0) {
        setTimeout(() => this.remove(id), timeout)
      }
      return id
    },
    remove (id) {
      const i = this.items.findIndex(it => it.id === id)
      if (i !== -1) this.items.splice(i, 1)
    },
  },
})
