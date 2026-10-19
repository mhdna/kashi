<template>
  <div class="page-root">
    <v-card class="bk-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-help-box-outline" />
        <span>Unknown items</span>
        <v-chip v-if="pending > 0" color="warning" size="small" variant="flat">{{ pending }} to resolve</v-chip>
        <v-spacer />
        <v-btn-toggle
          v-model="view"
          color="primary"
          density="compact"
          mandatory
          variant="outlined"
          @update:model-value="reload"
        >
          <v-btn value="waiting">
            Waiting
            <v-badge
              v-if="waitingCount > 0"
              class="ms-2"
              color="error"
              :content="waitingCount"
              inline
            />
          </v-btn>
          <v-btn value="pending">To resolve</v-btn>
          <v-btn value="resolved">Resolved</v-btn>
        </v-btn-toggle>
        <v-btn prepend-icon="mdi-refresh" text="Reload" variant="text" @click="reload" />
      </v-card-title>
      <v-divider />

      <v-card-text>
        <p v-if="view !== 'waiting'" class="text-body-2 text-medium-emphasis mb-4">
          Items a till sold without a code, with the picture its cashier took. Select the units that are the
          same product, pick the product, and the sold lines are relabelled and the stock is taken.
        </p>

        <template v-if="view === 'waiting'">
          <p class="text-body-2 text-medium-emphasis mb-3">
            Sales being rung up right now that are still missing a picture. If the cashier sent it another way,
            allow those units through and the till takes payment straight away; they stay under “To resolve”.
          </p>
          <div v-if="waiting.length === 0" class="text-center text-medium-emphasis py-10">Nobody is waiting.</div>
          <v-list v-else density="compact" lines="two">
            <v-list-item
              v-for="unit in waiting"
              :key="unit.id"
              @click="toggleWaiting(unit)"
            >
              <template #prepend>
                <v-checkbox-btn
                  v-if="canWaive && isMissing(unit)"
                  density="compact"
                  :model-value="selectedWaiting.includes(unit.id)"
                  @click.stop="toggleWaiting(unit)"
                />
                <v-avatar class="ms-1 me-3" color="surface-variant" rounded="lg" size="56">
                  <img v-if="unit.image_id && thumbs[unit.image_id]" alt="" :src="thumbs[unit.image_id]" @click.stop="preview = unit.image_id">
                  <v-icon v-else-if="isMissing(unit)" icon="mdi-timer-sand" />
                  <v-icon v-else-if="unit.waived_at" icon="mdi-check" />
                  <v-icon
                    v-else-if="unit.image_id && thumbFailed[unit.image_id]"
                    color="warning"
                    icon="mdi-alert-circle-outline"
                    :title="`${thumbFailed[unit.image_id]} — tap to retry`"
                    @click.stop="retryThumb(unit.image_id)"
                  />
                  <v-progress-circular v-else indeterminate size="20" width="2" />
                </v-avatar>
              </template>
              <v-list-item-title>
                {{ unit.branch_name }} · {{ unit.label || 'Unknown item' }}
                <v-chip
                  v-if="unit.help_requested_at"
                  class="ms-2"
                  color="warning"
                  size="x-small"
                  variant="flat"
                >asked for help</v-chip>
              </v-list-item-title>
              <v-list-item-subtitle>
                {{ formatAmount(unit.unit_price) }} · {{ formatWhen(unit.created_at) }} ·
                {{ unit.image_id ? 'picture taken' : unit.waived_at ? 'allowed without a picture' : 'no picture yet' }}
              </v-list-item-subtitle>
            </v-list-item>
          </v-list>
        </template>

        <v-skeleton-loader v-else-if="loading && items.length === 0" type="card, card, card" />
        <div v-else-if="items.length === 0" class="text-center text-medium-emphasis py-10">
          {{ resolved ? 'Nothing resolved yet.' : 'Nothing to resolve.' }}
        </div>

        <div v-if="view !== 'waiting' && items.length > 0" class="unknown-grid">
          <v-card
            v-for="item in items"
            :key="item.id"
            class="unknown-card"
            :class="{ 'unknown-selected': selected.includes(item.id) }"
            variant="outlined"
            @click="toggle(item)"
          >
            <div class="unknown-photo">
              <img v-if="thumbs[item.image_id]" alt="" :src="thumbs[item.image_id]" @click.stop="preview = item.image_id">
              <div v-else-if="!item.image_id" class="text-caption text-medium-emphasis text-center px-2">
                <template v-if="item.waived_at">No picture<br>allowed by {{ item.waived_by || 'a manager' }}</template>
                <template v-else>Picture deleted</template>
              </div>
              <div
                v-else-if="thumbFailed[item.image_id]"
                class="text-caption text-center px-2"
                @click.stop="retryThumb(item.image_id)"
              >
                <v-icon color="warning" icon="mdi-alert-circle-outline" /><br>
                {{ thumbFailed[item.image_id] }}<br>tap to retry
              </div>
              <v-progress-circular v-else indeterminate size="24" width="2" />
              <v-checkbox-btn
                v-if="view === 'pending' && canEdit"
                class="unknown-check"
                density="compact"
                :model-value="selected.includes(item.id)"
                @click.stop="toggle(item)"
              />
            </div>
            <div class="px-3 py-2">
              <div class="d-flex justify-space-between ga-2">
                <span class="font-weight-medium text-body-2">{{ item.branch_name }}</span>
                <span class="text-body-2">{{ formatAmount(item.unit_price) }}<span v-if="item.discount"> −{{ item.discount }}%</span></span>
              </div>
              <div class="text-caption text-medium-emphasis">
                {{ item.branch_invoice_code }} · {{ formatWhen(item.occurred_at) }}
              </div>
              <div v-if="resolved" class="text-caption mt-1">
                → {{ item.resolved_name || 'a product' }}
                <span class="text-medium-emphasis">{{ item.resolved_barcode }}</span>
              </div>
            </div>
          </v-card>
        </div>

        <div v-if="total > pageSize" class="d-flex justify-center mt-4">
          <v-pagination
            v-model="page"
            density="comfortable"
            :length="Math.ceil(total / pageSize)"
            total-visible="5"
            @update:model-value="load"
          />
        </div>
      </v-card-text>

      <v-card-actions v-if="view === 'waiting' && canWaive && selectedWaiting.length > 0" class="unknown-bar px-4 py-3">
        <span class="text-body-2">{{ selectedWaiting.length }} selected</span>
        <v-btn color="warning" :loading="saving" variant="flat" @click="waive">
          Allow {{ selectedWaiting.length }} without a picture
        </v-btn>
        <v-btn variant="text" @click="selectedWaiting = []">Clear</v-btn>
      </v-card-actions>

      <v-card-actions v-if="view === 'pending' && canEdit && selected.length > 0" class="unknown-bar px-4 py-3">
        <span class="text-body-2">{{ selected.length }} selected</span>
        <v-autocomplete
          v-model="variantId"
          class="ms-4"
          clearable
          density="compact"
          hide-details
          item-title="label"
          item-value="id"
          :items="variantItems"
          label="What is it? Search a product, code or barcode"
          :loading="searching"
          no-filter
          style="max-width: 460px"
          variant="outlined"
          @update:search="onSearch"
        />
        <v-btn
          color="primary"
          :disabled="!variantId"
          :loading="saving"
          variant="flat"
          @click="resolve"
        >
          Match {{ selected.length }}
        </v-btn>
        <v-btn variant="text" @click="selected = []">Clear</v-btn>
      </v-card-actions>
    </v-card>

    <v-dialog max-width="900" :model-value="preview != null" @update:model-value="preview = null">
      <v-card>
        <img v-if="full[preview]" alt="" class="unknown-full" :src="full[preview]">
        <v-progress-linear v-else indeterminate />
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
  import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { useUnknownItems } from '@/composables/useUnknownItems'
  import { useAuthStore } from '@/stores/auth'
  import { formatAmount } from '@/utils/money'

  const notify = useNotify()
  const { canEdit } = usePageAccess()
  const { fetchUnknownItems, fetchWaiting, waiveUnknownItems, searchVariants, resolveUnknownItems, fetchImageObjectUrl } = useUnknownItems()
  const auth = useAuthStore()
  const canWaive = computed(() => auth.canEdit('unknown_item_waive'))

  const view = ref('pending')
  const resolved = computed(() => view.value === 'resolved')
  const waiting = ref([])
  const waitingCount = ref(0)
  const selectedWaiting = ref([])
  const items = ref([])
  const total = ref(0)
  const pending = ref(0)
  const page = ref(1)
  const pageSize = 24
  const loading = ref(false)
  const selected = ref([])
  const thumbs = ref({})
  // Pictures that wouldn't load (by image id), so a failure shows instead of
  // an endless spinner; tapping one tries again.
  const thumbFailed = ref({})
  const full = ref({})
  const preview = ref(null)

  const variantId = ref(null)
  const variants = ref([])
  const searching = ref(false)
  const saving = ref(false)
  let searchTimer = null

  const variantItems = computed(() => variants.value.map(v => ({
    id: v.id,
    label: [v.product_name, v.color_name, v.size_name].filter(Boolean).join(' / ') + ` · ${v.product_code} · ${v.barcode}`,
  })))

  function formatWhen (value) {
    return new Date(value).toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' })
  }

  function toggle (item) {
    if (resolved.value || !canEdit.value) return
    selected.value = selected.value.includes(item.id)
      ? selected.value.filter(id => id !== item.id)
      : [...selected.value, item.id]
  }

  // A unit still needs a picture or a pass (pictured and allowed ones are
  // listed too, for context).
  function isMissing (unit) {
    return !unit.image_id && !unit.waived_at
  }

  function toggleWaiting (unit) {
    if (!canWaive.value || !isMissing(unit)) return
    selectedWaiting.value = selectedWaiting.value.includes(unit.id)
      ? selectedWaiting.value.filter(id => id !== unit.id)
      : [...selectedWaiting.value, unit.id]
  }

  async function loadWaiting () {
    try {
      waiting.value = await fetchWaiting()
      waitingCount.value = waiting.value.filter(unit => isMissing(unit)).length
      loadThumbs(waiting.value)
      selectedWaiting.value = selectedWaiting.value.filter(id => waiting.value.some(u => u.id === id))
    } catch {
      // The next poll tries again.
    }
  }

  async function waive () {
    saving.value = true
    try {
      const done = await waiveUnknownItems(selectedWaiting.value)
      notify.success(`${done} item(s) allowed without a picture`)
      selectedWaiting.value = []
      await loadWaiting()
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }

  async function load () {
    if (view.value === 'waiting') return loadWaiting()
    loadWaiting()
    loading.value = true
    try {
      const data = await fetchUnknownItems({ resolved: resolved.value, pageId: page.value - 1, pageSize })
      items.value = data.items
      total.value = data.total
      pending.value = data.pending
      selected.value = []
      loadThumbs(data.items)
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  // Units sharing one picture share one download.
  function loadThumbs (list) {
    for (const item of list) {
      if (!item.image_id || item.image_id in thumbs.value) continue
      thumbs.value[item.image_id] = ''
      fetchImageObjectUrl(item.image_id, { width: 480 })
        .then(url => { thumbs.value = { ...thumbs.value, [item.image_id]: url } })
        .catch(error => { thumbFailed.value = { ...thumbFailed.value, [item.image_id]: error.message } })
    }
  }

  function retryThumb (imageId) {
    const { [imageId]: _failed, ...rest } = thumbFailed.value
    thumbFailed.value = rest
    delete thumbs.value[imageId]
    loadThumbs([{ image_id: imageId }])
  }

  function reload () {
    page.value = 1
    return load()
  }

  function onSearch (text) {
    clearTimeout(searchTimer)
    if (!text || text.length < 2) return
    searchTimer = setTimeout(async () => {
      searching.value = true
      try {
        variants.value = await searchVariants(text)
      } catch {
        variants.value = []
      } finally {
        searching.value = false
      }
    }, 250)
  }

  async function resolve () {
    saving.value = true
    try {
      await resolveUnknownItems(selected.value, variantId.value)
      notify.success(`${selected.value.length} item(s) matched`)
      variantId.value = null
      await load()
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }

  watch(preview, async id => {
    if (id == null || full.value[id]) return
    try {
      full.value = { ...full.value, [id]: await fetchImageObjectUrl(id, { width: 1280 }) }
    } catch {
      preview.value = null
    }
  })

  // Cashiers wait on the Waiting tab, so it refreshes on its own.
  let waitingTimer = null
  onMounted(() => {
    load()
    waitingTimer = setInterval(loadWaiting, 4000)
  })
  const { onMessage } = useAdminSocket()
  const offMessage = onMessage(message => {
    if (message.type === 'unknown_item_help_requested') loadWaiting()
  })
  onBeforeUnmount(() => {
    offMessage()
    clearInterval(waitingTimer)
    clearTimeout(searchTimer)
    for (const url of [...Object.values(thumbs.value), ...Object.values(full.value)]) {
      if (url) URL.revokeObjectURL(url)
    }
  })
</script>

<style scoped>
  .unknown-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 12px;
  }
  .unknown-card { cursor: pointer; overflow: hidden; }
  .unknown-selected { border-color: rgb(var(--v-theme-primary)); box-shadow: 0 0 0 1px rgb(var(--v-theme-primary)); }
  .unknown-photo {
    position: relative;
    aspect-ratio: 4 / 3;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(var(--v-theme-on-surface), 0.06);
  }
  .unknown-photo img { width: 100%; height: 100%; object-fit: cover; }
  .unknown-check { position: absolute; top: 4px; left: 4px; background: rgb(var(--v-theme-surface)); border-radius: 8px; }
  .unknown-full { width: 100%; max-height: 80vh; object-fit: contain; display: block; }
  .unknown-bar {
    position: sticky;
    bottom: 0;
    flex-wrap: wrap;
    gap: 8px;
    background: rgb(var(--v-theme-surface));
    border-top: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  }
</style>
