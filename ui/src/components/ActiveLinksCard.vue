<template>
  <!-- The phone links tills have handed out that still work, so a manager can
       open or share one without the till in front of them. Shows nothing
       when there are none. -->
  <section v-if="links.length > 0" aria-label="Active links" class="active-links">
    <div class="d-flex align-center px-1 mb-2">
      <span class="text-body-2 font-weight-medium">Active links</span>
      <v-spacer />
      <span class="text-caption text-medium-emphasis" data-test="active-links-count">{{ links.length }} open</span>
    </div>
    <v-card variant="outlined">
      <v-list class="py-0" density="compact" lines="two">
        <template v-for="(link, index) in links" :key="link.path">
          <v-divider v-if="index > 0" />
          <v-list-item class="px-3" data-test="active-link">
            <v-list-item-title class="text-body-2">
              <span class="font-weight-medium">{{ linkTitle(link) }}</span>
              <span class="text-medium-emphasis ms-2">{{ linkDetail(link) }}</span>
            </v-list-item-title>
            <v-list-item-subtitle class="text-caption">
              <span class="active-link-url" data-test="active-link-url">{{ urlOf(link) }}</span>
              <span class="ms-3">{{ timeLeft(link.expires_at, now) }}</span>
            </v-list-item-subtitle>
            <template #append>
              <v-btn
                data-test="active-link-copy"
                size="small"
                :text="copied === link.path ? 'Copied' : 'Copy'"
                variant="text"
                @click="copy(link)"
              />
              <v-btn
                :href="urlOf(link)"
                rel="noopener"
                size="small"
                target="_blank"
                text="Open"
                variant="text"
              />
            </template>
          </v-list-item>
        </template>
      </v-list>
    </v-card>
  </section>
</template>

<script setup lang="ts">
  import { onMounted, onUnmounted, ref } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { API_BASE } from '@/config'
  import { linkDetail, linkTitle, linkUrl, timeLeft } from '@/utils/activeLinks'

  const links = ref([])
  const now = ref(Date.now())
  const copied = ref('')

  const urlOf = link => linkUrl(link, API_BASE)

  async function reload () {
    try {
      const res = await authFetch(`${API_BASE}/home/active_links`)
      if (!res.ok) {return}
      const body = await res.json()
      links.value = body.links ?? []
      now.value = Date.now()
    } catch {
      // Keep what's shown; the next poll tries again.
    }
  }

  let copiedTimer = null
  async function copy (link) {
    try {
      await navigator.clipboard.writeText(urlOf(link))
      copied.value = link.path
      clearTimeout(copiedTimer)
      copiedTimer = setTimeout(() => { copied.value = '' }, 2000)
    } catch {
      // Clipboard blocked: the link stays on screen to copy by hand.
    }
  }

  // Links come and go as tills hand them out and phones use them; the clock
  // keeps "Expires in" honest between polls.
  let poll = null
  let tick = null
  onMounted(() => {
    reload()
    poll = setInterval(reload, 30_000)
    tick = setInterval(() => { now.value = Date.now() }, 15_000)
  })
  onUnmounted(() => {
    clearInterval(poll)
    clearInterval(tick)
    clearTimeout(copiedTimer)
  })

  defineExpose({ reload })
</script>

<style scoped>
  .active-links {
    width: 100%;
  }
  .active-link-url {
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    user-select: all;
    overflow-wrap: anywhere;
  }
</style>
