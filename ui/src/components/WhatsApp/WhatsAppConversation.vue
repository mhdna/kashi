<template>
  <v-dialog max-width="1000" :model-value="!!messageId" @update:model-value="close">
    <v-card class="wc" :class="{ 'wc--dark': theme.current.value.dark }">
      <div class="wc__head d-flex align-center ga-3 px-4 py-2">
        <v-avatar color="grey-lighten-1" size="38">
          <v-icon color="white" icon="mdi-account" />
        </v-avatar>
        <div class="flex-grow-1 min-w-0">
          <div class="font-weight-medium text-truncate">{{ message?.client_name || formatPhone(message?.client_phone) || '…' }}</div>
          <div class="text-caption wc__sub text-truncate">
            {{ formatPhone(message?.client_phone) }}<template v-if="message?.branch_name"> · {{ message.branch_name }}</template>
          </div>
        </div>
        <v-btn icon="mdi-close" size="small" variant="text" @click="close" />
      </div>

      <div class="wc__layout">
        <div class="wc__chat pa-3">
          <PreparingState
            v-if="loading"
            kind="chat"
            min-height="200px"
            title="Loading the conversation"
          />
          <template v-else>
            <template v-for="(b, i) in bubbles" :key="i">
              <div v-if="dayLabel(i)" class="wc__day">{{ dayLabel(i) }}</div>

              <div v-if="b.from === 'note'" class="wc__note">{{ b.text }}</div>

              <div v-else class="wc__row" :class="`wc__row--${b.from}`">
                <div class="wc__bubble" :class="`wc__bubble--${b.from}`">
                  <img v-if="b.banner" alt="" class="wc__banner" :src="bannerURL">

                  <template v-if="b.kind === 'poll'">
                    <div class="font-weight-bold mb-1">{{ b.text }}</div>
                    <div class="wc__poll-hint">Select one</div>
                    <div v-for="o in b.options" :key="o" class="wc__option">
                      <v-icon
                        :color="o === b.chosen ? 'green-darken-1' : 'grey'"
                        :icon="o === b.chosen ? 'mdi-check-circle' : 'mdi-checkbox-blank-circle-outline'"
                        size="20"
                      />
                      <span class="flex-grow-1">{{ o }}</span>
                      <span v-if="o === b.chosen" class="wc__votes">1</span>
                    </div>
                  </template>

                  <div v-else-if="b.kind === 'template'" class="wc__text">
                    <span class="text-medium-emphasis">Meta template</span> <strong>{{ b.text }}</strong>
                  </div>

                  <!-- A voice note, photo, video or file the bridge kept. -->
                  <template v-else-if="b.media_url">
                    <audio
                      v-if="mediaKind(b) === 'audio'"
                      class="wc__audio"
                      controls
                      preload="metadata"
                      :src="media[b.media_url]"
                    />
                    <img v-else-if="mediaKind(b) === 'image'" alt="" class="wc__media" :src="media[b.media_url]">
                    <video
                      v-else-if="mediaKind(b) === 'video'"
                      class="wc__media"
                      controls
                      preload="metadata"
                      :src="media[b.media_url]"
                    />
                    <a v-else class="wc__text" :download="KIND_LABEL[b.kind] ?? 'file'" :href="media[b.media_url]">
                      <v-icon icon="mdi-paperclip" size="16" /> {{ KIND_LABEL[b.kind] ?? 'File' }}
                    </a>
                    <div v-if="mediaError[b.media_url]" class="text-caption text-medium-emphasis">{{ mediaError[b.media_url] }}</div>
                    <!-- eslint-disable-next-line vue/no-v-html -- escaped in format() -->
                    <div v-if="b.text" class="wc__text mt-1" v-html="format(b.text)" />
                  </template>
                  <div v-else-if="b.kind !== 'text' && !b.text" class="wc__text text-medium-emphasis">
                    <v-icon :icon="KIND_ICON[b.kind] ?? 'mdi-paperclip'" size="16" /> {{ KIND_LABEL[b.kind] ?? b.kind }}
                  </div>
                  <!-- eslint-disable-next-line vue/no-v-html -- escaped in format() -->
                  <div v-else class="wc__text" v-html="format(b.text)" />

                  <div class="wc__meta">
                    {{ time(b.at) }}
                    <v-icon
                      v-if="b.ticks"
                      :class="`wc__ticks--${b.ticks}`"
                      :icon="TICK_ICON[b.ticks]"
                      size="15"
                      :title="TICK_TITLE[b.ticks]"
                    />
                  </div>
                  <div v-if="b.read_at || b.delivered_at" class="wc__seen" :class="{ 'wc__seen--read': b.read_at }">
                    {{ b.read_at ? `Read ${time(b.read_at)}` : `Delivered ${time(b.delivered_at)}` }}
                  </div>
                </div>
              </div>
            </template>
            <div v-if="bubbles.length > 0 && message && !message.saved" class="wc__foot">
              Sent before kashi kept the text: rebuilt with today's wording.
            </div>
          </template>
        </div>

        <!-- The sale and the message, beside the chat. -->
        <aside class="wc__side pa-4">
          <template v-if="message">
            <div class="wc__side-title">Message</div>
            <div class="wc__facts">
              <span>Sent</span><span>{{ new Date(message.created_at).toLocaleString() }}</span>
              <span>Outcome</span>
              <span>
                {{ STATUS[message.status]?.label ?? message.status }}<template v-if="message.reason"> · {{ message.reason }}</template>
                <v-chip v-if="message.is_test" class="ms-1" size="x-small" variant="tonal">Test</v-chip>
              </span>
              <span>Rule</span><span>{{ message.rule_name || 'Default' }}</span>
              <template v-if="message.poll_answer">
                <span>Answer</span><span>{{ message.poll_answer }}</span>
              </template>
            </div>

            <div class="wc__side-title mt-5">
              Sale
              <v-chip
                v-if="invoice?.cancelled_at?.Valid"
                class="ms-1"
                color="error"
                size="x-small"
                variant="tonal"
              >Cancelled</v-chip>
            </div>
            <div v-if="!invoice" class="text-body-2 text-medium-emphasis">
              {{ message.is_test ? 'A test send: there is no sale.' : "This sale hasn't reached kashi yet." }}
            </div>
            <template v-else>
              <div class="wc__facts">
                <span>Invoice</span><span>{{ invoice.code }}</span>
                <span>When</span><span>{{ new Date(invoice.occurred_at).toLocaleString() }}</span>
                <span>Branch</span><span>{{ message.branch_name || '—' }}</span>
                <span>Salesperson</span><span>{{ invoice.salesperson || '—' }}</span>
              </div>

              <div class="wc__side-title mt-5">{{ invoice.items.length }} item{{ invoice.items.length === 1 ? '' : 's' }}</div>
              <div v-for="it in invoice.items" :key="it.id" class="wc__item">
                <div class="flex-grow-1 min-w-0">
                  <div class="text-body-2 font-weight-medium">{{ it.product_name || 'Unknown product' }}</div>
                  <div class="text-caption text-medium-emphasis">
                    {{ [it.product_code, it.color_name, it.size_name].filter(Boolean).join(' · ') }}
                  </div>
                  <div class="text-caption text-medium-emphasis">
                    {{ it.quantity }} × {{ formatDollars(it.unit_price) }}<template v-if="it.discount"> · −{{ it.discount }}%</template>
                  </div>
                </div>
                <div class="text-body-2 font-weight-medium">{{ formatDollars(it.line_total) }}</div>
              </div>

              <div class="wc__totals">
                <span>Subtotal</span><span>{{ formatDollars(invoice.subtotal) }}</span>
                <template v-if="invoice.discount">
                  <span>Discount</span><span>−{{ invoice.discount }}%</span>
                </template>
                <strong>Total</strong><strong>{{ formatDollars(invoice.grand_total) }}</strong>
                <template v-for="p in invoice.payments" :key="p.id">
                  <span class="text-medium-emphasis">Paid · {{ p.account_name }}</span>
                  <span class="text-medium-emphasis">{{ formatDollars(p.amount) }}</span>
                </template>
              </div>
            </template>
          </template>
        </aside>
      </div>
    </v-card>
  </v-dialog>
</template>

<script setup>
  import { ref, watch } from 'vue'
  import { useTheme } from 'vuetify'
  import { authFetch } from '@/composables/useApi'
  import { useNotify } from '@/composables/useNotify'
  import { useWhatsApp } from '@/composables/useWhatsApp'
  import { API_BASE } from '@/config'
  import { formatDollars, formatPhone, STATUS } from '@/utils/whatsapp'

  // One row of the Messages tab as a WhatsApp chat: what we sent, the poll
  // and its vote, the follow-up and what the client wrote back
  // (api/whatsapp_conversation.go).

  const props = defineProps({
    messageId: { type: Number, default: 0 },
    // Bumped when something new may have come in: reloads the open chat.
    refreshKey: { type: Number, default: 0 },
  })
  const emit = defineEmits(['close'])

  const notify = useNotify()
  const theme = useTheme()
  const { getConversation } = useWhatsApp()

  const message = ref(null)
  const bubbles = ref([])
  const invoice = ref(null)
  const loading = ref(false)

  const bannerURL = `${API_BASE}/review-assets/banner.jpg`
  const TICK_ICON = { queued: 'mdi-clock-outline', sent: 'mdi-check', delivered: 'mdi-check-all', read: 'mdi-check-all', failed: 'mdi-alert-circle-outline' }
  const TICK_TITLE = { queued: 'Waiting in the queue', sent: 'Sent', delivered: 'Delivered', read: 'Read', failed: 'Not delivered' }
  const KIND_ICON = { image: 'mdi-camera', video: 'mdi-video', audio: 'mdi-microphone', sticker: 'mdi-sticker-emoji', document: 'mdi-file-document-outline' }
  const KIND_LABEL = { image: 'Photo', video: 'Video', audio: 'Voice message', sticker: 'Sticker', document: 'Document', other: 'Message' }

  function close () {
    emit('close')
  }

  // quiet: a refresh of the open chat, without clearing it or a spinner.
  async function load (id, quiet = false) {
    if (!quiet) {
      message.value = null
      bubbles.value = []
      invoice.value = null
    }
    if (!id) return
    if (!quiet) loading.value = true
    try {
      const data = await getConversation(id)
      if (props.messageId !== id) return
      message.value = data.message
      bubbles.value = data.bubbles ?? []
      loadMedia()
      invoice.value = data.invoice ?? null
    } catch (error) {
      if (!quiet) {
        notify.error(error.message)
        close()
      }
    } finally {
      loading.value = false
    }
  }
  watch(() => props.messageId, id => load(id))
  watch(() => props.refreshKey, () => load(props.messageId, true))

  // Media comes through the API with the session's token, so it's fetched
  // here and handed to <audio>/<img> as a blob URL (kept per media_url).
  const media = ref({})
  const mediaError = ref({})
  const mediaKind = b => (b.media_type || '').split('/')[0]
  async function loadMedia () {
    for (const b of bubbles.value) {
      const path = b.media_url
      if (!path || media.value[path] || mediaError.value[path]) continue
      try {
        const res = await authFetch(`${API_BASE}${path}`)
        if (!res.ok) throw new Error(res.status === 404 ? 'No longer kept' : `Couldn't load it (${res.status})`)
        media.value = { ...media.value, [path]: URL.createObjectURL(await res.blob()) }
      } catch (error) {
        mediaError.value = { ...mediaError.value, [path]: error.message }
      }
    }
  }
  watch(() => props.messageId, () => {
    for (const url of Object.values(media.value)) URL.revokeObjectURL(url)
    media.value = {}
    mediaError.value = {}
  })

  function escapeHTML (s) {
    return s.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')
  }
  // WhatsApp's own formatting: *bold*, _italic_, links.
  function format (text) {
    return escapeHTML(text ?? '')
      .replaceAll(/\*([^*\n]+)\*/g, '<strong>$1</strong>')
      .replaceAll(/_([^_\n]+)_/g, '<em>$1</em>')
      .replaceAll(/(https?:\/\/\S+)/g, '<span class="wc__link">$1</span>')
  }
  const time = at => new Date(at).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
  // A day header above the first bubble of each day, like WhatsApp.
  function dayLabel (i) {
    const day = new Date(bubbles.value[i].at).toDateString()
    if (i > 0 && new Date(bubbles.value[i - 1].at).toDateString() === day) return ''
    if (day === new Date().toDateString()) return 'Today'
    return new Date(bubbles.value[i].at).toLocaleDateString(undefined, { day: 'numeric', month: 'long', year: 'numeric' })
  }
</script>

<style scoped>
.wc__head {
  background: #008069;
  color: #fff;
}
.wc__sub {
  opacity: 0.85;
}
.wc__layout {
  display: flex;
  height: min(78vh, 760px);
}
.wc__chat {
  flex: 1 1 480px;
  min-width: 0;
  overflow-y: auto;
  background: #efeae2;
}
.wc__side {
  flex: 0 0 360px;
  overflow-y: auto;
  border-left: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
@media (max-width: 760px) {
  .wc__layout {
    flex-direction: column;
    height: auto;
    max-height: 85vh;
    overflow-y: auto;
  }
  .wc__chat,
  .wc__side {
    flex: none;
    overflow: visible;
  }
}
.wc__side-title {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: rgb(var(--v-theme-on-surface), 0.6);
  margin-bottom: 8px;
}
.wc__facts,
.wc__totals {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 4px 12px;
  font-size: 0.875rem;
}
.wc__facts > :nth-child(odd) {
  color: rgb(var(--v-theme-on-surface), 0.6);
}
.wc__totals {
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.wc__totals > :nth-child(even) {
  text-align: end;
}
.wc__item {
  display: flex;
  gap: 12px;
  padding: 6px 0;
  border-bottom: 1px dashed rgba(var(--v-border-color), var(--v-border-opacity));
}
.wc__row {
  display: flex;
  margin: 2px 0;
}
.wc__row--us {
  justify-content: flex-end;
}
.wc__bubble {
  max-width: 85%;
  padding: 6px 8px 4px;
  border-radius: 8px;
  color: #111b21;
  box-shadow: 0 1px 0.5px rgba(11, 20, 26, 0.13);
  font-size: 0.9rem;
  line-height: 1.35;
}
.wc__bubble--us {
  background: #d9fdd3;
  border-top-right-radius: 0;
}
.wc__bubble--client {
  background: #fff;
  border-top-left-radius: 0;
}
.wc__text {
  white-space: pre-wrap;
  word-break: break-word;
  /* Each line in its own direction, so an Arabic line reads right to left. */
  unicode-bidi: plaintext;
}
.wc__text :deep(.wc__link) {
  color: #027eb5;
}
.wc__audio {
  display: block;
  width: 260px;
  max-width: 100%;
  height: 36px;
}
.wc__media {
  display: block;
  max-width: 260px;
  max-height: 320px;
  border-radius: 6px;
}
.wc__banner {
  display: block;
  width: 100%;
  border-radius: 6px;
  margin-bottom: 6px;
}
.wc__meta {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 3px;
  font-size: 0.68rem;
  color: #667781;
  margin-top: 2px;
}
.wc__ticks--read {
  color: #53bdeb;
}
.wc__ticks--failed {
  color: #ea0038;
}
.wc__poll-hint {
  font-size: 0.75rem;
  color: #667781;
  margin-bottom: 4px;
}
.wc__option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 0;
  min-width: 240px;
}
.wc__seen {
  text-align: end;
  font-size: 0.68rem;
  color: #667781;
}
.wc__seen--read {
  color: #53bdeb;
}
.wc__votes {
  font-size: 0.75rem;
  color: #667781;
}
.wc__note,
.wc__day {
  width: fit-content;
  margin: 8px auto;
  padding: 4px 10px;
  border-radius: 7px;
  font-size: 0.75rem;
  text-align: center;
  color: #54656f;
  background: #ffffffd9;
}
.wc__day {
  text-transform: uppercase;
  font-size: 0.7rem;
}
.wc__note {
  background: #fff5c4;
}
.wc__foot {
  margin-top: 12px;
  text-align: center;
  font-size: 0.68rem;
  color: #8696a0;
}
.min-w-0 {
  min-width: 0;
}
/* WhatsApp's own dark colours. */
.wc--dark .wc__head {
  background: #202c33;
}
.wc--dark .wc__chat {
  background: #0b141a;
}
.wc--dark .wc__bubble {
  color: #e9edef;
  box-shadow: none;
}
.wc--dark .wc__bubble--us {
  background: #005c4b;
}
.wc--dark .wc__bubble--client {
  background: #202c33;
}
.wc--dark .wc__meta,
.wc--dark .wc__seen,
.wc--dark .wc__poll-hint,
.wc--dark .wc__votes {
  color: #8696a0;
}
.wc--dark .wc__seen--read,
.wc--dark .wc__ticks--read {
  color: #53bdeb;
}
.wc--dark .wc__text :deep(.wc__link) {
  color: #53bdeb;
}
.wc--dark .wc__day,
.wc--dark .wc__note {
  background: #182229;
  color: #8696a0;
}
.wc--dark .wc__note {
  color: #ffd279;
}
.wc--dark .wc__foot {
  color: #667781;
}
</style>
