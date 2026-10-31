<template>
  <div class="reports-page">
    <!-- Picker and filters -->
    <aside class="reports-side">
      <div class="d-flex align-center ga-2 px-4 pt-4 pb-2">
        <v-icon icon="mdi-file-chart-outline" />
        <div class="text-h6 font-weight-medium">Reports</div>
        <v-spacer />
        <v-btn-toggle
          v-model="lang"
          density="compact"
          divided
          mandatory
          rounded="lg"
          variant="outlined"
        >
          <v-btn size="small" text="EN" value="en" />
          <v-btn size="small" text="عربي" value="ar" />
        </v-btn-toggle>
      </div>
      <v-list
        v-model:selected="selectedReport"
        class="py-0"
        density="compact"
        mandatory
        nav
      >
        <v-list-item
          v-for="r in REPORTS"
          :key="r.key"
          :prepend-icon="r.icon"
          rounded="lg"
          :subtitle="r.subtitle"
          :title="r.title"
          :value="r.key"
        />
      </v-list>

      <v-divider class="my-2" />

      <div class="d-flex flex-column ga-3 px-4 pb-4">
        <div>
          <div class="d-flex align-center">
            <div class="text-caption text-medium-emphasis">Include</div>
            <v-spacer />
            <v-btn
              density="compact"
              size="x-small"
              :text="allParts ? 'None' : 'All'"
              variant="text"
              @click="toggleAllParts"
            />
          </div>
          <!-- A tree: each block of the report, then its sections and parts.
               A box with something under it ticks or clears all of it. -->
          <div v-for="group in partGroups" :key="group.comp" class="part-tree">
            <v-checkbox
              v-if="partGroups.length > 1"
              class="part-check part-check--group"
              density="compact"
              hide-details
              :indeterminate="tickedCount(group.parts) > 0 && tickedCount(group.parts) < group.parts.length"
              :label="group.title"
              :model-value="tickedCount(group.parts) === group.parts.length"
              @update:model-value="setParts(group.parts, $event)"
            />
            <div :class="{ 'part-tree__children': partGroups.length > 1 }">
              <template v-for="node in group.nodes" :key="node.key">
                <v-checkbox
                  v-if="node.part"
                  v-model="parts"
                  class="part-check"
                  density="compact"
                  hide-details
                  :label="node.part.title"
                  :value="node.part.key"
                />
                <template v-else>
                  <div class="d-flex align-center">
                    <v-checkbox
                      class="part-check"
                      density="compact"
                      hide-details
                      :indeterminate="tickedCount(node.children) > 0 && tickedCount(node.children) < node.children.length"
                      :label="node.section"
                      :model-value="tickedCount(node.children) === node.children.length"
                      @update:model-value="setParts(node.children, $event)"
                    />
                    <v-spacer />
                    <v-btn
                      density="compact"
                      :icon="openNodes.includes(node.key) ? 'mdi-chevron-up' : 'mdi-chevron-down'"
                      size="x-small"
                      :title="openNodes.includes(node.key) ? 'Hide its parts' : 'Show its parts'"
                      variant="text"
                      @click="toggleOpen(node.key)"
                    />
                  </div>
                  <div v-if="openNodes.includes(node.key)" class="part-tree__children">
                    <v-checkbox
                      v-for="part in node.children"
                      :key="part.key"
                      v-model="parts"
                      class="part-check"
                      density="compact"
                      hide-details
                      :label="part.title"
                      :value="part.key"
                    />
                  </div>
                </template>
              </template>
            </div>
          </div>
        </div>

        <v-select
          v-model="period"
          density="compact"
          hide-details
          item-title="title"
          item-value="value"
          :items="PERIODS"
          label="Period"
          variant="outlined"
        />
        <div v-if="period === 'custom'" class="d-flex ga-2">
          <DateField
            v-model="customFrom"
            density="compact"
            hide-details
            label="From"
            variant="outlined"
          />
          <DateField
            v-model="customTo"
            density="compact"
            hide-details
            label="To"
            variant="outlined"
          />
        </div>
        <v-select
          v-model="branchIds"
          chips
          closable-chips
          density="compact"
          hide-details
          :items="branchOptions"
          label="Branches"
          multiple
          persistent-placeholder
          placeholder="All branches"
          variant="outlined"
        />
        <v-autocomplete
          v-if="report.employees"
          v-model="employees"
          chips
          closable-chips
          density="compact"
          hide-details
          item-title="title"
          item-value="value"
          :items="employeeOptions"
          label="Employees"
          multiple
          persistent-placeholder
          placeholder="Everyone"
          variant="outlined"
        />
        <div v-if="needsTopN || usesPart('bestsellers')" class="d-flex ga-2">
          <v-select
            v-if="needsTopN"
            v-model="topN"
            density="compact"
            hide-details
            :items="[10, 20, 25, 50, 100]"
            label="List length"
            variant="outlined"
          />
          <v-select
            v-if="usesPart('bestsellers')"
            v-model="sortBy"
            density="compact"
            hide-details
            :items="[{ title: 'Net sales', value: 'revenue' }, { title: 'Units', value: 'units' }]"
            label="Best sellers by"
            variant="outlined"
          />
        </div>

        <v-btn
          prepend-icon="mdi-filter-remove-outline"
          :text="excludedCount > 0 ? `Left out of reports (${excludedCount})` : 'Leave out products or customers'"
          variant="tonal"
          @click="exclusionsDialog = true"
        />
        <v-btn
          :disabled="!dirty || !canEdit"
          :loading="savingDefaults"
          prepend-icon="mdi-content-save-outline"
          :text="dirty ? 'Save as default' : 'Saved as default'"
          variant="text"
          @click="saveDefaults"
        />

        <div class="d-flex ga-2 mt-1">
          <v-btn
            class="flex-grow-1"
            color="primary"
            :disabled="!ready || parts.length === 0"
            prepend-icon="mdi-file-pdf-box"
            text="Export PDF"
            variant="flat"
            @click="exportPdf"
          />
          <v-btn
            :disabled="!ready || parts.length === 0"
            prepend-icon="mdi-file-delimited-outline"
            text="CSV"
            variant="tonal"
            @click="exportCsv"
          />
        </div>
        <div class="text-caption text-medium-emphasis">
          Export PDF opens the print dialog -- pick "Save as PDF".
        </div>
      </div>
    </aside>

    <!-- The sheet at A4 width, as it will print, zoomable like a PDF viewer -->
    <main ref="previewRef" class="reports-preview">
      <div class="zoom-bar">
        <v-btn
          density="comfortable"
          icon="mdi-magnify-minus-outline"
          size="small"
          title="Zoom out"
          variant="text"
          @click="zoomBy(-1)"
        />
        <span class="zoom-value">{{ Math.round(scale * 100) }}%</span>
        <v-btn
          density="comfortable"
          icon="mdi-magnify-plus-outline"
          size="small"
          title="Zoom in"
          variant="text"
          @click="zoomBy(1)"
        />
        <v-divider class="mx-1" vertical />
        <v-btn
          :active="zoom === 'fit'"
          density="comfortable"
          prepend-icon="mdi-arrow-expand-horizontal"
          size="small"
          text="Fit width"
          variant="text"
          @click="zoom = 'fit'"
        />
        <v-btn
          :active="zoom === 1"
          density="comfortable"
          size="small"
          text="100%"
          variant="text"
          @click="zoom = 1"
        />
        <v-divider class="mx-1" vertical />
        <v-btn
          :active="darkView"
          density="comfortable"
          icon="mdi-weather-night"
          size="small"
          :title="darkView ? 'Show the page white' : 'Show the page dark (the PDF stays white)'"
          variant="text"
          @click="darkView = !darkView"
        />
      </div>

      <PreparingState
        v-if="(loading && !ready) || !fontsReady || !settingsReady"
        kind="pdf"
        min-height="60vh"
        subtitle="Gathering the numbers and laying out the pages"
        title="Preparing your report"
      />
      <v-alert v-else-if="loadError" class="ma-6" type="error" variant="tonal">{{ loadError }}</v-alert>
      <v-alert v-else-if="parts.length === 0" class="ma-6" type="info" variant="tonal">
        Tick at least one part of the report on the left.
      </v-alert>
      <div v-else-if="ready" class="zoom-box" :style="{ width: `${pageWidth * scale}px`, height: `${sheetHeight * scale}px` }">
        <div class="zoom-inner" :class="{ 'zoom-inner--dark': darkView }" :style="{ width: `${pageWidth}px`, transform: `scale(${scale})` }">
          <div
            :key="`${report.key}|${sheetLang}`"
            ref="sheetRef"
            class="rpt-sheet"
            :class="{ 'reports-loading': loading }"
            :dir="sheetLang === 'ar' ? 'rtl' : 'ltr'"
            :lang="sheetLang"
            :style="{ width: `${pageWidth}px` }"
          >
            <header class="rpt-head">
              <div>
                <h1 class="rpt-title">{{ sheetTitle }}</h1>
                <p class="rpt-subtitle">{{ periodLabel }}</p>
              </div>
              <div class="rpt-meta">
                <div><strong>{{ branchLabel }}</strong></div>
                <div v-if="report.employees && employees.length > 0">{{ employeeLabel }}</div>
                <div>{{ t('Generated {at}', { at: generatedLabel }) }}</div>
                <div v-if="excludedCount > 0 && needsSales">
                  {{ t('Left out: {products} products, {clients} customers', { products: exclusions.products.length, clients: exclusions.clients.length }) }}
                </div>
              </div>
            </header>

            <div v-for="(block, index) in blocks" :key="block.comp" class="rpt-block">
              <h2 v-if="index > 0 && block.title" class="rpt-block-title">{{ block.title }}</h2>
              <component
                :is="COMPONENTS[block.comp]"
                :ref="el => (blockRefs[block.comp] = el)"
                :ctx="ctx"
                :parts="block.parts"
              />
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Products and customers left out of every report -->
    <v-dialog v-model="exclusionsDialog" max-width="620">
      <v-card>
        <v-card-title>Left out of reports</v-card-title>
        <v-card-text>
          <p class="text-body-2 text-medium-emphasis mb-4">
            Saved for everyone. A customer left out is gone from every figure (their sales too).
            A product left out is gone from best sellers, categories, sizes and item counts;
            the money it brought in stays in the sales totals. Pick from who and what sold in the loaded period.
          </p>

          <div class="text-subtitle-2 mb-1">Customers</div>
          <v-autocomplete
            v-model="pickClient"
            clearable
            density="compact"
            hide-details
            item-title="title"
            item-value="value"
            :items="clientChoices"
            no-data-text="No customer bought in the loaded period"
            placeholder="Search a customer by name or phone"
            prepend-inner-icon="mdi-account-search"
            return-object
            variant="outlined"
            @update:model-value="addExclusion('clients', $event)"
          />
          <div class="d-flex flex-wrap ga-1 mt-2 mb-4">
            <v-chip
              v-for="c in exclusions.clients"
              :key="c.id"
              closable
              size="small"
              :text="c.label"
              @click:close="removeExclusion('clients', c.id)"
            />
            <span v-if="exclusions.clients.length === 0" class="text-caption text-medium-emphasis">None</span>
          </div>

          <div class="text-subtitle-2 mb-1">Products</div>
          <v-autocomplete
            v-model="pickProduct"
            clearable
            density="compact"
            hide-details
            item-title="title"
            item-value="value"
            :items="productChoices"
            no-data-text="No product sold in the loaded period"
            placeholder="Search a product by name or code"
            prepend-inner-icon="mdi-package-variant"
            return-object
            variant="outlined"
            @update:model-value="addExclusion('products', $event)"
          />
          <div class="d-flex flex-wrap ga-1 mt-2">
            <v-chip
              v-for="p in exclusions.products"
              :key="p.id"
              closable
              size="small"
              :text="p.label"
              @click:close="removeExclusion('products', p.id)"
            />
            <span v-if="exclusions.products.length === 0" class="text-caption text-medium-emphasis">None</span>
          </div>
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text="Done" @click="exclusionsDialog = false" />
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup>
  import { computed, onBeforeUnmount, onMounted, provide, reactive, ref, shallowRef, watch } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import DateField from '@/components/DateField.vue'
  import AttendanceReport from '@/components/reports/AttendanceReport.vue'
  import BranchesReport from '@/components/reports/BranchesReport.vue'
  import BusyTimesReport from '@/components/reports/BusyTimesReport.vue'
  import CustomersReport from '@/components/reports/CustomersReport.vue'
  import FeedbackReport from '@/components/reports/FeedbackReport.vue'
  import PaymentsReport from '@/components/reports/PaymentsReport.vue'
  import ProductsReport from '@/components/reports/ProductsReport.vue'
  import { ARABIC_FONT, installReportFonts } from '@/components/reports/reportFonts'
  import { setReportLang, t } from '@/components/reports/reportI18n'
  import { buildPeople } from '@/components/reports/reportRows'
  import SalespersonReport from '@/components/reports/SalespersonReport.vue'
  import SummaryReport from '@/components/reports/SummaryReport.vue'
  import { useAttendance } from '@/composables/useAttendance'
  import { useBranches } from '@/composables/useBranches'
  import { useBranchSalespersons } from '@/composables/useBranchSalespersons'
  import { useNotify } from '@/composables/useNotify'
  import { usePageAccess } from '@/composables/usePageAccess'
  import { useReports } from '@/composables/useReports'
  import { useReviews } from '@/composables/useReviews'
  import { buildTimesheet } from '@/utils/attendanceSessions'
  import { downloadCsv, printReport } from '@/utils/reportExport'
  import '@/styles/report-sheet.css'

  // End-of-period reports, each one a printable A4 sheet in English or
  // Arabic: pick a report and the parts of it you want, a period, branches
  // (and employees where it applies), then export to PDF or CSV. What the
  // page opens with and what's left out of every report are saved for
  // everyone (/reports/settings). Other pages link here with
  // ?report=<key>[&parts=a,b].

  const COMPONENTS = {
    summary: SummaryReport,
    branches: BranchesReport,
    customers: CustomersReport,
    busy: BusyTimesReport,
    feedback: FeedbackReport,
    salespersons: SalespersonReport,
    attendance: AttendanceReport,
    payments: PaymentsReport,
    products: ProductsReport,
  }

  // Each report is a set of parts, shown as a tree on the left: a part
  // belongs to one component (its block on the sheet, the tree's top level)
  // and, when several parts share one heading on the sheet, to a `section`
  // (the tree's middle level), so a whole section or one piece of it can be
  // left out. The component draws the parts it is given. title is English
  // and goes through t() on the sheet.
  const REPORTS = [
    {
      key: 'overview', title: 'Monthly overview', subtitle: 'Sales, days, branches, best sellers', icon: 'mdi-view-dashboard-outline',
      parts: [
        { key: 'kpis.summary', comp: 'summary', title: 'Headline figures' },
        { key: 'compare', comp: 'summary', title: 'Compared with the period before' },
        { key: 'days.chart', comp: 'summary', section: 'Sales day by day', title: 'Daily chart' },
        { key: 'days.weekday', comp: 'summary', section: 'Sales day by day', title: 'Average by weekday' },
        { key: 'days.best', comp: 'summary', section: 'Sales day by day', title: 'Best and slowest days' },
        { key: 'branches', comp: 'summary', title: 'Branches' },
        { key: 'sold.products', comp: 'summary', section: 'What sold', title: 'Top 10 products' },
        { key: 'sold.categories', comp: 'summary', section: 'What sold', title: 'Categories' },
        { key: 'people.salespersons', comp: 'summary', section: 'People and payments', title: 'Top salespersons' },
        { key: 'people.payments', comp: 'summary', section: 'People and payments', title: 'Payment methods' },
        { key: 'comparison.table', comp: 'branches', title: 'Branches side by side', off: true },
        { key: 'comparison.trend', comp: 'branches', title: 'Day by day', off: true },
      ],
    },
    {
      key: 'customers', title: 'Customers', subtitle: 'Best customers, when they come, busy hours', icon: 'mdi-account-star-outline',
      parts: [
        { key: 'kpis.customers', comp: 'customers', title: 'Headline figures' },
        { key: 'best', comp: 'customers', title: 'Best customers' },
        { key: 'frequent', comp: 'customers', title: 'Most frequent visitors' },
        { key: 'who.visits', comp: 'customers', section: 'Who they are', title: 'By number of visits' },
        { key: 'who.spend', comp: 'customers', section: 'Who they are', title: 'By amount spent' },
        { key: 'who.home', comp: 'customers', section: 'Who they are', title: 'By home branch' },
        { key: 'who.top', comp: 'customers', section: 'Who they are', title: 'How much the top customers bring' },
        { key: 'when.heatmap', comp: 'customers', section: 'When customers come', title: 'Weekday and hour' },
        { key: 'when.busiest', comp: 'customers', section: 'When customers come', title: 'Busiest times' },
        { key: 'kpis.busy', comp: 'busy', title: 'Headline figures' },
        { key: 'busy.heatmap', comp: 'busy', section: 'Sales by weekday and hour', title: 'Weekday and hour' },
        { key: 'busy.busiest', comp: 'busy', section: 'Sales by weekday and hour', title: 'Busiest times' },
        { key: 'busy.hour', comp: 'busy', section: 'Through the day and the week', title: 'Sales per hour' },
        { key: 'busy.weekday', comp: 'busy', section: 'Through the day and the week', title: 'Net sales per weekday' },
        { key: 'busy.avg', comp: 'busy', section: 'Through the day and the week', title: 'Average sale by hour' },
        { key: 'footfall', comp: 'busy', title: 'Footfall and conversion' },
      ],
    },
    {
      key: 'feedback', title: 'Customer feedback', subtitle: 'Poll answers, replies, Google reviews', icon: 'mdi-star-outline',
      parts: [
        { key: 'kpis.feedback', comp: 'feedback', title: 'Headline figures' },
        { key: 'ratings.answers', comp: 'feedback', section: 'Poll answers', title: 'Answers' },
        { key: 'ratings.daily', comp: 'feedback', section: 'Poll answers', title: 'Answers by day' },
        { key: 'ratings.branches', comp: 'feedback', section: 'Poll answers', title: 'By branch' },
        { key: 'reviews.funnel', comp: 'feedback', section: 'Reviews', title: 'Review requests' },
        { key: 'reviews.google', comp: 'feedback', section: 'Reviews', title: 'Google' },
        { key: 'unhappy', comp: 'feedback', title: 'Unhappy customers' },
        { key: 'replies', comp: 'feedback', title: 'What customers wrote back' },
      ],
    },
    {
      key: 'staff', title: 'Salespersons & attendance', subtitle: 'Sales per person, hours, $/hour', icon: 'mdi-account-cash', employees: true,
      parts: [
        { key: 'kpis.salespersons', comp: 'salespersons', title: 'Headline figures' },
        { key: 'ranking', comp: 'salespersons', title: 'Ranking' },
        { key: 'categories', comp: 'salespersons', title: 'Category mix' },
        { key: 'staffDiscounts', comp: 'payments', title: 'Discounts by salesperson', off: true },
        { key: 'hours', comp: 'attendance', title: 'Hours per employee' },
        { key: 'grid', comp: 'attendance', title: 'Month calendar' },
        { key: 'unlinked', comp: 'attendance', title: 'Punches nobody is linked to' },
      ],
    },
    {
      key: 'attendanceDetail', title: 'Attendance by employee', subtitle: 'A page per employee, every day', icon: 'mdi-account-clock-outline', employees: true,
      parts: [
        { key: 'unlinked', comp: 'attendance', title: 'Punches nobody is linked to' },
        { key: 'detail', comp: 'attendance', title: 'Every day per employee' },
      ],
    },
    {
      key: 'sales', title: 'Sales & goods', subtitle: 'Best sellers, categories, payments, discounts', icon: 'mdi-package-variant',
      parts: [
        { key: 'kpis.products', comp: 'products', title: 'Headline figures' },
        { key: 'categories', comp: 'products', title: 'Categories' },
        { key: 'bestsellers', comp: 'products', title: 'Best sellers' },
        { key: 'profitable', comp: 'products', section: 'Profit', title: 'Most profitable' },
        { key: 'lowMargin', comp: 'products', section: 'Profit', title: 'Lowest margins' },
        { key: 'by.brand', comp: 'products', section: 'Breakdowns', title: 'By brand' },
        { key: 'by.type', comp: 'products', section: 'Breakdowns', title: 'By product type' },
        { key: 'by.audience', comp: 'products', section: 'Breakdowns', title: 'By who it\'s for' },
        { key: 'by.origin', comp: 'products', section: 'Breakdowns', title: 'By country of origin' },
        { key: 'by.year', comp: 'products', section: 'Breakdowns', title: 'By collection year' },
        { key: 'by.season', comp: 'products', section: 'Breakdowns', title: 'By season' },
        { key: 'by.supplier', comp: 'products', section: 'Breakdowns', title: 'By supplier' },
        { key: 'priceBands', comp: 'products', section: 'Breakdowns', title: 'Price ranges' },
        { key: 'attrs.sizes', comp: 'products', section: 'Sizes and colours', title: 'Units by size' },
        { key: 'attrs.colours', comp: 'products', section: 'Sizes and colours', title: 'Units by colour' },
        { key: 'returned', comp: 'products', title: 'Most returned' },
        { key: 'kpis.payments', comp: 'payments', title: 'Headline figures' },
        { key: 'payments', comp: 'payments', title: 'Payment methods' },
        { key: 'discounts', comp: 'payments', title: 'Discounts' },
        { key: 'returns', comp: 'payments', title: 'Returns by branch' },
      ],
    },
  ]
  // A part saved (or linked) before parts were split stands for its pieces.
  const OLD_PARTS = {
    days: ['days.chart', 'days.weekday', 'days.best'],
    sold: ['sold.products', 'sold.categories'],
    people: ['people.salespersons', 'people.payments'],
    comparison: ['comparison.table', 'comparison.trend'],
    who: ['who.visits', 'who.spend', 'who.home'],
    when: ['when.heatmap', 'when.busiest'],
    busy: ['busy.heatmap', 'busy.busiest', 'busy.hour', 'busy.weekday'],
    ratings: ['ratings.answers', 'ratings.daily', 'ratings.branches'],
    reviews: ['reviews.funnel', 'reviews.google'],
    attrs: ['attrs.sizes', 'attrs.colours'],
    by: ['by.brand', 'by.type', 'by.audience', 'by.origin', 'by.year', 'by.season', 'by.supplier', 'priceBands'],
  }
  // Headings over the second and later blocks of a report, and the group
  // labels of the part checkboxes.
  const BLOCK_TITLES = {
    summary: 'Overview',
    branches: 'Branch comparison',
    customers: 'Customers',
    busy: 'Busy times',
    feedback: 'Customer feedback',
    salespersons: 'Salesperson performance',
    attendance: 'Attendance & hours',
    products: 'Goods',
    payments: 'Payments, discounts and returns',
  }
  // Report keys from before the reports were grouped, for old links.
  const OLD_KEYS = {
    summary: ['overview'],
    branches: ['overview', ['comparison']],
    attendance: ['staff', ['hours', 'grid']],
    salespersons: ['staff', ['ranking', 'categories']],
    busy: ['customers', ['kpis.busy', 'busy', 'footfall']],
    products: ['sales', ['kpis.products', 'categories', 'bestsellers', 'attrs', 'returned']],
    payments: ['sales', ['kpis.payments', 'payments', 'discounts', 'returns']],
  }

  const route = useRoute()
  const router = useRouter()
  const notify = useNotify()
  const { canEdit } = usePageAccess()
  const { getReports, getReportSettings, saveReportSettings } = useReports()
  const { listEventsInRange } = useAttendance()
  const { listBranchSalespersons } = useBranchSalespersons()
  const { getReviews } = useReviews()
  const { branches, fetchBranches } = useBranches()

  function stored (key, fallback) {
    try {
      return localStorage.getItem(key) ?? fallback
    } catch {
      return fallback
    }
  }
  function store (key, value) {
    try {
      localStorage.setItem(key, value)
    } catch {
      // private window: the zoom just isn't remembered
    }
  }

  // ---- language -----------------------------------------------------------
  // lang is the switch; sheetLang is what the sheet is drawn in, which
  // follows only once the Arabic face is loaded -- charts draw on canvas
  // once and wouldn't pick it up later.
  const lang = ref('en')
  const sheetLang = ref('en')
  setReportLang('en')
  const fontsReady = ref(true)
  async function loadArabicFont () {
    fontsReady.value = false
    installReportFonts()
    await Promise.all([400, 500, 700].map(w => document.fonts.load(`${w} 16px "${ARABIC_FONT}"`, 'ع').catch(() => {})))
    fontsReady.value = true
  }
  watch(lang, async value => {
    if (value === 'ar') {await loadArabicFont()}
    setReportLang(value)
    sheetLang.value = value
  })

  // ---- report and parts -----------------------------------------------------
  const [linkedKey, linkedParts] = REPORTS.some(r => r.key === route.query.report)
    ? [route.query.report, null]
    : (OLD_KEYS[route.query.report] ?? ['overview'])
  const selectedReport = ref([linkedKey])
  const report = computed(() => REPORTS.find(r => r.key === selectedReport.value[0]) ?? REPORTS[0])
  const queryParts = typeof route.query.parts === 'string' ? route.query.parts.split(',') : linkedParts

  // Known part keys only, old combined parts opened into their pieces.
  function validParts (r, keys) {
    return keys
      .flatMap(k => OLD_PARTS[k] ?? [k])
      .filter((k, i, all) => all.indexOf(k) === i && r.parts.some(p => p.key === k))
  }
  // A report's parts: the saved default, else every part not marked off. A
  // default saved before the headline figures were a part of their own
  // keeps them.
  function defaultParts (r) {
    const saved = settings.value.defaults?.parts?.[r.key]
    if (!Array.isArray(saved)) {return r.parts.filter(p => !p.off).map(p => p.key)}
    const keys = validParts(r, saved)
    if (!saved.some(k => k.startsWith('kpis.'))) {
      keys.push(...r.parts.filter(p => p.key.startsWith('kpis.') && keys.some(k => r.parts.find(q => q.key === k)?.comp === p.comp)).map(p => p.key))
    }
    // A part added since the default was saved starts ticked (unless it is
    // off by default) rather than missing. Defaults saved before parts were
    // recorded only predate 'unlinked'.
    const seen = settings.value.defaults?.parts_seen?.[r.key] ?? r.parts.map(p => p.key).filter(k => k !== 'unlinked')
    keys.push(...r.parts.filter(p => !p.off && !seen.includes(p.key) && !keys.includes(p.key)).map(p => p.key))
    return keys
  }
  const parts = ref([])
  watch(report, r => {
    parts.value = defaultParts(r)
    employees.value = []
    router.replace({ query: { report: r.key } })
  })

  const allParts = computed(() => parts.value.length === report.value.parts.length)
  function toggleAllParts () {
    parts.value = allParts.value ? [] : report.value.parts.map(p => p.key)
  }
  // The tree on the left: components, then sections (parts sharing one
  // heading on the sheet) or single parts.
  const partGroups = computed(() => {
    const groups = []
    for (const p of report.value.parts) {
      let group = groups.find(g => g.comp === p.comp)
      if (!group) {
        group = { comp: p.comp, title: BLOCK_TITLES[p.comp], parts: [], nodes: [] }
        groups.push(group)
      }
      group.parts.push(p)
      const node = p.section && group.nodes.find(n => n.section === p.section)
      if (node) {
        node.children.push(p)
      } else {
        group.nodes.push(p.section ? { section: p.section, key: `${p.comp}|${p.section}`, children: [p] } : { key: p.key, part: p })
      }
    }
    return groups
  })
  // Ticking a group or section ticks everything under it.
  const keysOf = list => list.map(p => p.key)
  const tickedCount = list => list.filter(p => parts.value.includes(p.key)).length
  function setParts (list, on) {
    const keys = keysOf(list)
    parts.value = on
      ? [...parts.value, ...keys.filter(k => !parts.value.includes(k))]
      : parts.value.filter(k => !keys.includes(k))
  }
  const openNodes = ref([])
  function toggleOpen (key) {
    openNodes.value = openNodes.value.includes(key) ? openNodes.value.filter(k => k !== key) : [...openNodes.value, key]
  }
  // The sheet's Hide buttons untick their part(s).
  provide('reportHideParts', keys => {
    parts.value = parts.value.filter(k => !keys.includes(k))
  })
  // The components to draw, in report order, each with its ticked parts.
  const blocks = computed(() => partGroups.value
    .map(g => ({ comp: g.comp, title: sheetLang.value && t(BLOCK_TITLES[g.comp]), parts: g.parts.filter(p => parts.value.includes(p.key)).map(p => p.key) }))
    .filter(b => b.parts.length > 0))
  const usesComp = comp => blocks.value.some(b => b.comp === comp)
  const usesPart = key => parts.value.includes(key) && report.value.parts.some(p => p.key === key)
  const needsTopN = computed(() => usesComp('customers') || usesComp('products'))

  // ---- filters ----------------------------------------------------------------
  const pad = n => String(n).padStart(2, '0')
  const ymd = d => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
  const PERIODS = [
    { title: 'This month', value: 'this-month' },
    { title: 'Last month', value: 'last-month' },
    { title: 'This week', value: 'this-week' },
    { title: 'Last week', value: 'last-week' },
    { title: 'Last 30 days', value: 'last-30' },
    { title: 'This year', value: 'this-year' },
    { title: 'Custom…', value: 'custom' },
  ]
  // Early in a month the month that just ended is the one people report on.
  const autoPeriod = () => (new Date().getDate() <= 5 ? 'last-month' : 'this-month')
  const period = ref(autoPeriod())
  const customFrom = ref(ymd(new Date(new Date().getFullYear(), new Date().getMonth(), 1)))
  const customTo = ref(ymd(new Date()))

  const range = computed(() => {
    const today = new Date()
    const y = today.getFullYear()
    const m = today.getMonth()
    const monday = new Date(today)
    monday.setDate(today.getDate() - ((today.getDay() + 6) % 7))
    switch (period.value) {
      case 'last-month': {
        return { from: ymd(new Date(y, m - 1, 1)), to: ymd(new Date(y, m, 0)) }
      }
      case 'this-week': {
        return { from: ymd(monday), to: ymd(today) }
      }
      case 'last-week': {
        const start = new Date(monday)
        start.setDate(monday.getDate() - 7)
        const end = new Date(monday)
        end.setDate(monday.getDate() - 1)
        return { from: ymd(start), to: ymd(end) }
      }
      case 'last-30': {
        const start = new Date(today)
        start.setDate(today.getDate() - 29)
        return { from: ymd(start), to: ymd(today) }
      }
      case 'this-year': {
        return { from: `${y}-01-01`, to: ymd(today) }
      }
      case 'custom': {
        return { from: customFrom.value, to: customTo.value }
      }
      default: {
        return { from: ymd(new Date(y, m, 1)), to: ymd(today) }
      }
    }
  })

  const branchIds = ref([])
  const branchOptions = computed(() => branches.value.map(b => ({ title: b.name, value: b.id })))
  const branchNames = computed(() => Object.fromEntries(branches.value.map(b => [b.id, b.name])))
  const employees = ref([])
  const topN = ref(25)
  const sortBy = ref('revenue')

  // ---- saved settings ---------------------------------------------------------
  // { defaults: { branch_ids, period, lang, top_n, sort_by, parts: { report: [..] } },
  //   exclusions: { products: [{ id, label }], clients: [{ id, label }] } }
  const settings = ref({})
  const settingsReady = ref(false)
  const exclusions = computed(() => ({
    products: settings.value.exclusions?.products ?? [],
    clients: settings.value.exclusions?.clients ?? [],
  }))
  const excludedCount = computed(() => exclusions.value.products.length + exclusions.value.clients.length)

  // Poll replies hidden from the feedback report by hand (small talk the
  // automatic filter let through). Saved for everyone, like the rest.
  const hiddenReplies = computed(() => settings.value.hidden_replies ?? [])
  async function toggleReply (id) {
    const list = hiddenReplies.value.includes(id)
      ? hiddenReplies.value.filter(x => x !== id)
      : [...hiddenReplies.value, id]
    try {
      await saveSettings({ ...settings.value, hidden_replies: list })
    } catch (error) {
      notify.error(error.message)
    }
  }

  // People left out of the attendance reports (the owner, a test user),
  // saved for everyone like the hidden replies. [{ key, label }]: the label
  // names them in the "Hidden" list even in a period they didn't work.
  const hiddenPeople = computed(() => settings.value.hidden_people ?? [])
  async function togglePerson (key, label) {
    const list = hiddenPeople.value.some(x => x.key === key)
      ? hiddenPeople.value.filter(x => x.key !== key)
      : [...hiddenPeople.value, { key, label }]
    try {
      await saveSettings({ ...settings.value, hidden_people: list })
    } catch (error) {
      notify.error(error.message)
    }
  }

  async function loadSettings () {
    try {
      settings.value = (await getReportSettings()) ?? {}
    } catch (error) {
      notify.error(error.message)
    }
    const d = settings.value.defaults ?? {}
    if (Array.isArray(d.branch_ids)) {branchIds.value = d.branch_ids.filter(id => Number.isInteger(id))}
    // A link's period wins over the saved one.
    if (PERIODS.some(p => p.value === route.query.period)) {
      period.value = route.query.period
    } else if (PERIODS.some(p => p.value === d.period) && d.period !== 'custom') {
      period.value = d.period
    }
    if ([10, 20, 25, 50, 100].includes(d.top_n)) {topN.value = d.top_n}
    if (d.sort_by === 'units' || d.sort_by === 'revenue') {sortBy.value = d.sort_by}
    parts.value = queryParts ? validParts(report.value, queryParts) : defaultParts(report.value)
    if (d.lang === 'ar') {lang.value = 'ar'}
    settingsReady.value = true
  }

  // What Save as default would store, to tell whether anything changed.
  const currentDefaults = computed(() => ({
    branch_ids: [...branchIds.value].toSorted((a, b) => a - b),
    period: period.value === 'custom' ? autoPeriod() : period.value,
    lang: lang.value,
    top_n: topN.value,
    sort_by: sortBy.value,
    parts: { ...settings.value.defaults?.parts, [report.value.key]: parts.value },
    parts_seen: { ...settings.value.defaults?.parts_seen, [report.value.key]: report.value.parts.map(p => p.key) },
  }))
  const dirty = computed(() => {
    const d = settings.value.defaults ?? {}
    const c = currentDefaults.value
    return JSON.stringify([c.branch_ids, c.period, c.lang, c.top_n, c.sort_by, [...parts.value].toSorted()])
      !== JSON.stringify([[...(d.branch_ids ?? [])].toSorted((a, b) => a - b), d.period ?? autoPeriod(), d.lang ?? 'en', d.top_n ?? 25, d.sort_by ?? 'revenue', [...defaultParts(report.value)].toSorted()])
  })

  const savingDefaults = ref(false)
  async function saveSettings (next) {
    const saved = await saveReportSettings(next)
    settings.value = saved ?? next
  }
  async function saveDefaults () {
    savingDefaults.value = true
    try {
      await saveSettings({ ...settings.value, defaults: currentDefaults.value })
      notify.success('Saved: the Reports page opens like this from now on.')
    } catch (error) {
      notify.error(error.message)
    } finally {
      savingDefaults.value = false
    }
  }

  // ---- left out of reports ---------------------------------------------------
  const exclusionsDialog = ref(false)
  const pickClient = ref(null)
  const pickProduct = ref(null)
  const clientChoices = computed(() => (data.value?.clients ?? [])
    .filter(c => !exclusions.value.clients.some(x => x.id === c.client_id))
    .map(c => ({ title: c.phone ? `${c.name} · ${c.phone}` : c.name, value: c.client_id })))
  const productChoices = computed(() => (data.value?.products ?? [])
    .filter(p => p.product_id > 0 && !exclusions.value.products.some(x => x.id === p.product_id))
    .map(p => ({ title: p.product_code ? `${p.product_name} · ${p.product_code}` : p.product_name, value: p.product_id })))

  async function changeExclusions (kind, list) {
    try {
      await saveSettings({ ...settings.value, exclusions: { ...exclusions.value, [kind]: list } })
      // The figures change: fetch them again.
      loadedKeys.sales = ''
      load()
    } catch (error) {
      notify.error(error.message)
    }
  }
  function addExclusion (kind, item) {
    if (!item) {return}
    changeExclusions(kind, [...exclusions.value[kind], { id: item.value, label: item.title }])
    // Clear the box once the pick is taken.
    setTimeout(() => {
      pickClient.value = null
      pickProduct.value = null
    })
  }
  function removeExclusion (kind, id) {
    changeExclusions(kind, exclusions.value[kind].filter(x => x.id !== id))
  }

  // ---- data -------------------------------------------------------------------
  // Each source is loaded once per filter set and kept while switching
  // reports; attendance and feedback only when a ticked part needs them.
  const loading = ref(false)
  const loadError = ref('')
  const data = shallowRef(null)
  const attendance = shallowRef([])
  const attendanceError = ref('')
  const reviews = shallowRef(null)
  const reviewsError = ref('')
  const staff = shallowRef([])
  const loadedKeys = { sales: '', attendance: '', reviews: '' }

  const filterKey = computed(() => `${range.value.from}|${range.value.to}|${branchIds.value.join(',')}`)
  // Sales figures feed everything but feedback; attendance feeds the
  // attendance parts and salespersons' hours.
  const needsSales = computed(() => blocks.value.some(b => b.comp !== 'feedback'))
  const needsAttendance = computed(() => usesComp('attendance') || usesComp('salespersons'))
  const needsReviews = computed(() => usesComp('feedback'))

  let seq = 0
  async function load () {
    if (!settingsReady.value) {return}
    const { from, to } = range.value
    if (!from || !to || from > to) {return}
    const key = filterKey.value
    const jobs = []
    if (needsSales.value && loadedKeys.sales !== key) {
      loadedKeys.sales = key
      jobs.push(getReports({ from, to, branchIds: branchIds.value }).then(body => {
        data.value = body
      }).catch(error => {
        loadedKeys.sales = ''
        throw error
      }))
    }
    if (needsAttendance.value && loadedKeys.attendance !== key) {
      loadedKeys.attendance = key
      // The attendance endpoint takes one branch or all; several picked
      // branches are filtered here.
      const one = branchIds.value.length === 1 ? branchIds.value[0] : null
      jobs.push(listEventsInRange({ branchId: one, from, to }).then(events => {
        attendance.value = branchIds.value.length > 1 ? events.filter(e => branchIds.value.includes(e.branch_id)) : events
        attendanceError.value = ''
      }).catch(error => {
        attendance.value = []
        attendanceError.value = error.message
      }))
    }
    if (needsReviews.value && loadedKeys.reviews !== key) {
      loadedKeys.reviews = key
      const params = { from, to }
      if (branchIds.value.length === 1) {params.branch_id = branchIds.value[0]}
      jobs.push(getReviews(params).then(body => {
        reviews.value = body
        reviewsError.value = ''
      }).catch(error => {
        reviews.value = {}
        reviewsError.value = error.message
      }))
    }
    if (jobs.length === 0) {return}

    const mine = ++seq
    loading.value = true
    loadError.value = ''
    try {
      await Promise.all(jobs)
      if (mine === seq) {await loadStaff()}
      if (mine === seq) {generatedAt.value = new Date()}
    } catch (error) {
      if (mine === seq) {loadError.value = error.message}
    } finally {
      if (mine === seq) {loading.value = false}
    }
  }

  // kashi's salesperson list for every branch that shows up, so sales and
  // punches of one person can be put together.
  const staffBranches = new Set()
  async function loadStaff () {
    const ids = new Set([
      ...(data.value?.salespersons ?? []).map(s => s.branch_id),
      ...attendance.value.map(e => e.branch_id),
    ])
    const missing = [...ids].filter(id => !staffBranches.has(id)).slice(0, 30)
    if (missing.length === 0) {return}
    const lists = await Promise.all(missing.map(id => listBranchSalespersons(id).catch(() => [])))
    for (const id of missing) {staffBranches.add(id)}
    staff.value = [...staff.value, ...lists.flat()]
  }

  watch([filterKey, blocks], load)
  onMounted(async () => {
    fetchBranches().catch(error => notify.error(error.message))
    await loadSettings()
    load()
  })

  const timesheet = computed(() => buildTimesheet(attendance.value))
  const people = computed(() => buildPeople({ salespersons: data.value?.salespersons ?? [], staff: staff.value, timesheet: timesheet.value }))
  const employeeOptions = computed(() => people.value.list.map(p => ({
    title: branchIds.value.length === 1 ? p.name : `${p.name} · ${branchNames.value[p.branchId] ?? ''}`,
    value: p.key,
  })))

  const ready = computed(() => {
    if (needsSales.value && !data.value) {return false}
    if (needsReviews.value && !reviews.value) {return false}
    return true
  })

  const ctx = computed(() => ({
    data: data.value,
    from: range.value.from,
    to: range.value.to,
    branchIds: branchIds.value,
    branchNames: branchNames.value,
    multiBranch: branchIds.value.length !== 1 && (data.value?.branches?.length ?? 0) > 1,
    attendance: attendance.value,
    attendanceError: attendanceError.value,
    reviews: reviews.value,
    reviewsError: reviewsError.value,
    people: people.value,
    employees: employees.value,
    options: { topN: topN.value, sortBy: sortBy.value },
    hiddenReplies: hiddenReplies.value,
    toggleReply: canEdit.value ? toggleReply : null,
    hiddenPeople: hiddenPeople.value,
    togglePerson: canEdit.value ? togglePerson : null,
  }))

  // ---- A4 preview and zoom ----------------------------------------------------
  // Every report is portrait A4: 210mm at 96 dpi, with the print's 10mm
  // margins inside, so the preview wraps lines exactly as the PDF does.
  // Zoom scales the drawn sheet, like a PDF viewer, rather than redrawing it.
  const pageWidth = 794
  const ZOOMS = [0.5, 0.67, 0.75, 0.9, 1, 1.1, 1.25, 1.5, 1.75, 2]
  const savedZoom = stored('reports.zoom', 'fit')
  const zoom = ref(savedZoom === 'fit' ? 'fit' : (Number(savedZoom) || 'fit'))
  watch(zoom, value => store('reports.zoom', String(value)))
  // Dark viewing only: the frame around the sheet is inverted on screen,
  // and the PDF copies the sheet itself, which stays white.
  const darkView = ref(stored('reports.dark', '0') === '1')
  watch(darkView, value => store('reports.dark', value ? '1' : '0'))
  const previewRef = ref(null)
  const previewWidth = ref(0)
  const sheetHeight = ref(0)
  const scale = computed(() => {
    if (zoom.value !== 'fit') {return zoom.value}
    if (!previewWidth.value) {return 1}
    return Math.max(0.3, Math.min(2, (previewWidth.value - 48) / pageWidth))
  })
  function zoomBy (step) {
    const now = scale.value
    const next = step > 0 ? ZOOMS.find(z => z > now + 0.001) : ZOOMS.findLast(z => z < now - 0.001)
    zoom.value = next ?? (step > 0 ? ZOOMS.at(-1) : ZOOMS[0])
  }

  const sheetRef = ref(null)
  let observer
  onMounted(() => {
    observer = new ResizeObserver(entries => {
      for (const entry of entries) {
        if (entry.target === previewRef.value) {previewWidth.value = entry.contentRect.width + 48}
        if (entry.target === sheetRef.value) {sheetHeight.value = entry.target.offsetHeight}
      }
    })
    if (previewRef.value) {observer.observe(previewRef.value)}
  })
  watch(sheetRef, (el, old) => {
    if (old) {observer?.unobserve(old)}
    if (el) {
      observer?.observe(el)
      sheetHeight.value = el.offsetHeight
    }
  })
  onBeforeUnmount(() => observer?.disconnect())

  // ---- header and export ------------------------------------------------------
  // Each reads sheetLang so the header follows a language switch.
  const generatedAt = ref(new Date())
  const dateLocale = () => (sheetLang.value === 'ar' ? 'ar-LB-u-nu-latn' : 'en-GB')
  const listSep = () => (sheetLang.value === 'ar' ? '، ' : ', ')
  const sheetTitle = computed(() => sheetLang.value && t(report.value.title))
  const longDate = d => new Date(`${d}T00:00:00`).toLocaleDateString(dateLocale(), { day: 'numeric', month: 'long', year: 'numeric' })
  const periodLabel = computed(() => {
    const { from, to } = range.value
    const start = new Date(`${from}T00:00:00`)
    const end = new Date(`${to}T00:00:00`)
    const lastOfMonth = new Date(start.getFullYear(), start.getMonth() + 1, 0)
    if (start.getDate() === 1 && ymd(end) === ymd(lastOfMonth)) {
      return start.toLocaleDateString(dateLocale(), { month: 'long', year: 'numeric' })
    }
    return from === to ? longDate(from) : `${longDate(from)} – ${longDate(to)}`
  })
  const branchLabel = computed(() => (branchIds.value.length === 0
    ? sheetLang.value && t('All branches')
    : branchIds.value.map(id => branchNames.value[id] ?? `#${id}`).join(listSep())))
  const employeeLabel = computed(() => {
    const names = new Map(people.value.list.map(p => [p.key, p.name]))
    return employees.value.map(k => names.get(k) ?? k).join(listSep())
  })
  const fileTitle = computed(() => `${sheetTitle.value} - ${periodLabel.value}${branchIds.value.length > 0 ? ` - ${branchLabel.value}` : ''}`)
  // Shown in the sheet's language, refreshed when the figures load.
  const generatedLabel = computed(() => generatedAt.value.toLocaleString(dateLocale(), { dateStyle: 'medium', timeStyle: 'short' }))

  const blockRefs = reactive({})
  async function exportPdf () {
    if (!sheetRef.value) {return}
    try {
      await printReport(sheetRef.value, { title: fileTitle.value, lang: sheetLang.value })
    } catch (error) {
      notify.error(error.message)
    }
  }
  function exportCsv () {
    const sections = blocks.value.flatMap(b => blockRefs[b.comp]?.csv?.() ?? [])
    downloadCsv(`${fileTitle.value.replaceAll(/[\\/:*?"<>|]/g, '-')}.csv`, sections)
  }
</script>

<style scoped>
.reports-page {
  display: flex;
  flex: 1 1 auto;
  height: 100%;
  min-height: 0;
}
.reports-side {
  flex: 0 0 300px;
  overflow-y: auto;
  border-right: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.reports-preview {
  position: relative;
  flex: 1 1 auto;
  min-width: 0;
  overflow: auto;
  padding: 0 24px 24px;
  background: rgba(var(--v-theme-on-surface), 0.06);
}
.zoom-bar {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 2px;
  margin: 0 -24px 16px;
  padding: 4px 8px;
  background: rgb(var(--v-theme-surface));
  border-bottom: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.zoom-value {
  min-width: 48px;
  font-size: 0.85rem;
  text-align: center;
  font-variant-numeric: tabular-nums;
}
/* The box takes the zoomed size so the page scrolls right; the sheet
   inside keeps its A4 layout and is only scaled. */
.zoom-box {
  margin: 0 auto;
}
.zoom-inner {
  transform-origin: top left;
}
/* Inverting with a half-turn of hue keeps chart and tag colours close to
   their own hues on a dark page. */
.zoom-inner--dark {
  filter: invert(0.9) hue-rotate(180deg);
}
.reports-preview .rpt-sheet {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.18);
  border-radius: 2px;
}
.reports-loading {
  opacity: 0.55;
  transition: opacity 0.2s;
}
.part-check :deep(.v-selection-control) {
  min-height: 30px;
}
.part-check :deep(.v-label) {
  font-size: 0.85rem;
}
.part-check--group :deep(.v-label) {
  font-weight: 600;
}
.part-tree__children {
  padding-inline-start: 22px;
}
@media (max-width: 960px) {
  .reports-page { flex-direction: column; height: auto; }
  .reports-side { flex: 0 0 auto; border-right: none; }
  .reports-preview { padding: 0 8px 8px; }
  .zoom-bar { margin: 0 -8px 8px; }
}
</style>
