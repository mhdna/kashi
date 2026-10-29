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
                    the Meta Cloud API section above.
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
            Sending after sales is off: switch it on at the top of this page.
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
              A branch left on "Default number" uses the number marked Default. Switch a branch off and it
              sends no messages.
            </p>
            <div v-for="b in draft.branches" :key="b.branch_id" class="d-flex align-center ga-3 mb-2">
              <v-select
                v-model="b.number_id"
                density="compact"
                :disabled="!canEdit || b.enabled === false"
                hide-details
                :items="numberOptions"
                :label="b.branch_name"
                variant="outlined"
              />
              <v-switch
                class="flex-grow-0"
                color="primary"
                density="compact"
                :disabled="!canEdit"
                hide-details
                inset
                :model-value="b.enabled !== false"
                :title="b.enabled === false ? 'Off: sends no messages' : 'On'"
                @update:model-value="b.enabled = !!$event"
              />
            </div>
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
                Each client gets their own link to your Google review box (set up in
                <router-link to="/settings?tab=reviews">Settings › Reviews</router-link>), with your banner card.
              </template>
            </p>
            <v-row class="mt-2" density="compact">
              <v-col cols="12" md="6">
                <v-text-field
                  v-model="draft.whatsapp_channel_url"
                  density="compact"
                  :disabled="!canEdit"
                  hint="For {channel} in the message; left empty, that line is left out"
                  label="WhatsApp channel link"
                  persistent-hint
                  placeholder="https://whatsapp.com/channel/…"
                  variant="outlined"
                />
              </v-col>
              <template v-if="pollMode">
                <v-col class="d-flex align-center" cols="12" md="6">
                  <v-switch
                    v-model="draft.whatsapp_message_logo"
                    color="primary"
                    density="compact"
                    :disabled="!canEdit"
                    hide-details
                    label="Banner on top of the message (not a link)"
                  />
                </v-col>
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
            <v-switch
              v-model="draft.whatsapp_delete_for_me"
              class="mb-1"
              color="primary"
              density="compact"
              :disabled="!canEdit"
              hide-details
              inset
              label="Delete sent messages from the number's own chat (the client keeps them; votes still reach kashi)"
            />
            <v-switch
              v-if="pollMode"
              v-model="draft.whatsapp_delete_poll_for_me"
              class="mb-1"
              color="primary"
              density="compact"
              :disabled="!canEdit || !draft.whatsapp_delete_for_me"
              hide-details
              inset
              label="Also delete the rating poll (turn off if votes stop arriving)"
            />
            <!-- New clients: the channel; returning ones: their receipt. -->
            <v-switch
              v-model="draft.whatsapp_channel_first"
              class="mb-1"
              color="primary"
              density="compact"
              :disabled="!canEdit"
              hide-details
              inset
              label="Channel link for a client's first message, receipt details after that"
            />
            <div v-if="draft.whatsapp_channel_first" class="d-flex align-center ga-2 mb-2">
              <span class="text-caption text-medium-emphasis">Preview as</span>
              <v-btn-toggle v-model="previewFirst" density="compact" mandatory variant="outlined">
                <v-btn size="small" text="New client" :value="true" />
                <v-btn size="small" text="Returning client" :value="false" />
              </v-btn-toggle>
            </div>
            <!-- The till sends a new client's first message once it has their number. -->
            <v-switch
              v-if="draft.whatsapp_channel_first"
              v-model="draft.whatsapp_welcome_early"
              class="mb-1"
              color="primary"
              density="compact"
              :disabled="!canEdit"
              hide-details
              inset
              label="Send a new client's first message when the till gets their number, not after checkout"
            />
            <!-- Details to include; off ones are left out (saved with Save). -->
            <div class="d-flex align-center flex-wrap ga-3 mt-1">
              <span class="text-caption text-medium-emphasis">Include in the message for</span>
              <v-select
                v-model="fieldsBranch"
                density="compact"
                hide-details
                :items="fieldsBranchOptions"
                style="max-width: 240px"
                variant="outlined"
              />
              <template v-if="fieldsBranchRow">
                <span v-if="fieldsBranchRow.hidden_fields" class="text-caption">
                  Its own choice ·
                  <a class="text-primary cursor-pointer" @click="fieldsBranchRow.hidden_fields = null">use the default</a>
                </span>
                <span v-else class="text-caption text-medium-emphasis">Follows the default until you change one</span>
              </template>
            </div>
            <!-- The message part by part: each line its own box, a detail's
                 switch beside it (whatsappMessageParts.js). Hand-written text
                 that isn't made of these parts is edited as one. -->
            <template v-if="!showAsText">
              <div v-for="part in visibleParts" :key="part.key" class="d-flex align-center ga-2 mb-2">
                <v-switch
                  v-if="SWITCHED_FIELDS.has(part.field)"
                  class="flex-grow-0"
                  color="primary"
                  density="compact"
                  :disabled="!canEdit"
                  hide-details
                  inset
                  :model-value="fieldOn(part.field)"
                  :title="fieldOn(part.field) ? 'In the message' : 'Left out'"
                  @update:model-value="togglePart(part)"
                />
                <div v-else class="wm-switch-gap" />
                <v-text-field
                  :class="{ 'wm-part--off': SWITCHED_FIELDS.has(part.field) && !fieldOn(part.field) }"
                  density="compact"
                  :disabled="!canEdit"
                  :hint="partHint(part)"
                  :label="part.title"
                  :model-value="messageParts[part.key]"
                  persistent-hint
                  :placeholder="part.hint"
                  :suffix="PART_SUFFIX[part.key]"
                  variant="outlined"
                  @update:model-value="setPart(part.key, $event)"
                />
              </div>
              <a class="text-caption text-primary cursor-pointer" @click="textMode = true">Edit as one text</a>
            </template>
            <template v-else>
              <v-alert
                v-if="!messageParts"
                class="mb-2"
                density="compact"
                type="info"
                variant="tonal"
              >
                This message was written by hand, so it's edited as one text.
                <a class="text-primary cursor-pointer" @click="resetToParts">Start over from parts</a>
              </v-alert>
              <v-textarea
                v-model="draft[messageKey]"
                auto-grow
                density="compact"
                :disabled="!canEdit"
                hide-details
                :placeholder="pollMode ? defaultMessage : defaultLinkMessage"
                rows="8"
                variant="outlined"
              />
              <a v-if="messageParts" class="text-caption text-primary cursor-pointer" @click="textMode = false">Edit by parts</a>
            </template>
          </v-card>
        </v-col>
        <v-col cols="12" md="5">
          <v-card class="card" variant="flat">
            <div class="card__title mb-2">Preview</div>
            <div class="bubble">
              <img
                v-if="pollMode && draft.whatsapp_message_logo"
                alt=""
                class="bubble__banner"
                :src="bannerURL"
              >
              <div v-html="previewHTML" />
            </div>
            <div v-if="pollMode" class="bubble poll mt-1">
              <div class="font-weight-medium mb-1">{{ pollQuestion }}</div>
              <div class="text-caption poll__hint mb-1">Select one</div>
              <div v-for="o in pollOptions" :key="o" class="poll__option">
                <span class="poll__radio" />{{ o }}
              </div>
            </div>
          </v-card>
        </v-col>

        <!-- Which message goes to which kind of client -->
        <v-col cols="12">
          <v-card class="card" variant="flat">
            <div class="card__title">Who gets what</div>
            <p class="text-caption text-medium-emphasis mb-2">
              Each message can be switched on or off separately for a client messaged for the first time and
              for a returning one (messaged before).
            </p>
            <v-table density="compact">
              <thead>
                <tr>
                  <th />
                  <th class="text-center">New clients</th>
                  <th class="text-center">Returning clients</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="r in audienceRows" :key="r.label">
                  <td>{{ r.label }}</td>
                  <td v-for="k in r.keys" :key="k" class="text-center">
                    <v-switch
                      v-model="draft[k]"
                      class="d-inline-flex"
                      color="primary"
                      density="compact"
                      :disabled="!canEdit"
                      hide-details
                    />
                  </td>
                </tr>
              </tbody>
            </v-table>
          </v-card>
        </v-col>

        <!-- After they answer the poll -->
        <v-col v-if="pollMode" cols="12">
          <v-card class="card" variant="flat">
            <div class="card__title">After they answer</div>
            <p class="text-caption text-medium-emphasis mb-2">
              A client's first answer to the poll gets one message back, asking them to reply (switch it on
              under "Who gets what"). Only one per poll, and only to clients who answered, so it doesn't add
              unprompted messages. What they write in the next 3 days shows on the Reviews page; you answer
              them from the phone as usual.
            </p>
            <v-row v-if="draft.whatsapp_followup || draft.whatsapp_followup_returning" class="mt-1" density="compact">
              <v-col v-for="f in FOLLOWUPS" :key="f.key" cols="12" md="4">
                <v-textarea
                  v-model="draft[`whatsapp_followup_${f.key}`]"
                  auto-grow
                  density="compact"
                  :disabled="!canEdit"
                  :hint="f.hint"
                  :label="f.label"
                  persistent-hint
                  :placeholder="defaultFollowups[f.key]"
                  rows="4"
                  variant="outlined"
                />
              </v-col>
            </v-row>
          </v-card>
        </v-col>

        <!-- Try it -->
        <v-col cols="12">
          <v-card class="card" variant="flat">
            <div class="card__title">Send a test</div>
            <p class="text-caption text-medium-emphasis mb-3">
              Sends the saved message{{ pollMode ? ' and poll' : '' }} to a phone as if a sale had just been
              made at the branch, from that branch's number. It's listed on the Messages tab with a TEST
              invoice{{ pollMode ? ', and the answer shows on the Reviews page' : '' }}. Save your changes first.
            </p>
            <div class="d-flex flex-wrap align-center ga-3">
              <v-text-field
                v-model="testPhone"
                density="compact"
                hide-details
                label="Phone"
                placeholder="+961 70 123456"
                style="max-width: 220px"
                variant="outlined"
              />
              <v-select
                v-model="testBranch"
                density="compact"
                hide-details
                :items="testBranchOptions"
                label="As if sold at"
                style="max-width: 240px"
                variant="outlined"
              />
              <v-btn
                color="primary"
                :disabled="!testPhone.trim() || dirty"
                :loading="testing"
                prepend-icon="mdi-send-outline"
                text="Send test"
                variant="tonal"
                @click="sendTestMessage"
              />
              <span v-if="testResult" class="text-body-2" :class="testResult.ok ? 'text-success' : 'text-error'">
                {{ testResult.text }}
              </span>
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
  import { API_BASE } from '@/config'
  import { composeMessage, MESSAGE_PARTS, parseMessage } from '@/utils/whatsappMessageParts'

  // How post-sale messages go out: our own numbers through the WhatsApp bridge
  // (api/whatsapp_bridge.go), or Meta's Cloud API.

  const props = defineProps({
    settings: { type: Object, default: null },
  })

  const notify = useNotify()
  const { canEdit } = usePageAccess()
  const { getSender, saveSender, sendTest, addNumber, numberQR, updateNumber, deleteNumber } = useWhatsApp()

  const PLACEHOLDERS = ['name', 'invoice', 'total', 'date', 'branch', 'review', 'channel']
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
  const defaultFollowups = ref({})
  // The follow-up for each kind of answer, by what it counts as.
  // One row per message: its switch for a new client, then for a returning one.
  const audienceRows = computed(() => [
    { label: pollMode.value ? 'Message' : 'Message with the review link', keys: ['whatsapp_message_new', 'whatsapp_message_returning'] },
    ...(pollMode.value ? [{ label: 'Rating poll', keys: ['whatsapp_poll_new', 'whatsapp_poll_returning'] }] : []),
    ...(pollMode.value ? [{ label: 'Message after they answer', keys: ['whatsapp_followup', 'whatsapp_followup_returning'] }] : []),
  ])

  const FOLLOWUPS = [
    { key: 'good', label: 'After a top answer', hint: '4 or 5 stars, like 😍 Loved it' },
    { key: 'okay', label: 'After a middle answer', hint: '3 stars, like 🙂 It was okay' },
    { key: 'bad', label: 'After a low answer', hint: '1 or 2 stars, like 😕 Not great: they go on the call-back list' },
  ]
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
    defaultFollowups.value = body.default_followups || {}
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
  // ---- the message, part by part -------------------------------------------
  const messageKey = computed(() => (pollMode.value ? 'whatsapp_bridge_message' : 'whatsapp_bridge_link_message'))
  const currentTemplate = computed(() =>
    draft.value?.[messageKey.value] || (pollMode.value ? defaultMessage.value : defaultLinkMessage.value))
  // null: hand-written text the parts can't make, edited as one.
  const messageParts = computed(() => parseMessage(currentTemplate.value, !pollMode.value))
  const textMode = ref(false)
  const showAsText = computed(() => textMode.value || !messageParts.value)
  const visibleParts = computed(() => MESSAGE_PARTS.filter(p => !p.linkOnly || !pollMode.value))
  const SWITCHED_FIELDS = new Set(['invoice', 'date', 'total', 'branch', 'channel'])
  // What the server puts after a part's text.
  const PART_SUFFIX = { greeting: '{name},', invoice: '↵ {invoice}', date: '{date}', total: '{total}', branch: '{branch}', review: '↵ {review}', channel: '{channel}' }
  function setPart (key, value) {
    draft.value[messageKey.value] = composeMessage({ ...messageParts.value, [key]: value ?? '' }, !pollMode.value)
  }
  // Switching a detail on with nothing written for it starts it off with the
  // usual wording, so it shows up.
  function togglePart (part) {
    const turningOn = !fieldOn(part.field)
    toggleField(part.field)
    if (turningOn && !messageParts.value[part.key]?.trim()) setPart(part.key, part.hint)
  }
  function partHint (part) {
    if (SWITCHED_FIELDS.has(part.field) && fieldOn(part.field) && !messageParts.value[part.key]?.trim()) {
      return 'Empty: not in the message'
    }
    return ''
  }
  function resetToParts () {
    draft.value[messageKey.value] = ''
    textMode.value = false
  }

  // The details that can be switched off (the server's whatsAppMessageFields).
  const MESSAGE_FIELDS = [
    { key: 'invoice', label: 'Receipt code' },
    { key: 'date', label: 'Date' },
    { key: 'total', label: 'Total' },
    { key: 'branch', label: 'Branch' },
    { key: 'channel', label: 'Channel link' },
  ]
  // Whose switches are shown: 0 the default, else a branch, which follows the
  // default (hidden_fields null) until one of its switches is changed.
  const fieldsBranch = ref(0)
  const fieldsBranchOptions = computed(() => [
    { title: 'All branches (default)', value: 0 },
    ...(draft.value?.branches ?? []).map(b => ({
      title: b.hidden_fields ? `${b.branch_name} · own choice` : b.branch_name,
      value: b.branch_id,
    })),
  ])
  const fieldsBranchRow = computed(() => (draft.value?.branches ?? []).find(b => b.branch_id === fieldsBranch.value))
  // Like the server's whatsAppFirstOrReturning: a first message swaps the
  // receipt details for the channel link, a later one the other way round.
  const RECEIPT_FIELDS = ['invoice', 'date', 'total']
  const previewFirst = ref(true)
  function firstOrReturning (hidden, first) {
    const rest = hidden.filter(f => f !== 'channel' && !RECEIPT_FIELDS.includes(f))
    return first
      ? [...rest, ...RECEIPT_FIELDS]
      : [...rest, ...hidden.filter(f => RECEIPT_FIELDS.includes(f)), 'channel']
  }
  const hiddenFields = computed(() => fieldsBranchRow.value?.hidden_fields ?? draft.value?.whatsapp_hidden_fields ?? [])
  const fieldOn = key => !hiddenFields.value.includes(key)
  function toggleField (key) {
    const hidden = hiddenFields.value
    const next = hidden.includes(key) ? hidden.filter(k => k !== key) : [...hidden, key]
    if (fieldsBranchRow.value) fieldsBranchRow.value.hidden_fields = next
    else draft.value.whatsapp_hidden_fields = next
  }

  // Like the server's whatsAppBridgeText: lines whose placeholder is empty
  // (or switched off) go, then labels left with nothing to label, and a
  // blank line doubled by that.
  function fillMessage (message) {
    const values = {
      ...SAMPLE,
      review: pollMode.value ? '' : SAMPLE.review,
      channel: draft.value?.whatsapp_channel_url?.trim() || '',
    }
    if (fieldsBranchRow.value) values.branch = fieldsBranchRow.value.branch_name
    let hidden = hiddenFields.value
    if (draft.value?.whatsapp_channel_first) hidden = firstOrReturning(hidden, previewFirst.value)
    for (const key of hidden) values[key] = ''
    const lines = message.split('\n')
    const dropped = lines.map(line => [...line.matchAll(/\{(\w+)\}/g)].some(m => m[1] in values && !values[m[1]]))
    const isLabel = line => line.trim().endsWith(':') && !/\{\w+\}/.test(line)
    for (let i = lines.length - 1; i >= 0; i--) {
      if (dropped[i] || !isLabel(lines[i]) || !lines[i + 1]?.trim()) continue
      const valueOnly = lines[i + 1].replaceAll(/\{\w+\}/g, '').replaceAll(/[\s*_~]/g, '') === ''
      let rest = true
      for (let j = i + 1; j < lines.length && lines[j].trim(); j++) rest &&= dropped[j]
      dropped[i] = (dropped[i + 1] && valueOnly) || rest
    }
    const out = []
    for (const [i, line] of lines.entries()) {
      if (dropped[i]) continue
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

  // The banner kashi sends on top of the message (api/review_link.go).
  const bannerURL = `${API_BASE}/review-assets/banner.jpg`

  // ---- test ------------------------------------------------------------------
  const testPhone = ref('')
  const testBranch = ref(0)
  const testing = ref(false)
  const testResult = ref(null)
  const testBranchOptions = computed(() => [
    { title: 'No branch (default number)', value: 0 },
    ...(draft.value?.branches ?? []).map(b => ({ title: b.branch_name, value: b.branch_id })),
  ])
  async function sendTestMessage () {
    testing.value = true
    testResult.value = null
    try {
      const res = await sendTest({ phone: testPhone.value.trim(), branch_id: testBranch.value })
      testResult.value = res.sent
        ? { ok: true, text: `Queued${res.from ? ` from "${res.from}"` : ''} — it arrives in about 20–45 seconds.` }
        : { ok: false, text: res.reason || 'Not sent.' }
      await reloadNumbers()
    } catch (error) {
      testResult.value = { ok: false, text: error.message }
    } finally {
      testing.value = false
    }
  }

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
.wm-switch-gap {
  width: 52px;
  flex: none;
}
.wm-part--off {
  opacity: 0.5;
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
.bubble__banner {
  display: block;
  width: calc(100% + 12px);
  margin: -5px -6px 8px;
  border-radius: 6px;
}
:deep(.bubble__link) {
  color: #027eb5;
  text-decoration: underline;
  word-break: break-all;
}
</style>
