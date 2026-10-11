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
          <AppLogo v-else class="mx-auto home-phone-logo" />
          <p v-if="appVersion || appCommit" class="mt-0 mb-0 text-caption text-disabled build-info">
            {{ [appVersion, appCommit].filter(Boolean).join(' · ') }}
          </p>
          <h1 class="mt-1 font-weight-medium" :class="xs ? 'text-h6' : 'text-h5'">Welcome to Kashi Cloud</h1>
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

          <v-card class="home-activities-card" rounded="lg" variant="outlined">
            <div class="home-activities-scroll" @scroll.passive="onFeedScroll($event, loadMoreKashi)">
              <v-skeleton-loader
                v-if="!kashiLoaded"
                color="transparent"
                type="list-item-avatar-two-line@8"
              />
              <v-list v-else class="py-1" density="compact" lines="two">
                <v-virtual-scroll :item-height="71" :items="shown(kashiActivities)" renderless>
                  <template #default="{ item, itemRef }">
                    <v-list-item :ref="itemRef" class="px-3">
                      <template #prepend>
                        <v-avatar
                          class="me-3 text-white"
                          :color="userColor(item)"
                          size="34"
                        >
                          <span class="text-caption font-weight-bold">{{ initialsOf(item.user_name) }}</span>
                        </v-avatar>
                      </template>
                      <v-list-item-title class="text-body-2 activity-sentence">
                        <span class="font-weight-medium" :title="item.user_name">{{ firstNameOf(item.user_name) || 'Someone' }}</span>
                        {{ kashiSentence(item) }}
                        <!-- The same change made again straight after (within the
                         hour) is counted on one entry, not listed again. -->
                        <v-chip
                          v-if="item.repeat_count > 1"
                          class="ms-1"
                          label
                          size="x-small"
                          :title="`${item.repeat_count} times in a row`"
                          variant="tonal"
                        >
                          ×{{ item.repeat_count }}
                        </v-chip>
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
                  </template>
                </v-virtual-scroll>
              </v-list>

              <div
                v-if="kashiLoaded && kashiActivities.length === 0"
                class="text-body-2 text-medium-emphasis py-4 text-center"
              >
                Nothing changed in kashi yet.
              </div>
              <div v-if="kashiLoading" class="d-flex justify-center py-2">
                <v-progress-circular indeterminate size="18" width="2" />
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

          <v-card class="home-activities-card" rounded="lg" variant="outlined">
            <div class="home-activities-scroll" @scroll.passive="onFeedScroll($event, loadMoreActivities)">
              <v-skeleton-loader
                v-if="!activitiesLoaded"
                color="transparent"
                type="list-item-avatar-two-line@8"
              />
              <v-list v-else class="py-0" density="compact" lines="two">
                <v-virtual-scroll :item-height="71" :items="shown(activities)" renderless>
                  <template #default="{ item, itemRef }">
                    <v-list-item
                      :ref="itemRef"
                      class="px-3 activity-row"
                      :class="rowTone(item)"
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
                          <span class="font-weight-medium text-high-emphasis text-no-wrap" :title="item.person">{{ firstNameOf(item.person) }}</span>
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
                  </template>
                </v-virtual-scroll>
              </v-list>

              <div
                v-if="activitiesLoaded && activities.length === 0"
                class="text-body-2 text-medium-emphasis py-4 text-center"
              >
                No recent activity yet.
              </div>
              <div v-if="activitiesLoading" class="d-flex justify-center py-2">
                <v-progress-circular indeterminate size="18" width="2" />
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

          <v-card class="home-activities-card" rounded="lg" variant="outlined">
            <div class="home-activities-scroll" @scroll.passive="onFeedScroll($event, loadMoreLatest)">
              <v-skeleton-loader
                v-if="!latestLoaded"
                color="transparent"
                type="list-item-avatar-two-line@8"
              />
              <v-list v-else class="py-1" density="compact" lines="two">
                <v-virtual-scroll :item-height="81" :items="shownLatest" renderless>
                  <template #default="{ item, itemRef }">
                    <v-list-item
                      :ref="itemRef"
                      class="px-3 activity-row"
                      @click="openLatest(item)"
                    >
                      <template #prepend>
                        <!-- Thumbnails load as rows scroll into view, not all 500 up front. -->
                        <v-avatar
                          class="me-3 latest-thumb"
                          :color="latestVisual(item).color"
                          rounded="lg"
                          size="56"
                          variant="tonal"
                          @vue:mounted="ensureThumb(item)"
                        >
                          <v-img v-if="thumbs[latestKey(item)]" cover :src="thumbs[latestKey(item)]" />
                          <v-icon v-else :icon="latestVisual(item).icon" size="18" />
                        </v-avatar>
                      </template>
                      <v-list-item-title class="text-body-2 font-weight-medium">
                        {{ item.title }}
                      </v-list-item-title>
                      <v-list-item-subtitle class="text-caption">
                        {{ [latestVisual(item).subtitleLabel === false ? '' : latestVisual(item).label, item.subtitle].filter(Boolean).join(' · ') }}
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
                  </template>
                </v-virtual-scroll>
              </v-list>

              <div
                v-if="latestLoaded && latestItems.length === 0"
                class="text-body-2 text-medium-emphasis py-4 text-center"
              >
                Nothing added yet.
              </div>
              <div v-else-if="latestLoading" class="d-flex justify-center py-2">
                <v-progress-circular indeterminate size="18" width="2" />
              </div>
              <div v-else-if="latestLoaded && canShowMoreLatest && !mdAndUp" class="d-flex justify-center py-2">
                <v-btn
                  size="small"
                  text="Show more"
                  variant="text"
                  @click="showMoreLatest"
                />
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
          <ZoomableImage
            v-if="photoUrl"
            :alt="photoItem?.title"
            :busy="photoChanging"
            :progress="uploadProgress"
            :replaceable="canReplace"
            :rotatable="canRotate"
            :src="photoUrl"
            @replace="file => changeLatestPhoto(item => replacePhoto(item.kind, item.id, file))"
            @rotate="delta => changeLatestPhoto(item => rotatePhoto(item.kind, item.id, delta))"
          />
          <div v-else class="d-flex justify-center pa-8">
            <v-progress-circular indeterminate />
          </div>
        </div>
        <template v-if="photoItem && latestTarget(photoItem)">
          <v-divider />
          <v-card-actions class="px-4">
            <span class="text-caption text-medium-emphasis">{{ photoItem.subtitle }}</span>
            <v-spacer />
            <v-btn
              color="primary"
              :text="latestTarget(photoItem).label"
              variant="tonal"
              @click="goToLatestTarget(photoItem)"
            />
          </v-card-actions>
        </template>
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
  import { feedQuery, nearFeedEnd, useKeysetFeed } from '@/composables/useKeysetFeed'
  import { useNotify } from '@/composables/useNotify'
  import { PHOTO_FETCH, usePhotoActions } from '@/composables/usePhotoActions'
  import { hashColor, initialsOf, useUserAvatar } from '@/composables/useUserAvatar'
  import { API_BASE } from '@/config'
  import { useAuthStore } from '@/stores/auth'

  // The feed row is narrow: the first name is enough beside the avatar,
  // whose initials and hover title still carry the full name.
  function firstNameOf (name) {
    return String(name ?? '').trim().split(/\s+/)[0] ?? ''
  }

  const { mdAndUp, xs } = useDisplay()
  const theme = useTheme()
  const logoColor = computed(() => theme.global.current.value.dark
    ? 'rgb(var(--v-theme-on-surface))'
    : 'rgba(var(--v-theme-on-surface), 0.7)')

  const auth = useAuthStore()
  const { color: myAvatarColor } = useUserAvatar()

  // Each row in the colour its user picked in Settings (avatar_color comes
  // with the row), else the one from their name. The signed-in user's own
  // rows follow their user, so a new pick shows before the feed reloads.
  function userColor (item) {
    const id = item.user_id?.Int64 ?? item.user_id
    const mine = auth.user && ((id && id === auth.user.id) || (item.user_name && item.user_name === auth.user.name))
    if (mine) {return myAvatarColor.value}
    return item.user_avatar_color || hashColor(item.user_name || 'kashi')
  }

  // Stamped at build time from git (see vite.config.mjs).
  const appVersion = import.meta.env.VITE_APP_VERSION
  const appCommit = import.meta.env.VITE_APP_COMMIT


  // Recent activity: a read-only glance at the latest branch-sync activity
  // (/home/activities, newest received first). The panel fills the bottom
  // of the (non-scrolling) home page and scrolls internally, and a click on
  // any row opens a details dialog. It refreshes itself off the shared admin
  // websocket (the same push stream the dashboard listens to), so there's no
  // manual refresh button.
  // Each feed loads 30 rows at a time as it's scrolled (useKeysetFeed) and
  // renders them in a virtual scroller, so only the rows in view are in the
  // DOM.

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

  const notify = useNotify()

  async function fetchFeedPage (path, cursor, what) {
    const response = await authFetch(`${API_BASE}${path}?${feedQuery(cursor)}`)
    if (!response.ok) {
      const body = await response.json().catch(() => ({}))
      throw new Error(body.error || `Failed to load ${what} (${response.status})`)
    }
    return response.json()
  }

  const {
    items: activities,
    loaded: activitiesLoaded,
    loading: activitiesLoading,
    loadMore: loadMoreActivities,
    refresh: refreshActivities,
  } = useKeysetFeed({
    fetchPage: async last => {
      const cursor = last && { at: last.synced_at, kind: last.activity, id: last.id }
      const data = await fetchFeedPage('/home/activities', cursor, 'recent activity')
      return data.activities ?? []
    },
    keyOf: item => `${item.activity}-${item.id}`,
    onError: error_ => notify.error(error_.message),
  })

  function reload () {
    if (canViewActivity.value) refreshActivities()
  }

  // Fetch the next page once a feed is scrolled near its bottom.
  function onFeedScroll (event, loadMore) {
    if (nearFeedEnd(event.target)) loadMore()
  }

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

  // Abnormal activity gets a tinted row, graded by how much it should worry
  // whoever is watching -- routine punches and sales stay untinted:
  //   high    red     money moved after the fact: a settlement edited, or a
  //                   shift closed short (negative USD variance)
  //   medium  orange  an approved attendance edit, an exchange invoice
  //   low     amber   an attendance complaint filed or rejected -- nothing
  //                   changed yet
  // Exchanges and the attendance status only live in the details string,
  // which the feed builds with ' · ' separators (see DashboardActivitiesList).
  function rowSeverity (item) {
    const parts = String(item.details || '').split(' · ')
    switch (item.activity) {
      case 'Settlement changed': {
        return 'high'
      }
      case 'Shift closed': {
        return item.amount < 0 ? 'high' : null
      }
      case 'Attendance changed': {
        return parts.includes('approved') ? 'medium' : 'low'
      }
      case 'Invoice': {
        return parts[0] === 'Exchange' ? 'medium' : null
      }
      default: {
        return null
      }
    }
  }

  function rowTone (item) {
    const severity = rowSeverity(item)
    return severity ? `activity-row--${severity}` : null
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
  // for every successful change (api/admin_activity.go). Paged like the
  // branch feed, refreshed by the admin_activity push.
  const canViewKashiActivity = computed(() => auth.canView('system'))
  let kashiRefreshTimer = null

  const {
    items: kashiActivities,
    loaded: kashiLoaded,
    loading: kashiLoading,
    loadMore: loadMoreKashi,
    refresh: refreshKashi,
  } = useKeysetFeed({
    fetchPage: async last => {
      const cursor = last && { at: last.created_at, id: last.id }
      const data = await fetchFeedPage('/admin_activities', cursor, 'kashi activity')
      return data.activities ?? []
    },
    keyOf: item => item.id,
    onError: error_ => notify.error(error_.message),
  })

  function reloadKashi () {
    if (canViewKashiActivity.value) refreshKashi()
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
  const thumbs = ref({})

  const LATEST_KINDS = {
    expense_image: { label: 'Expense photo', icon: 'mdi-receipt-text', color: 'error', file: id => `/branch_expense_images/${id}/file` },
    // Its title already says "Close-shift receipt", so the label stays out
    // of the subtitle (it's still on the viewer's Open button).
    shift_receipt: { label: 'Shift photo', subtitleLabel: false, icon: 'mdi-cash-register', color: 'info', file: id => `/branch_shift_receipt_images/${id}/file` },
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
    const res = await authFetch(`${API_BASE}${path}${width ? `?w=${width}` : ''}`, PHOTO_FETCH)
    if (!res.ok) return null
    return URL.createObjectURL(await res.blob())
  }

  // Called as each row mounts -- the virtual scroller only mounts the rows
  // in view -- so a thumbnail is fetched once, when it's first seen.
  const thumbsPending = new Set()
  async function ensureThumb (item) {
    const key = latestKey(item)
    if (!item.has_image || thumbs.value[key] || thumbsPending.has(key)) return
    thumbsPending.add(key)
    try {
      const url = await imageObjectUrl(item, 96).catch(() => null)
      if (url) thumbs.value = { ...thumbs.value, [key]: url }
    } finally {
      thumbsPending.delete(key)
    }
  }

  // "Show more" (phones only): first expands the short preview to the rows
  // loaded, then fetches the next page each time -- the card doesn't scroll
  // on a phone, so there's no scrolling to the end to trigger it.
  const latestExpanded = ref(false)
  const shownLatest = computed(() =>
    latestExpanded.value ? latestItems.value : shown(latestItems.value),
  )
  const canShowMoreLatest = computed(() => latestExpanded.value
    ? !latestDone.value
    : shownLatest.value.length < latestItems.value.length)

  function showMoreLatest () {
    if (latestExpanded.value) loadMoreLatest()
    else latestExpanded.value = true
  }

  // Where a feed item leads: the day page's "Shifts & Expenses" tab for the
  // branch and local day a shift receipt or expense photo belongs to (the
  // only place a single shift or branch expense is shown), the transfer
  // itself, or the page an expense/purchase lives on.
  function latestTarget (item) {
    if ((item.kind === 'shift_receipt' || item.kind === 'expense_image') && item.ref_day && item.branch_id) {
      return {
        label: item.kind === 'shift_receipt' ? 'Open shift day' : 'Open expense day',
        to: { path: `/daily-income/${item.ref_day}`, query: { branch_id: item.branch_id, tab: 'cash' } },
      }
    }
    if (item.kind === 'transfer' && item.ref_id) {
      return { label: 'Open transfer', to: { path: '/transfers', query: { open: item.ref_id } } }
    }
    const to = latestVisual(item).to
    return to ? { label: 'Open', to } : null
  }
  function goToLatestTarget (item) {
    const target = latestTarget(item)
    if (!target) return
    photoOpen.value = false
    router.push(target.to)
  }

  const {
    items: latestItems,
    loaded: latestLoaded,
    loading: latestLoading,
    done: latestDone,
    loadMore: loadMoreLatest,
    refresh: refreshLatest,
  } = useKeysetFeed({
    fetchPage: async last => {
      const cursor = last && { at: last.added_at, kind: last.kind, id: last.id }
      const data = await fetchFeedPage('/home/latest', cursor, 'latest additions')
      return data.items ?? []
    },
    keyOf: latestKey,
    onError: error_ => notify.error(error_.message),
  })

  function reloadLatest () {
    if (canViewLatest.value) refreshLatest()
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
    const target = latestTarget(item)
    if (target) router.push(target.to)
  }

  // Saves a rotation or replacement (save gets the feed item), then
  // re-fetches the photo and its row thumbnail as the server now serves
  // them. Photo kinds here are usePhotoActions'.
  const { canRotate, canReplace, rotatePhoto, replacePhoto, uploadProgress } = usePhotoActions()
  const photoChanging = ref(false)
  async function changeLatestPhoto (save) {
    const item = photoItem.value
    if (!item) return
    photoChanging.value = true
    try {
      await save(item)
      const [url, thumb] = await Promise.all([
        imageObjectUrl(item, 0),
        imageObjectUrl(item, 96).catch(() => null),
      ])
      const key = latestKey(item)
      if (thumb) {
        if (thumbs.value[key]) URL.revokeObjectURL(thumbs.value[key])
        thumbs.value = { ...thumbs.value, [key]: thumb }
      }
      if (photoItem.value !== item) {
        if (url) URL.revokeObjectURL(url)
        return
      }
      if (url) {
        if (photoUrl.value) URL.revokeObjectURL(photoUrl.value)
        photoUrl.value = url
      }
    } catch (error_) {
      notify.error(error_.message)
    } finally {
      photoChanging.value = false
    }
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

  // Phones get a short preview of each feed rather than the whole list.
  const PHONE_FEED_ROWS = 5
  function shown (items) {
    return mdAndUp.value ? items : items.slice(0, PHONE_FEED_ROWS)
  }

  // Invoices carry no currency code (their totals are USD), so no code
  // means dollars: "$28", "−$5". Any other currency keeps its code after
  // the number: "200,000 LBP".
  function money (amount, currency = '') {
    const value = Number(amount || 0) / 100
    const formatted = new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(Math.abs(value))
    const sign = amount < 0 ? '−' : ''
    if (!currency || currency === 'USD') return `${sign}$${formatted}`
    return `${sign}${formatted} ${currency}`
  }

  // Branch and details, minus what the row already shows elsewhere: the
  // person's name (beside their avatar) and the date and time the till wrote
  // into an attendance punch's details (the "just now" on the right).
  const DATE_PART = /^\d{4}-\d{2}-\d{2}(?:[ T]\d{1,2}:\d{2}(?::\d{2})?)?$/
  const TIME_PART = /^\d{1,2}:\d{2}(?::\d{2})?$/
  function activityContext (item) {
    const details = (item.details || '')
      .split(' · ')
      .filter(part => part !== item.person && !DATE_PART.test(part) && !TIME_PART.test(part))
      .join(' · ')
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

  /* The logo image is square but "K" fills only its middle -- the bottom
     quarter is empty -- so each logo pulls the version line up into that
     space (a fifth of the logo's width) to sit just under the letters. */
  .home-desktop-logo {
    width: clamp(72px, 9vw, 132px);
    margin-bottom: calc(clamp(72px, 9vw, 132px) * -0.2);
    aspect-ratio: 1;
    background-color: var(--home-logo-color);
    mask: var(--home-logo-image) center / contain no-repeat;
    -webkit-mask: var(--home-logo-image) center / contain no-repeat;
  }

  .home-phone-logo {
    margin-bottom: calc(clamp(100px, 13vw, 180px) * -0.2);
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

  .activity-row--high {
    background-color: rgba(var(--v-theme-error), 0.24);
  }

  .activity-row--medium {
    background-color: rgba(255, 109, 0, 0.2);
  }

  .activity-row--low {
    background-color: rgba(255, 193, 7, 0.12);
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

    /* Each card is just as tall as its few rows -- no inner scrolling. */
    .home-activities-card {
      flex: 0 0 auto;
    }

    .home-activities-scroll {
      flex: 0 0 auto;
      overflow: visible;
    }

    .home-activities {
      flex: 0 0 auto;
    }

    /* One line per row; the long sentences truncate. */
    .activity-sentence {
      white-space: nowrap;
    }

    .latest-thumb {
      height: 40px !important;
      width: 40px !important;
    }
  }

  @media (max-width: 600px) {
    .home-empty-state {
      padding-inline: 8px;
    }
  }
</style>
