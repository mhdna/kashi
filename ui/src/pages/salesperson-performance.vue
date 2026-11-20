<template>
  <v-container class="performance-page px-4 px-md-6 pb-4 pb-md-6 pt-1 pt-md-2" fluid>
    <div class="d-flex align-center flex-wrap ga-3 mb-5">
      <h1 class="text-h4 font-weight-bold">Salesperson performance</h1>
      <v-spacer />
      <v-btn
        prepend-icon="mdi-file-chart-outline"
        text="Monthly report (PDF)"
        to="/reports?report=staff"
        variant="tonal"
      />
    </div>

    <v-alert class="formula-card mb-5" color="primary" icon="mdi-information" variant="tonal">
      <div class="font-weight-bold mb-1">Formula</div>
      <div class="text-body-2">
        Score = 50% × (sold amount ÷ highest amount) + 25% × (invoices ÷ highest invoices)
        + 25% × (goods sold ÷ highest goods)
      </div>
    </v-alert>

    <v-card class="mb-5" rounded="lg">
      <v-card-text>
        <div class="d-flex align-center ga-2 mb-4">
          <v-icon color="primary" icon="mdi-filter" />
          <span class="font-weight-bold">Filters</span>
          <v-spacer />
          <v-btn
            v-if="hasFilters"
            size="small"
            text="Clear all"
            variant="text"
            @click="clearFilters"
          />
        </div>
        <v-row density="compact">
          <v-col cols="12" lg="3" sm="6">
            <v-select
              v-model="branchId"
              clearable
              density="comfortable"
              hide-details
              item-title="name"
              item-value="id"
              :items="branches"
              label="Employee branch"
              prepend-inner-icon="mdi-store-outline"
              variant="outlined"
            />
          </v-col>
          <v-col cols="12" lg="3" sm="6">
            <v-autocomplete
              v-model="salespersonId"
              clearable
              density="comfortable"
              hide-details
              item-title="name"
              item-value="key"
              :items="availableSalespersons"
              label="Employee name"
              prepend-inner-icon="mdi-account-outline"
              variant="outlined"
            />
          </v-col>
          <v-col cols="12" lg="2" sm="6">
            <v-select
              v-model="period"
              density="comfortable"
              hide-details
              item-title="title"
              item-value="value"
              :items="periods"
              label="Period"
              prepend-inner-icon="mdi-calendar"
              variant="outlined"
            />
          </v-col>
          <v-col v-if="period === 'month'" cols="12" lg="2" sm="6">
            <v-menu v-model="monthMenu" :close-on-content-click="false">
              <template #activator="{ props }">
                <v-text-field
                  v-bind="props"
                  density="comfortable"
                  hide-details
                  label="Month"
                  :model-value="formattedMonth"
                  prepend-inner-icon="mdi-calendar"
                  readonly
                  variant="outlined"
                />
              </template>
              <v-card>
                <v-date-picker
                  v-model:month="monthPickerMonth"
                  v-model:year="monthPickerYear"
                  view-mode="months"
                  @update:month="selectMonth"
                />
              </v-card>
            </v-menu>
          </v-col>
          <v-col v-if="period === 'range'" cols="12" lg="4" sm="6">
            <DateRangePicker v-model="dateRange" clearable empty-label="All time" />
          </v-col>
        </v-row>
      </v-card-text>
    </v-card>

    <v-row class="mb-3" density="compact">
      <v-col v-for="summary in summaries" :key="summary.label" cols="12" sm="4">
        <v-card class="h-100" rounded="lg">
          <v-card-text>
            <div class="text-caption text-medium-emphasis">{{ summary.label }}</div>
            <div class="text-h5 font-weight-bold mt-1">{{ summary.value }}</div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <v-card class="mb-5" rounded="lg">
      <v-card-title class="d-flex align-center py-4">
        <span>Performance ranking</span>
        <v-spacer />
        <v-chip color="primary" size="small" variant="tonal">{{ rankedRows.length }} employees</v-chip>
      </v-card-title>
      <v-divider />
      <PreparingState v-if="loading" kind="data" min-height="260px" title="Ranking salespersons" />
      <v-card-text v-else-if="rankedRows.length === 0" class="py-10 text-center text-medium-emphasis">
        No salesperson activity matches these filters.
      </v-card-text>
      <v-card-text v-else class="pt-2">
        <div
          v-for="(employee, index) in rankedRows"
          :key="employee.key"
          class="performance-row py-4"
          role="button"
          tabindex="0"
          @click="openAdjustments(employee)"
          @keydown.enter="openAdjustments(employee)"
        >
          <div class="d-flex align-center ga-3 mb-3">
            <div class="rank text-caption font-weight-bold">{{ index + 1 }}</div>
            <v-avatar color="primary" size="38" variant="tonal">{{ initials(employee.name) }}</v-avatar>
            <div class="min-width-0">
              <div class="font-weight-bold text-truncate">{{ employee.name }}</div>
              <div class="text-caption text-medium-emphasis">{{ employee.branchName }}</div>
            </div>
            <v-spacer />
            <div class="text-end">
              <div class="text-h6 font-weight-bold">{{ employee.score.toFixed(1) }}</div>
              <div class="text-caption text-medium-emphasis">out of 100</div>
            </div>
          </div>
          <v-progress-linear :color="scoreColor(employee.score)" height="12" :model-value="employee.score" rounded />
          <div class="d-flex flex-wrap ga-2 mt-3">
            <v-chip size="small" variant="tonal">Amount {{ employee.amountScore.toFixed(0) }}%</v-chip>
            <v-chip size="small" variant="tonal">Invoices {{ employee.invoiceScore.toFixed(0) }}%</v-chip>
            <v-chip size="small" variant="tonal">Goods {{ employee.goodsScore.toFixed(0) }}%</v-chip>
            <v-chip color="warning" size="small" variant="tonal">
              Discounts {{ formatMoney(employee.totalDiscounts) }}
            </v-chip>
            <v-chip color="error" size="small" variant="tonal">
              Price overrides {{ formatMoney(employee.totalPriceOverrides) }}
            </v-chip>
          </div>
        </div>
      </v-card-text>
    </v-card>

    <v-card rounded="lg">
      <v-card-title class="py-4">Performance details</v-card-title>
      <v-divider />
      <v-data-table
        class="performance-table"
        :headers="headers"
        :items="rankedRows"
        :loading="loading"
        no-data-text="No salesperson activity found"
        @click:row="openAdjustmentsFromTable"
      >
        <template #item.employee="{ item }">
          <div class="py-2">
            <div class="font-weight-medium">{{ item.name }}</div>
            <div class="text-caption text-medium-emphasis">{{ item.branchName }}</div>
          </div>
        </template>
        <template #item.score="{ item }">
          <v-chip :color="scoreColor(item.score)" size="small" variant="tonal">{{ item.score.toFixed(1) }}</v-chip>
        </template>
        <template #item.totalAmount="{ item }">{{ formatMoney(item.totalAmount) }}</template>
        <template #item.totalDiscounts="{ item }">{{ formatMoney(item.totalDiscounts) }}</template>
        <template #item.totalPriceOverrides="{ item }">{{ formatMoney(item.totalPriceOverrides) }}</template>
        <template #bottom />
      </v-data-table>
    </v-card>

    <!-- The salesperson score: the full rule (utils/salespersonScore.js). -->
    <v-card class="mt-5" data-test="score-card" rounded="lg">
      <v-card-title class="d-flex align-center py-4">
        <span>Performance score</span>
        <v-spacer />
        <span class="text-body-2 text-medium-emphasis" data-test="score-count">{{ rankedCount }} ranked</span>
      </v-card-title>
      <v-divider />
      <v-card-text>
        <v-alert class="formula-card mb-4" color="primary" icon="mdi-scale-balance" variant="tonal">
          <div class="font-weight-bold mb-1">Formula</div>
          <div class="text-body-2" data-test="score-formula">
            Score = Target reached ({{ weight('target') }}) + Sales per hour ({{ weight('perHour') }})
            + Average sale ({{ weight('avgSale') }}) + Items per sale ({{ weight('itemsPerSale') }})
            + Return rate ({{ weight('returnRate') }}), out of 100
          </div>
          <ul class="formula-list text-body-2 mt-2">
            <li><strong>Net sales</strong> = what was sold minus what came back. A return or exchange counts for whoever made the sale.</li>
            <li><strong>Target reached</strong> = net sales ÷ their target (the branch target shared by hours worked), up to 150%: points = min(reached, 150%) ÷ 150% × {{ weight('target') }}.</li>
            <li><strong>Sales per hour, average sale, items per sale</strong> are compared with the team, up to 1.5 times it: points = min(theirs ÷ team, 1.5) ÷ 1.5 × weight.</li>
            <li><strong>Return rate</strong>, lower is better: points = (1 − theirs ÷ (2 × team)) × {{ weight('returnRate') }}, never under 0.</li>
            <li>Discounts don't count: most come from the discount list, not the salesperson, and selling at a discount already lowers net sales against the target and per hour.</li>
            <li>Ranked with at least {{ SCORE_RULE.minInvoices }} invoices and {{ SCORE_RULE.minHours }} hours worked. A part with no data (no target, no hours) is left out and the score is out of the rest.</li>
            <li>Grades: A from 85, B from 70, C from 55, D under 55.</li>
          </ul>
        </v-alert>
      </v-card-text>
      <PreparingState v-if="loading" kind="data" min-height="200px" title="Working out the scores" />
      <v-data-table
        v-else
        class="performance-table"
        data-test="score-table"
        :headers="scoreHeaders"
        :items="scorePeople"
        :items-per-page="-1"
        no-data-text="No salesperson activity found"
        @click:row="openScoreFromTable"
      >
        <template #item.rank="{ item }">{{ item.rank ?? '' }}</template>
        <template #item.employee="{ item }">
          <div class="py-2">
            <div class="font-weight-medium">{{ item.name }}</div>
            <div class="text-caption text-medium-emphasis">{{ branchName(item.branchId) }}</div>
          </div>
        </template>
        <template #item.score="{ item }">
          <span v-if="item.score !== null" class="font-weight-bold" :class="gradeClass(item.grade)">
            {{ item.score.toFixed(1) }} {{ item.grade }}
          </span>
          <span v-else class="text-caption text-medium-emphasis">{{ item.notRankedWhy }}</span>
        </template>
        <template #item.net="{ item }">{{ formatMoney(item.net) }}</template>
        <template #item.target="{ item }">
          <template v-if="item.target > 0">
            {{ formatMoney(item.target) }}
            <div class="text-caption text-medium-emphasis">{{ formatPercent(item.net / item.target) }} reached</div>
          </template>
          <span v-else class="text-medium-emphasis">None</span>
        </template>
        <template #item.hours="{ item }">{{ item.hours > 0 ? item.hours.toFixed(1) : '' }}</template>
        <template #item.perHour="{ item }">{{ item.perHour === null ? '' : formatMoney(item.perHour) }}</template>
        <template #item.avgSale="{ item }">{{ formatMoney(item.avgSale) }}</template>
        <template #item.itemsPerSale="{ item }">{{ item.itemsPerSale.toFixed(2) }}</template>
        <template #item.returnRate="{ item }">{{ formatPercent(item.returnRate) }}</template>
        <template #bottom />
      </v-data-table>
    </v-card>

    <v-dialog v-model="scoreDialog" max-width="820">
      <v-card v-if="scorePerson" data-test="score-dialog" rounded="lg">
        <v-card-title class="d-flex align-center py-4">
          <div>
            <div>{{ scorePerson.name }}</div>
            <div class="text-caption text-medium-emphasis">How the score was worked out</div>
          </div>
          <v-spacer />
          <v-btn aria-label="Close" icon="mdi-close" variant="text" @click="scoreDialog = false" />
        </v-card-title>
        <v-divider />
        <v-card-text>
          <div class="score-facts text-body-2 mb-4" data-test="score-facts">
            <div><span class="text-medium-emphasis">Sold</span> {{ formatMoney(scorePerson.sold) }}</div>
            <div><span class="text-medium-emphasis">Came back</span> {{ formatMoney(scorePerson.returned) }}</div>
            <div><span class="text-medium-emphasis">Net sales</span> {{ formatMoney(scorePerson.net) }}</div>
            <div><span class="text-medium-emphasis">Invoices</span> {{ scorePerson.invoices }}</div>
            <div><span class="text-medium-emphasis">Items</span> {{ scorePerson.units }}</div>
            <div><span class="text-medium-emphasis">Hours</span> {{ scorePerson.hours.toFixed(1) }}</div>
            <div><span class="text-medium-emphasis">Target</span> {{ scorePerson.target > 0 ? formatMoney(scorePerson.target) : 'None' }}</div>
          </div>
          <v-table density="compact">
            <thead>
              <tr><th>Part</th><th>Calculation</th><th class="text-end">Points</th></tr>
            </thead>
            <tbody>
              <tr v-for="p in scorePerson.parts" :key="p.key" :data-test="`score-part-${p.key}`">
                <td class="text-no-wrap">{{ p.label }}</td>
                <td class="text-body-2">{{ explainPart(p, showPart) }}</td>
                <td class="text-end text-no-wrap">{{ p.available ? `${p.points.toFixed(2)} of ${p.weight}` : '' }}</td>
              </tr>
            </tbody>
          </v-table>
          <p class="text-body-2 mt-4 mb-0" data-test="score-total">
            <template v-if="scorePerson.score !== null">
              {{ scorePerson.points.toFixed(2) }} points out of {{ scorePerson.outOf }},
              so the score is {{ scorePerson.points.toFixed(2) }} ÷ {{ scorePerson.outOf }} × 100 =
              <strong :class="gradeClass(scorePerson.grade)">{{ scorePerson.score.toFixed(1) }}, grade {{ scorePerson.grade }}</strong>
            </template>
            <template v-else>Not ranked: {{ scorePerson.notRankedWhy.toLowerCase() }} in this period.</template>
          </p>
        </v-card-text>
      </v-card>
    </v-dialog>

    <v-dialog v-model="adjustmentsDialog" max-width="1100">
      <v-card class="adjustments-dialog-card" rounded="lg">
        <v-card-title class="d-flex align-center py-4">
          <div>
            <div>{{ selectedEmployee?.name }}</div>
            <div class="text-caption text-medium-emphasis">Manual discounts and price overrides</div>
          </div>
          <v-spacer />
          <v-btn icon="mdi-close" variant="text" @click="adjustmentsDialog = false" />
        </v-card-title>
        <v-divider />
        <v-card-text class="adjustments-dialog-content">
          <v-row class="mb-3" density="compact">
            <v-col cols="12" sm="6">
              <v-card color="warning" variant="tonal">
                <v-card-text>
                  <div class="text-caption">Total manual discounts</div>
                  <div class="text-h5 font-weight-bold">{{ formatMoney(selectedEmployee?.totalDiscounts) }}</div>
                </v-card-text>
              </v-card>
            </v-col>
            <v-col cols="12" sm="6">
              <v-card color="error" variant="tonal">
                <v-card-text>
                  <div class="text-caption">Total price deductions from overrides</div>
                  <div class="text-h5 font-weight-bold">{{ formatMoney(selectedEmployee?.totalPriceOverrides) }}</div>
                </v-card-text>
              </v-card>
            </v-col>
          </v-row>

          <v-alert
            v-if="selectedAdjustments.length === 0"
            type="info"
            variant="tonal"
          >
            No manual discounts or price overrides were recorded for this salesperson in the selected period.
          </v-alert>
          <template v-else>
            <v-text-field
              v-model="adjustmentSearch"
              class="mb-3"
              clearable
              data-discard-ignore
              density="compact"
              hide-details
              label="Search adjustments"
              prepend-inner-icon="mdi-magnify"
              variant="outlined"
            />
            <v-data-table
              :headers="adjustmentHeaders"
              :items="selectedAdjustments"
              :search="adjustmentSearch"
            >
              <template #item.type="{ item }">
                <v-chip :color="item.type === 'Discount' ? 'warning' : 'error'" size="small" variant="tonal">
                  {{ item.type }}
                </v-chip>
              </template>
              <template #item.date="{ item }">{{ formatDate(item.date) }}</template>
              <template #item.originalPrice="{ item }">{{ item.originalPrice == null ? '—' : formatMoney(item.originalPrice) }}</template>
              <template #item.finalPrice="{ item }">{{ formatMoney(item.finalPrice) }}</template>
              <template #item.amount="{ item }"><strong>{{ formatMoney(item.amount) }}</strong></template>
            </v-data-table>
          </template>
        </v-card-text>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref, watch } from 'vue'
  import { buildPeople } from '@/components/reports/reportRows'
  import { authFetch } from '@/composables/useApi'
  import { useAttendance } from '@/composables/useAttendance'
  import { useAttendanceAutoOut } from '@/composables/useAttendanceAutoOut'
  import { useBranches } from '@/composables/useBranches'
  import { useBranchSalespersons } from '@/composables/useBranchSalespersons'
  import { useNotify } from '@/composables/useNotify'
  import { API_BASE } from '@/config'
  import { buildTimesheet } from '@/utils/attendanceSessions'
  import { ymd } from '@/utils/dates'
  import { explainPart, hoursFromTimesheet, SCORE_RULE, scoreSalespersons } from '@/utils/salespersonScore'

  const { branches, fetchBranches } = useBranches()
  const { listEventsInRange } = useAttendance()
  const { autoInMinutes, autoOutMinutes, ensureAutoDefaults } = useAttendanceAutoOut()
  const { listBranchSalespersons } = useBranchSalespersons()
  const invoices = ref([])
  // For the score: returns and exchanges, each branch's target for the
  // period, the period's punches and the salespersons they belong to.
  const returns = ref([])
  const branchTargets = ref([])
  const attendance = ref([])
  const staff = ref([])
  const scoreDialog = ref(false)
  const scorePerson = ref(null)
  const loading = ref(false)
  const notify = useNotify()
  const branchId = ref(null)
  const salespersonId = ref(null)
  const period = ref('month')
  const now = new Date()
  const month = ref(`${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`)
  const monthMenu = ref(false)
  const monthPickerMonth = ref(now.getMonth())
  const monthPickerYear = ref(now.getFullYear())
  const dateRange = ref([])
  const adjustmentsDialog = ref(false)
  const selectedEmployee = ref(null)
  const adjustmentSearch = ref('')

  const periods = [
    { title: 'This month', value: 'month' },
    { title: 'Custom date range', value: 'range' },
    { title: 'All time', value: 'all' },
  ]
  const headers = [
    { title: 'Employee', key: 'employee', align: 'start' },
    { title: 'Performance', key: 'score', align: 'center' },
    { title: 'Total Sold Amount', key: 'totalAmount', align: 'end' },
    { title: 'Total Invoices', key: 'totalInvoices', align: 'end' },
    { title: 'Total Number of goods', key: 'totalGoods', align: 'end' },
    { title: 'Total Discounts', key: 'totalDiscounts', align: 'end' },
    { title: 'Total Price Deductions', key: 'totalPriceOverrides', align: 'end' },
  ]
  const adjustmentHeaders = [
    { title: 'Type', key: 'type', align: 'start' },
    { title: 'Date', key: 'date', align: 'start' },
    { title: 'Invoice', key: 'invoiceCode', align: 'start' },
    { title: 'Product', key: 'product', align: 'start' },
    { title: 'Qty', key: 'quantity', align: 'end' },
    { title: 'Original Price', key: 'originalPrice', align: 'end' },
    { title: 'Final Price', key: 'finalPrice', align: 'end' },
    { title: 'Discount', key: 'percentage', align: 'end' },
    { title: 'Deducted Amount', key: 'amount', align: 'end' },
  ]

  function dateKey (value) {
    if (typeof value === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(value)) return value
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? '' : ymd(date)
  }

  const formattedMonth = computed(() => new Date(`${month.value}-01T00:00:00`).toLocaleDateString(undefined, {
    month: 'long',
    year: 'numeric',
  }))
  const selectedRange = computed(() => dateRange.value.map(value => dateKey(value)).filter(Boolean).toSorted())
  // One person per salesperson id, so a renamed salesperson isn't counted
  // twice; an invoice not linked to one (an old till, or a name kashi never
  // had) still groups by name.
  function personKey (invoice) {
    const id = Number(invoice.salesperson_id) || 0
    return id > 0 ? `id:${id}` : `name:${invoice.salesperson_name || 'Unassigned'}`
  }

  function goodsCount (invoice) {
    if (invoice.total_goods != null) return Number(invoice.total_goods) || 0
    if (invoice.items_count != null) return Number(invoice.items_count) || 0
    const items = invoice.items || invoice.invoice_items || []
    return items.reduce((total, item) => total + (Number(item.quantity) || 0), 0)
  }

  function manualDiscountAmount (item) {
    const source = item.discount_source || item.discount_type
    if (source && !['manual', 'salesperson', 'cashier'].includes(String(source).toLowerCase())) return 0
    if (!(Number(item.discount) > 0 || item.is_manual_discount || item.manual_discount)) return 0
    if (item.manual_discount_amount != null) return Math.max(0, Number(item.manual_discount_amount) || 0)
    const gross = (Number(item.unit_price) || 0) * Math.abs(Number(item.quantity) || 0)
    const net = Math.abs(Number(item.line_total) || 0)
    return Math.max(0, gross - net)
  }

  function originalUnitPrice (item) {
    const value = item.original_unit_price ?? item.list_price ?? item.regular_price ?? item.base_price
    return value == null ? null : Number(value)
  }

  function manualOverrideAmount (item) {
    if (item.price_override_amount != null) return Math.max(0, Number(item.price_override_amount) || 0)
    const source = item.price_override_source || item.override_source
    const isManual = item.is_manual_price_override || item.price_overridden || item.manual_price_override
      || ['manual', 'salesperson', 'cashier'].includes(String(source || '').toLowerCase())
    const original = originalUnitPrice(item)
    if (!isManual || original == null) return 0
    return Math.max(0, original - (Number(item.unit_price) || 0)) * Math.abs(Number(item.quantity) || 0)
  }

  function adjustmentRecords (invoice) {
    return (invoice.items || invoice.invoice_items || []).flatMap((item, index) => {
      const records = []
      const common = {
        id: `${invoice.id}-${item.id ?? index}`,
        date: invoice.occurred_at || invoice.created_at || invoice.received_at,
        invoiceCode: invoice.branch_invoice_code || invoice.invoice_code || `#${invoice.id}`,
        product: item.product_name || item.product_code || `Product #${item.branch_product_id ?? item.product_id ?? '—'}`,
        quantity: Math.abs(Number(item.quantity) || 0),
        finalPrice: Number(item.unit_price) || 0,
      }
      const discountAmount = manualDiscountAmount(item)
      if (discountAmount > 0) {
        records.push({
          ...common,
          id: `${common.id}-discount`,
          type: 'Discount',
          originalPrice: Number(item.unit_price) || 0,
          percentage: `${Number(item.discount) || 0}%`,
          amount: discountAmount,
        })
      }
      const overrideAmount = manualOverrideAmount(item)
      if (overrideAmount > 0) {
        records.push({
          ...common,
          id: `${common.id}-override`,
          type: 'Price override',
          originalPrice: originalUnitPrice(item),
          percentage: '—',
          amount: overrideAmount,
        })
      }
      return records
    })
  }

  const aggregatedRows = computed(() => {
    const result = new Map()
    for (const invoice of invoices.value) {
      if (invoice.kind && invoice.kind !== 'sale' && invoice.kind !== 'sales') continue
      if (branchId.value != null && String(invoice.branch_id) !== String(branchId.value)) continue
      const key = personKey(invoice)
      const row = result.get(key) || {
        key,
        name: invoice.salesperson_name || invoice.salesperson?.name || 'Unassigned',
        branchId: invoice.branch_id,
        totalAmount: 0,
        totalInvoices: 0,
        totalGoods: 0,
        totalDiscounts: 0,
        totalPriceOverrides: 0,
        adjustments: [],
      }
      row.totalAmount += Number(invoice.grand_total ?? invoice.total_amount ?? invoice.total ?? 0) || 0
      row.totalInvoices += 1
      row.totalGoods += goodsCount(invoice)
      const adjustments = adjustmentRecords(invoice)
      row.adjustments.push(...adjustments)
      row.totalDiscounts += adjustments.filter(item => item.type === 'Discount').reduce((sum, item) => sum + item.amount, 0)
      row.totalPriceOverrides += adjustments.filter(item => item.type === 'Price override').reduce((sum, item) => sum + item.amount, 0)
      result.set(key, row)
    }
    return [...result.values()].map(row => ({
      ...row,
      branchName: branches.value.find(branch => String(branch.id) === String(row.branchId))?.name || 'Main branch',
    }))
  })

  const availableSalespersons = computed(() => aggregatedRows.value.map(row => ({ key: row.key, name: row.name })))
  const rankedRows = computed(() => {
    const rows = salespersonId.value ? aggregatedRows.value.filter(row => row.key === salespersonId.value) : aggregatedRows.value
    const maxAmount = Math.max(...rows.map(row => row.totalAmount), 0)
    const maxInvoices = Math.max(...rows.map(row => row.totalInvoices), 0)
    const maxGoods = Math.max(...rows.map(row => row.totalGoods), 0)
    return rows.map(row => {
      const amountScore = maxAmount ? row.totalAmount / maxAmount * 100 : 0
      const invoiceScore = maxInvoices ? row.totalInvoices / maxInvoices * 100 : 0
      const goodsScore = maxGoods ? row.totalGoods / maxGoods * 100 : 0
      return { ...row, amountScore, invoiceScore, goodsScore, score: amountScore * 0.5 + invoiceScore * 0.25 + goodsScore * 0.25 }
    }).toSorted((a, b) => b.score - a.score)
  })

  // The score's table: worked out over the chosen branch (the team is
  // everyone ranked there), then narrowed to the chosen salesperson.
  const inBranch = row => branchId.value == null || String(row.branch_id) === String(branchId.value)
  const scoreHours = computed(() => {
    const timesheet = buildTimesheet(attendance.value.filter(row => inBranch(row)), { autoIn: autoInMinutes.value, autoOut: autoOutMinutes.value })
    const people = buildPeople({ staff: staff.value, timesheet })
    return hoursFromTimesheet(timesheet, people.forAttendance)
  })
  const scoreResult = computed(() => scoreSalespersons({
    invoices: invoices.value.filter(row => inBranch(row)),
    returns: returns.value.filter(row => inBranch(row)),
    branchTargets: branchTargets.value.filter(row => inBranch(row)),
    hours: scoreHours.value,
  }))
  const scorePeople = computed(() => salespersonId.value
    ? scoreResult.value.people.filter(p => p.key === salespersonId.value)
    : scoreResult.value.people)
  const rankedCount = computed(() => scorePeople.value.filter(p => p.score !== null).length)
  const scoreHeaders = [
    { title: '#', key: 'rank', align: 'start', width: 40 },
    { title: 'Employee', key: 'employee', align: 'start' },
    { title: 'Score', key: 'score', align: 'start' },
    { title: 'Net sales', key: 'net', align: 'end' },
    { title: 'Target', key: 'target', align: 'end' },
    { title: 'Hours', key: 'hours', align: 'end' },
    { title: 'Per hour', key: 'perHour', align: 'end' },
    { title: 'Average sale', key: 'avgSale', align: 'end' },
    { title: 'Items per sale', key: 'itemsPerSale', align: 'end' },
    { title: 'Return rate', key: 'returnRate', align: 'end' },
  ]
  function weight (key) {
    return SCORE_RULE.parts.find(p => p.key === key)?.weight ?? 0
  }
  function formatPercent (ratio) {
    return `${((Number(ratio) || 0) * 100).toFixed(1)}%`
  }
  // A part's value in the calculation dialog.
  function showPart (key, value) {
    if (key === 'itemsPerSale') {return (Number(value) || 0).toFixed(2)}
    if (key === 'returnRate') {return formatPercent(value)}
    return formatMoney(value)
  }
  function gradeClass (g) {
    return { A: 'text-success', B: 'text-primary', C: 'text-warning', D: 'text-error' }[g] ?? ''
  }
  function branchName (id) {
    return branches.value.find(branch => String(branch.id) === String(id))?.name || ''
  }
  function openScoreFromTable (_event, row) {
    scorePerson.value = row.item
    scoreDialog.value = true
  }

  const summaries = computed(() => [
    { label: 'Total sold amount', value: formatMoney(rankedRows.value.reduce((sum, row) => sum + row.totalAmount, 0)) },
    { label: 'Total invoices', value: rankedRows.value.reduce((sum, row) => sum + row.totalInvoices, 0).toLocaleString() },
    { label: 'Total goods sold', value: rankedRows.value.reduce((sum, row) => sum + row.totalGoods, 0).toLocaleString() },
  ])
  const selectedAdjustments = computed(() => selectedEmployee.value?.adjustments || [])
  const hasFilters = computed(() => branchId.value != null || salespersonId.value != null || period.value !== 'month' || month.value !== `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`)

  function formatMoney (cents) {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: 'USD' }).format((Number(cents) || 0) / 100)
  }
  function formatDate (value) {
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString()
  }
  function initials (name) {
    return name.split(/\s+/).map(part => part[0]).slice(0, 2).join('').toUpperCase()
  }
  function scoreColor (score) {
    return score >= 75 ? 'success' : (score >= 45 ? 'warning' : 'error')
  }
  function selectMonth (monthIndex) {
    month.value = `${monthPickerYear.value}-${String(Number(monthIndex) + 1).padStart(2, '0')}`
    monthMenu.value = false
  }
  function openAdjustments (employee) {
    selectedEmployee.value = employee
    adjustmentSearch.value = ''
    adjustmentsDialog.value = true
  }
  function openAdjustmentsFromTable (_event, row) {
    openAdjustments(row.item)
  }
  function clearFilters () {
    branchId.value = null
    salespersonId.value = null
    period.value = 'month'
    month.value = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
    dateRange.value = []
  }

  // The selected period as inclusive days for the API; none = all time.
  const periodBounds = computed(() => {
    if (period.value === 'month') {
      const [year, monthNumber] = month.value.split('-').map(Number)
      const lastDay = new Date(year, monthNumber, 0).getDate()
      return { from: `${month.value}-01`, to: `${month.value}-${String(lastDay).padStart(2, '0')}` }
    }
    if (period.value === 'range' && selectedRange.value.length > 0) {
      return { from: selectedRange.value[0], to: selectedRange.value.at(-1) }
    }
    return {}
  })

  // Every sale of the period with its lines, in one request (see
  // api/salesperson_performance.go).
  async function loadData () {
    loading.value = true
    try {
      const [, response] = await Promise.all([
        fetchBranches(),
        authFetch(`${API_BASE}/salesperson_performance?${new URLSearchParams(periodBounds.value)}`),
      ])
      if (!response.ok) throw new Error(`Failed to load sales activity (${response.status})`)
      const data = await response.json()
      invoices.value = data?.invoices ?? []
      returns.value = data?.returns ?? []
      branchTargets.value = data?.branch_targets ?? []
    } catch (loadError) {
      notify.error(loadError.message || 'Failed to load salesperson performance.')
      invoices.value = []
      returns.value = []
      branchTargets.value = []
    } finally {
      loading.value = false
    }
    loadHours()
  }

  // Hours for the score, after the sales are on screen: the period's punches
  // and each branch's salespersons, to tell whose they are. All time has no
  // hours (the score leaves that part out).
  async function loadHours () {
    const { from, to } = periodBounds.value
    try {
      await ensureAutoDefaults()
      attendance.value = from && to ? await listEventsInRange({ from, to }) : []
      const ids = [...new Set([...invoices.value, ...attendance.value].map(r => r.branch_id))].slice(0, 30)
      staff.value = (await Promise.all(ids.map(id => listBranchSalespersons(id).catch(() => [])))).flat()
    } catch {
      attendance.value = []
    }
  }

  watch(branchId, () => {
    salespersonId.value = null
  })
  watch(() => JSON.stringify(periodBounds.value), loadData)
  watch(monthMenu, isOpen => {
    if (!isOpen) return
    const [year, selectedMonth] = month.value.split('-').map(Number)
    monthPickerYear.value = year
    monthPickerMonth.value = selectedMonth - 1
  })
  onMounted(loadData)
</script>

<style scoped>
  .performance-page { max-width: 1500px; }
  .formula-card { border-inline-start: 4px solid rgb(var(--v-theme-primary)); }
  .performance-row { cursor: pointer; border-radius: 8px; transition: background-color .15s ease; }
  .performance-row:hover, .performance-row:focus-visible { background: rgba(var(--v-theme-primary), .04); outline: none; }
  .performance-row + .performance-row { border-top: 1px solid rgba(var(--v-border-color), var(--v-border-opacity)); }
  .performance-table :deep(tbody tr) { cursor: pointer; }
  .adjustments-dialog-card {
    display: flex;
    min-height: 660px;
    max-height: 660px !important;
    height: 660px;
    flex-direction: column;
  }
  .adjustments-dialog-content { flex: 1 1 auto; overflow-y: auto; }
  .rank { display: grid; place-items: center; width: 26px; height: 26px; border-radius: 50%; background: rgba(var(--v-theme-primary), .1); color: rgb(var(--v-theme-primary)); }
  .min-width-0 { min-width: 0; }
  .formula-list { margin: 0; padding-inline-start: 18px; }
  .formula-list li + li { margin-top: 2px; }
  .score-facts { display: flex; flex-wrap: wrap; gap: 6px 20px; }
</style>
