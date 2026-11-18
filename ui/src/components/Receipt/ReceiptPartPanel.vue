<!--
  The panel for one part of the receipt: every line in it, each with what the
  till lets it change (words, capitals, size, bold, alignment, printed or
  not), and the part itself left out or not.
-->
<template>
  <div class="rpp" data-no-discard-guard>
    <template v-if="info">
      <div class="d-flex align-center ga-2 mb-3">
        <div class="text-subtitle-1 font-weight-medium flex-grow-1" data-test="panel-title">{{ info.label }}</div>
        <v-btn
          v-if="changed"
          data-test="reset-part"
          density="comfortable"
          prepend-icon="mdi-restore"
          size="small"
          text="Back to default"
          variant="text"
          @click="update(resetGroup(design, catalog, group))"
        />
      </div>

      <v-switch
        v-if="info.section"
        class="mb-2"
        color="primary"
        data-test="part-printed"
        density="compact"
        hide-details
        inset
        label="Print this part"
        :model-value="!isSectionHidden(design, info.section)"
        @update:model-value="update(toggleSection(design, info.section))"
      />

      <v-select
        v-if="group === 'meta'"
        class="mb-4"
        data-test="date-format"
        density="compact"
        hide-details
        item-title="label"
        item-value="key"
        :items="catalog.date_formats"
        label="Date format"
        :model-value="design.date_format || ''"
        variant="outlined"
        @update:model-value="v => update({ ...design, date_format: v || '' })"
      />

      <div
        v-for="e in lines"
        :key="e.key"
        class="rpp-line"
        :class="{ 'rpp-line--off': isSectionHidden(design, info.section) }"
        :data-test="`line-${e.key}`"
      >
        <div class="d-flex align-center ga-2 mb-2">
          <div class="text-body-2 font-weight-medium flex-grow-1">{{ e.label }}</div>
          <v-switch
            v-if="can(catalog, e, 'hide')"
            color="primary"
            :data-test="`printed-${e.key}`"
            density="compact"
            hide-details
            inset
            :label="element(e.key).hide ? 'Left out' : 'Printed'"
            :model-value="!element(e.key).hide"
            @update:model-value="v => set(e.key, { hide: !v })"
          />
        </div>

        <template v-if="can(catalog, e, 'text')">
          <v-textarea
            v-if="long(e)"
            auto-grow
            :data-test="`text-${e.key}`"
            density="compact"
            :hint="hint(e)"
            :maxlength="catalog.max_length"
            :model-value="element(e.key).text ?? ''"
            :persistent-hint="Boolean(hint(e))"
            persistent-placeholder
            :placeholder="e.default || 'The branch\'s own, from Branch Settings'"
            rows="2"
            variant="outlined"
            @update:model-value="v => set(e.key, { text: v })"
          />
          <v-text-field
            v-else
            :data-test="`text-${e.key}`"
            density="compact"
            :hint="hint(e)"
            :maxlength="catalog.max_length"
            :model-value="element(e.key).text ?? ''"
            :persistent-hint="Boolean(hint(e))"
            persistent-placeholder
            :placeholder="e.default || 'The branch\'s own, from Branch Settings'"
            variant="outlined"
            @update:model-value="v => set(e.key, { text: v })"
          />
        </template>

        <div class="rpp-controls">
          <v-select
            v-if="can(catalog, e, 'case')"
            class="rpp-case"
            :data-test="`case-${e.key}`"
            density="compact"
            hide-details
            :items="caseOptions"
            label="Capitals"
            :model-value="element(e.key).case || ''"
            variant="outlined"
            @update:model-value="v => set(e.key, { case: v })"
          />

          <div v-if="can(catalog, e, 'size')" class="rpp-size">
            <span class="text-caption text-medium-emphasis me-1">Size</span>
            <v-btn
              :aria-label="`Smaller ${e.label}`"
              :data-test="`smaller-${e.key}`"
              density="comfortable"
              :disabled="(element(e.key).size ?? 0) <= catalog.size_min"
              icon="mdi-minus"
              size="x-small"
              variant="outlined"
              @click="update(sizeStep(design, catalog, e.key, -1))"
            />
            <span class="rpp-size-value" :data-test="`size-${e.key}`">{{ sizeText(element(e.key).size) }}</span>
            <v-btn
              :aria-label="`Bigger ${e.label}`"
              :data-test="`bigger-${e.key}`"
              density="comfortable"
              :disabled="(element(e.key).size ?? 0) >= catalog.size_max"
              icon="mdi-plus"
              size="x-small"
              variant="outlined"
              @click="update(sizeStep(design, catalog, e.key, 1))"
            />
          </div>

          <v-btn-toggle
            v-if="can(catalog, e, 'bold')"
            color="primary"
            :data-test="`bold-${e.key}`"
            density="compact"
            divided
            mandatory
            :model-value="boldState(design, e.key)"
            variant="outlined"
            @update:model-value="v => update(setBold(design, e.key, v))"
          >
            <v-btn size="small" value="look">Look's own</v-btn>
            <v-btn size="small" value="bold"><strong>Bold</strong></v-btn>
            <v-btn size="small" value="regular">Regular</v-btn>
          </v-btn-toggle>

          <v-btn-toggle
            v-if="can(catalog, e, 'align')"
            color="primary"
            :data-test="`align-${e.key}`"
            density="compact"
            divided
            mandatory
            :model-value="element(e.key).align || ''"
            variant="outlined"
            @update:model-value="v => set(e.key, { align: v })"
          >
            <v-btn v-for="a in alignOptions" :key="a.value" size="small" :value="a.value">{{ a.value ? a.title : 'Look\'s own' }}</v-btn>
          </v-btn-toggle>
        </div>
      </div>
    </template>
    <div v-else class="text-body-2 text-medium-emphasis" data-test="panel-empty">
      Click any part of the receipt to change its words, capitals, size, weight or alignment, or to leave it out.
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import {
    alignOptions, boldState, can, caseOptions, elementsOf, groupInfo, isGroupChanged,
    isSectionHidden, resetGroup, setBold, setElement, sizeStep, toggleSection,
  } from '@/utils/receiptDesign'

  const props = defineProps({
    catalog: { type: Object, required: true },
    design: { type: Object, required: true },
    group: { type: String, default: '' },
  })
  const emit = defineEmits(['update:design'])

  const info = computed(() => groupInfo(props.catalog, props.group))
  const lines = computed(() => elementsOf(props.catalog, props.group))
  const changed = computed(() => isGroupChanged(props.design, props.catalog, props.group))

  function element (key) {
    return props.design.elements[key] ?? {}
  }
  function update (design) {
    emit('update:design', design)
  }
  function set (key, patch) {
    update(setElement(props.design, key, patch))
  }
  function long (e) {
    return e.key === 'policy_text' || e.key === 'referral_note'
  }
  function hint (e) {
    if ((e.default ?? '').includes('{percent}')) {return '{percent} prints as the reward, like 15%.'}
    if ((e.default ?? '').includes('{currency}')) {return '{currency} prints as the currency, like LBP.'}
    return ''
  }
  function sizeText (size) {
    const n = Number(size ?? 0)
    return n > 0 ? `+${n}` : String(n)
  }
</script>

<style scoped>
.rpp-line {
  padding: 12px 0;
  border-top: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.rpp-line--off {
  opacity: 0.5;
}
.rpp-controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
}
.rpp-case {
  flex: 0 1 170px;
}
.rpp-size {
  display: flex;
  align-items: center;
  gap: 4px;
}
.rpp-size-value {
  min-width: 28px;
  text-align: center;
  font-variant-numeric: tabular-nums;
}
</style>
