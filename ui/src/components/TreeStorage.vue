<template>
  <div class="d-flex" style="height: 100%">
    <!-- PageSidebar folds into a chip strip above the grid on phones. -->
    <PageSidebar
      v-model="selectedKind"
      all-icon="mdi-view-gallery"
      :items="items"
      show-all
      :width="240"
    />

    <v-divider vertical />

    <div class="flex-grow-1 pa-4" style="overflow-y: auto">
      <template v-if="source">
        <div class="d-flex flex-wrap align-center justify-space-between ga-2 mb-4">
          <div class="text-h6 text-no-wrap">{{ source.title }}</div>
          <div class="d-flex align-center ga-2 storage-search">
            <v-text-field
              v-model="search"
              clearable
              density="compact"
              hide-details
              placeholder="Search"
              prepend-inner-icon="mdi-magnify"
              variant="outlined"
            />
            <v-btn
              v-if="canDelete"
              color="error"
              :disabled="selectedIds.length === 0"
              prepend-icon="mdi-delete"
              size="small"
              variant="tonal"
              @click="confirmDelete = true"
            >
              Delete{{ selectedIds.length > 0 ? ` (${selectedIds.length})` : '' }}
            </v-btn>
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
            <v-checkbox-btn
              v-if="canDelete"
              class="image-select"
              density="compact"
              :model-value="selectedIds.includes(image.id)"
              @click.stop="toggleSelected(image.id)"
            />
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

    <v-dialog v-model="confirmDelete" max-width="420">
      <v-card>
        <v-card-title>Delete {{ selectedIds.length }} picture(s)?</v-card-title>
        <v-card-text>
          The pictures are removed for good. Pictures of items from a sale that are still waiting to be matched are kept.
          Items already matched keep their match.
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="confirmDelete = false">Cancel</v-btn>
          <v-btn color="error" :loading="deleting" variant="flat" @click="deleteSelected">Delete</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="previewDialog" max-width="720">
      <v-card>
        <!-- Fixed-height stage: the dialog keeps its size while the photo
          loads, then the photo fits inside it. -->
        <div class="image-stage d-flex align-center justify-center">
          <v-progress-circular v-if="previewLoading" indeterminate size="32" />
          <ZoomableImage
            v-else-if="previewUrl"
            :busy="changingPhoto"
            height="100%"
            :progress="uploadProgress"
            :replaceable="canReplace && !!previewPhotoKind"
            :rotatable="canRotate && !!previewPhotoKind"
            :src="previewUrl"
            @replace="file => changePreview((kind, id) => replacePhoto(kind, id, file))"
            @rotate="delta => changePreview((kind, id) => rotatePhoto(kind, id, delta))"
          />
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
  import PageSidebar from '@/components/PageSidebar.vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { useExpenseImages } from '@/composables/useExpenseImages'
  import { useNotify } from '@/composables/useNotify'
  import { usePhotoActions } from '@/composables/usePhotoActions'
  import { useShiftReceiptImages } from '@/composables/useShiftReceiptImages'
  import { useTransferAttachments } from '@/composables/useTransferAttachments'
  import { useUnknownItems } from '@/composables/useUnknownItems'

  const { fetchExpenseImages, fetchExpenseImageObjectUrl } = useExpenseImages()
  const { fetchShiftReceiptImages, fetchShiftReceiptImageObjectUrl } = useShiftReceiptImages()
  const { fetchTransferAttachments, fetchTransferAttachmentObjectUrl } = useTransferAttachments()
  const { fetchUnknownItemImages, deleteUnknownItemImages, fetchImageObjectUrl: fetchUnknownItemImageObjectUrl } = useUnknownItems()
  const { ensureConnected, onMessage } = useAdminSocket()
  const { canRotate, canReplace, rotatePhoto, replacePhoto, uploadProgress } = usePhotoActions()
  const notify = useNotify()

  // One entry per real image kind. Every handler takes the whole image row
  // so the generic panel -- and the "All" source below -- can stay kind
  // agnostic. Adding a storage kind is a new entry here plus its composable.
  // photoKind is the kind usePhotoActions rotates or replaces it as; none
  // means the photo can't be turned or swapped.
  const KINDS = {
    expenses: {
      title: 'Expense receipts',
      icon: 'mdi-receipt-text-check',
      fetchPage: fetchExpenseImages,
      fetchObjectUrl: (image, opts) => fetchExpenseImageObjectUrl(image.id, opts),
      refreshOn: 'branch_expense_images_uploaded',
      photoKind: 'expense_image',
      primaryText: image => image.expense_description,
    },
    shiftReceipts: {
      title: 'Shift receipts',
      icon: 'mdi-camera',
      fetchPage: fetchShiftReceiptImages,
      fetchObjectUrl: (image, opts) => fetchShiftReceiptImageObjectUrl(image.id, opts),
      refreshOn: 'branch_shift_receipt_uploaded',
      photoKind: 'shift_receipt',
      primaryText: image => `Shift #${image.shift_local_id} · ${image.closing_person_name || '—'}`,
    },
    transfers: {
      title: 'Transfer photos',
      icon: 'mdi-truck-delivery',
      fetchPage: fetchTransferAttachments,
      fetchObjectUrl: (image, opts) => fetchTransferAttachmentObjectUrl(image.id, opts),
      refreshOn: 'branch_transfer_photo_uploaded',
      photoKind: 'transfer',
      primaryText: image => `${image.code || `Transfer #${image.id}`} · ${image.from_inventory_name} → ${image.to_inventory_name}`,
    },
    unknownItems: {
      title: 'Unknown item photos',
      icon: 'mdi-help-box-outline',
      fetchPage: fetchUnknownItemImages,
      fetchObjectUrl: (image, opts) => fetchUnknownItemImageObjectUrl(image.id, opts),
      refreshOn: 'unknown_item_uploaded',
      photoKind: null,
      deletable: true,
      primaryText: image => `Unknown item · $${(image.unit_price / 100).toFixed(2)}${image.branch_invoice_code ? ` · ${image.branch_invoice_code}` : ''}`,
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
      // A kind the role can't read (e.g. transfers need Inventory) just
      // drops out instead of failing the whole view.
      const settled = await Promise.allSettled(
        KIND_KEYS.map(kind => KINDS[kind].fetchPage(pageId, pageSize)),
      )
      const results = settled.map(r => (r.status === 'fulfilled' ? r.value : { images: [], total: 0 }))
      const merged = results.flatMap((data, index) =>
        data.images.map(image => ({ ...image, __kind: KIND_KEYS[index] })),
      ).sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
      const total = results.reduce((sum, data) => sum + data.total, 0)
      return { images: merged, total }
    },
  }

  const SOURCES = { all: ALL, ...KINDS }

  const items = [
    { value: 'expenses', title: 'Expenses', prependIcon: 'mdi-receipt-text-check' },
    { value: 'shiftReceipts', title: 'Shift receipts', prependIcon: 'mdi-camera' },
    { value: 'transfers', title: 'Transfers', prependIcon: 'mdi-truck-delivery' },
    { value: 'unknownItems', title: 'Unknown items', prependIcon: 'mdi-help-box-outline' },
  ]

  // null is PageSidebar's "All" row.
  const selectedKind = ref(null)
  const activeNode = computed(() => selectedKind.value ?? 'all')
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

  // Delete is offered on the kinds that support it (unknown items), to
  // whoever may replace photos.
  const canDelete = computed(() => canReplace.value && !!source.value?.deletable)
  const selectedIds = ref([])
  const confirmDelete = ref(false)
  const deleting = ref(false)

  function toggleSelected (id) {
    selectedIds.value = selectedIds.value.includes(id)
      ? selectedIds.value.filter(selected => selected !== id)
      : [...selectedIds.value, id]
  }

  async function deleteSelected () {
    deleting.value = true
    try {
      const { deleted, kept } = await deleteUnknownItemImages(selectedIds.value)
      notify.success(kept
        ? `${deleted} deleted, ${kept} kept (their sale is still waiting to be matched)`
        : `${deleted} deleted`)
      confirmDelete.value = false
      selectedIds.value = []
      resetAndLoad()
    } catch (error) {
      notify.error(error.message)
    } finally {
      deleting.value = false
    }
  }

  const previewDialog = ref(false)
  const previewImage = ref(null)
  // The photo kind the open preview can be turned or replaced as (none for
  // kinds that can't be).
  const previewPhotoKind = computed(() => {
    const image = previewImage.value
    if (!image) return null
    return (image.__kind ? KINDS[image.__kind] : source.value)?.photoKind ?? null
  })
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

  // Saves a rotation or replacement (save gets the photo's kind and id),
  // then re-fetches the photo and its grid thumbnail as the server now
  // serves them.
  const changingPhoto = ref(false)
  async function changePreview (save) {
    const image = previewImage.value
    if (!image) return
    const kind = image.__kind ? KINDS[image.__kind] : source.value
    changingPhoto.value = true
    try {
      await save(kind.photoKind, image.id)
      const [url, thumb] = await Promise.all([
        source.value.fetchObjectUrl(image),
        source.value.fetchObjectUrl(image, { width: THUMB_WIDTH }).catch(() => null),
      ])
      const key = imageKey(image)
      if (thumb) {
        if (thumbnailUrls.value[key]) URL.revokeObjectURL(thumbnailUrls.value[key])
        thumbnailUrls.value[key] = thumb
      }
      if (previewImage.value !== image) {
        URL.revokeObjectURL(url)
        return
      }
      revokePreviewUrl()
      previewUrl.value = url
    } catch (error) {
      notify.error(error.message)
    } finally {
      changingPhoto.value = false
    }
  }

  function resetAndLoad () {
    search.value = ''
    selectedIds.value = []
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
.storage-search {
  flex: 1 1 200px;
  max-width: 320px;
}

.image-stage {
  height: min(70vh, 640px);
  width: 100%;
}

.image-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
}

.image-card {
  cursor: pointer;
  position: relative;
}

.image-select {
  background: rgba(var(--v-theme-surface), 0.85);
  border-radius: 8px;
  left: 6px;
  position: absolute;
  top: 6px;
  z-index: 1;
}
</style>
