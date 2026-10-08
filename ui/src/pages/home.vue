<template>
  <main class="home-empty-state">
    <div class="home-content">
      <div class="home-top">
        <section class="home-hero text-center">
          <div
            v-if="mdAndUp"
            aria-label="K"
            class="home-desktop-logo mx-auto"
            role="img"
            :style="{ '--home-logo-color': logoColor, '--home-logo-image': `url(${appLogo})` }"
          />
          <AppLogo v-else class="mx-auto" />
          <p v-if="appVersion || appCommit" class="mt-2 mb-0 text-caption text-disabled build-info">
            {{ [appVersion, appCommit].filter(Boolean).join(' · ') }}
          </p>
          <h1 class="mt-2 text-h5 font-weight-medium">Welcome to Kashi Cloud</h1>
          <p class="mt-2 mb-0 text-body-2 text-medium-emphasis">
            Take a look at the latest updates.
          </p>
          <!-- TEMP: month summary + feed-count note hidden
          <p class="mt-2 text-body-2 text-medium-emphasis month-summary">
            {{ monthSummary || "Here's what's new." }}
          </p>
          <p class="mt-1 mb-0 text-caption text-disabled">
            Feed counts cover the last week.
          </p>
          -->
        </section>


      </div>

      <div v-if="canViewKashiActivity || canViewActivity || canViewLatest" class="home-feeds">
        <section v-if="canViewKashiActivity" aria-label="IT Management" class="home-activities">
          <div class="home-activities-head d-flex align-center px-1 mb-2">
            <v-icon class="me-2 text-medium-emphasis" icon="mdi-account-group" size="18" />
            <span class="text-body-2 font-weight-medium">IT Management</span>
            <v-spacer />
            <span
              v-if="recentCounts"
              class="recent-count text-caption text-medium-emphasis"
            >
              {{ recentLabel(recentCounts.admin_activities, 'changes') }}
            </span>
            <v-skeleton-loader
              v-else
              class="today-skeleton"
              color="transparent"
              type="text"
              width="96"
            />
          </div>

          <v-card v-if="mdAndUp" class="home-activities-card" rounded="lg" variant="outlined">
            <div class="home-activities-scroll">
              <v-skeleton-loader
                v-if="!kashiLoaded"
                color="transparent"
                type="list-item-avatar-two-line@8"
              />
              <v-list v-else class="py-1" density="compact" lines="two">
                <v-list-item v-for="item in kashiActivities" :key="item.id" class="px-3">
                  <template #prepend>
                    <v-avatar
                      class="me-3 text-white"
                      :color="hashColor(item.user_name || 'kashi')"
                      size="34"
                    >
                      <span class="text-caption font-weight-bold">{{ initialsOf(item.user_name) }}</span>
                    </v-avatar>
                  </template>
                  <v-list-item-title class="text-body-2 activity-sentence">
                    <span class="font-weight-medium">{{ item.user_name || 'Someone' }}</span>
                    {{ kashiSentence(item) }}
                  </v-list-item-title>
                  <v-list-item-subtitle v-if="item.detail" class="text-caption">
                    {{ item.detail }}
                  </v-list-item-subtitle>
                  <template #append>
                    <div class="text-caption text-medium-emphasis text-no-wrap ms-2" :title="dateTime(item.created_at)">
                      {{ timeAgo(item.created_at) }}
                    </div>
                  </template>
                </v-list-item>
              </v-list>

              <div
                v-if="kashiLoaded && kashiActivities.length === 0"
                class="text-body-2 text-medium-emphasis py-4 text-center"
              >
                Nothing changed in kashi yet.
              </div>
            </div>
          </v-card>
        </section>

        <section v-if="canViewActivity" aria-label="Recent activity" class="home-activities">
          <div class="home-activities-head d-flex align-center px-1 mb-2">
            <v-icon class="me-2 text-medium-emphasis" icon="mdi-history" size="18" />
            <span class="text-body-2 font-weight-medium">Recent activity</span>
            <v-spacer />
            <span
              v-if="recentCounts"
              class="recent-count text-caption text-medium-emphasis"
            >
              {{ recentLabel(recentCounts.activities, 'synced') }}
            </span>
            <v-skeleton-loader
              v-else
              class="today-skeleton"
              color="transparent"
              type="text"
              width="96"
            />
          </div>

          <v-card v-if="mdAndUp" class="home-activities-card" rounded="lg" variant="outlined">
            <div class="home-activities-scroll">
              <v-skeleton-loader
                v-if="!activitiesLoaded"
                color="transparent"
                type="list-item-avatar-two-line@8"
              />
              <v-list v-else class="py-1" density="compact" lines="two">
                <v-list-item
                  v-for="(item, index) in activities"
                  :key="index"
                  class="px-3 activity-row"
                  @click="openDetail(item)"
                >
                  <!-- TEMP: left icon hidden
                  <template #prepend>
                    <v-avatar
                      class="me-3"
                      :color="visual(item).color"
                      rounded="lg"
                      size="34"
                      variant="tonal"
                    >
                      <v-icon :icon="visual(item).icon" size="18" />
                    </v-avatar>
                  </template>
                  -->
                  <v-list-item-title class="text-body-2 font-weight-medium">
                    {{ item.activity || 'Activity' }}
                  </v-list-item-title>
                  <v-list-item-subtitle class="text-caption d-flex align-center ga-1">
                    <template v-if="item.person">
                      <v-avatar
                        class="text-white flex-shrink-0"
                        :color="item.person_color || hashColor(item.person)"
                        size="18"
                      >
                        <span class="person-initials">{{ initialsOf(item.person) }}</span>
                      </v-avatar>
                      <span class="font-weight-medium text-high-emphasis">{{ item.person }}</span>
                      <span>·</span>
                    </template>
                    <span class="text-truncate">{{ activityContext(item) }}</span>
                  </v-list-item-subtitle>
                  <template #append>
                    <div class="text-right">
                      <div
                        v-if="hasAmount(item)"
                        class="text-body-2 font-weight-medium"
                        :class="amountClass(item.amount)"
                      >
                        {{ money(item.amount, item.currency_code) }}
                      </div>
                      <div class="text-caption text-medium-emphasis">
                        {{ timeAgo(item.synced_at) }}
                      </div>
                    </div>
                  </template>
                </v-list-item>
              </v-list>

              <div
                v-if="activitiesLoaded && activities.length === 0"
                class="text-body-2 text-medium-emphasis py-4 text-center"
              >
                No recent activity yet.
              </div>
            </div>
          </v-card>
        </section>

        <section v-if="canViewLatest" aria-label="Latest attachments" class="home-activities">
          <div class="home-activities-head d-flex align-center px-1 mb-2">
            <v-icon class="me-2 text-medium-emphasis" icon="mdi-image-multiple-outline" size="18" />
            <span class="text-body-2 font-weight-medium">Latest attachments</span>
            <v-spacer />
            <span
              v-if="recentCounts"
              class="recent-count text-caption text-medium-emphasis"
            >
              {{ recentLabel(recentCounts.latest, 'added') }}
            </span>
            <v-skeleton-loader
              v-else
              class="today-skeleton"
              color="transparent"
              type="text"
              width="96"
            />
          </div>

          <v-card v-if="mdAndUp" class="home-activities-card" rounded="lg" variant="outlined">
            <div class="home-activities-scroll">
              <v-skeleton-loader
                v-if="!latestLoaded"
                color="transparent"
                type="list-item-avatar-two-line@8"
              />
              <v-list v-else class="py-1" density="compact" lines="two">
                <v-list-item
                  v-for="item in latestItems"
                  :key="latestKey(item)"
                  class="px-3 activity-row"
                  @click="openLatest(item)"
                >
                  <template #prepend>
                    <v-avatar
                      class="me-3"
                      :color="latestVisual(item).color"
                      rounded="lg"
                      size="56"
                      variant="tonal"
                    >
                      <v-img v-if="thumbs[latestKey(item)]" cover :src="thumbs[latestKey(item)]" />
                      <v-icon v-else :icon="latestVisual(item).icon" size="18" />
                    </v-avatar>
                  </template>
                  <v-list-item-title class="text-body-2 font-weight-medium">
                    {{ item.title }}
                  </v-list-item-title>
                  <v-list-item-subtitle class="text-caption">
                    {{ [latestVisual(item).label, item.subtitle].filter(Boolean).join(' · ') }}
                  </v-list-item-subtitle>
                  <template #append>
                    <div class="text-right">
                      <div v-if="item.amount" class="text-body-2 font-weight-medium">
                        {{ money(item.amount, item.currency_code) }}
                      </div>
                      <div class="text-caption text-medium-emphasis">
                        {{ timeAgo(item.added_at) }}
                      </div>
                    </div>
                  </template>
                </v-list-item>
              </v-list>

              <div
                v-if="latestLoaded && latestItems.length === 0"
                class="text-body-2 text-medium-emphasis py-4 text-center"
              >
                Nothing added yet.
              </div>
            </div>
          </v-card>
        </section>
      </div>
    </div>

    <v-dialog v-model="detailOpen" max-width="460">
      <v-card v-if="detail" rounded="lg">
        <v-card-title class="d-flex align-center ga-3 px-4 py-3">
          <v-avatar :color="visual(detail).color" rounded="lg" size="38" variant="tonal">
            <v-icon :icon="visual(detail).icon" size="20" />
          </v-avatar>
          <span class="text-subtitle-1 font-weight-medium">{{ detail.activity || 'Activity' }}</span>
          <v-spacer />
          <v-btn
            aria-label="Close"
            icon="mdi-close"
            size="small"
            variant="text"
            @click="detailOpen = false"
          />
        </v-card-title>
        <v-divider />
        <v-list class="py-2" density="compact">
          <v-list-item>
            <v-list-item-subtitle class="text-caption">Branch</v-list-item-subtitle>
            <v-list-item-title class="text-body-2">{{ detail.branch_name || '—' }}</v-list-item-title>
          </v-list-item>
          <v-list-item>
            <v-list-item-subtitle class="text-caption">Details</v-list-item-subtitle>
            <v-list-item-title class="text-body-2" style="white-space: normal">
              {{ detail.details || '—' }}
            </v-list-item-title>
          </v-list-item>
          <v-list-item v-if="hasAmount(detail)">
            <v-list-item-subtitle class="text-caption">Amount</v-list-item-subtitle>
            <v-list-item-title class="text-body-2" :class="amountClass(detail.amount)">
              {{ money(detail.amount, detail.currency_code) }}
            </v-list-item-title>
          </v-list-item>
          <v-list-item>
            <v-list-item-subtitle class="text-caption">Synced at</v-list-item-subtitle>
            <v-list-item-title class="text-body-2">{{ dateTime(detail.synced_at) }}</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-card>
    </v-dialog>

    <v-dialog v-model="photoOpen" max-width="720">
      <v-card rounded="lg">
        <v-card-title class="d-flex align-center px-4 py-3">
          <span class="text-subtitle-1 font-weight-medium">{{ photoItem?.title }}</span>
          <v-spacer />
          <v-btn
            aria-label="Close"
            icon="mdi-close"
            size="small"
            variant="text"
            @click="photoOpen = false"
          />
        </v-card-title>
        <v-divider />
        <div class="pa-2">
          <ZoomableImage v-if="photoUrl" :alt="photoItem?.title" :src="photoUrl" />
          <div v-else class="d-flex justify-center pa-8">
            <v-progress-circular indeterminate />
          </div>
        </div>
      </v-card>
    </v-dialog>
  </main>
</template>

<script setup lang="ts">
  import { computed, onMounted, onUnmounted, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { useDisplay, useTheme } from 'vuetify'
  import appLogo from '@/assets/app-logo.png'
  import AppLogo from '@/components/AppLogo.vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { authFetch } from '@/composables/useApi'
  import { BRANCH_ACTIVITY_TYPES } from '@/composables/useBranchActivityMessage'
  import { useNotify } from '@/composables/useNotify'
  import { hashColor, initialsOf } from '@/composables/useUserAvatar'
  import { API_BASE } from '@/config'
  import { useAuthStore } from '@/stores/auth'

  const { mdAndUp } = useDisplay()
  const theme = useTheme()
  const logoColor = computed(() => theme.global.current.value.dark
    ? 'rgb(var(--v-theme-on-surface))'
    : 'rgba(var(--v-theme-on-surface), 0.7)')

  const auth = useAuthStore()

  // Stamped at build time from git (see vite.config.mjs).
  const appVersion = import.meta.env.VITE_APP_VERSION
  const appCommit = import.meta.env.VITE_APP_COMMIT


  // Recent activity: a read-only glance at the latest branch-sync activity.
  // Reuses the dashboard's /activities endpoint, which defaults to the
  // current month ordered newest-first when no filters are passed. This is a
  // fixed, short feed -- just the newest page -- not a paginated log: the
  // full history lives on the dashboard. The panel fills the bottom third of
  // the (non-scrolling) home page and scrolls internally, and a click on any
  // row opens a details dialog. It refreshes itself off the shared admin
  // websocket (the same push stream the dashboard listens to), so there's no
  // manual refresh button.
  const PAGE_SIZE = 20

  // The feed comes from the dashboard endpoints, which need Analytics access.
  const canViewActivity = computed(() => auth.canView('analytics'))

  // The line under the title: this month so far in one sentence, compared
  // with the same days of last month (1st..today vs 1st..same day), from the
  // dashboard summary (Analytics access). Falls back to a plain greeting.
  const monthSummary = ref('')

  function isoDay (d) {
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  }

  function usd (cents) {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: 'USD', maximumFractionDigits: 0 })
      .format(Number(cents || 0) / 100)
  }

  async function fetchMonthSummary (from, to) {
    const res = await authFetch(`${API_BASE}/dashboard/summary?date_from=${isoDay(from)}&date_to=${isoDay(to)}`)
    if (!res.ok) throw new Error(`summary ${res.status}`)
    return res.json()
  }

  async function loadMonthSummary () {
    if (!canViewActivity.value) return
    const today = new Date()
    const monthStart = new Date(today.getFullYear(), today.getMonth(), 1)
    const prevStart = new Date(today.getFullYear(), today.getMonth() - 1, 1)
    // Same day last month, or that month's last day when it's shorter.
    const prevEnd = new Date(today.getFullYear(), today.getMonth() - 1,
                             Math.min(today.getDate(), new Date(today.getFullYear(), today.getMonth(), 0).getDate()))
    try {
      const [now, prev] = await Promise.all([
        fetchMonthSummary(monthStart, today),
        fetchMonthSummary(prevStart, prevEnd),
      ])
      const month = today.toLocaleString(undefined, { month: 'long' })
      let text = `${month} so far: ${usd(now.revenue)} in sales from ${now.invoices} invoice${now.invoices === 1 ? '' : 's'}`
      if (prev.revenue > 0) {
        const change = Math.round((now.revenue - prev.revenue) * 100 / prev.revenue)
        text += change === 0
          ? ', level with the same days last month'
          : `, ${Math.abs(change)}% ${change > 0 ? 'more' : 'less'} than the same days last month`
      }
      const extras = [`${usd(now.expenses)} in expenses`]
      if (now.new_clients > 0) extras.push(`${now.new_clients} new client${now.new_clients === 1 ? '' : 's'}`)
      if (now.targets_pct != null) extras.push(`targets at ${now.targets_pct}%`)
      monthSummary.value = `${text}. ${extras.join(' · ')}.`
    } catch {
      monthSummary.value = ''
    }
  }

  const activities = ref([])
  const activitiesLoaded = ref(false)
  const notify = useNotify()

  const detailOpen = ref(false)
  const detail = ref(null)

  function openDetail (item) {
    detail.value = item
    detailOpen.value = true
  }

  // Per-activity-type colour + glyph. `activity` is the union label the SQL
  // feed stamps on every row (see DashboardActivitiesList); invoices and
  // loans additionally swing colour on the sign of the amount.
  const ACTIVITY_VISUALS = {
    'Invoice': { icon: 'mdi-sale', color: 'success' },
    'Expense': { icon: 'mdi-cash-minus', color: 'error' },
    'Loan': { icon: 'mdi-hand-coin', color: 'warning' },
    'Shift closed': { icon: 'mdi-cash-register', color: 'info' },
    'Settlement changed': { icon: 'mdi-cash-edit', color: 'purple' },
    'Attendance': { icon: 'mdi-fingerprint', color: 'primary' },
    'Attendance changed': { icon: 'mdi-clock-edit-outline', color: 'primary' },
  }

  function visual (item) {
    const base = ACTIVITY_VISUALS[item.activity] ?? { icon: 'mdi-pulse', color: 'info' }
    if ((item.activity === 'Invoice' || item.activity === 'Loan') && item.amount < 0) {
      return { ...base, color: 'error' }
    }
    return base
  }

  // Fetch the newest page and replace the list. There's no "load more" --
  // the dashboard is where the full, filterable history lives.
  async function reload () {
    if (!canViewActivity.value) return
    try {
      const response = await authFetch(
        `${API_BASE}/dashboard/activities?page_size=${PAGE_SIZE}&page_id=0`,
      )
      if (!response.ok) {
        const body = await response.json().catch(() => ({}))
        throw new Error(body.error || `Failed to load recent activity (${response.status})`)
      }
      const data = await response.json()
      activities.value = data.activities ?? []
    } catch (error_) {
      notify.error(error_.message)
    } finally {
      activitiesLoaded.value = true
    }
  }

  // A branch catch-up can fire a burst of pushes; debounce so the feed
  // reloads once after the dust settles.
  const { ensureConnected, onMessage } = useAdminSocket()
  let refreshTimer = null
  ensureConnected()
  const unsubscribe = onMessage(message => {
    if (message?.type === 'admin_activity') {
      clearTimeout(kashiRefreshTimer)
      // A change in kashi (a new transfer, an expense) may be a new
      // "Latest added" row too.
      kashiRefreshTimer = setTimeout(() => {
        reloadKashi()
        reloadLatest()
        reloadRecentCounts()
      }, 500)
      return
    }
    if (!BRANCH_ACTIVITY_TYPES.includes(message?.type)) return
    clearTimeout(refreshTimer)
    refreshTimer = setTimeout(() => {
      reload()
      reloadRecentCounts()
    }, 500)
  })

  // "N in the last week" beside each feed title (api/home_latest.go
  // getHomeToday, which counts from whatever `since` it's given). A rolling
  // window rather than "today", so the counts don't drop to zero first thing
  // in the morning; a feed the user can't view comes back null. Refreshed
  // alongside the feeds.
  const RECENT_DAYS = 7
  const recentCounts = ref(null)

  async function reloadRecentCounts () {
    const since = new Date(Date.now() - RECENT_DAYS * 24 * 60 * 60 * 1000)
    try {
      const response = await authFetch(`${API_BASE}/home/today?since=${encodeURIComponent(since.toISOString())}`)
      if (response.ok) recentCounts.value = await response.json()
    } catch {
      // Informational -- the next refresh tries again.
    }
  }

  // Just the count ("33,184 synced"); the one-week window is spelled out once,
  // under the month summary.
  function recentLabel (count, kind) {
    const n = count?.toLocaleString() ?? '—'
    if (kind === 'changes') return `${n} change${count === 1 ? '' : 's'}`
    return `${n} ${kind}`
  }

  // "In kashi": what people changed in kashi itself -- a transfer created,
  // a conflict resolved, settings sent to a branch -- recorded server-side
  // for every successful change (api/admin_activity.go). Same fixed newest
  // page as the branch feed, refreshed by the admin_activity push.
  const canViewKashiActivity = computed(() => auth.canView('system'))
  const kashiActivities = ref([])
  const kashiLoaded = ref(false)
  let kashiRefreshTimer = null

  async function reloadKashi () {
    if (!canViewKashiActivity.value) return
    try {
      const response = await authFetch(`${API_BASE}/admin_activities?page_size=${PAGE_SIZE}&page_id=0`)
      if (!response.ok) {
        const body = await response.json().catch(() => ({}))
        throw new Error(body.error || `Failed to load kashi activity (${response.status})`)
      }
      const data = await response.json()
      kashiActivities.value = data.activities ?? []
    } catch (error_) {
      notify.error(error_.message)
    } finally {
      kashiLoaded.value = true
    }
  }

  function kashiSentence (item) {
    return item.entity_id?.Valid ? `${item.summary} #${item.entity_id.Int64}` : item.summary
  }

  // "Latest added": the newest photos (expense, close-shift receipt,
  // transfer), transfers, expenses and purchases in one list
  // (api/home_latest.go). The server only returns kinds this user's role can
  // view. Photo rows show a small thumbnail and open full size; the rest
  // open their page. Polled, since most of these have no live push.
  const router = useRouter()
  const canViewLatest = computed(() =>
    auth.canView('accounting') || auth.canView('hr') || auth.canView('inventory'))
  const latestItems = ref([])
  const latestLoaded = ref(false)
  const thumbs = ref({})

  const LATEST_KINDS = {
    expense_image: { label: 'Expense photo', icon: 'mdi-receipt-text', color: 'error', file: id => `/branch_expense_images/${id}/file` },
    shift_receipt: { label: 'Shift photo', icon: 'mdi-cash-register', color: 'info', file: id => `/branch_shift_receipt_images/${id}/file` },
    transfer: { label: 'Transfer', icon: 'mdi-transfer', color: 'primary', file: id => `/transfers/${id}/attachment`, to: '/transfers' },
    expense: { label: 'Expense', icon: 'mdi-cash-minus', color: 'error', to: '/expenses' },
    purchase: { label: 'Purchase', icon: 'mdi-truck-delivery', color: 'success', to: '/purchase-invoice' },
  }

  function latestVisual (item) {
    return LATEST_KINDS[item.kind] ?? { label: '', icon: 'mdi-plus-box-outline', color: 'info' }
  }
  function latestKey (item) {
    return `${item.kind}-${item.id}`
  }

  async function imageObjectUrl (item, width) {
    const path = latestVisual(item).file?.(item.id)
    if (!path) return null
    const res = await authFetch(`${API_BASE}${path}${width ? `?w=${width}` : ''}`)
    if (!res.ok) return null
    return URL.createObjectURL(await res.blob())
  }

  async function loadThumbs () {
    for (const item of latestItems.value) {
      const key = latestKey(item)
      if (!item.has_image || thumbs.value[key]) continue
      const url = await imageObjectUrl(item, 96).catch(() => null)
      if (url) thumbs.value = { ...thumbs.value, [key]: url }
    }
  }

  async function reloadLatest () {
    if (!canViewLatest.value) return
    try {
      const response = await authFetch(`${API_BASE}/home/latest?page_size=${PAGE_SIZE}`)
      if (!response.ok) {
        const body = await response.json().catch(() => ({}))
        throw new Error(body.error || `Failed to load latest additions (${response.status})`)
      }
      const data = await response.json()
      latestItems.value = data.items ?? []
      loadThumbs()
    } catch (error_) {
      notify.error(error_.message)
    } finally {
      latestLoaded.value = true
    }
  }

  const photoOpen = ref(false)
  const photoItem = ref(null)
  const photoUrl = ref(null)

  async function openLatest (item) {
    if (item.has_image) {
      photoItem.value = item
      if (photoUrl.value) URL.revokeObjectURL(photoUrl.value)
      photoUrl.value = null
      photoOpen.value = true
      photoUrl.value = await imageObjectUrl(item, 0).catch(() => null)
      return
    }
    const to = latestVisual(item).to
    if (to) router.push(to)
  }

  // "5h ago" for the first 48 hours, then "Oct 7th" (year added when it
  // isn't this one).
  function timeAgo (value) {
    if (!value) return '—'
    const date = new Date(value)
    const seconds = Math.max(0, Math.round((Date.now() - date.getTime()) / 1000))
    if (seconds < 60) return 'just now'
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`
    if (seconds < 172_800) return `${Math.floor(seconds / 3600)}h ago`
    const day = date.getDate()
    const suffix = day % 100 >= 11 && day % 100 <= 13 ? 'th' : ({ 1: 'st', 2: 'nd', 3: 'rd' }[day % 10] ?? 'th')
    const month = date.toLocaleString([], { month: 'short' })
    const year = date.getFullYear() === new Date().getFullYear() ? '' : ` ${date.getFullYear()}`
    return `${month} ${day}${suffix}${year}`
  }

  function money (amount, currency = '') {
    const value = Number(amount || 0) / 100
    const formatted = new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(Math.abs(value))
    return `${amount < 0 ? '−' : ''}${formatted} ${currency}`.trim()
  }

  // Branch and details, minus the person's name the till also wrote into
  // the details -- it's already shown beside their avatar.
  function activityContext (item) {
    let details = item.details || ''
    if (item.person) {
      details = details.split(' · ').filter(part => part !== item.person).join(' · ')
    }
    return [item.branch_name, details].filter(Boolean).join(' · ') || '—'
  }

  function hasAmount (item) {
    return item.amount != null && item.amount !== 0 && !String(item.activity).startsWith('Attendance')
  }

  function amountClass (amount) {
    if (amount > 0) return 'text-success'
    if (amount < 0) return 'text-error'
    return 'text-medium-emphasis'
  }

  function dateTime (value) {
    return value ? new Date(value).toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' }) : '—'
  }

  let latestPoll = null
  onMounted(() => {
    reload()
    reloadKashi()
    reloadLatest()
    loadMonthSummary()
    reloadRecentCounts()
    // `make deploy-notes` inserts straight into the database, so no
    // admin_activity push fires for it; the poll picks those rows up.
    latestPoll = setInterval(() => {
      reloadKashi()
      reloadLatest()
      reloadRecentCounts()
    }, 60_000)
  })
  onUnmounted(() => {
    clearTimeout(refreshTimer)
    clearTimeout(kashiRefreshTimer)
    clearInterval(latestPoll)
    for (const url of Object.values(thumbs.value)) URL.revokeObjectURL(url)
    if (photoUrl.value) URL.revokeObjectURL(photoUrl.value)
    unsubscribe()
  })
</script>

<style scoped>
  .recent-count {
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  .home-empty-state {
    display: flex;
    justify-content: center;
    width: 100%;
    height: 100%;
    min-height: 0;
    padding: clamp(16px, 4vh, 40px) 24px 24px;
    overflow: hidden;
  }

  /* The page is a fixed-height, non-scrolling column: the hero on top, the
     feeds side by side filling the rest, each scrolling within itself. */
  .home-content {
    display: flex;
    flex-direction: column;
    width: 100%;
    min-height: 0;
  }

  .home-top {
    flex: 0 0 auto;
  }

  .home-hero {
    margin-inline: auto;
    max-width: 760px;
  }

  .month-summary {
    line-height: 1.6;
  }

  .home-desktop-logo {
    width: clamp(72px, 9vw, 132px);
    aspect-ratio: 1;
    background-color: var(--home-logo-color);
    mask: var(--home-logo-image) center / contain no-repeat;
    -webkit-mask: var(--home-logo-image) center / contain no-repeat;
  }

  /* The two feeds share the bottom third of the page, side by side. */
  .home-feeds {
    display: flex;
    flex: 1 1 0;
    gap: 16px;
    min-height: 0;
    max-width: 1600px;
    margin: 32px auto 0;
    width: 100%;
  }

  .home-activities {
    display: flex;
    flex: 1 1 0;
    flex-direction: column;
    min-height: 0;
    min-width: 0;
  }

  .activity-sentence {
    white-space: normal;
  }

  .today-skeleton :deep(.v-skeleton-loader__text) {
    margin: 0;
  }

  .home-activities-card {
    display: flex;
    flex: 1 1 0;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
  }

  .home-activities-scroll {
    flex: 1 1 0;
    min-height: 0;
    overflow-y: auto;
  }

  .activity-row {
    cursor: pointer;
  }

  .person-initials {
    font-size: 0.55rem;
    font-weight: 700;
  }

  /* Below md there's no room for the feeds side by side: stack them, give
     each card its own height (still scrolling within itself) and let the
     page scroll instead. */
  @media (max-width: 959.98px) {
    .home-empty-state {
      align-items: flex-start;
      overflow-y: auto;
    }

    .home-feeds {
      flex: 0 0 auto;
      flex-direction: column;
      gap: 24px;
    }

    .home-activities-card {
      flex: 0 0 auto;
      height: min(440px, 65vh);
    }
  }

  /* Below md the lists are hidden and only each feed's title and total show,
     so the header rows become the content. */
  @media (max-width: 959.98px) {
    .home-feeds {
      gap: 0;
    }

    .home-activities-head {
      border-bottom: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
      margin-bottom: 0 !important;
      padding-block: 14px;
    }
  }

  @media (max-width: 600px) {
    .home-empty-state {
      padding-inline: 8px;
    }
  }
</style>
