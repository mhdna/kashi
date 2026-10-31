<template>
  <!-- Fills the space a report, chart or list will take while it's being
       prepared: a small animated picture of what's coming and a line saying
       so. CSS only, and only transform/opacity move, so it costs next to
       nothing; it holds still for people who turn motion off. -->
  <div class="preparing" role="status" aria-live="polite" :style="{ minHeight }">
    <div aria-hidden="true" class="preparing__art">
      <div v-if="kind === 'pdf'" class="doc">
        <span class="doc__corner" />
        <span v-for="i in 4" :key="i" class="doc__line" :style="{ '--i': i }" />
        <span class="doc__badge">PDF</span>
      </div>
      <div v-else-if="kind === 'chart'" class="bars">
        <span v-for="(h, i) in [45, 75, 55, 95]" :key="i" class="bar" :style="{ '--i': i, height: `${h}%` }" />
      </div>
      <div v-else class="rows">
        <span v-for="i in 3" :key="i" class="row" :style="{ '--i': i }" />
      </div>
    </div>
    <div class="text-subtitle-1 font-weight-medium">{{ title }}</div>
    <div v-if="subtitle" class="text-caption text-medium-emphasis">{{ subtitle }}</div>
  </div>
</template>

<script setup>
  defineProps({
    // What's being prepared: a 'pdf' (a page writing itself), a 'chart'
    // (bars growing) or 'data' (rows filling in).
    kind: { type: String, default: 'data' },
    title: { type: String, default: 'Preparing…' },
    subtitle: { type: String, default: '' },
    minHeight: { type: String, default: '320px' },
  })
</script>

<style scoped>
.preparing {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 24px;
  text-align: center;
}
.preparing__art {
  display: flex;
  align-items: flex-end;
  justify-content: center;
  height: 84px;
  margin-bottom: 10px;
}

/* A page writing itself, then stamped PDF. */
.doc {
  position: relative;
  width: 58px;
  height: 74px;
  border: 2px solid rgba(var(--v-theme-primary), 0.7);
  border-radius: 6px;
  background: rgb(var(--v-theme-surface));
  animation: preparing-float 2.8s ease-in-out infinite;
}
.doc__corner {
  position: absolute;
  top: -2px;
  right: -2px;
  width: 16px;
  height: 16px;
  border-bottom-left-radius: 4px;
  background: linear-gradient(225deg, rgb(var(--v-theme-background)) 50%, rgba(var(--v-theme-primary), 0.35) 50%);
}
.doc__line {
  position: absolute;
  left: 9px;
  right: 9px;
  top: calc(10px + var(--i) * 8px);
  height: 4px;
  border-radius: 2px;
  background: rgba(var(--v-theme-primary), 0.4);
  transform-origin: left center;
  animation: preparing-write 2.8s ease-out infinite;
  animation-delay: calc(var(--i) * 0.22s);
}
.doc__line:nth-of-type(5) { right: 22px; }
.doc__badge {
  position: absolute;
  bottom: 7px;
  left: 50%;
  padding: 1px 6px;
  border-radius: 3px;
  background: rgb(var(--v-theme-primary));
  color: rgb(var(--v-theme-on-primary));
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.06em;
  line-height: 14px;
  transform: translateX(-50%);
  animation: preparing-stamp 2.8s ease-out infinite;
}

/* Bars growing, one after another. */
.bars {
  display: flex;
  align-items: flex-end;
  gap: 7px;
  height: 64px;
  padding: 0 4px;
  border-bottom: 2px solid rgba(var(--v-theme-on-surface), 0.25);
}
.bar {
  width: 11px;
  border-radius: 3px 3px 0 0;
  background: rgba(var(--v-theme-primary), 0.75);
  transform-origin: bottom center;
  animation: preparing-grow 1.4s ease-in-out infinite alternate;
  animation-delay: calc(var(--i) * 0.16s);
}

/* Rows filling in. */
.rows {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 72px;
  padding-bottom: 6px;
}
.row {
  height: 9px;
  border-radius: 5px;
  background: rgba(var(--v-theme-primary), 0.55);
  transform-origin: left center;
  animation: preparing-write 2s ease-out infinite;
  animation-delay: calc(var(--i) * 0.2s);
}
.row:nth-child(2) { width: 80%; }
.row:nth-child(3) { width: 55%; }

@keyframes preparing-float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-5px); }
}
@keyframes preparing-write {
  0% { opacity: 0; transform: scaleX(0); }
  30% { opacity: 1; transform: scaleX(1); }
  80% { opacity: 1; transform: scaleX(1); }
  100% { opacity: 0; transform: scaleX(1); }
}
@keyframes preparing-stamp {
  0%, 50% { opacity: 0; transform: translateX(-50%) scale(1.6); }
  62% { opacity: 1; transform: translateX(-50%) scale(1); }
  88% { opacity: 1; transform: translateX(-50%) scale(1); }
  100% { opacity: 0; transform: translateX(-50%) scale(1); }
}
@keyframes preparing-grow {
  from { transform: scaleY(0.2); }
  to { transform: scaleY(1); }
}

@media (prefers-reduced-motion: reduce) {
  .preparing * { animation: none !important; }
}
</style>
