<template>
  <div class="security pa-4">
    <div class="d-flex align-center flex-wrap ga-2 mb-1">
      <v-icon icon="mdi-shield-lock-outline" />
      <div class="text-h6 font-weight-medium">Security</div>
      <v-progress-circular v-if="loading" indeterminate size="18" width="2" />
      <v-spacer />
      <v-btn prepend-icon="mdi-refresh" variant="text" @click="refreshAll">Refresh</v-btn>
    </div>
    <div v-if="overview" class="text-body-2 text-medium-emphasis mb-4">
      You're on <b>{{ overview.my_ip }}</b>
      <template v-if="overview.my_location?.country">
        · {{ flag(overview.my_location.country_code) }} {{ place(overview.my_location) }}
        <span v-if="overview.my_location.network">· {{ overview.my_location.network }}</span>
      </template>
    </div>

    <v-alert
      v-if="overview && !overview.geo_ready"
      class="mb-4"
      density="compact"
      type="info"
      variant="tonal"
    >
      The server is still downloading the IP location database (it fetches it on start, then monthly).
      Until it's there, locations show as unknown and the country filter lets everyone through.
    </v-alert>

    <div class="top-grid mb-4">
      <!-- Alerts -->
      <v-card class="section" variant="flat">
        <div class="d-flex align-center ga-2 mb-2">
          <v-icon color="warning" icon="mdi-shield-alert-outline" size="20" />
          <div class="section__title">Alerts</div>
          <v-chip v-if="overview?.open_alerts" color="warning" size="x-small" variant="flat">
            {{ overview.open_alerts }} new
          </v-chip>
          <v-spacer />
          <v-btn
            v-if="canEdit && openUnpinned > 0"
            prepend-icon="mdi-check-all"
            size="small"
            variant="text"
            @click="acknowledge([])"
          >
            Mark all seen
          </v-btn>
          <v-btn size="small" variant="text" @click="showSeen = !showSeen">
            {{ showSeen ? 'Hide seen' : 'Show seen' }}
          </v-btn>
        </div>
        <div v-if="shownAlerts.length === 0" class="text-body-2 text-medium-emphasis py-6 text-center">
          {{ showSeen ? 'No alerts yet.' : 'Nothing new. 👍' }}
        </div>
        <v-list v-else class="alert-list pa-0" density="compact">
          <v-list-item
            v-for="a in shownAlerts"
            :key="a.id"
            class="alert-item"
            :class="{ 'alert--seen': a.acknowledged_at, 'alert--pinned': a.pinned }"
            :style="{ '--alert-severity': severityOf(a).bar }"
          >
            <template #prepend>
              <v-icon :color="ALERT_KINDS[a.kind]?.color ?? 'grey'" :icon="ALERT_KINDS[a.kind]?.icon ?? 'mdi-alert'" size="20" />
            </template>
            <v-list-item-title class="text-body-2">
              <v-chip
                class="me-1"
                :color="severityOf(a).color"
                label
                size="x-small"
                variant="flat"
              >
                {{ severityOf(a).label }}
              </v-chip>
              <b>{{ ALERT_KINDS[a.kind]?.label ?? a.kind }}</b>
              <span v-if="a.count > 1" class="text-medium-emphasis"> ×{{ a.count }}</span>
              — {{ a.detail }}
            </v-list-item-title>
            <v-list-item-subtitle class="text-caption">
              {{ flag(a.country_code) }} {{ a.ip }} · {{ ago(a.last_at) }}
              <template v-if="a.acknowledged_at && a.acknowledged_by">
                · {{ a.severity === 'critical' ? 'resolved' : 'seen' }} by {{ a.acknowledged_by }}
              </template>
            </v-list-item-subtitle>
            <template #append>
              <span class="d-flex align-center ga-1">
                <template v-if="canEdit">
                  <v-btn
                    v-if="!isBanned(a.ip)"
                    color="error"
                    size="x-small"
                    variant="text"
                    @click="openBan(a.ip)"
                  >
                    Ban
                  </v-btn>
                  <v-btn
                    v-if="a.pinned"
                    color="error"
                    prepend-icon="mdi-check"
                    size="x-small"
                    variant="tonal"
                    @click="acknowledge([a.id])"
                  >
                    Resolve
                  </v-btn>
                  <v-btn
                    v-else-if="!a.acknowledged_at"
                    aria-label="Mark seen"
                    icon="mdi-check"
                    size="x-small"
                    variant="text"
                    @click="acknowledge([a.id])"
                  />
                </template>
                <v-icon
                  v-if="a.pinned"
                  aria-label="Pinned until resolved"
                  color="error"
                  icon="mdi-pin"
                  size="18"
                />
              </span>
            </template>
          </v-list-item>
        </v-list>
      </v-card>

      <!-- Countries -->
      <v-card class="section" variant="flat">
        <div class="d-flex align-center ga-2 mb-2">
          <v-icon icon="mdi-earth" size="20" />
          <div class="section__title">Where visitors are</div>
        </div>
        <div v-if="countries.length === 0" class="text-body-2 text-medium-emphasis py-6 text-center">No visitors recorded yet.</div>
        <div v-for="c in countries" :key="c.country_code" class="country-row" @click="filterCountry(c.country_code)">
          <span class="country-row__name">
            {{ flag(c.country_code) }} {{ c.country || 'Unknown' }}
            <v-icon
              v-if="c.country_code && !allowedSet.has(c.country_code)"
              color="warning"
              icon="mdi-alert"
              size="14"
              title="Not one of your allowed countries"
            />
          </span>
          <span class="country-row__bar"><span :style="{ width: barWidth(c.requests) }" /></span>
          <span class="country-row__num text-caption">
            {{ c.ips }} IP{{ c.ips === 1 ? '' : 's' }} · {{ c.requests.toLocaleString() }}
            <span v-if="c.blocked" class="text-error"> · {{ c.blocked }} refused</span>
          </span>
        </div>
      </v-card>
    </div>

    <!-- Visitors -->
    <v-card class="section mb-4" variant="flat">
      <div class="d-flex align-center flex-wrap ga-2 mb-2">
        <v-icon icon="mdi-web" size="20" />
        <div class="section__title">Visitors</div>
        <span class="text-caption text-medium-emphasis">{{ visitorTotal.toLocaleString() }} IPs</span>
        <v-spacer />
        <v-chip
          v-if="countryFilter"
          closable
          size="small"
          @click:close="filterCountry('')"
        >
          {{ flag(countryFilter) }} {{ countryName(countryFilter) }}
        </v-chip>
        <v-text-field
          v-model="search"
          clearable
          density="compact"
          hide-details
          placeholder="IP, city, network, user…"
          prepend-inner-icon="mdi-magnify"
          style="max-width: 280px"
          variant="outlined"
        />
      </div>
      <v-data-table-server
        v-model:items-per-page="pageSize"
        v-model:page="page"
        density="compact"
        :headers="visitorHeaders"
        item-value="ip"
        :items="visitors"
        :items-length="visitorTotal"
        :items-per-page-options="tablePageSizeOptions(pageSize)"
        :loading="visitorsLoading"
        show-expand
      >
        <template #item.ip="{ item }">
          <span class="font-weight-medium">{{ item.ip }}</span>
          <v-chip
            v-if="isBanned(item.ip)"
            class="ms-1"
            color="error"
            size="x-small"
            variant="flat"
          >banned</v-chip>
        </template>
        <template #item.place="{ item }">
          <span :class="{ 'text-warning': item.country_code && !allowedSet.has(item.country_code) }">
            {{ flag(item.country_code) }} {{ place(item) || 'Unknown' }}
          </span>
        </template>
        <template #item.who="{ item }">
          <v-chip :color="KIND_COLORS[item.last_kind]" size="x-small" variant="tonal">{{ item.last_kind }}</v-chip>
          <span class="ms-1">{{ item.last_user_name || item.last_branch_name }}</span>
        </template>
        <template #item.requests="{ item }">{{ item.requests.toLocaleString() }}</template>
        <template #item.failed_logins="{ item }">
          <span :class="{ 'text-error font-weight-bold': item.failed_logins >= 5 }">{{ item.failed_logins || '' }}</span>
        </template>
        <template #item.blocked="{ item }">
          <span :class="{ 'text-error': item.blocked }">{{ item.blocked || '' }}</span>
        </template>
        <template #item.last_seen_at="{ item }">
          <span :title="new Date(item.last_seen_at).toLocaleString()">{{ ago(item.last_seen_at) }}</span>
        </template>
        <template v-if="canEdit" #item.actions="{ item }">
          <v-btn
            v-if="isBanned(item.ip)"
            size="x-small"
            variant="text"
            @click="unban(item.ip)"
          >
            Unban
          </v-btn>
          <v-btn
            v-else-if="item.ip !== overview?.my_ip"
            color="error"
            size="x-small"
            variant="text"
            @click="openBan(item.ip)"
          >
            Ban
          </v-btn>
        </template>
        <template #expanded-row="{ columns, item }">
          <tr>
            <td class="py-2 text-caption" :colspan="columns.length">
              <div><b>Network:</b> {{ item.network || '—' }}<span v-if="item.asn"> (AS{{ item.asn }})</span></div>
              <div><b>Last request:</b> {{ item.last_method }} {{ item.last_path }} → {{ item.last_status }}</div>
              <div><b>Browser:</b> {{ item.user_agent || '—' }}</div>
              <div>
                <b>First seen:</b> {{ new Date(item.first_seen_at).toLocaleString() }} ·
                <b>sign-ins:</b> {{ item.logins }} · <b>not found:</b> {{ item.not_found }}
              </div>
            </td>
          </tr>
        </template>
      </v-data-table-server>
    </v-card>

    <div class="top-grid">
      <!-- Settings -->
      <v-card class="section" variant="flat">
        <div class="section__title mb-3">Protection</div>
        <v-switch
          v-model="draft.country_filter_enabled"
          color="primary"
          density="compact"
          :disabled="!canEdit"
          hide-details
          inset
          label="Only allow these countries"
        />
        <div class="text-caption text-medium-emphasis mb-2">
          Everyone else gets "access denied" from the API, so they can't sign in or read anything. Tills keep
          syncing regardless, and the server refuses a setting that would block you.
        </div>
        <v-autocomplete
          v-model="draft.allowed_countries"
          chips
          class="mb-3"
          closable-chips
          density="compact"
          :disabled="!canEdit"
          hide-details
          :item-title="c => `${flag(c)} ${countryName(c)}`"
          :item-value="c => c"
          :items="COUNTRY_CODES"
          label="Allowed countries"
          multiple
          variant="outlined"
        />
        <v-combobox
          v-model="draft.always_allowed"
          chips
          class="mb-1"
          closable-chips
          density="compact"
          :disabled="!canEdit"
          hide-details
          label="Always allow these IPs or ranges"
          multiple
          placeholder="e.g. 198.51.100.7 or 198.51.100.0/24"
          variant="outlined"
        />
        <div class="text-caption text-medium-emphasis mb-3">
          For someone who must get in from abroad. Type an address and press Enter.
          <a v-if="canEdit && overview && !draft.always_allowed.includes(overview.my_ip)" href="#" @click.prevent="draft.always_allowed.push(overview.my_ip)">
            Add my IP
          </a>
        </div>
        <v-divider class="mb-3" />
        <v-switch
          v-model="draft.auto_ban_enabled"
          color="primary"
          density="compact"
          :disabled="!canEdit"
          hide-details
          inset
          label="Ban an IP after repeated failed sign-ins"
        />
        <div class="d-flex ga-3 mt-2 mb-3">
          <v-text-field
            v-model.number="draft.auto_ban_failed_logins"
            density="compact"
            :disabled="!canEdit || !draft.auto_ban_enabled"
            hide-details
            label="Failed sign-ins (15 min)"
            min="3"
            type="number"
            variant="outlined"
          />
          <v-text-field
            v-model.number="draft.auto_ban_minutes"
            density="compact"
            :disabled="!canEdit || !draft.auto_ban_enabled"
            hide-details
            label="Ban for (minutes)"
            min="1"
            type="number"
            variant="outlined"
          />
        </div>
        <div v-if="canEdit" class="d-flex">
          <v-spacer />
          <v-btn color="primary" :disabled="!settingsDirty" :loading="saving" @click="saveSettings">Save</v-btn>
        </div>
      </v-card>

      <!-- Bans -->
      <v-card class="section" variant="flat">
        <div class="d-flex align-center ga-2 mb-2">
          <v-icon icon="mdi-cancel" size="20" />
          <div class="section__title">Banned IPs</div>
          <v-spacer />
          <v-btn v-if="canEdit" size="small" variant="text" @click="openBan('')">Ban an IP</v-btn>
        </div>
        <div v-if="bans.length === 0" class="text-body-2 text-medium-emphasis py-6 text-center">No one is banned.</div>
        <v-list v-else class="pa-0" density="compact">
          <v-list-item v-for="b in bans" :key="b.ip" class="px-1">
            <v-list-item-title class="text-body-2"><b>{{ b.ip }}</b> — {{ b.reason || 'no reason given' }}</v-list-item-title>
            <v-list-item-subtitle class="text-caption">
              {{ b.until ? `until ${new Date(b.until).toLocaleString()}` : 'until lifted' }}
              · by {{ b.created_by || '—' }}
            </v-list-item-subtitle>
            <template v-if="canEdit" #append>
              <v-btn
                prepend-icon="mdi-lock-open-variant-outline"
                size="x-small"
                variant="text"
                @click="unban(b.ip)"
              >
                Unban
              </v-btn>
            </template>
          </v-list-item>
        </v-list>
      </v-card>
    </div>

    <div class="text-caption text-disabled mt-4">
      Counts every request that reaches the API (signing in, data, tills), not loads of the web app's own files.
      <a href="https://db-ip.com" rel="noopener" target="_blank">IP Geolocation by DB-IP</a>.
    </div>

    <v-dialog v-model="banDialog" max-width="420">
      <v-card title="Ban an IP">
        <v-card-text>
          <v-text-field
            v-model="banForm.ip"
            class="mb-3"
            density="compact"
            hide-details
            label="IP address"
            variant="outlined"
          />
          <v-text-field
            v-model="banForm.reason"
            class="mb-3"
            density="compact"
            hide-details
            label="Reason (optional)"
            variant="outlined"
          />
          <v-select
            v-model="banForm.minutes"
            density="compact"
            hide-details
            :items="BAN_LENGTHS"
            label="For"
            variant="outlined"
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="banDialog = false">Cancel</v-btn>
          <v-btn color="error" :loading="banning" variant="flat" @click="ban">Ban</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
  import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
  import { useAdminSocket } from '@/composables/useAdminSocket'
  import { authFetch } from '@/composables/useApi'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { useSecurityAlerts } from '@/composables/useSecurityAlerts'
  import { API_BASE } from '@/config'
  import { tablePageSizeOptions, useSettingsStore } from '@/stores/settings'

  // Who reaches the API, from where, and what to do about it -- see
  // api/security.go. Needs the Security section; editing it to ban, unban
  // or change protection.

  const notify = useNotify()
  const { canEdit } = usePageAccess()

  const ALERT_KINDS = {
    failed_logins: { label: 'Failed sign-ins', icon: 'mdi-account-outline', color: 'warning' },
    auto_ban: { label: 'Banned automatically', icon: 'mdi-cancel', color: 'error' },
    probe: { label: 'Probing', icon: 'mdi-magnify', color: 'error' },
    login_abroad: { label: 'Sign-in from abroad', icon: 'mdi-earth', color: 'error' },
    foreign_visit: { label: 'Visit from abroad', icon: 'mdi-earth', color: 'warning' },
    blocked_country: { label: 'Refused (country)', icon: 'mdi-shield-lock-outline', color: 'info' },
  }
  // Severity comes from the server (alertSeverities in api/security.go).
  // color is the chip's, bar the strip down the card's left edge.
  const SEVERITIES = {
    critical: { label: 'Critical', color: 'error', bar: 'rgb(var(--v-theme-error))' },
    high: { label: 'High', color: 'deep-orange', bar: '#ff5722' },
    medium: { label: 'Medium', color: 'warning', bar: 'rgb(var(--v-theme-warning))' },
    low: { label: 'Low', color: 'info', bar: 'rgb(var(--v-theme-info))' },
  }
  function severityOf (a) {
    return SEVERITIES[a.severity] ?? SEVERITIES.medium
  }
  const KIND_COLORS = { admin: 'primary', till: 'success', public: 'grey' }
  const BAN_LENGTHS = [
    { title: '1 hour', value: 60 },
    { title: '1 day', value: 1440 },
    { title: '1 week', value: 10_080 },
    { title: 'Until lifted', value: 0 },
  ]

  // ISO 3166-1 alpha-2 codes; names come from the browser (Intl).
  const COUNTRY_CODES = 'AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR GA GB GD GE GF GG GH GI GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO JP KE KG KH KI KM KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH MK ML MM MN MO MP MQ MR MS MT MU MV MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM PN PR PS PT PW PY QA RE RO RS RU RW SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV TW TZ UA UG UM US UY UZ VA VC VE VG VI VN VU WF WS YE YT ZA ZM ZW'.split(' ')
  const regionNames = new Intl.DisplayNames(['en'], { type: 'region' })
  function countryName (code) {
    try {
      return regionNames.of(code) ?? code
    } catch {
      return code
    }
  }
  // 🇱🇧 from "LB" (regional indicator letters); nothing for an unknown place.
  function flag (code) {
    if (!code || code.length !== 2) return ''
    return String.fromCodePoint(...[...code.toUpperCase()].map(ch => 0x1_F1_E6 + ch.codePointAt(0) - 65))
  }
  function place (loc) {
    return [loc.city, loc.region === loc.city ? '' : loc.region, loc.country].filter(Boolean).join(', ')
  }
  function ago (value) {
    const s = Math.max(0, (Date.now() - new Date(value)) / 1000)
    if (s < 60) return 'just now'
    if (s < 3600) return `${Math.floor(s / 60)}m ago`
    if (s < 86_400) return `${Math.floor(s / 3600)}h ago`
    return `${Math.floor(s / 86_400)}d ago`
  }

  async function request (path, options) {
    const res = await authFetch(`${API_BASE}${path}`, options)
    const body = await res.json().catch(() => ({}))
    if (!res.ok) throw new Error(body.error || `Request failed (${res.status})`)
    return body
  }

  // ---- Overview: alerts, countries, bans, settings ---------------------------
  const overview = ref(null)
  const loading = ref(false)
  const showSeen = ref(false)

  // Pinned (unresolved critical) alerts first; the server already sends them
  // first, this keeps it so whatever the filter.
  const shownAlerts = computed(() => (overview.value?.alerts ?? [])
    .filter(a => showSeen.value || !a.acknowledged_at)
    .sort((x, y) => Number(y.pinned) - Number(x.pinned)))
  // What "Mark all seen" would clear: open alerts that aren't pinned.
  const openUnpinned = computed(() => (overview.value?.alerts ?? []).filter(a => !a.acknowledged_at && !a.pinned).length)
  const countries = computed(() => overview.value?.countries ?? [])
  const bans = computed(() => overview.value?.bans ?? [])
  const bannedSet = computed(() => new Set(bans.value.map(b => b.ip)))
  const isBanned = ip => bannedSet.value.has(ip)
  const maxRequests = computed(() => Math.max(1, ...countries.value.map(c => c.requests)))
  const barWidth = n => `${Math.max(2, (n / maxRequests.value) * 100)}%`

  const draft = reactive({
    country_filter_enabled: false,
    allowed_countries: [],
    always_allowed: [],
    auto_ban_enabled: true,
    auto_ban_failed_logins: 10,
    auto_ban_minutes: 60,
  })
  const allowedSet = computed(() => new Set(overview.value?.settings?.allowed_countries))
  function resetDraft () {
    const s = overview.value?.settings
    if (!s) return
    Object.assign(draft, {
      country_filter_enabled: s.country_filter_enabled,
      allowed_countries: [...s.allowed_countries],
      always_allowed: [...s.always_allowed],
      auto_ban_enabled: s.auto_ban_enabled,
      auto_ban_failed_logins: s.auto_ban_failed_logins,
      auto_ban_minutes: s.auto_ban_minutes,
    })
  }
  const settingsDirty = computed(() => {
    const s = overview.value?.settings
    if (!s) return false
    return JSON.stringify([s.country_filter_enabled, s.allowed_countries, s.always_allowed, s.auto_ban_enabled, s.auto_ban_failed_logins, s.auto_ban_minutes])
      !== JSON.stringify([draft.country_filter_enabled, draft.allowed_countries, draft.always_allowed, draft.auto_ban_enabled, draft.auto_ban_failed_logins, draft.auto_ban_minutes])
  })

  const { openSecurityAlerts } = useSecurityAlerts()
  async function loadOverview () {
    loading.value = true
    try {
      const keepDraft = settingsDirty.value
      overview.value = await request('/security/overview')
      openSecurityAlerts.value = overview.value?.open_alerts ?? 0
      if (!keepDraft) resetDraft()
    } catch (error) {
      notify.error(error.message)
    } finally {
      loading.value = false
    }
  }

  async function acknowledge (ids) {
    try {
      await request('/security/alerts/acknowledge', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ids }),
      })
      await loadOverview()
    } catch (error) {
      notify.error(error.message)
    }
  }

  const saving = ref(false)
  async function saveSettings () {
    saving.value = true
    try {
      const { settings } = await request('/security/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...draft }),
      })
      overview.value = { ...overview.value, settings }
      resetDraft()
      notify.success('Protection settings saved.')
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }

  // ---- Bans ---------------------------------------------------------------------
  const banDialog = ref(false)
  const banning = ref(false)
  const banForm = reactive({ ip: '', reason: '', minutes: 1440 })
  function openBan (ip) {
    Object.assign(banForm, { ip, reason: '', minutes: 1440 })
    banDialog.value = true
  }
  async function ban () {
    banning.value = true
    try {
      await request('/security/bans', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...banForm }),
      })
      banDialog.value = false
      await loadOverview()
    } catch (error) {
      notify.error(error.message)
    } finally {
      banning.value = false
    }
  }
  async function unban (ip) {
    try {
      await request(`/security/bans/${encodeURIComponent(ip)}`, { method: 'DELETE' })
      await loadOverview()
    } catch (error) {
      notify.error(error.message)
    }
  }

  // ---- Visitors -------------------------------------------------------------------
  const visitorHeaders = [
    { title: 'IP', key: 'ip', sortable: false },
    { title: 'Location', key: 'place', sortable: false },
    { title: 'Network', key: 'network', sortable: false },
    { title: 'Who', key: 'who', sortable: false },
    { title: 'Requests', key: 'requests', sortable: false, align: 'end' },
    { title: 'Failed sign-ins', key: 'failed_logins', sortable: false, align: 'end' },
    { title: 'Refused', key: 'blocked', sortable: false, align: 'end' },
    { title: 'Last seen', key: 'last_seen_at', sortable: false },
    { title: '', key: 'actions', sortable: false, align: 'end' },
  ]
  const visitors = ref([])
  const visitorTotal = ref(0)
  const visitorsLoading = ref(false)
  const page = ref(1)
  const pageSize = ref(useSettingsStore().tablePageSize)
  const search = ref('')
  const countryFilter = ref('')

  async function loadVisitors () {
    visitorsLoading.value = true
    try {
      const params = new URLSearchParams({
        page_size: pageSize.value,
        page_id: (page.value - 1) * pageSize.value,
        search: search.value ?? '',
        country: countryFilter.value,
      })
      const data = await request(`/security/visitors?${params}`)
      visitors.value = data.visitors ?? []
      visitorTotal.value = data.total ?? 0
    } catch (error) {
      notify.error(error.message)
    } finally {
      visitorsLoading.value = false
    }
  }
  function filterCountry (code) {
    countryFilter.value = code
  }

  let searchTimer = 0
  watch(search, () => {
    clearTimeout(searchTimer)
    searchTimer = setTimeout(() => {
      page.value = 1
      loadVisitors()
    }, 300)
  })
  watch(countryFilter, () => {
    page.value = 1
    loadVisitors()
  })
  watch([page, pageSize], loadVisitors)

  function refreshAll () {
    loadOverview()
    loadVisitors()
  }

  // A new alert comes over the admin socket (security_alert).
  const { ensureConnected, onMessage } = useAdminSocket()
  let unsubscribe = null
  onMounted(() => {
    refreshAll()
    ensureConnected()
    unsubscribe = onMessage(message => {
      if (message.type === 'security_alert') loadOverview()
    })
  })
  onUnmounted(() => {
    unsubscribe?.()
    clearTimeout(searchTimer)
  })
</script>

<style scoped>
.top-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

@media (max-width: 959.98px) {
  .top-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

.section {
  padding: 12px 16px;
  border: 1px solid rgba(128, 128, 128, 0.18);
}

.section__title {
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  opacity: 0.75;
}

.alert-list {
  max-height: 340px;
  overflow-y: auto;
}

.alert-item {
  border-left: 4px solid var(--alert-severity);
  border-radius: 4px;
  margin-bottom: 4px;
  padding-inline: 8px 4px !important;
}

.alert--pinned {
  background: rgba(var(--v-theme-error), 0.08);
}

.alert--seen {
  opacity: 0.55;
}

.country-row {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  padding: 4px 2px;
  cursor: pointer;
  border-radius: 6px;
}

.country-row:hover {
  background: rgba(var(--v-theme-on-surface), 0.05);
}

.country-row__name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.country-row__bar {
  height: 8px;
  background: rgba(var(--v-theme-on-surface), 0.06);
  border-radius: 4px;
  overflow: hidden;
}

.country-row__bar > span {
  display: block;
  height: 100%;
  background: rgb(var(--v-theme-primary));
  border-radius: 4px;
}
</style>
