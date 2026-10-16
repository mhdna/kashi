<template>
  <div class="pa-4 pt-3">
    <div v-if="!draft" class="d-flex justify-center pa-12">
      <v-progress-circular color="primary" indeterminate />
    </div>

    <v-row v-else density="compact">
      <!-- How messages go out -->
      <v-col cols="12">
        <v-card class="card" variant="flat">
          <div class="card__title">Send with</div>
          <v-radio-group v-model="draft.whatsapp_sender" class="mt-1" :disabled="!canEdit" hide-details>
            <v-radio value="bridge">
              <template #label>
                <div>
                  <div>Our WhatsApp numbers</div>
                  <div class="text-caption text-medium-emphasis">
                    Free, any text, from numbers linked below by scanning a QR. Unofficial: WhatsApp can
                    restrict a number that messages many people who never wrote to it first.
                  </div>
                </div>
              </template>
            </v-radio>
            <v-radio class="mt-2" value="meta">
              <template #label>
                <div>
                  <div>Meta's WhatsApp Cloud API</div>
                  <div class="text-caption text-medium-emphasis">
                    Official, paid per message, sends the approved template set in
                    <router-link to="/settings?tab=whatsapp">Settings › WhatsApp</router-link>.
                  </div>
                </div>
              </template>
            </v-radio>
          </v-radio-group>
          <v-alert
            v-if="settingsOff"
            class="mt-3"
            density="compact"
            type="warning"
            variant="tonal"
          >
            Sending after sales is off. Turn it on in <router-link to="/settings?tab=whatsapp">Settings › WhatsApp</router-link>.
          </v-alert>
        </v-card>
      </v-col>

      <template v-if="draft.whatsapp_sender === 'bridge'">
        <!-- Numbers -->
        <v-col cols="12" md="7">
          <v-card class="card" variant="flat">
            <div class="d-flex align-center">
              <div class="card__title">Numbers</div>
              <v-spacer />
              <v-btn
                v-if="canEdit"
                color="primary"
                :disabled="!hasSavedKey"
                prepend-icon="mdi-plus"
                size="small"
                text="Add number"
                variant="tonal"
                @click="openAdd"
              />
            </div>
            <v-alert
              v-if="!hasSavedKey"
              class="my-2"
              density="compact"
              type="info"
              variant="tonal"
            >
              First paste the bridge's API key under <strong>Bridge</strong> at the bottom of this tab and
              <strong>Save</strong>; then numbers can be added.
            </v-alert>
            <v-alert
              v-else-if="bridgeError"
              class="my-2"
              density="compact"
              type="warning"
              variant="tonal"
            >
              {{ bridgeError }}
            </v-alert>
            <div v-if="numbers.length === 0" class="text-body-2 text-medium-emphasis py-6 text-center">
              No numbers yet. Add one and scan its QR from WhatsApp on that phone
              (Settings › Linked devices › Link a device).
            </div>
            <v-list v-else class="py-0" density="compact">
              <v-list-item v-for="n in numbers" :key="n.id" class="px-0">
                <template #prepend>
                  <v-icon class="me-3" :color="STATE[n.state]?.color" icon="mdi-cellphone" />
                </template>
                <v-list-item-title>
                  {{ n.label || 'Unnamed number' }}
                  <v-chip
                    v-if="n.is_default"
                    class="ms-1"
                    color="primary"
                    size="x-small"
                    text="Default"
                    variant="tonal"
                  />
                </v-list-item-title>
                <v-list-item-subtitle>
                  {{ n.phone ? `+${n.phone}` : 'Not linked yet' }} ·
                  <span :class="`text-${STATE[n.state]?.color || 'medium-emphasis'}`">{{ STATE[n.state]?.label || n.state }}</span>
                  <span v-if="n.queued"> · {{ n.queued }} waiting to send</span>
                </v-list-item-subtitle>
                <template v-if="canEdit" #append>
                  <v-btn
                    v-if="n.state !== 'open'"
                    icon="mdi-qrcode"
                    size="small"
                    title="Link: show its QR"
                    variant="text"
                    @click="showQR(n)"
                  />
                  <v-btn
                    v-if="!n.is_default"
                    icon="mdi-star-outline"
                    size="small"
                    title="Make default"
                    variant="text"
                    @click="makeDefault(n)"
                  />
                  <v-btn
                    icon="mdi-pencil"
                    size="small"
                    title="Rename"
                    variant="text"
                    @click="rename(n)"
                  />
                  <v-btn
                    color="error"
                    icon="mdi-delete"
                    size="small"
                    title="Remove"
                    variant="text"
                    @click="remove(n)"
                  />
                </template>
              </v-list-item>
            </v-list>
          </v-card>
        </v-col>

        <!-- Which number each branch sends from -->
        <v-col cols="12" md="5">
          <v-card class="card" variant="flat">
            <div class="card__title">Each branch sends from</div>
            <p class="text-caption text-medium-emphasis mb-3">
              A branch left on "Default number" uses the number marked Default.
            </p>
            <v-select
              v-for="b in draft.branches"
              :key="b.branch_id"
              v-model="b.number_id"
              class="mb-2"
              density="compact"
              :disabled="!canEdit"
              hide-details
              :items="numberOptions"
              :label="b.branch_name"
              variant="outlined"
            />
          </v-card>
        </v-col>

        <!-- How the message asks for feedback -->
        <v-col cols="12">
          <v-card class="card" variant="flat">
            <div class="card__title">Ask for feedback with</div>
            <v-radio-group
              v-model="draft.whatsapp_feedback_mode"
              class="mt-1"
              :disabled="!canEdit"
              hide-details
              inline
            >
              <v-radio label="A star-rating poll after the message" value="poll" />
              <v-radio class="ms-4" label="The Google review link in the message" value="link" />
            </v-radio-group>
            <p class="text-caption text-medium-emphasis mt-2 mb-0">
              <template v-if="pollMode">
                One tap in the chat, no link or sign-in. Answers show on the
                <router-link to="/reviews">Reviews</router-link> page by branch, with the low ones to call back.
              </template>
              <template v-else>
                Each client gets their own link to your Google review box (set up on the
                <router-link to="/reviews">Reviews</router-link> page), with your banner card.
              </template>
            </p>
            <v-row class="mt-2" density="compact">
              <v-col cols="12" md="6">
                <v-text-field
                  v-model="draft.whatsapp_instagram_url"
                  density="compact"
                  :disabled="!canEdit"
                  hint="For {instagram} in the message; left empty, that line is left out"
                  label="Instagram page"
                  persistent-hint
                  placeholder="https://instagram.com/yourshop"
                  variant="outlined"
                />
              </v-col>
              <template v-if="pollMode">
                <v-col cols="12" md="6">
                  <v-text-field
                    v-model="draft.whatsapp_poll_question"
                    density="compact"
                    :disabled="!canEdit"
                    hint="Shown in bold on top of the poll"
                    label="Poll question"
                    persistent-hint
                    :placeholder="defaultPollQuestion"
                    variant="outlined"
                  />
                </v-col>
                <v-col cols="12" md="6">
                  <v-textarea
                    v-model="draft.whatsapp_poll_options"
                    auto-grow
                    density="compact"
                    :disabled="!canEdit"
                    hint="3 to 5 lines, best first: the first counts as 5 stars, the last as 1"
                    label="Poll options"
                    persistent-hint
                    :placeholder="defaultPollOptions"
                    rows="3"
                    variant="outlined"
                  />
                </v-col>
              </template>
            </v-row>
          </v-card>
        </v-col>

        <!-- The message -->
        <v-col cols="12" md="7">
          <v-card class="card" variant="flat">
            <div class="card__title">Message {{ pollMode ? '(with the poll)' : '(with the review link)' }}</div>
            <p class="text-caption text-medium-emphasis mb-2">
              Filled in per sale: <code v-for="p in placeholders" :key="p" class="me-1">{{ placeholder(p) }}</code>.
              A line whose placeholder has nothing to fill is left out. *bold* works as in WhatsApp.
            </p>
            <v-textarea
              v-if="pollMode"
              v-model="draft.whatsapp_bridge_message"
              auto-grow
              density="compact"
              :disabled="!canEdit"
              hide-details
              :placeholder="defaultMessage"
              rows="8"
              variant="outlined"
            />
            <v-textarea
              v-else
              v-model="draft.whatsapp_bridge_link_message"
              auto-grow
              density="compact"
              :disabled="!canEdit"
              hide-details
              :placeholder="defaultLinkMessage"
              rows="8"
              variant="outlined"
            />
          </v-card>
        </v-col>
        <v-col cols="12" md="5">
          <v-card class="card" variant="flat">
            <div class="card__title mb-2">Preview</div>
            <div class="bubble" v-html="previewHTML" />
            <div v-if="pollMode" class="bubble poll mt-1">
              <div class="font-weight-medium mb-1">{{ pollQuestion }}</div>
              <div class="text-caption poll__hint mb-1">Select one</div>
              <div v-for="o in pollOptions" :key="o" class="poll__option">
                <span class="poll__radio" />{{ o }}
              </div>
            </div>
          </v-card>
        </v-col>

        <!-- The bridge -->
        <v-col cols="12">
          <v-card class="card" variant="flat">
            <div class="card__title">Bridge</div>
            <p class="text-caption text-medium-emphasis mb-3">
              The WhatsApp bridge holds the numbers' sessions. Its API key is <code>API_KEY</code> in
              <code>/home/wabridge/app/bridge.env</code> on the server.
            </p>
            <v-row density="compact">
              <v-col cols="12" md="6">
                <v-text-field
                  v-model="draft.whatsapp_bridge_url"
                  density="compact"
                  :disabled="!canEdit"
                  hide-details
                  label="Address"
                  placeholder="http://127.0.0.1:8099"
                  variant="outlined"
                />
              </v-col>
              <v-col cols="12" md="6">
                <v-text-field
                  v-model="draft.whatsapp_bridge_key"
                  :append-inner-icon="showKey ? 'mdi-eye-off' : 'mdi-eye'"
                  autocomplete="off"
                  density="compact"
                  :disabled="!canEdit"
                  hide-details
                  label="API key"
                  :type="showKey ? 'text' : 'password'"
                  variant="outlined"
                  @click:append-inner="showKey = !showKey"
                />
              </v-col>
            </v-row>
          </v-card>
        </v-col>
      </template>

      <v-col v-if="canEdit" class="d-flex justify-end" cols="12">
        <v-btn
          color="primary"
          :disabled="!dirty"
          :loading="saving"
          text="Save"
          variant="flat"
          @click="save"
        />
      </v-col>
    </v-row>

    <!-- Add a number -->
    <v-dialog v-model="addOpen" max-width="420">
      <v-card title="Add a WhatsApp number">
        <v-card-text>
          <v-text-field
            v-model="addLabel"
            autofocus
            density="compact"
            hint="What you call it, e.g. Centro or Main"
            label="Name"
            persistent-hint
            variant="outlined"
            @keyup.enter="add"
          />
          <v-alert
            v-if="addError"
            class="mt-3"
            density="compact"
            type="error"
            variant="tonal"
          >
            {{ addError }}
          </v-alert>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Cancel" @click="addOpen = false" />
          <v-btn
            color="primary"
            :loading="adding"
            text="Add and show QR"
            variant="flat"
            @click="add"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Link a number: its QR, refreshed until it's scanned -->
    <v-dialog v-model="qrOpen" max-width="380" @after-leave="stopQR">
      <v-card :title="`Link ${qrNumber?.label || 'number'}`">
        <v-card-text class="text-center">
          <template v-if="qrState === 'open'">
            <v-icon color="success" icon="mdi-check-circle" size="64" />
            <div class="mt-2">Linked.</div>
          </template>
          <template v-else>
            <p class="text-body-2 text-medium-emphasis mb-3">
              On that phone: WhatsApp › Settings › Linked devices › Link a device, then scan.
            </p>
            <v-img
              v-if="qrImage"
              class="mx-auto"
              :src="qrImage"
              width="260"
            />
            <v-progress-circular v-else class="my-12" color="primary" indeterminate />
            <div v-if="qrError" class="text-caption text-error mt-2">{{ qrError }}</div>
          </template>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Close" @click="qrOpen = false" />
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
  import { computed, onUnmounted, ref } from 'vue'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { useWhatsApp } from '@/composables/useWhatsApp'

  // How post-sale messages go out: our own numbers through the WhatsApp bridge
  // (api/whatsapp_bridge.go), or Meta's Cloud API.

  const props = defineProps({
    settings: { type: Object, default: null },
  })

  const notify = useNotify()
  const { canEdit } = usePageAccess()
  const { getSender, saveSender, addNumber, numberQR, updateNumber, deleteNumber } = useWhatsApp()

  const PLACEHOLDERS = ['name', 'invoice', 'total', 'date', 'branch', 'review', 'instagram']
  const placeholder = p => '{' + p + '}'
  const STATE = {
    open: { label: 'Linked', color: 'success' },
    close: { label: 'Waiting for its QR to be scanned', color: 'warning' },
    connecting: { label: 'Connecting…', color: 'info' },
    missing: { label: 'Not on the bridge — show its QR to link it again', color: 'error' },
    unknown: { label: 'Status unknown', color: undefined },
  }

  const saved = ref(null)
  const draft = ref(null)
  const numbers = ref([])
  const bridgeError = ref('')
  const defaultMessage = ref('')
  const defaultLinkMessage = ref('')
  const defaultPollQuestion = ref('')
  const defaultPollOptions = ref('')
  const showKey = ref(false)
  const saving = ref(false)

  // Numbers live on the bridge, which only answers with its key.
  const hasSavedKey = computed(() => Boolean(saved.value?.whatsapp_bridge_key?.trim()))
  const settingsOff = computed(() => props.settings && !props.settings.whatsapp_enabled)
  const dirty = computed(() => JSON.stringify(draft.value) !== JSON.stringify(saved.value))
  const numberOptions = computed(() => [
    { title: 'Default number', value: null },
    ...numbers.value.map(n => ({ title: n.label || (n.phone ? `+${n.phone}` : n.instance), value: n.id })),
  ])

  function apply (body) {
    saved.value = body.settings
    draft.value = structuredClone(body.settings)
    numbers.value = body.numbers ?? []
    bridgeError.value = body.bridge_error || ''
    defaultMessage.value = body.default_message || ''
    defaultLinkMessage.value = body.default_link_message || ''
    defaultPollQuestion.value = body.default_poll_question || ''
    defaultPollOptions.value = body.default_poll_options || ''
  }
  async function load () {
    try {
      apply(await getSender())
    } catch (error) {
      notify.error(error.message)
    }
  }
  async function reloadNumbers () {
    try {
      const body = await getSender()
      numbers.value = body.numbers ?? []
      bridgeError.value = body.bridge_error || ''
    } catch (error) {
      notify.error(error.message)
    }
  }
  async function save () {
    saving.value = true
    try {
      apply(await saveSender(draft.value))
      notify.success('Saved.')
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }

  // ---- preview ---------------------------------------------------------------
  const pollMode = computed(() => draft.value?.whatsapp_feedback_mode !== 'link')
  const placeholders = computed(() => PLACEHOLDERS.filter(p => p !== 'review' || !pollMode.value))
  const SAMPLE = {
    name: 'Sara',
    invoice: 'SA/CT/2026/00482',
    total: '$156.25',
    date: new Date().toLocaleDateString('en-GB'),
    branch: 'Centro',
    review: 'https://grandbz.com/review/k7Pq2xWm',
  }
  // Like the server's whatsAppBridgeText: lines whose placeholder is empty go,
  // and so does a blank line doubled by that.
  function fillMessage (message) {
    const values = {
      ...SAMPLE,
      review: pollMode.value ? '' : SAMPLE.review,
      instagram: draft.value?.whatsapp_instagram_url?.trim() || '',
    }
    const out = []
    for (const line of message.split('\n')) {
      const keys = [...line.matchAll(/\{(\w+)\}/g)].map(m => m[1])
      if (keys.some(k => k in values && !values[k])) continue
      const filled = line.replaceAll(/\{(\w+)\}/g, (m, key) => values[key] ?? m)
      if (!filled.trim() && out.length > 0 && !out.at(-1).trim()) continue
      out.push(filled)
    }
    return out.join('\n').trim()
  }
  function escapeHTML (s) {
    return s.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;')
  }
  const pollQuestion = computed(() => draft.value?.whatsapp_poll_question?.trim() || defaultPollQuestion.value)
  const pollOptions = computed(() => {
    const lines = (draft.value?.whatsapp_poll_options || '').split('\n').map(l => l.trim()).filter(Boolean)
    return lines.length >= 3 && lines.length <= 5 ? lines : defaultPollOptions.value.split('\n')
  })
  const previewHTML = computed(() => {
    const message = pollMode.value
      ? (draft.value?.whatsapp_bridge_message || defaultMessage.value)
      : (draft.value?.whatsapp_bridge_link_message || defaultLinkMessage.value)
    return escapeHTML(fillMessage(message))
      .replaceAll(/\*([^*\n]+)\*/g, '<strong>$1</strong>')
      .replaceAll(/_([^_\n]+)_/g, '<em>$1</em>')
      .replaceAll(/(https?:\/\/\S+)/g, '<span class="bubble__link">$1</span>')
  })

  // ---- numbers ---------------------------------------------------------------
  const addOpen = ref(false)
  const addLabel = ref('')
  const adding = ref(false)
  function openAdd () {
    addLabel.value = ''
    addError.value = ''
    addOpen.value = true
  }
  const addError = ref('')
  async function add () {
    adding.value = true
    addError.value = ''
    try {
      const { number, qr } = await addNumber(addLabel.value.trim())
      addOpen.value = false
      await reloadNumbers()
      showQR(number, qr)
    } catch (error) {
      addError.value = error.message
    } finally {
      adding.value = false
    }
  }

  const qrOpen = ref(false)
  const qrNumber = ref(null)
  const qrImage = ref('')
  const qrState = ref('')
  const qrError = ref('')
  let qrTimer = 0
  // WhatsApp's QR changes every ~20 s; ask again every few seconds until the
  // number is linked.
  async function pollQR () {
    try {
      const { qr, state } = await numberQR(qrNumber.value.id)
      qrState.value = state
      if (qr) qrImage.value = qr
      qrError.value = ''
      if (state === 'open') {
        await reloadNumbers()
        return
      }
    } catch (error) {
      qrError.value = error.message
    }
    if (qrOpen.value) qrTimer = setTimeout(pollQR, 4000)
  }
  function showQR (number, firstQR = '') {
    qrNumber.value = number
    qrImage.value = firstQR
    qrState.value = ''
    qrError.value = ''
    qrOpen.value = true
    clearTimeout(qrTimer)
    qrTimer = setTimeout(pollQR, firstQR ? 4000 : 0)
  }
  function stopQR () {
    clearTimeout(qrTimer)
    reloadNumbers()
  }

  async function makeDefault (n) {
    try {
      await updateNumber(n.id, { label: n.label, is_default: true })
      await reloadNumbers()
    } catch (error) {
      notify.error(error.message)
    }
  }
  async function rename (n) {
    const label = window.prompt('Name', n.label)
    if (label == null) return
    try {
      await updateNumber(n.id, { label: label.trim(), is_default: n.is_default })
      await reloadNumbers()
    } catch (error) {
      notify.error(error.message)
    }
  }
  async function remove (n) {
    if (!window.confirm(`Remove ${n.label || 'this number'}? It is unlinked from WhatsApp, and branches sending from it go back to the default number.`)) return
    try {
      await deleteNumber(n.id)
      await load()
    } catch (error) {
      notify.error(error.message)
    }
  }

  onUnmounted(() => clearTimeout(qrTimer))
  load()
</script>

<style scoped>
.card {
  padding: 12px 14px;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 10px;
  height: 100%;
}
.card__title {
  font-size: 0.9rem;
  font-weight: 600;
}
.bubble {
  background: #dcf8c6;
  color: #111;
  border-radius: 8px;
  padding: 8px 10px;
  white-space: pre-line;
  font-size: 0.9rem;
  max-width: 340px;
}
.poll__hint {
  color: #667781;
}
.poll__option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
}
.poll__radio {
  width: 16px;
  height: 16px;
  border: 2px solid #8696a0;
  border-radius: 50%;
  flex: none;
}
:deep(.bubble__link) {
  color: #027eb5;
  text-decoration: underline;
  word-break: break-all;
}
</style>
