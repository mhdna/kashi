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

        <div v-if="!loading && images.length === 0" class="text-medium-emphasis">
          No photos uploaded yet.
        </div>

        <div v-else-if="!loading && filteredImages.length === 0" class="text-medium-emphasis">
          No photos match your search.
        </div>

        <div class="image-grid">
          <v-card
            v-for="image in filteredImages"
            :key="imageKey(image)"
            class="image-card"
            hover
            @click="openPreview(image)"
          >
            <v-img aspect-ratio="1" cover :src="thumbnailUrls[imageKey(image)]">
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
        <!-- Fixed-height stage: the dialog keeps its size while the photo
          loads, then the photo fits inside it. -->
        <div class="image-stage d-flex align-center justify-center">
          <v-progress-circular v-if="previewLoading" indeterminate size="32" />
          <v-img
            v-else-if="previewUrl"
            class="image-stage__img"
            contain
            height="100%"
            :src="previewUrl"
          >
            <template #placeholder>
              <div class="d-flex align-center justify-center fill-height">
                <v-progress-circular indeterminate size="32" />
              </div>
            </template>
          </v-img>
          <span v-else class="text-medium-emphasis">Couldn't load this image.</span>
        </div>
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
  import { useNotify } from '@/composables/useNotify'
  import { useExpenseImages } from '@/composables/useExpenseImages'
  import { useShiftReceiptImages } from '@/composables/useShiftReceiptImages'

  const { fetchExpenseImages, fetchExpenseImageObjectUrl } = useExpenseImages()
  const { fetchShiftReceiptImages, fetchShiftReceiptImageObjectUrl } = useShiftReceiptImages()
  const { ensureConnected, onMessage } = useAdminSocket()
  const notify = useNotify()

  // One entry per real image kind. Every handler takes the whole image row
  // so the generic panel -- and the "All" source below -- can stay kind
  // agnostic. Adding a storage kind is a new entry here plus its composable.
  const KINDS = {
    expenses: {
      title: 'Expense receipts',
      icon: 'mdi-receipt-text-check',
      fetchPage: fetchExpenseImages,
      fetchObjectUrl: (image, opts) => fetchExpenseImageObjectUrl(image.id, opts),
      refreshOn: 'branch_expense_images_uploaded',
      primaryText: image => image.expense_description,
    },
    shiftReceipts: {
      title: 'Shift receipts',
      icon: 'mdi-camera',
      fetchPage: fetchShiftReceiptImages,
      fetchObjectUrl: (image, opts) => fetchShiftReceiptImageObjectUrl(image.id, opts),
      refreshOn: 'branch_shift_receipt_uploaded',
      primaryText: image => `Shift #${image.shift_local_id} · ${image.closing_person_name || '—'}`,
    },
  }

  const KIND_KEYS = Object.keys(KINDS)

  // "All" fans one page request out to every kind, tags each row with its
  // kind, and interleaves them newest-first. Pagination is per-kind, so a
  // "Load more" pulls the next page from each -- ordering across page
  // boundaries is approximate, which is fine for a photo grid.
  const ALL = {
    title: 'All storage',
    icon: 'mdi-view-gallery',
    refreshOn: KIND_KEYS.map(kind => KINDS[kind].refreshOn),
    primaryText: image => KINDS[image.__kind].primaryText(image),
    fetchObjectUrl: (image, opts) => KINDS[image.__kind].fetchObjectUrl(image, opts),
    async fetchPage (pageId, pageSize) {
      const results = await Promise.all(
        KIND_KEYS.map(kind => KINDS[kind].fetchPage(pageId, pageSize)),
      )
      const merged = results.flatMap((data, index) =>
        data.images.map(image => ({ ...image, __kind: KIND_KEYS[index] })),
      ).sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
      const total = results.reduce((sum, data) => sum + data.total, 0)
      return { images: merged, total }
    },
  }

  const SOURCES = { all: ALL, ...KINDS }

  const items = [
    { id: 'all', title: 'All', icon: 'mdi-view-gallery' },
    { id: 'expenses', title: 'Expenses', icon: 'mdi-receipt-text-check' },
    { id: 'shiftReceipts', title: 'Shift receipts', icon: 'mdi-camera' },
  ]

  const activated = ref(['all'])
  const activeNode = computed(() => activated.value[0] ?? null)
  const source = computed(() => (activeNode.value ? SOURCES[activeNode.value] : null))

  const images = ref([])
  const search = ref('')
  const total = ref(0)
  const pageId = ref(0)
  const pageSize = 24
  const loading = ref(false)
  const thumbnailUrls = ref({})
  const previewUrl = ref('')
  const previewLoading = ref(false)

  // The grid tiles are ~160px; ask the server for a long-edge-capped
  // thumbnail (see serveStoredImage) instead of the multi-MB original, and
  // fetch them a few at a time so the first rows paint fast.
  const THUMB_WIDTH = 480
  const THUMB_CONCURRENCY = 5

  // Ids are only unique within a kind, so the "All" grid keys on both.
  function imageKey (image) {
    return image.__kind ? `${image.__kind}:${image.id}` : String(image.id)
  }

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

  function revokePreviewUrl () {
    if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = ''
  }

  async function loadThumbnails (rows) {
    if (!source.value) return
    const pending = rows.filter(image => !thumbnailUrls.value[imageKey(image)])
    let cursor = 0
    async function worker () {
      while (cursor < pending.length) {
        const image = pending[cursor++]
        const key = imageKey(image)
        try {
          thumbnailUrls.value[key] = await source.value.fetchObjectUrl(image, { width: THUMB_WIDTH })
        } catch {
          // Leave it unset -- the v-img placeholder just stays put.
        }
      }
    }
    await Promise.all(
      Array.from({ length: Math.min(THUMB_CONCURRENCY, pending.length) }, worker),
    )
  }

  async function loadImages ({ reset = false } = {}) {
    if (!source.value) return
    loading.value = true
    try {
      const page = reset ? 0 : pageId.value
      const data = await source.value.fetchPage(page, pageSize)
      images.value = reset ? data.images : [...images.value, ...data.images]
      total.value = data.total
      pageId.value = page + 1
      await loadThumbnails(data.images)
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  function loadMore () {
    loadImages()
  }

  async function openPreview (image) {
    previewImage.value = image
    previewDialog.value = true
    revokePreviewUrl()
    previewLoading.value = true
    try {
      const url = await source.value.fetchObjectUrl(image)
      // A newer preview may have been opened while this one was fetching.
      if (previewImage.value !== image) {
        URL.revokeObjectURL(url)
        return
      }
      previewUrl.value = url
    } catch {
      // Fall back to the already-loaded thumbnail rather than a blank dialog.
      if (previewImage.value === image) {
        previewUrl.value = thumbnailUrls.value[imageKey(image)] ?? ''
      }
    } finally {
      if (previewImage.value === image) previewLoading.value = false
    }
  }

  function resetAndLoad () {
    search.value = ''
    images.value = []
    total.value = 0
    pageId.value = 0
    revokeThumbnails()
    loadImages({ reset: true })
  }

  function refreshMatches (messageType) {
    const on = source.value?.refreshOn
    return Array.isArray(on) ? on.includes(messageType) : on === messageType
  }

  watch(activeNode, () => {
    if (source.value) resetAndLoad()
  })

  watch(previewDialog, open => {
    if (!open) revokePreviewUrl()
  })

  onMounted(() => {
    if (source.value) loadImages({ reset: true })
    ensureConnected()
    unsubscribe = onMessage(message => {
      if (source.value && refreshMatches(message.type)) {
        pageId.value = 0
        loadImages({ reset: true })
      }
    })
  })

  onUnmounted(() => {
    unsubscribe?.()
    revokeThumbnails()
    revokePreviewUrl()
  })
</script>

<style scoped>
.image-stage {
  height: min(70vh, 640px);
  width: 100%;
}

.image-stage__img {
  width: 100%;
}

.image-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
}

.image-card {
  cursor: pointer;
}
</style>
