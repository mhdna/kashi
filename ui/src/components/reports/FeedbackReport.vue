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
        <RptCard :title="t('Answers')">
          <RptTable :columns="answerColumns" compact :rows="answerRows" />
        </RptCard>
        <RptCard :title="t('Answers by day')">
          <RptChart :height="200" :option="dailyChart" />
        </RptCard>
      </RptGrid>
      <RptTable v-if="ratingBranches.length > 1" :columns="branchColumns" compact :rows="ratingBranches" />
    </RptSection>

    <template v-if="has('reviews')">
      <RptSection v-if="funnel.with_link > 0" :note="t('Post-sale WhatsApp messages carrying the Google review link.')" :title="t('Review requests')">
        <RptTable :columns="funnelColumns" compact :rows="funnelRows" />
      </RptSection>

      <RptSection v-if="places.length > 0" :title="t('Google')">
        <RptTable :columns="placeColumns" compact :rows="places" />
        <template v-if="googleReviews.length > 0">
          <h3 class="rpt-h3">{{ t('Latest Google reviews') }}</h3>
          <blockquote v-for="g in googleReviews" :key="g.review_name" class="rpt-quote">
            <div>{{ '★'.repeat(g.rating || 0) }} {{ g.body || t('(no text)') }}</div>
            <div class="rpt-quote__meta">{{ g.author }} · {{ dateTime(nv(g.published_at)) }}</div>
          </blockquote>
        </template>
      </RptSection>
    </template>

    <RptSection v-if="has('unhappy') && low.length > 0" :note="t('Customers who gave the lowest answers -- worth a call.')" :title="t('Unhappy customers')">
      <RptTable :columns="lowColumns" compact :rows="low" />
    </RptSection>

    <RptSection
      v-if="has('replies') && replies.length > 0"
      :note="t('Only answers to the follow-up question, within a day of it.')"
      :title="t('What customers wrote back')"
    >
      <blockquote v-for="reply in replies.slice(0, 40)" :key="reply.id" class="rpt-quote">
        <div>{{ reply.body || `(${reply.kind})` }}</div>
        <div class="rpt-quote__meta">
          {{ reply.client_name || reply.client_phone }}<template v-if="answerOf(reply)"> · {{ answerOf(reply) }}</template>
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
  import { barOption, chartDay, csvSection, dateTime, int, nv, pct, ratio, SERIES } from './reportKit'
  import RptCard from './RptCard.vue'
  import RptChart from './RptChart.vue'
  import RptGrid from './RptGrid.vue'
  import RptKpis from './RptKpis.vue'
  import RptSection from './RptSection.vue'
  import RptTable from './RptTable.vue'

  // What customers told us: the WhatsApp rating poll (in its own answers --
  // "Loved it", "It was okay", "Not great" -- not stars), their replies to
  // the follow-up question, the review-link funnel and the Google ratings,
  // all from /reviews. Parts: ratings, reviews, unhappy, replies.
  const props = defineProps({
    ctx: { type: Object, required: true },
    parts: { type: Array, required: true },
  })
  const has = part => props.parts.includes(part)
  const r = computed(() => props.ctx.reviews)

  const totals = computed(() => r.value.ratings?.totals ?? {})
  const funnel = computed(() => r.value.funnel ?? {})
  const places = computed(() => (r.value.places ?? []).filter(p => p.checked).map(p => ({ ...p, branches: p.branches.join(', ') })))
  const googleReviews = computed(() => (r.value.reviews ?? []).slice(0, 8))
  const replies = computed(() => r.value.ratings?.replies ?? [])
  const ratingBranches = computed(() => r.value.ratings?.branches ?? [])

  // The poll's answers as configured, best first. Each counts as a number of
  // stars underneath (5 loved it, 3 okay, 1 not great by default); the label
  // is what the customer pressed. A leading emoji is kept, the words are
  // translated when they are the usual ones.
  function translateAnswer (label) {
    const match = /^(\P{L}*)(.*)$/u.exec(label ?? '')
    return `${match[1]}${t(match[2].trim())}`.trim()
  }
  const options = computed(() => {
    const list = (r.value.ratings?.options ?? []).filter(o => o.stars > 0)
    const fallback = [{ label: 'Loved it', stars: 5 }, { label: 'It was okay', stars: 3 }, { label: 'Not great', stars: 1 }]
    return (list.length > 0 ? list : fallback)
      .map(o => ({ stars: o.stars, label: translateAnswer(o.label) }))
      .toSorted((a, b) => b.stars - a.stars)
  })
  const labelFor = stars => options.value.find(o => o.stars === stars)?.label ?? `${stars}★`
  // What a reply's or a low answer's customer pressed.
  const answerOf = row => (nv(row.poll_rating) ? labelFor(nv(row.poll_rating)) : '')
  const count = (row, stars) => row?.[`stars${stars}`] ?? 0

  const answerRows = computed(() => options.value.map(o => ({ answer: o.label, count: count(totals.value, o.stars), share: ratio(count(totals.value, o.stars), totals.value.answers) })))
  const low = computed(() => (r.value.ratings?.low ?? []).map(l => ({ ...l, answer: answerOf(l), at: nv(l.poll_voted_at) })))

  const kpis = computed(() => {
    const tt = totals.value
    const items = [
      { label: t('Ratings asked'), value: int(tt.polls) },
      { label: t('Answered'), value: int(tt.answers), hint: pct(ratio(tt.answers, tt.polls)) },
      // One tile per answer: what share of customers pressed it.
      ...answerRows.value.map(a => ({ label: a.answer, value: pct(a.share), hint: t('{n} customers', { n: int(a.count) }) })),
      { label: t('Replies received'), value: int(replies.value.length) },
    ]
    if (funnel.value.with_link > 0) {
      items.push({ label: t('Review links opened'), value: int(funnel.value.clicked), hint: t('{pct} of links sent', { pct: pct(ratio(funnel.value.clicked, funnel.value.with_link)) }) })
    }
    for (const p of places.value.slice(0, 2)) {
      items.push({
        label: `${t('Google')} · ${p.name || t('rating')}`,
        value: `${p.rating.toFixed(1)} ★`,
        hint: p.new_reviews === null
          ? t('{n} reviews', { n: int(p.review_count) })
          : t('{n} reviews, +{new} this period', { n: int(p.review_count), new: int(p.new_reviews) }),
      })
    }
    return items
  })

  const answerColumns = [
    { key: 'answer', title: t('Answer') },
    { key: 'count', title: t('Customers'), type: 'int' },
    { key: 'share', title: t('Share'), type: 'pct' },
  ]
  // Each day's answers stacked, best answer first in the legend.
  const dailyChart = computed(() => {
    const days = r.value.ratings?.days ?? []
    return barOption({
      categories: days.map(d => chartDay(d.day)),
      series: options.value.map((o, i) => ({ name: o.label, color: [SERIES[2], SERIES[3], SERIES[7], SERIES[0], SERIES[6]][i], data: days.map(d => count(d, o.stars)) })),
      stack: true,
    })
  })

  const branchColumns = computed(() => [
    { key: 'branch_name', title: t('Branch') },
    { key: 'polls', title: t('Asked'), type: 'int' },
    { key: 'answers', title: t('Answered'), type: 'int' },
    ...options.value.map(o => ({ key: `stars${o.stars}`, title: o.label, type: 'int' })),
  ])
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
    { key: 'answer', title: t('Answer') },
    { key: 'replies', title: t('They wrote') },
  ]

  defineExpose({
    csv: () => [
      has('ratings') && csvSection(t('Answers'), answerColumns, answerRows.value),
      has('ratings') && csvSection(t('Ratings by branch'), branchColumns.value, ratingBranches.value),
      has('reviews') && csvSection(t('Review requests'), funnelColumns, funnelRows.value),
      has('unhappy') && csvSection(t('Unhappy customers'), lowColumns, low.value),
      has('replies') && csvSection(t('What customers wrote back'), [
        { key: 'received_at', title: t('Date'), type: 'datetime' },
        { key: 'client_name', title: t('Customer') },
        { key: 'client_phone', title: t('Phone') },
        { key: 'branch_name', title: t('Branch') },
        { key: 'answer', title: t('Answer'), value: x => answerOf(x) },
        { key: 'body', title: t('Message') },
      ], replies.value),
    ].filter(Boolean),
  })
</script>
