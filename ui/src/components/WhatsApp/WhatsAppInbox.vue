<template>
  <div class="wi" :class="{ 'wi--dark': theme.current.value.dark }">
    <!-- Customers, the most recently active first. -->
    <div class="wi__list" :class="{ 'wi__list--hidden': !!phone }">
      <div class="pa-3 flex-shrink-0">
        <v-text-field
          v-model="search"
          clearable
          density="compact"
          hide-details
          placeholder="Name, phone or what they wrote…"
          prepend-inner-icon="mdi-magnify"
          variant="outlined"
        />
      </div>
      <div class="wi__scroll">
        <div v-if="loadingChats && chats.length === 0" class="d-flex justify-center pa-6">
          <v-progress-circular indeterminate />
        </div>
        <div v-else-if="chats.length === 0" class="pa-6 text-body-2 text-medium-emphasis text-center">
          {{ search ? 'No conversation matches.' : 'No conversations yet. Messages show up here as they come in on your numbers.' }}
        </div>
        <button
          v-for="c in chats"
          :key="c.phone"
          class="wi__chat"
          :class="{ 'wi__chat--open': c.phone === phone }"
          type="button"
          @click="open(c.phone)"
        >
          <v-avatar color="grey-lighten-1" size="40">
            <v-icon color="white" icon="mdi-account" />
          </v-avatar>
          <div class="flex-grow-1 min-w-0">
            <div class="d-flex ga-2">
              <span class="font-weight-medium text-truncate flex-grow-1 wi__title">{{ title(c) }}</span>
              <span class="text-caption text-medium-emphasis flex-shrink-0">{{ when(c.last_at) }}</span>
            </div>
            <div class="text-body-2 text-medium-emphasis text-truncate wi__preview">
              <span v-if="c.last_from_me">You: </span>{{ preview(c) }}
            </div>
          </div>
        </button>
        <div v-if="moreChats" class="pa-3 text-center">
          <v-btn :loading="loadingChats" size="small" variant="text" @click="loadChats(true)">Show more</v-btn>
        </div>
      </div>
    </div>

    <!-- The open conversation. -->
    <div class="wi__thread" :class="{ 'wi__thread--hidden': !phone }">
      <template v-if="phone">
        <div class="wi__head d-flex align-center ga-3 px-3 py-2">
          <v-btn
            class="wi__back"
            icon="mdi-arrow-left"
            size="small"
            variant="text"
            @click="phone = ''"
          />
          <v-avatar color="grey-lighten-1" size="38">
            <v-icon color="white" icon="mdi-account" />
          </v-avatar>
          <div class="min-w-0">
            <div class="font-weight-medium text-truncate">{{ openTitle }}</div>
            <div class="text-caption wi__sub text-truncate">{{ formatPhone(phone) }}</div>
          </div>
        </div>
        <div ref="chatEl" class="wi__chatbox pa-3">
          <div v-if="moreMessages" class="text-center mb-2">
            <v-btn :loading="loadingMessages" size="small" variant="tonal" @click="loadEarlier">Earlier messages</v-btn>
          </div>
          <div v-if="loadingMessages && messages.length === 0" class="d-flex justify-center pa-6">
            <v-progress-circular indeterminate />
          </div>
          <template v-for="(m, i) in messages" :key="m.id">
            <div v-if="dayLabel(i)" class="wi__day">{{ dayLabel(i) }}</div>
            <div class="wi__row" :class="{ 'wi__row--us': m.from_me }">
              <div class="wi__bubble" :class="m.from_me ? 'wi__bubble--us' : 'wi__bubble--client'">
                <template v-if="m.media_url">
                  <audio
                    v-if="mediaKind(m) === 'audio'"
                    class="wi__audio"
                    controls
                    preload="metadata"
                    :src="media[m.media_url]"
                  />
                  <img v-else-if="mediaKind(m) === 'image' && media[m.media_url]" alt="" class="wi__media" :src="media[m.media_url]">
                  <video
                    v-else-if="mediaKind(m) === 'video'"
                    class="wi__media"
                    controls
                    preload="metadata"
                    :src="media[m.media_url]"
                  />
                  <a v-else-if="media[m.media_url]" class="wi__text" download :href="media[m.media_url]">
                    <v-icon icon="mdi-paperclip" size="16" /> {{ KIND_LABEL[m.kind] ?? 'File' }}
                  </a>
                  <div v-else class="wi__text text-medium-emphasis">
                    <v-icon :icon="KIND_ICON[m.kind] ?? 'mdi-paperclip'" size="16" /> {{ mediaError[m.media_url] || KIND_LABEL[m.kind] || 'File' }}
                  </div>
                </template>
                <div v-else-if="!m.text" class="wi__text text-medium-emphasis">
                  <v-icon :icon="KIND_ICON[m.kind] ?? 'mdi-paperclip'" size="16" /> {{ KIND_LABEL[m.kind] ?? m.kind }}
                </div>
                <!-- eslint-disable-next-line vue/no-v-html -- escaped in format() -->
                <div v-if="m.text" class="wi__text" :class="{ 'mt-1': m.media_url }" v-html="format(m.text)" />
                <div class="wi__meta">{{ time(m.at) }}</div>
              </div>
            </div>
          </template>
        </div>
      </template>
      <div v-else class="wi__empty text-medium-emphasis">Pick a conversation.</div>
    </div>
  </div>
</template>

<script setup>
  import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
  import { useTheme } from 'vuetify'
  import { authFetch } from '@/composables/useApi'
  import { useNotify } from '@/composables/useNotify'
  import { useWhatsApp } from '@/composables/useWhatsApp'
  import { API_BASE } from '@/config'
  import { formatPhone } from '@/utils/whatsapp'

  // Every message on our own WhatsApp numbers, both ways, one thread per
  // customer, including what staff type on the phone (api/whatsapp_chat.go).

  const props = defineProps({
    // Bumped when something new may have come in: reloads the open lists.
    refreshKey: { type: Number, default: 0 },
  })

  const notify = useNotify()
  const theme = useTheme()
  const { listChats, getChat, clientsByPhones } = useWhatsApp()

  const KIND_ICON = { image: 'mdi-camera', video: 'mdi-video', audio: 'mdi-microphone', sticker: 'mdi-sticker-emoji', document: 'mdi-file-document-outline' }
  const KIND_LABEL = { image: 'Photo', video: 'Video', audio: 'Voice message', sticker: 'Sticker', document: 'Document', other: 'Message' }

  // ---- customers ----
  const search = ref('')
  const chats = ref([])
  const moreChats = ref(false)
  const loadingChats = ref(false)
  const chatPage = ref(1)
  // Names from kashi's clients, for phones whose owner hasn't set a WhatsApp name.
  const clientNames = ref({})

  async function loadChats (next = false) {
    // A refresh reloads every page already shown, so the list doesn't jump back.
    const pages = next ? 1 : chatPage.value
    const page = next ? chatPage.value + 1 : 1
    loadingChats.value = true
    try {
      const wanted = next ? [page] : Array.from({ length: pages }, (_, i) => i + 1)
      let rows = next ? chats.value : []
      let more = false
      for (const p of wanted) {
        const data = await listChats({ search: search.value ?? '', page: p, page_size: 50 })
        rows = [...rows, ...(data.chats ?? [])]
        more = data.more
      }
      chats.value = rows
      moreChats.value = more
      chatPage.value = next ? page : pages
      loadClientNames()
    } catch (error) {
      notify.error(error.message)
    } finally {
      loadingChats.value = false
    }
  }

  async function loadClientNames () {
    const missing = chats.value.map(c => c.phone).filter(p => !(p in clientNames.value))
    if (missing.length === 0) return
    try {
      const found = await clientsByPhones(missing.slice(0, 100))
      const next = { ...clientNames.value }
      for (const p of missing) next[p] = ''
      for (const c of found) {
        // Clients are matched on the last 8 digits, whatever the country prefix.
        const phone = missing.find(p => p.endsWith(c.digits.slice(-8)))
        if (phone) next[phone] = c.name
      }
      clientNames.value = next
    } catch { /* numbers are shown instead of names */ }
  }

  const title = c => clientNames.value[c.phone] || c.name || formatPhone(c.phone)
  const preview = c => c.last_body || KIND_LABEL[c.last_kind] || ''

  let searchTimer = 0
  watch(search, () => {
    clearTimeout(searchTimer)
    searchTimer = setTimeout(() => loadChats(), 300)
  })

  // ---- the open thread ----
  const phone = ref('')
  const messages = ref([])
  const moreMessages = ref(false)
  const loadingMessages = ref(false)
  const chatEl = ref(null)

  const openTitle = computed(() => {
    const c = chats.value.find(x => x.phone === phone.value)
    return c ? title(c) : formatPhone(phone.value)
  })

  function open (p) {
    phone.value = p
  }

  async function loadThread (quiet = false) {
    const p = phone.value
    if (!p) return
    if (!quiet) {
      messages.value = []
      moreMessages.value = false
      loadingMessages.value = true
    }
    try {
      const data = await getChat(p, { page_size: 100 })
      if (phone.value !== p) return
      // A refresh keeps the older pages already loaded and only swaps in the latest.
      const latest = data.messages ?? []
      const firstId = latest[0]?.id ?? 0
      const kept = quiet ? messages.value.filter(m => m.id < firstId) : []
      const grew = latest.length + kept.length > messages.value.length
      messages.value = [...kept, ...latest]
      if (!quiet || kept.length === 0) moreMessages.value = data.more
      loadMedia()
      if (!quiet || grew) scrollToEnd()
    } catch (error) {
      if (!quiet) notify.error(error.message)
    } finally {
      loadingMessages.value = false
    }
  }

  async function loadEarlier () {
    const p = phone.value
    loadingMessages.value = true
    try {
      const data = await getChat(p, { before: messages.value[0].id, page_size: 100 })
      if (phone.value !== p) return
      const el = chatEl.value
      const before = el ? el.scrollHeight - el.scrollTop : 0
      messages.value = [...(data.messages ?? []), ...messages.value]
      moreMessages.value = data.more
      loadMedia()
      // Stay on the message that was at the top.
      await nextTick()
      if (el) el.scrollTop = el.scrollHeight - before
    } catch (error) {
      notify.error(error.message)
    } finally {
      loadingMessages.value = false
    }
  }

  async function scrollToEnd () {
    await nextTick()
    if (chatEl.value) chatEl.value.scrollTop = chatEl.value.scrollHeight
  }

  watch(phone, () => {
    for (const url of Object.values(media.value)) URL.revokeObjectURL(url)
    media.value = {}
    mediaError.value = {}
    loadThread()
  })

  // ---- refreshing ----
  watch(() => props.refreshKey, () => {
    loadChats()
    loadThread(true)
  })
  loadChats()

  // Media comes through the API with the session's token, so it's fetched
  // here and handed to <audio>/<img> as a blob URL (kept per media_url).
  const media = ref({})
  const mediaError = ref({})
  const mediaKind = m => (m.media_type || '').split('/')[0]
  async function loadMedia () {
    const p = phone.value
    for (const m of messages.value) {
      const path = m.media_url
      if (!path || media.value[path] || mediaError.value[path]) continue
      try {
        const res = await authFetch(`${API_BASE}${path}`)
        if (!res.ok) throw new Error(res.status === 404 ? 'No longer kept' : `Couldn't load it (${res.status})`)
        const url = URL.createObjectURL(await res.blob())
        if (phone.value !== p) {
          URL.revokeObjectURL(url)
          return
        }
        media.value = { ...media.value, [path]: url }
      } catch (error) {
        mediaError.value = { ...mediaError.value, [path]: error.message }
      }
    }
  }
  onUnmounted(() => {
    clearTimeout(searchTimer)
    for (const url of Object.values(media.value)) URL.revokeObjectURL(url)
  })

  function escapeHTML (s) {
    return s.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')
  }
  // WhatsApp's own formatting: *bold*, _italic_, links.
  function format (text) {
    return escapeHTML(text ?? '')
      .replaceAll(/\*([^*\n]+)\*/g, '<strong>$1</strong>')
      .replaceAll(/_([^_\n]+)_/g, '<em>$1</em>')
      .replaceAll(/(https?:\/\/\S+)/g, '<span class="wi__link">$1</span>')
  }
  const time = at => new Date(at).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
  // Today as a time, this week as a weekday, else the date: like WhatsApp's list.
  function when (at) {
    const d = new Date(at)
    const days = Math.floor((new Date().setHours(0, 0, 0, 0) - new Date(d).setHours(0, 0, 0, 0)) / 86_400_000)
    if (days <= 0) return time(at)
    if (days === 1) return 'Yesterday'
    if (days < 7) return d.toLocaleDateString(undefined, { weekday: 'long' })
    return d.toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' })
  }
  // A day header above the first bubble of each day.
  function dayLabel (i) {
    const day = new Date(messages.value[i].at).toDateString()
    if (i > 0 && new Date(messages.value[i - 1].at).toDateString() === day) return ''
    if (day === new Date().toDateString()) return 'Today'
    return new Date(messages.value[i].at).toLocaleDateString(undefined, { day: 'numeric', month: 'long', year: 'numeric' })
  }
</script>

<style scoped>
.wi {
  display: flex;
  height: calc(100vh - 230px);
  min-height: 420px;
  border-top: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.wi__list {
  flex: 0 0 340px;
  display: flex;
  flex-direction: column;
  min-width: 0;
  border-inline-end: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.wi__scroll {
  flex: 1;
  overflow-y: auto;
}
.wi__chat {
  /* A <button>: drop the browser's grey fill, border and font. */
  background: transparent;
  border: 0;
  border-bottom: 1px solid rgba(var(--v-border-color), 0.06);
  color: inherit;
  font: inherit;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 10px 12px;
  text-align: start;
}
.wi__chat:hover {
  background: rgba(var(--v-theme-on-surface), 0.04);
}
.wi__chat--open {
  background: rgba(var(--v-theme-primary), 0.1);
}
.wi__title,
.wi__preview {
  /* Arabic runs read right to left inside, but the line stays left-aligned
     under the avatar instead of jumping to the right edge. */
  text-align: left;
  unicode-bidi: isolate;
}
.wi__thread {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.wi__empty {
  margin: auto;
}
.wi__head {
  flex-shrink: 0;
  background: #008069;
  color: #fff;
}
.wi__sub {
  opacity: 0.85;
}
.wi__back {
  display: none;
}
.wi__chatbox {
  flex: 1;
  overflow-y: auto;
  background: #efeae2;
}
.wi__row {
  display: flex;
  margin: 2px 0;
}
.wi__row--us {
  justify-content: flex-end;
}
.wi__bubble {
  max-width: 85%;
  padding: 6px 8px 4px;
  border-radius: 8px;
  color: #111b21;
  box-shadow: 0 1px 0.5px rgba(11, 20, 26, 0.13);
  font-size: 0.9rem;
  line-height: 1.35;
}
.wi__bubble--us {
  background: #d9fdd3;
  border-top-right-radius: 0;
}
.wi__bubble--client {
  background: #fff;
  border-top-left-radius: 0;
}
.wi__text {
  white-space: pre-wrap;
  word-break: break-word;
  /* Each line in its own direction, so an Arabic line reads right to left. */
  unicode-bidi: plaintext;
}
.wi__text :deep(.wi__link) {
  color: #027eb5;
}
.wi__audio {
  display: block;
  width: 260px;
  max-width: 100%;
  height: 36px;
}
.wi__media {
  display: block;
  max-width: 260px;
  max-height: 320px;
  border-radius: 6px;
}
.wi__meta {
  text-align: end;
  font-size: 0.68rem;
  color: #667781;
  margin-top: 2px;
}
.wi__day {
  width: fit-content;
  margin: 8px auto;
  padding: 4px 10px;
  border-radius: 7px;
  font-size: 0.7rem;
  text-align: center;
  text-transform: uppercase;
  color: #54656f;
  background: #ffffffd9;
}

.wi--dark .wi__head {
  background: #202c33;
}
.wi--dark .wi__chatbox {
  background: #0b141a;
}
.wi--dark .wi__bubble {
  color: #e9edef;
  box-shadow: none;
}
.wi--dark .wi__bubble--us {
  background: #005c4b;
}
.wi--dark .wi__bubble--client {
  background: #202c33;
}
.wi--dark .wi__meta {
  color: #8696a0;
}
.wi--dark .wi__text :deep(.wi__link) {
  color: #53bdeb;
}
.wi--dark .wi__day {
  background: #182229;
  color: #8696a0;
}

/* Phones: the list and the thread take turns. */
@media (max-width: 760px) {
  .wi__list {
    flex-basis: 100%;
  }
  .wi__list--hidden,
  .wi__thread--hidden {
    display: none;
  }
  .wi__back {
    display: inline-flex;
  }
}
</style>
