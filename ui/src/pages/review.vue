<script setup lang="ts">
  import { computed, ref } from 'vue'

  // Swap for the real Google "write a review" URL when ready.
  const PUBLIC_REVIEW_URL = ''

  const VERDICTS: Record<number, string> = {
    1: 'Poor — we let you down',
    2: 'Below par',
    3: 'Fair',
    4: 'Good',
    5: 'Wonderful — thank you!',
  }

  const STORE_KEY = 'gb-review-sent'

  const rating = ref(0)
  const note = ref('')
  const submitted = ref(false)
  const alreadySent = ref(false)

  try {
    if (localStorage.getItem(STORE_KEY)) {
      alreadySent.value = true
      submitted.value = true
    }
  } catch { /* storage blocked — show the form */ }

  const verdict = computed(() => (rating.value ? VERDICTS[rating.value] : 'Tap a star'))

  const doneTitle = computed(() => {
    if (alreadySent.value) return "You've already shared this"
    if (rating.value >= 4) return 'Thank you — that means a lot'
    return 'Thank you for the honesty'
  })

  const doneBody = computed(() => {
    if (alreadySent.value) return 'Your feedback for the Beirut branch is in. See you next visit.'
    if (rating.value >= 4) return "We're glad the visit landed well. A public review helps other shoppers find us."
    return 'This goes straight to the branch team so we can put it right.'
  })

  const showPublicLink = computed(() => !alreadySent.value && rating.value >= 4 && !!PUBLIC_REVIEW_URL)

  function submit () {
    if (!rating.value) return
    const entry = { rating: rating.value, note: note.value.trim(), at: new Date().toISOString() }
    try {
      localStorage.setItem(STORE_KEY, JSON.stringify(entry))
    } catch { /* not fatal */ }
    // TODO: POST `entry` to a public /api/reviews endpoint once it exists.
    submitted.value = true
  }
</script>

<template>
  <div class="review-wrap">
    <v-card class="review-card" elevation="8" rounded="lg">
      <h1 class="brand">Grand<span>Bazaar</span></h1>
      <div class="flourish" />

      <template v-if="!submitted">
        <p class="eyebrow">Your visit</p>
        <p class="question">How did we do today?</p>
        <p class="sub">One tap tells the branch team how your experience felt.</p>

        <v-rating
          v-model="rating"
          active-color="amber-darken-2"
          color="grey-lighten-1"
          density="comfortable"
          hover
          size="52"
          class="my-2"
        />
        <p class="verdict" :class="{ filled: rating > 0 }">{{ verdict }}</p>

        <v-expand-transition>
          <v-textarea
            v-if="rating > 0"
            v-model="note"
            auto-grow
            class="mt-2 text-start"
            counter="400"
            :label="rating >= 4 ? 'What stood out?' : 'What could we do better?'"
            maxlength="400"
            rows="3"
            variant="outlined"
          />
        </v-expand-transition>

        <v-btn
          block
          class="mt-4 submit-btn"
          color="red-darken-2"
          :disabled="rating === 0"
          size="large"
          @click="submit"
        >
          Send feedback
        </v-btn>

        <p class="foot">Demo Store &middot; Beirut</p>
      </template>

      <template v-else>
        <div class="done">
          <v-icon class="done-tick" color="amber-darken-2" icon="mdi-check" size="32" />
          <h2>{{ doneTitle }}</h2>
          <p>{{ doneBody }}</p>
          <v-btn
            v-if="showPublicLink"
            class="mt-4"
            color="red-darken-2"
            :href="PUBLIC_REVIEW_URL"
            target="_blank"
            variant="tonal"
          >
            Leave a public review
          </v-btn>
        </div>
      </template>
    </v-card>
  </div>
</template>

<style scoped>
.review-wrap {
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 16px;
  background: #fbf7ef;
}

.review-card {
  width: 100%;
  max-width: 430px;
  padding: 40px 32px 32px;
  text-align: center;
  background: #fff;
}

.brand {
  font-family: 'Belleza', 'Trebuchet MS', sans-serif;
  font-weight: 400;
  font-size: 30px;
  line-height: 1.1;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  margin: 0;
  color: #23302e;
}
.brand span { display: block; margin-top: 2px; }

.flourish {
  width: 64px;
  height: 1px;
  margin: 16px auto 0;
  background: #e7dec9;
  position: relative;
}
.flourish::after {
  content: '';
  position: absolute;
  left: 50%;
  top: 50%;
  width: 6px;
  height: 6px;
  background: #db9b08;
  transform: translate(-50%, -50%) rotate(45deg);
}

.eyebrow {
  margin: 22px 0 0;
  font-family: 'Belleza', 'Trebuchet MS', sans-serif;
  font-size: 12px;
  letter-spacing: 0.28em;
  text-transform: uppercase;
  color: #5f6b63;
}

.question {
  margin: 6px 0 4px;
  font-size: 23px;
  font-weight: 600;
  color: #23302e;
}

.sub {
  margin: 0 auto;
  max-width: 32ch;
  color: #5f6b63;
  font-size: 14px;
}

.verdict {
  margin: 6px 0 0;
  min-height: 1.4em;
  font-size: 14px;
  font-style: italic;
  color: #5f6b63;
}
.verdict.filled { color: #23302e; }

.submit-btn {
  letter-spacing: 0.14em;
  text-transform: uppercase;
  font-family: 'Belleza', 'Trebuchet MS', sans-serif;
}

.foot {
  margin: 18px 0 0;
  font-size: 12px;
  letter-spacing: 0.04em;
  color: #5f6b63;
}

.done { padding: 8px 0 4px; }
.done-tick {
  border: 1px solid #db9b08;
  border-radius: 50%;
  padding: 8px;
  margin-bottom: 6px;
}
.done h2 {
  margin: 12px 0 6px;
  font-size: 22px;
  font-weight: 600;
  color: #23302e;
}
.done p {
  margin: 0 auto;
  max-width: 32ch;
  color: #5f6b63;
  font-size: 14px;
}

@media (max-width: 400px) {
  .review-card { padding: 32px 20px 26px; }
  .brand { font-size: 25px; letter-spacing: 0.13em; }
}
</style>
