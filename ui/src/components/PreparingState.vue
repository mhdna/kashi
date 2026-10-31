<template>
  <!-- Fills the space a report, chart or list will take while it's being
       prepared: a small animated picture of what's coming and a line saying
       so. CSS only, and only transform/opacity move, so it costs next to
       nothing; it holds still for people who turn motion off. -->
  <div
    :aria-label="title"
    aria-live="polite"
    class="preparing"
    :class="{ 'preparing--small': size }"
    role="status"
    :style="size ? undefined : { minHeight }"
  >
    <div aria-hidden="true" class="preparing__art" :style="size ? { '--scale': size / 84, width: `${size}px`, height: `${size}px` } : undefined">
      <div class="preparing__scale">
        <div v-if="kind === 'pdf'" class="doc">
          <span v-for="i in 4" :key="i" class="doc__line" :style="{ '--i': i }" />
          <span class="doc__badge">PDF</span>
        </div>
        <div v-else-if="kind === 'chart'" class="bars">
          <span v-for="(h, i) in [45, 75, 55, 95]" :key="i" class="bar" :style="{ '--i': i, height: `${h}%` }" />
        </div>
        <div v-else-if="kind === 'people'" class="people">
          <span v-for="i in 3" :key="i" class="person" :style="{ '--i': i }" />
        </div>
        <div v-else-if="kind === 'chat'" class="chat">
          <span class="bubble bubble--in"><span v-for="i in 3" :key="i" class="dot" :style="{ '--i': i }" /></span>
          <span class="bubble bubble--out" />
        </div>
        <div v-else-if="kind === 'settings'" class="gears">
          <span class="gear gear--big" />
          <span class="gear gear--small" />
        </div>
        <div v-else-if="kind === 'box'" class="box">
          <span class="box__item" />
          <span class="box__body" />
        </div>
        <div v-else-if="kind === 'receipt'" class="printer">
          <span class="printer__paper"><span v-for="i in 3" :key="i" class="printer__line" /></span>
          <span class="printer__slot" />
        </div>
        <div v-else-if="kind === 'update'" class="update">
          <span class="update__ring" />
          <span class="update__cloud" />
          <span class="update__arrow" />
        </div>
        <div v-else class="rows">
          <span v-for="i in 3" :key="i" class="row" :style="{ '--i': i }" />
        </div>
      </div>
    </div>
    <template v-if="!size">
      <div class="text-subtitle-1 font-weight-medium">{{ title }}</div>
      <div v-if="subtitle" class="text-caption text-medium-emphasis">{{ subtitle }}</div>
    </template>
  </div>
</template>

<script setup>
  defineProps({
    // What's being prepared, which picks the picture: 'pdf' (a page writing
    // itself), 'chart' (bars growing), 'people' (heads bobbing), 'chat'
    // (bubbles and typing dots), 'settings' (gears turning), 'box' (an item
    // dropping into a box), 'receipt' (a receipt printing), 'update' (an
    // arrow rising through a cloud in a turning ring) or 'data' (rows
    // filling in).
    kind: { type: String, default: 'data' },
    title: { type: String, default: 'Preparing…' },
    subtitle: { type: String, default: '' },
    minHeight: { type: String, default: '320px' },
    // A size in px makes it a small inline icon with no text, in place of a
    // small spinner (a header, a button row, a list row).
    size: { type: Number, default: 0 },
  })
</script>

<style scoped>
/* The pictures are grey, drawn in the text colour at low strength; the
   update one is in the theme colour. */
.preparing {
  --prep: var(--v-theme-on-surface);
  --prep-a: 0.5;
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
.preparing__scale {
  display: flex;
  align-items: flex-end;
  justify-content: center;
}
.preparing--small {
  display: inline-flex;
  padding: 0;
  vertical-align: middle;
}
.preparing--small .preparing__art {
  position: relative;
  margin: 0;
}
.preparing--small .preparing__scale {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%) scale(var(--scale));
}

/* A page writing itself, then stamped PDF. */
.doc {
  position: relative;
  width: 58px;
  height: 74px;
  border: 2px solid rgba(var(--prep), calc(0.7 * var(--prep-a)));
  border-radius: 6px;
  background: rgb(var(--v-theme-surface));
  animation: preparing-float 2.8s ease-in-out infinite;
}
.doc__line {
  position: absolute;
  left: 9px;
  right: 9px;
  top: calc(10px + var(--i) * 8px);
  height: 4px;
  border-radius: 2px;
  background: rgba(var(--prep), calc(0.4 * var(--prep-a)));
  transform-origin: left center;
  animation: preparing-write 2.8s ease-out infinite;
  animation-delay: calc(var(--i) * 0.22s);
}
.doc__line:nth-of-type(4) { right: 22px; }
.doc__badge {
  position: absolute;
  bottom: 7px;
  left: 50%;
  padding: 1px 6px;
  border-radius: 3px;
  background: rgba(var(--prep), var(--prep-a));
  color: rgb(var(--v-theme-surface));
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
  background: rgba(var(--prep), calc(0.75 * var(--prep-a)));
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
  background: rgba(var(--prep), calc(0.55 * var(--prep-a)));
  transform-origin: left center;
  animation: preparing-write 2s ease-out infinite;
  animation-delay: calc(var(--i) * 0.2s);
}
.row:nth-child(2) { width: 80%; }
.row:nth-child(3) { width: 55%; }

/* Three people bobbing in turn. */
.people {
  display: flex;
  align-items: flex-end;
  gap: 8px;
}
.person {
  position: relative;
  width: 20px;
  height: 38px;
  animation: preparing-bob 1.5s ease-in-out infinite;
  animation-delay: calc(var(--i) * 0.18s);
}
.person::before,
.person::after {
  content: '';
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(var(--prep), calc(0.75 * var(--prep-a)));
}
.person::before {
  top: 0;
  width: 14px;
  height: 14px;
  border-radius: 50%;
}
.person::after {
  bottom: 0;
  width: 20px;
  height: 20px;
  border-radius: 10px 10px 3px 3px;
}
.person:nth-child(2) { transform: scale(1.15); }

/* A message coming in (typing dots), then the reply. */
.chat {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 76px;
}
.bubble {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  height: 22px;
  border-radius: 11px;
}
.bubble--in {
  align-self: flex-start;
  width: 46px;
  border-bottom-left-radius: 3px;
  background: rgba(var(--v-theme-on-surface), 0.14);
}
.bubble--out {
  align-self: flex-end;
  width: 52px;
  border-bottom-right-radius: 3px;
  background: rgba(var(--prep), calc(0.75 * var(--prep-a)));
  transform-origin: right center;
  animation: preparing-reply 2.4s ease-out infinite;
}
.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: rgba(var(--v-theme-on-surface), 0.6);
  animation: preparing-typing 1.2s ease-in-out infinite;
  animation-delay: calc(var(--i) * 0.15s);
}

/* Two gears turning against each other. */
.gears {
  position: relative;
  width: 76px;
  height: 64px;
}
.gear {
  position: absolute;
  border-style: dashed;
  border-color: rgba(var(--prep), calc(0.8 * var(--prep-a)));
  border-radius: 50%;
  box-shadow: inset 0 0 0 3px rgba(var(--prep), calc(0.35 * var(--prep-a)));
}
.gear--big {
  left: 0;
  bottom: 0;
  width: 42px;
  height: 42px;
  border-width: 6px;
  animation: preparing-spin 4s linear infinite;
}
.gear--small {
  right: 0;
  top: 0;
  width: 26px;
  height: 26px;
  border-width: 5px;
  animation: preparing-spin 2.6s linear infinite reverse;
}

/* An item dropping into a box. */
.box {
  position: relative;
  width: 64px;
  height: 70px;
}
.box__body {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 34px;
  border-radius: 4px;
  background: rgba(var(--prep), calc(0.75 * var(--prep-a)));
}
.box__item {
  position: absolute;
  left: 50%;
  bottom: 22px;
  width: 22px;
  height: 22px;
  margin-left: -11px;
  border-radius: 4px;
  background: rgba(var(--prep), calc(0.35 * var(--prep-a)));
  animation: preparing-drop 1.8s ease-in infinite;
}

/* A receipt printing out of a slot. */
.printer {
  position: relative;
  display: flex;
  justify-content: center;
  width: 70px;
  height: 76px;
  overflow: hidden;
}
.printer__slot {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 16px;
  border-radius: 5px;
  background: rgba(var(--v-theme-on-surface), 0.3);
}
.printer__paper {
  position: absolute;
  bottom: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 44px;
  height: 54px;
  padding: 8px 7px;
  border-radius: 3px 3px 0 0;
  background: rgb(var(--v-theme-surface));
  border: 2px solid rgba(var(--prep), calc(0.6 * var(--prep-a)));
  animation: preparing-print 2.4s ease-out infinite;
}
.printer__line {
  height: 3px;
  border-radius: 2px;
  background: rgba(var(--prep), calc(0.45 * var(--prep-a)));
}
.printer__line:nth-child(2) { width: 70%; }

/* The app updating: an arrow rising through a cloud, in a turning ring. */
.update {
  --prep: var(--v-theme-primary);
  --prep-a: 1;
  position: relative;
  width: 84px;
  height: 84px;
}
.update__ring {
  position: absolute;
  inset: 0;
  border: 3px solid rgba(var(--prep), calc(0.15 * var(--prep-a)));
  border-top-color: rgba(var(--prep), var(--prep-a));
  border-right-color: rgba(var(--prep), calc(0.55 * var(--prep-a)));
  border-radius: 50%;
  animation: preparing-spin 1.6s linear infinite;
}
.update__cloud {
  position: absolute;
  left: 18px;
  top: 44px;
  width: 48px;
  height: 18px;
  border-radius: 9px;
  background: rgba(var(--prep), calc(0.35 * var(--prep-a)));
}
.update__cloud::before,
.update__cloud::after {
  content: '';
  position: absolute;
  border-radius: 50%;
  background: inherit;
}
.update__cloud::before {
  left: 6px;
  top: -9px;
  width: 18px;
  height: 18px;
}
.update__cloud::after {
  left: 18px;
  top: -15px;
  width: 24px;
  height: 24px;
}
.update__arrow {
  position: absolute;
  left: 39px;
  top: 26px;
  width: 6px;
  height: 22px;
  border-radius: 3px;
  background: rgba(var(--prep), var(--prep-a));
  animation: preparing-rise 1.6s ease-in-out infinite;
}
.update__arrow::before {
  content: '';
  position: absolute;
  left: -6px;
  top: -9px;
  width: 18px;
  height: 12px;
  background: inherit;
  clip-path: polygon(50% 0, 100% 100%, 0 100%);
}

@keyframes preparing-rise {
  0% { opacity: 0; transform: translateY(10px); }
  30% { opacity: 1; }
  70% { opacity: 1; transform: translateY(-4px); }
  100% { opacity: 0; transform: translateY(-10px); }
}
@keyframes preparing-bob {
  0%, 100% { translate: 0 0; }
  50% { translate: 0 -6px; }
}
@keyframes preparing-typing {
  0%, 60%, 100% { opacity: 0.35; transform: translateY(0); }
  30% { opacity: 1; transform: translateY(-3px); }
}
@keyframes preparing-reply {
  0%, 45% { opacity: 0; transform: scale(0.6); }
  60% { opacity: 1; transform: scale(1); }
  90% { opacity: 1; transform: scale(1); }
  100% { opacity: 0; transform: scale(1); }
}
@keyframes preparing-spin {
  to { transform: rotate(360deg); }
}
@keyframes preparing-drop {
  0% { opacity: 0; transform: translateY(-40px); }
  20% { opacity: 1; }
  60% { opacity: 1; transform: translateY(0); }
  80%, 100% { opacity: 0; transform: translateY(14px); }
}
@keyframes preparing-print {
  0% { transform: translateY(60px); }
  70% { transform: translateY(0); }
  90% { transform: translateY(0); opacity: 1; }
  100% { transform: translateY(0); opacity: 0; }
}
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
