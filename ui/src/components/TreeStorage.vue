<template>
  <div class="d-flex" style="height: 100%">
    <v-sheet class="flex-shrink-0" width="240">
      <v-treeview
        v-model:activated="activated"
        activatable
        item-value="id"
        :items="items"
        open-on-click
      >
        <template #prepend="{ item }">
          <v-icon :icon="item.icon" />
        </template>
      </v-treeview>
    </v-sheet>

    <v-divider vertical />

    <div class="flex-grow-1 pa-4" style="overflow-y: auto">
      <template v-if="source">
        <div class="d-flex align-center justify-space-between mb-4">
          <div class="text-h6">{{ source.title }}</div>
          <div class="d-flex align-center ga-2">
            <v-text-field
              v-model="search"
              clearable
              density="compact"
              hide-details
              placeholder="Search receipts"
              prepend-inner-icon="mdi-magnify"
              variant="outlined"
              width="280"
            />
            <v-progress-circular v-if="loading" indeterminate size="20" />
          </div>
        </div>

        <v-alert v-if="loadError" class="mb-4" type="error" variant="tonal">{{ loadError }}</v-alert>

        <div v-if="!loading && images.length === 0" class="text-medium-emphasis">
          No photos uploaded yet.
        </div>

        <div v-else-if="!loading && filteredImages.length === 0" class="text-medium-emphasis">
          No photos match your search.
        </div>

        <div class="image-grid">
          <v-card
            v-for="image in filteredImages"
            :key="image.id"
            class="image-card"
            hover
            @click="openPreview(image)"
          >
            <v-img aspect-ratio="1" cover :src="thumbnailUrls[image.id]">
              <template #placeholder>
                <div class="d-flex align-center justify-center fill-height">
                  <v-progress-circular indeterminate size="24" />
                </div>
              </template>
            </v-img>
            <v-card-text class="pa-2">
              <div class="text-body-2 text-truncate">{{ source.primaryText(image) }}</div>
              <div class="text-caption text-medium-emphasis">{{ image.branch_name }}</div>
              <div class="text-caption text-medium-emphasis">{{ formatDate(image.created_at) }}</div>
            </v-card-text>
          </v-card>
        </div>

        <div v-if="!loading && images.length < total" class="text-center mt-4">
          <v-btn variant="text" @click="loadMore">Load more</v-btn>
        </div>
      </template>

      <template v-else>
        <div class="text-medium-emphasis">Select a folder on the left.</div>
      </template>
    </div>

    <v-dialog v-model="previewDialog" max-width="720">
      <v-card>
        <v-img v-if="previewImage" contain max-height="80vh" :src="thumbnailUrls[previewImage.id]" />
        <v-card-text v-if="previewImage">
          <div class="text-subtitle-1">{{ source?.primaryText(previewImage) }}</div>
          <div class="text-caption text-medium-emphasis">
            {{ previewImage.branch_name }} — {{ formatDate(previewImage.created_at) }}
          </div>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="previewDialog = false">Close</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
  import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useExpenseImages } from '@/composables/useExpenseImages'
  import { useShiftReceiptImages } from '@/composables/useShiftReceiptImages'

  const { fetchExpenseImages, fetchExpenseImageObjectUrl } = useExpenseImages()
  const { fetchShiftReceiptImages, fetchShiftReceiptImageObjectUrl } = useShiftReceiptImages()
  const { ensureConnected, onMessage } = useAdminSocket()

  // Each tree node is one image source. Adding a storage kind later is a
  // new entry here plus its composable -- the panel below is generic.
  const SOURCES = {
    expenses: {
      title: 'Expense receipts',
      icon: 'mdi-receipt-text-check',
      fetchPage: fetchExpenseImages,
      fetchObjectUrl: fetchExpenseImageObjectUrl,
      refreshOn: 'branch_expense_images_uploaded',
      primaryText: image => image.expense_description,
    },
    shiftReceipts: {
      title: 'Shift receipts',
      icon: 'mdi-camera',
      fetchPage: fetchShiftReceiptImages,
      fetchObjectUrl: fetchShiftReceiptImageObjectUrl,
      refreshOn: 'branch_shift_receipt_uploaded',
      primaryText: image => `Shift #${image.shift_local_id} · ${image.closing_person_name || '—'}`,
    },
  }

  const items = [
    { id: 'expenses', title: 'Expenses', icon: 'mdi-receipt-text-check' },
    { id: 'shiftReceipts', title: 'Shift receipts', icon: 'mdi-camera' },
  ]

  const activated = ref(['expenses'])
  const activeNode = computed(() => activated.value[0] ?? null)
  const source = computed(() => (activeNode.value ? SOURCES[activeNode.value] : null))

  const images = ref([])
  const search = ref('')
  const total = ref(0)
  const pageId = ref(0)
  const pageSize = 24
  const loading = ref(false)
  const loadError = ref('')
  const thumbnailUrls = ref({})

  const filteredImages = computed(() => {
    const query = search.value.trim().toLowerCase()
    if (!query) return images.value

    return images.value.filter(image => [
      source.value?.primaryText(image),
      image.branch_name,
      formatDate(image.created_at),
    ].some(value => String(value ?? '').toLowerCase().includes(query)))
  })

  const previewDialog = ref(false)
  const previewImage = ref(null)
  let unsubscribe = null

  function formatDate (value) {
    return new Date(value).toLocaleString()
  }

  function revokeThumbnails () {
    for (const url of Object.values(thumbnailUrls.value)) URL.revokeObjectURL(url)
    thumbnailUrls.value = {}
  }

  async function loadThumbnails (rows) {
    if (!source.value) return
    await Promise.all(rows.map(async image => {
      if (thumbnailUrls.value[image.id]) return
      try {
        thumbnailUrls.value[image.id] = await source.value.fetchObjectUrl(image.id)
      } catch {
        // Leave it unset -- the v-img placeholder just stays put.
      }
    }))
  }

  async function loadImages ({ reset = false } = {}) {
    if (!source.value) return
    loading.value = true
    loadError.value = ''
    try {
      const page = reset ? 0 : pageId.value
      const data = await source.value.fetchPage(page, pageSize)
      images.value = reset ? data.images : [...images.value, ...data.images]
      total.value = data.total
      pageId.value = page + 1
      await loadThumbnails(data.images)
    } catch (error) {
      loadError.value = error.message
    } finally {
      loading.value = false
    }
  }

  function loadMore () {
    loadImages()
  }

  function openPreview (image) {
    previewImage.value = image
    previewDialog.value = true
  }

  function resetAndLoad () {
    search.value = ''
    images.value = []
    total.value = 0
    pageId.value = 0
    revokeThumbnails()
    loadImages({ reset: true })
  }

  watch(activeNode, () => {
    if (source.value) resetAndLoad()
  })

  onMounted(() => {
    if (source.value) loadImages({ reset: true })
    ensureConnected()
    unsubscribe = onMessage(message => {
      if (source.value && message.type === source.value.refreshOn) {
        pageId.value = 0
        loadImages({ reset: true })
      }
    })
  })

  onUnmounted(() => {
    unsubscribe?.()
    revokeThumbnails()
  })
</script>

<style scoped>
.image-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
}

.image-card {
  cursor: pointer;
}
</style>
