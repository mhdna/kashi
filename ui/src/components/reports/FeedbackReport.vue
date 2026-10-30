<template>
  <p v-if="ctx.reviewsError" class="rpt-empty">{{ t('Feedback couldn\'t be loaded: {error}', { error: ctx.reviewsError }) }}</p>
  <template v-else-if="r">
    <p v-if="ctx.branchIds.length > 1" class="rpt-note">
      {{ t('Feedback can be shown for one branch or for all of them; with several branches picked this covers every branch.') }}
    </p>
    <RptKpis :items="kpis" />
    <p v-if="!totals.polls && !funnel.with_link && places.length === 0" class="rpt-empty">
      {{ t('No feedback was collected in this period: no rating polls or review links went out, and no Google place is set up.') }}
    </p>

    <RptSection v-if="has('ratings') && totals.answers > 0" :note="t('Answers to the rating poll sent on WhatsApp after a sale.')" :title="t('How customers rated us')">
      <RptGrid>
        <RptCard :title="t('Answers by rating')">
          <RptChart :height="200" :option="starsChart" />
        </RptCard>
        <RptCard :title="t('Average rating by day')">
          <RptChart :height="200" :option="averageChart" />
        </RptCard>
      </RptGrid>
      <RptTable v-if="ratingBranches.length > 1" :columns="branchColumns" compact :rows="ratingBranches" />
    </RptSection>

    <template v-if="has('reviews')">
      <RptSection v-if="funnel.with_link > 0" :note="t('Post-sale WhatsApp messages carrying the Google review link.')" :title="t('Review requests')">
        <RptGrid>
          <RptChart :height="180" :option="funnelChart" />
          <RptTable :columns="funnelColumns" compact :rows="funnelRows" />
        </RptGrid>
      </RptSection>

      <RptSection v-if="places.length > 0" :title="t('Google')">
        <RptTable :columns="placeColumns" compact :rows="places" />
        <template v-if="googleReviews.length > 0">
          <h3 class="rpt-h3">{{ t('Latest Google reviews') }}</h3>
          <blockquote v-for="g in googleReviews" :key="g.review_name" class="rpt-quote">
            <div>{{ stars(g.rating) }} {{ g.body || t('(no text)') }}</div>
            <div class="rpt-quote__meta">{{ g.author }} · {{ dateTime(nv(g.published_at)) }}</div>
          </blockquote>
        </template>
      </RptSection>
    </template>

    <RptSection v-if="has('unhappy') && low.length > 0" :note="t('Customers who gave 1 or 2 stars -- worth a call.')" :title="t('Unhappy customers')">
      <RptTable :columns="lowColumns" compact :rows="low" />
    </RptSection>

    <RptSection v-if="has('replies') && replies.length > 0" :title="t('What customers wrote back')">
      <blockquote v-for="reply in replies.slice(0, 40)" :key="reply.id" class="rpt-quote">
        <div>{{ reply.body || `(${reply.kind})` }}</div>
        <div class="rpt-quote__meta">
          {{ reply.client_name || reply.client_phone }}<template v-if="nv(reply.poll_rating)"> · {{ stars(nv(reply.poll_rating)) }}</template>
          · {{ reply.branch_name }} · {{ dateTime(reply.received_at) }}
        </div>
      </blockquote>
      <p v-if="replies.length > 40" class="rpt-note">{{ t('And {n} more; the CSV export has them all.', { n: replies.length - 40 }) }}</p>
    </RptSection>
  </template>
</template>

<script setup>
  import { computed } from 'vue'
  import { t } from './reportI18n'
  import { barOption, csvSection, dateTime, dayLabel, int, lineOption, num1, nv, pct, ratio } from './reportKit'
  import RptCard from './RptCard.vue'
  import RptChart from './RptChart.vue'
  import RptGrid from './RptGrid.vue'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // What customers told us: the WhatsApp rating poll, their replies, the
  // review-link funnel and the Google ratings (all from /reviews).
  // Parts: ratings, reviews, unhappy, replies.
  const props = defineProps({
    ctx: { type: Object, required: true },
    parts: { type: Array, required: true },
  })
  const has = part => props.parts.includes(part)
  const r = computed(() => props.ctx.reviews)

  const stars = n => '★'.repeat(Number(n) || 0)
  const totals = computed(() => r.value.ratings?.totals ?? {})
  const funnel = computed(() => r.value.funnel ?? {})
  const places = computed(() => (r.value.places ?? []).filter(p => p.checked).map(p => ({ ...p, branches: p.branches.join(', ') })))
  const googleReviews = computed(() => (r.value.reviews ?? []).slice(0, 8))
  const replies = computed(() => r.value.ratings?.replies ?? [])
  const low = computed(() => (r.value.ratings?.low ?? []).map(l => ({ ...l, rating: nv(l.poll_rating), at: nv(l.poll_voted_at) })))
  const ratingBranches = computed(() => r.value.ratings?.branches ?? [])

  const happy = computed(() => (totals.value.stars5 ?? 0) + (totals.value.stars4 ?? 0))
  const unhappy = computed(() => (totals.value.stars1 ?? 0) + (totals.value.stars2 ?? 0))
  const kpis = computed(() => {
    const tt = totals.value
    const items = [
      { label: t('Ratings asked'), value: int(tt.polls) },
      { label: t('Answered'), value: int(tt.answers), hint: pct(ratio(tt.answers, tt.polls)) },
      { label: t('Average rating'), value: tt.answers ? `${num1(tt.average)} / 5` : '—' },
      { label: t('Loved it (4–5★)'), value: pct(ratio(happy.value, tt.answers)), hint: t('{n} customers', { n: int(happy.value) }) },
      { label: t('Unhappy (1–2★)'), value: pct(ratio(unhappy.value, tt.answers)), hint: t('{n} customers', { n: int(unhappy.value) }) },
      { label: t('Replies received'), value: int(replies.value.length) },
    ]
    if (funnel.value.with_link > 0) {
      items.push({ label: t('Review links opened'), value: int(funnel.value.clicked), hint: t('{pct} of links sent', { pct: pct(ratio(funnel.value.clicked, funnel.value.with_link)) }) })
    }
    for (const p of places.value.slice(0, 2)) {
      items.push({
        label: `${t('Google')} · ${p.name || t('rating')}`,
        value: `${num1(p.rating)} ★`,
        hint: p.new_reviews === null
          ? t('{n} reviews', { n: int(p.review_count) })
          : t('{n} reviews, +{new} this period', { n: int(p.review_count), new: int(p.new_reviews) }),
      })
    }
    return items
  })

  const starsChart = computed(() => {
    const tt = totals.value
    return barOption({
      categories: ['5★', '4★', '3★', '2★', '1★'],
      series: [{ name: t('Answers'), data: [tt.stars5, tt.stars4, tt.stars3, tt.stars2, tt.stars1].map(v => v ?? 0) }],
      labels: true,
    })
  })
  const averageChart = computed(() => {
    const days = r.value.ratings?.days ?? []
    return lineOption({ categories: days.map(d => dayLabel(d.day, { day: 'numeric', month: 'short' })), series: [{ name: t('Average stars'), data: days.map(d => Math.round(d.average * 10) / 10) }], kind: 'num1' })
  })

  const branchColumns = [
    { key: 'branch_name', title: t('Branch') },
    { key: 'polls', title: t('Asked'), type: 'int' },
    { key: 'answers', title: t('Answered'), type: 'int' },
    { key: 'average', title: t('Average'), type: 'num1' },
    { key: 'stars5', title: '5★', type: 'int' },
    { key: 'stars4', title: '4★', type: 'int' },
    { key: 'stars3', title: '3★', type: 'int' },
    { key: 'stars2', title: '2★', type: 'int' },
    { key: 'stars1', title: '1★', type: 'int' },
  ]
  const funnelRows = computed(() => {
    const f = funnel.value
    return [
      { step: t('Sent with a review link'), count: f.with_link },
      { step: t('Delivered'), count: f.delivered },
      { step: t('Read'), count: f.read },
      { step: t('Opened the link'), count: f.clicked },
    ].map(row => ({ ...row, rate: ratio(row.count, f.with_link) }))
  })
  const funnelColumns = [
    { key: 'step', title: t('Step') },
    { key: 'count', title: t('Messages'), type: 'int' },
    { key: 'rate', title: t('Of sent'), type: 'pct' },
  ]
  const funnelChart = computed(() => barOption({ categories: funnelRows.value.map(f => f.step), series: [{ name: t('Messages'), data: funnelRows.value.map(f => f.count ?? 0) }], horizontal: true, labels: true }))
  const placeColumns = [
    { key: 'name', title: t('Place') },
    { key: 'branches', title: t('Branches') },
    { key: 'rating', title: t('Rating'), type: 'num1' },
    { key: 'review_count', title: t('Reviews'), type: 'int' },
    { key: 'new_reviews', title: t('New this period'), value: p => (p.new_reviews === null ? '—' : int(p.new_reviews)) },
  ]
  const lowColumns = [
    { key: 'at', title: t('Date'), type: 'datetime' },
    { key: 'client_name', title: t('Customer'), value: l => l.client_name || '—' },
    { key: 'client_phone', title: t('Phone') },
    { key: 'branch_name', title: t('Branch') },
    { key: 'invoice_code', title: t('Invoice') },
    { key: 'rating', title: t('Stars'), value: l => stars(l.rating) },
    { key: 'replies', title: t('They wrote') },
  ]

  defineExpose({
    csv: () => [
      has('ratings') && csvSection(t('Ratings by branch'), branchColumns, ratingBranches.value),
      has('reviews') && csvSection(t('Review requests'), funnelColumns, funnelRows.value),
      has('unhappy') && csvSection(t('Unhappy customers'), lowColumns, low.value),
      has('replies') && csvSection(t('What customers wrote back'), [
        { key: 'received_at', title: t('Date'), type: 'datetime' },
        { key: 'client_name', title: t('Customer') },
        { key: 'client_phone', title: t('Phone') },
        { key: 'branch_name', title: t('Branch') },
        { key: 'rating', title: t('Stars'), value: x => nv(x.poll_rating) ?? '' },
        { key: 'body', title: t('Message') },
      ], replies.value),
    ].filter(Boolean),
  })
</script>
