<template>
  <div>
    <h3 class="text-subtitle-1 mb-1">Google reviews</h3>
    <p class="text-caption text-medium-emphasis mb-1">
      The Google side of reviews: links to your Google review box in the message (instead of the poll),
      and each place's rating and latest reviews on the Reviews page. Off, the page shows only the poll.
    </p>
    <v-switch
      v-if="draft"
      v-model="draft.google_reviews_enabled"
      color="primary"
      density="compact"
      :disabled="!canEdit"
      hide-details
      label="Show Google reviews"
    />
    <p v-if="draft?.google_reviews_enabled" class="text-caption text-medium-emphasis mt-2 mb-3">
      Each message's link opens the Google "write a review" box of the branch that made the sale:
      its own place when set, else the main one. Paste a place id, or a whole review link.
      WhatsApp shows the title and line below on the link's card.
    </p>
    <v-form v-if="draft" :disabled="!canEdit" @submit.prevent="save">
      <v-row v-if="draft.google_reviews_enabled" density="compact">
        <v-col cols="12" md="6">
          <v-text-field
            v-model="draft.review_place_id"
            density="compact"
            hint="Google place id, e.g. ChIJ…"
            label="Main Google place"
            persistent-hint
            variant="outlined"
          />
        </v-col>
        <v-col cols="12" md="6">
          <v-text-field
            v-model="draft.review_base_url"
            density="compact"
            hint="Where the links live; each message adds its own code"
            label="Link address"
            persistent-hint
            placeholder="https://grandbz.com/review"
            variant="outlined"
          />
        </v-col>
        <v-col cols="12" md="6">
          <v-text-field
            v-model="draft.review_title"
            density="compact"
            hide-details
            label="Card title"
            placeholder="Grand Bazaar Feedback"
            variant="outlined"
          />
        </v-col>
        <v-col cols="12" md="6">
          <v-text-field
            v-model="draft.review_description"
            density="compact"
            hide-details
            label="Card line (optional)"
            variant="outlined"
          />
        </v-col>
        <v-col cols="12" md="6">
          <v-text-field
            v-model="draft.google_places_api_key"
            :append-inner-icon="showKey ? 'mdi-eye-off' : 'mdi-eye'"
            autocomplete="off"
            density="compact"
            hint="Google Cloud > Places API (New). Read every 3 hours."
            label="Places API key"
            persistent-hint
            :type="showKey ? 'text' : 'password'"
            variant="outlined"
            @click:append-inner="showKey = !showKey"
          />
        </v-col>
      </v-row>

      <div class="text-body-2 font-weight-medium mt-4 mb-1">Branches with their own Google place</div>
      <v-row density="compact">
        <v-col
          v-for="b in draft.branches"
          :key="b.branch_id"
          cols="12"
          md="6"
        >
          <v-text-field
            v-model="b.place_id"
            density="compact"
            hide-details
            :label="b.branch_name"
            placeholder="Uses the main place"
            variant="outlined"
          />
        </v-col>
      </v-row>

      <div v-if="canEdit" class="d-flex justify-end mt-3">
        <v-btn
          color="primary"
          :disabled="!dirty"
          :loading="saving"
          text="Save"
          type="submit"
          variant="flat"
        />
      </div>
    </v-form>
    <div v-if="canEdit && saved?.google_reviews_enabled && saved?.google_places_api_key" class="mt-2">
      <v-btn
        :loading="checking"
        prepend-icon="mdi-refresh"
        size="small"
        text="Check Google now"
        variant="text"
        @click="checkNow"
      />
    </div>
  </div>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { useReviews } from '@/composables/useReviews'

  // Settings › Reviews: where review links go (the main Google place and each
  // branch's own), the link card, and the Places API key (api/review_link.go).

  const notify = useNotify()
  const { canEdit } = usePageAccess()
  const { refreshReviews, getReviewSettings, saveReviewSettings } = useReviews()

  const saved = ref(null)
  const draft = ref(null)
  const saving = ref(false)
  const checking = ref(false)
  const showKey = ref(false)
  const dirty = computed(() => JSON.stringify(draft.value) !== JSON.stringify(saved.value))

  function seed (settings) {
    saved.value = settings
    draft.value = structuredClone(settings)
  }
  async function loadSettings () {
    try {
      seed(await getReviewSettings())
    } catch (error) {
      notify.error(error.message)
    }
  }
  async function save () {
    saving.value = true
    try {
      seed(await saveReviewSettings(draft.value))
      notify.success('Review link saved.')
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }
  async function checkNow () {
    checking.value = true
    try {
      await refreshReviews()
      notify.success('Read from Google.')
    } catch (error) {
      notify.error(`Google: ${error.message}`)
    } finally {
      checking.value = false
    }
  }

  onMounted(loadSettings)
</script>
