<template>
  <p v-if="ctx.reviewsError" class="rpt-empty">Feedback couldn't be loaded: {{ ctx.reviewsError }}</p>
  <template v-else-if="r">
    <p v-if="ctx.branchIds.length > 1" class="rpt-note">
      Feedback can be shown for one branch or for all of them; with several branches picked this covers every branch.
    </p>
    <RptKpis :items="kpis" />

    <p v-if="!totals.polls && !funnel.with_link && places.length === 0" class="rpt-empty">
      No feedback was collected in this period: no rating polls or review links went out, and no Google place is set up.
      Turn them on under WhatsApp &amp; Reviews.
    </p>
    <RptSection v-if="totals.answers > 0" note="Answers to the rating poll sent on WhatsApp after a sale." title="How customers rated us">
      <RptGrid>
        <RptCard title="Answers by rating">
          <RptChart :height="200" :option="starsChart" />
        </RptCard>
        <RptCard title="Average rating by day">
          <RptChart :height="200" :option="averageChart" />
        </RptCard>
      </RptGrid>
      <RptTable v-if="ratingBranches.length > 1" :columns="branchColumns" compact :rows="ratingBranches" />
    </RptSection>

    <RptSection v-if="funnel.with_link > 0" note="Post-sale WhatsApp messages carrying the Google review link." title="Review requests">
      <RptGrid>
        <RptChart :height="180" :option="funnelChart" />
        <RptTable :columns="funnelColumns" compact :rows="funnelRows" />
      </RptGrid>
    </RptSection>

    <RptSection v-if="places.length > 0" title="Google">
      <RptTable :columns="placeColumns" compact :rows="places" />
      <template v-if="googleReviews.length > 0">
        <h3 class="rpt-h3">Latest Google reviews</h3>
        <blockquote v-for="g in googleReviews" :key="g.review_name" class="rpt-quote">
          <div>{{ stars(g.rating) }} {{ g.body || '(no text)' }}</div>
          <div class="rpt-quote__meta">{{ g.author }} · {{ dateTime(nv(g.published_at)) }}</div>
        </blockquote>
      </template>
    </RptSection>

    <RptSection v-if="low.length > 0" break-before note="Customers who gave 1 or 2 stars -- worth a call." title="Unhappy customers">
      <RptTable :columns="lowColumns" compact :rows="low" />
    </RptSection>

    <RptSection v-if="replies.length > 0" title="What customers wrote back">
      <blockquote v-for="reply in replies.slice(0, 40)" :key="reply.id" class="rpt-quote">
        <div>{{ reply.body || `(${reply.kind})` }}</div>
        <div class="rpt-quote__meta">
          {{ reply.client_name || reply.client_phone }}<template v-if="nv(reply.poll_rating)"> · rated {{ stars(nv(reply.poll_rating)) }}</template>
          · {{ reply.branch_name }} · {{ dateTime(reply.received_at) }}
        </div>
      </blockquote>
      <p v-if="replies.length > 40" class="rpt-note">And {{ replies.length - 40 }} more; the CSV export has them all.</p>
    </RptSection>
  </template>
</template>

<script setup>
  import { computed } from 'vue'
  import { barOption, csvSection, dateTime, dayLabel, int, lineOption, num1, nv, pct, ratio } from './reportKit'
  import RptCard from './RptCard.vue'
  import RptChart from './RptChart.vue'
  import RptGrid from './RptGrid.vue'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // What customers told us: the WhatsApp rating poll, their replies, the
  // review-link funnel and the Google ratings (all from /reviews).
  const props = defineProps({ ctx: { type: Object, required: true } })
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
    const t = totals.value
    const items = [
      { label: 'Ratings asked', value: int(t.polls) },
      { label: 'Answered', value: int(t.answers), hint: pct(ratio(t.answers, t.polls)) },
      { label: 'Average rating', value: t.answers ? `${num1(t.average)} / 5` : '—' },
      { label: 'Loved it (4–5★)', value: pct(ratio(happy.value, t.answers)), hint: `${int(happy.value)} customers` },
      { label: 'Unhappy (1–2★)', value: pct(ratio(unhappy.value, t.answers)), hint: `${int(unhappy.value)} customers` },
      { label: 'Replies received', value: int(replies.value.length) },
    ]
    if (funnel.value.with_link > 0) {
      items.push({ label: 'Review links opened', value: int(funnel.value.clicked), hint: `${pct(ratio(funnel.value.clicked, funnel.value.with_link || funnel.value.sent))} of links sent` })
    }
    for (const p of places.value.slice(0, 2)) {
      items.push({ label: `Google · ${p.name || 'rating'}`, value: `${num1(p.rating)} ★`, hint: p.new_reviews === null ? `${int(p.review_count)} reviews` : `${int(p.review_count)} reviews, +${int(p.new_reviews)} this period` })
    }
    return items
  })

  const starsChart = computed(() => {
    const t = totals.value
    return barOption({
      categories: ['5★', '4★', '3★', '2★', '1★'],
      series: [{ name: 'Answers', data: [t.stars5, t.stars4, t.stars3, t.stars2, t.stars1].map(v => v ?? 0) }],
      labels: true,
    })
  })
  const averageChart = computed(() => {
    const days = r.value.ratings?.days ?? []
    return lineOption({ categories: days.map(d => dayLabel(d.day, { day: 'numeric', month: 'short' })), series: [{ name: 'Average stars', data: days.map(d => Math.round(d.average * 10) / 10) }] })
  })

  const branchColumns = [
    { key: 'branch_name', title: 'Branch' },
    { key: 'polls', title: 'Asked', type: 'int' },
    { key: 'answers', title: 'Answered', type: 'int' },
    { key: 'average', title: 'Average', type: 'num1' },
    { key: 'stars5', title: '5★', type: 'int' },
    { key: 'stars4', title: '4★', type: 'int' },
    { key: 'stars3', title: '3★', type: 'int' },
    { key: 'stars2', title: '2★', type: 'int' },
    { key: 'stars1', title: '1★', type: 'int' },
  ]
  const funnelRows = computed(() => {
    const f = funnel.value
    const base = f.with_link || f.sent
    return [
      { step: 'Sent with a review link', count: f.with_link ?? f.sent },
      { step: 'Delivered', count: f.delivered },
      { step: 'Read', count: f.read },
      { step: 'Opened the link', count: f.clicked },
    ].map(row => ({ ...row, rate: ratio(row.count, base) }))
  })
  const funnelColumns = [
    { key: 'step', title: 'Step' },
    { key: 'count', title: 'Messages', type: 'int' },
    { key: 'rate', title: 'Of sent', type: 'pct' },
  ]
  const funnelChart = computed(() => barOption({ categories: funnelRows.value.map(f => f.step), series: [{ name: 'Messages', data: funnelRows.value.map(f => f.count ?? 0) }], horizontal: true, labels: true }))
  const placeColumns = [
    { key: 'name', title: 'Place' },
    { key: 'branches', title: 'Branches' },
    { key: 'rating', title: 'Rating', type: 'num1' },
    { key: 'review_count', title: 'Reviews', type: 'int' },
    { key: 'new_reviews', title: 'New this period', value: p => (p.new_reviews === null ? '—' : int(p.new_reviews)) },
  ]
  const lowColumns = [
    { key: 'at', title: 'Date', type: 'datetime' },
    { key: 'client_name', title: 'Customer', value: l => l.client_name || '—' },
    { key: 'client_phone', title: 'Phone' },
    { key: 'branch_name', title: 'Branch' },
    { key: 'invoice_code', title: 'Invoice' },
    { key: 'rating', title: 'Stars', value: l => stars(l.rating) },
    { key: 'replies', title: 'They wrote' },
  ]

  defineExpose({
    csv: () => [
      csvSection('Ratings by branch', branchColumns, ratingBranches.value),
      csvSection('Review requests', funnelColumns, funnelRows.value),
      csvSection('Unhappy customers', lowColumns, low.value),
      csvSection('Replies', [
        { key: 'received_at', title: 'Date', type: 'datetime' },
        { key: 'client_name', title: 'Customer' },
        { key: 'client_phone', title: 'Phone' },
        { key: 'branch_name', title: 'Branch' },
        { key: 'rating', title: 'Stars', value: x => nv(x.poll_rating) ?? '' },
        { key: 'body', title: 'Message' },
      ], replies.value),
    ],
  })
</script>
