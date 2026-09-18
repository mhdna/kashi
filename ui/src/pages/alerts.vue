<template>
  <div class="page-root">
    <v-card class="alerts-card" flat>
      <v-card-title class="page-heading d-flex flex-wrap align-center ga-3 px-4 py-3">
        <v-icon icon="mdi-alert-circle-outline" />
        <span>Alerts</span>
        <v-chip-group v-model="viewIndex" class="ms-2" mandatory selected-class="text-primary">
          <v-chip filter size="small" variant="tonal">Active</v-chip>
          <v-chip filter size="small" variant="tonal">
            Ignored
            <v-badge
              v-if="ignoredCount"
              class="ml-2"
              color="grey"
              :content="ignoredCount"
              inline
            />
          </v-chip>
        </v-chip-group>
        <v-spacer />
        <v-btn
          prepend-icon="mdi-cog-outline"
          variant="text"
          @click="defaultDialog = true"
        >
          Default threshold: {{ defaultThresholdSaved ?? '—' }}
        </v-btn>
      </v-card-title>
      <v-divider />

      <div class="alerts-content">
        <div class="pa-4 alerts-body">
          <!-- Active view: Out of Stock and Low Stock side by side, each an
            infinite-scroll column with its own search. Tap a row to edit its
            threshold or mute it. -->
          <v-row class="alerts-columns" density="compact">
            <template v-if="!showIgnored">
              <v-col cols="12" md="6">
                <AlertList
                  ref="outListRef"
                  color="error"
                  empty-text="Nothing is out of stock."
                  :endpoint="activeURL"
                  icon="mdi-close-circle"
                  status="out_of_stock"
                  title="Out of Stock"
                  @select="item => openEdit(item, false)"
                />
              </v-col>
              <v-col cols="12" md="6">
                <AlertList
                  ref="lowListRef"
                  color="warning"
                  empty-text="Nothing is running low."
                  :endpoint="activeURL"
                  icon="mdi-alert"
                  status="low_stock"
                  title="Low Stock"
                  @select="item => openEdit(item, false)"
                />
              </v-col>
            </template>
            <v-col v-else cols="12">
              <AlertList
                ref="ignoredListRef"
                color="grey"
                empty-text="No ignored products."
                :endpoint="ignoredURL"
                icon="mdi-bell-off-outline"
                title="Ignored Products"
                @select="item => openEdit(item, true)"
              />
            </v-col>
          </v-row>
        </div>
      </div>
    </v-card>

    <v-dialog v-model="defaultDialog" max-width="380">
      <v-card>
        <v-card-title class="d-flex align-center justify-space-between ga-2">
          Default low-stock threshold
          <v-icon-btn icon="mdi-close" variant="text" @click="defaultDialog = false" />
        </v-card-title>
        <v-card-text>
          <p class="text-caption text-medium-emphasis mb-4">
            Applied to every product that has no threshold of its own.
          </p>
          <v-text-field
            v-model.number="defaultThresholdInput"
            density="compact"
            hide-details
            label="Threshold"
            min="0"
            type="number"
            variant="outlined"
            @keyup.enter="saveDefaultThreshold"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Cancel" variant="text" @click="defaultDialog = false" />
          <v-btn
            color="primary"
            :disabled="!defaultThresholdDirty"
            :loading="savingDefault"
            text="Save"
            @click="saveDefaultThreshold"
          />
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="editOpen" max-width="420">
      <v-card v-if="editItem">
        <v-card-title class="d-flex align-center justify-space-between ga-2">
          <span class="text-truncate">{{ editItem.name }}</span>
          <v-icon-btn icon="mdi-close" variant="text" @click="editOpen = false" />
        </v-card-title>
        <v-card-text>
          <div class="text-caption text-medium-emphasis mb-4">
            {{ editItem.code }} · On hand: {{ editItem.on_hand }}
          </div>
          <v-text-field
            v-model.number="editThreshold"
            density="compact"
            hide-details
            label="Low-stock threshold"
            min="0"
            type="number"
            variant="outlined"
          />
          <div class="text-caption text-medium-emphasis mt-2">
            {{ editHasOverride
              ? 'Custom threshold set for this product.'
              : `Currently using the default (${defaultThresholdSaved ?? '—'}).` }}
          </div>
        </v-card-text>
        <v-card-actions>
          <v-btn
            v-if="editHasOverride"
            :loading="editSaving"
            text="Reset to default"
            variant="text"
            @click="resetEditThreshold"
          />
          <v-spacer />
          <v-btn
            :loading="editMuting"
            :prepend-icon="editMuted ? 'mdi-bell-outline' : 'mdi-bell-off-outline'"
            :text="editMuted ? 'Resume alerts' : 'Mute alerts'"
            variant="tonal"
            @click="toggleEditMute"
          />
          <v-btn color="primary" :loading="editSaving" text="Save" @click="saveEditThreshold" />
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue'
  import AlertList from '@/components/Alerts/AlertList.vue'
  import { useAppSettings } from '@/composables/useAppSettings'
  import { useLowStockAlerts } from '@/composables/useLowStockAlerts'
  import { API_BASE } from '@/config'

  const { getAppSettings } = useAppSettings()
  const { fetchIgnoredCount, setProductThreshold, setDefaultThreshold, setAlertsEnabled } = useLowStockAlerts()

  const activeURL = `${API_BASE}/low_stock`
  const ignoredURL = `${API_BASE}/low_stock/ignored`

  const viewIndex = ref(0)
  const showIgnored = computed(() => viewIndex.value === 1)

  const outListRef = ref(null)
  const lowListRef = ref(null)
  const ignoredListRef = ref(null)

  function refreshLists () {
    outListRef.value?.reload()
    lowListRef.value?.reload()
    ignoredListRef.value?.reload()
    loadIgnoredCount()
  }

  const ignoredCount = ref(0)

  async function loadIgnoredCount () {
    try {
      ignoredCount.value = await fetchIgnoredCount()
    } catch {
      ignoredCount.value = 0
    }
  }

  const defaultThresholdInput = ref(null)
  const defaultThresholdSaved = ref(null)
  const defaultThresholdDirty = computed(() =>
    defaultThresholdInput.value != null
    && Number(defaultThresholdInput.value) !== Number(defaultThresholdSaved.value),
  )
  const savingDefault = ref(false)
  const defaultDialog = ref(false)

  async function loadDefaultThreshold () {
    try {
      const settings = await getAppSettings()
      defaultThresholdSaved.value = settings?.default_low_stock_threshold ?? 5
      defaultThresholdInput.value = defaultThresholdSaved.value
    } catch {
      defaultThresholdSaved.value = 5
      defaultThresholdInput.value = 5
    }
  }

  async function saveDefaultThreshold () {
    if (defaultThresholdInput.value == null) return
    savingDefault.value = true
    try {
      const settings = await setDefaultThreshold(Number(defaultThresholdInput.value))
      defaultThresholdSaved.value = settings?.default_low_stock_threshold ?? defaultThresholdInput.value
      defaultDialog.value = false
      refreshLists()
    } finally {
      savingDefault.value = false
    }
  }

  // Go's sql.NullInt64 marshals as {Int64, Valid} (no json tags on that
  // type) -- mirrors the nullableString pattern used elsewhere in this app.
  function nullableInt (value) {
    if (value == null) return null
    if (typeof value !== 'object') return value
    const valid = value.Valid ?? value.valid
    if (valid === false) return null
    return value.Int64 ?? value.int64 ?? null
  }

  const editOpen = ref(false)
  const editItem = ref(null)
  const editMuted = ref(false)
  const editThreshold = ref(null)
  const editSaving = ref(false)
  const editMuting = ref(false)

  const editHasOverride = computed(() => nullableInt(editItem.value?.low_stock_threshold) != null)

  function openEdit (item, muted) {
    editItem.value = item
    editMuted.value = muted
    editThreshold.value = nullableInt(item.low_stock_threshold) ?? item.effective_threshold ?? defaultThresholdSaved.value
    editOpen.value = true
  }

  async function saveEditThreshold () {
    if (!editItem.value || editThreshold.value == null || editThreshold.value === '') return
    editSaving.value = true
    try {
      await setProductThreshold(editItem.value.id, Number(editThreshold.value))
      editOpen.value = false
      refreshLists()
    } finally {
      editSaving.value = false
    }
  }

  async function resetEditThreshold () {
    if (!editItem.value) return
    editSaving.value = true
    try {
      await setProductThreshold(editItem.value.id, null)
      editOpen.value = false
      refreshLists()
    } finally {
      editSaving.value = false
    }
  }

  async function toggleEditMute () {
    if (!editItem.value) return
    editMuting.value = true
    try {
      await setAlertsEnabled(editItem.value.id, editMuted.value)
      editOpen.value = false
      refreshLists()
    } finally {
      editMuting.value = false
    }
  }

  onMounted(() => {
    loadDefaultThreshold()
    loadIgnoredCount()
  })
</script>

<style scoped>
.page-root {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.alerts-card {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.alerts-content {
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}
/* Let the columns row grow into whatever height is left below the KPI
   cards; each AlertList caps itself with a max-height. */
.alerts-body {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.alerts-columns {
  flex: 1 1 auto;
  min-height: 0;
}
.alerts-columns > :deep(.v-col-12) {
  display: flex;
  min-height: 0;
}
</style>
