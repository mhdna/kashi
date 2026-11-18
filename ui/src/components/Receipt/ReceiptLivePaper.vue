<!--
  The receipt as the till prints it, drawn by kashi with the till's own code
  (/receipt_designer/preview), and every change made on it:
  - click a line's words: type over them in place, with a toolbar right under
    the line for its capitals, size, weight, alignment and printing;
  - click elsewhere in a part: a toolbar under the part for all its lines at
    once, its lines and box, its place and printing.
  Enter keeps typed words, Esc doesn't; Esc or the cross closes a toolbar.
-->
<template>
  <div class="rlive" :style="{ width: `${width}px` }" @keydown.esc="close">
    <div v-if="image" class="rlive-paper">
      <img
        alt="The receipt as it prints"
        class="rlive-img"
        data-test="receipt-image"
        :src="image"
      >
      <button
        v-for="m in marks"
        :key="m.group"
        :aria-label="`Change ${groupLabel(m.group)}`"
        class="rlive-part"
        :class="{ 'rlive-part--picked': m.group === picked }"
        :data-test="`pick-${m.group}`"
        :disabled="!editable"
        :style="bandStyle(m)"
        type="button"
        @click="pickPart(m.group)"
      >
        <!-- Outside the paper, so it never covers the receipt's words. -->
        <span class="rlive-label">{{ groupLabel(m.group) }}</span>
      </button>
      <template v-if="editable">
        <button
          v-for="(l, i) in lines"
          :key="`${l.key}-${i}`"
          :aria-label="`Change ${l.text}`"
          class="rlive-line"
          :class="{ 'rlive-line--picked': line && line.key === l.key && line.top === l.top }"
          :data-test="`words-${l.key}`"
          :style="lineStyle(l)"
          :title="`Change: ${l.text}`"
          type="button"
          @click="pickLine(l)"
        />
      </template>

      <div v-if="line && typing" class="rlive-edit" :style="lineStyle(line)">
        <textarea
          v-if="multiline(line.key)"
          ref="editor"
          v-model="draft"
          class="rlive-input"
          data-test="edit-words"
          :maxlength="maxLength"
          rows="2"
          :style="inputStyle(line)"
          @blur="commit"
          @keydown.enter.exact.prevent="commit"
          @keydown.esc.prevent.stop="cancelTyping"
        />
        <input
          v-else
          ref="editor"
          v-model="draft"
          class="rlive-input"
          data-test="edit-words"
          :maxlength="maxLength"
          :style="inputStyle(line)"
          @blur="commit"
          @keydown.enter.prevent="commit"
          @keydown.esc.prevent.stop="cancelTyping"
        >
      </div>

      <!-- The toolbar opens right under what was clicked. -->
      <div v-if="line && lineInfo" class="rlive-tools" :style="toolsStyle(line)">
        <ReceiptLineTools
          :catalog="catalog"
          :design="design"
          :info="lineInfo"
          @close="close"
          @update:design="d => emit('update:design', d)"
        />
      </div>
      <div v-else-if="partBand" class="rlive-tools" :style="toolsStyle(partBand)">
        <ReceiptPartTools
          :catalog="catalog"
          :design="design"
          :group="picked"
          @close="close"
          @update:design="d => emit('update:design', d)"
        />
      </div>
    </div>
    <div v-else class="rlive-empty">Drawing the receipt</div>
    <v-progress-linear v-if="loading" class="rlive-loading" color="primary" indeterminate />
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, ref, watch } from 'vue'
  import ReceiptLineTools from '@/components/Receipt/ReceiptLineTools.vue'
  import ReceiptPartTools from '@/components/Receipt/ReceiptPartTools.vue'
  import { can, setElement } from '@/utils/receiptDesign'

  type Band = { top: number, bottom: number }
  type Line = Band & { key: string, group: string, text: string, size_px: number, align: string, left?: number, right?: number }

  const props = withDefaults(defineProps<{
    image?: string
    height?: number
    // The picture's width in its own pixels.
    imageWidth?: number
    marks?: Array<Band & { group: string }>
    lines?: Line[]
    catalog?: Record<string, any> | null
    design?: Record<string, any>
    // The part picked (its toolbar shows unless a line of it is picked).
    picked?: string
    loading?: boolean
    editable?: boolean
    width?: number
    maxLength?: number
  }>(), {
    image: '', height: 0, imageWidth: 576, marks: () => [], lines: () => [], catalog: null,
    design: () => ({ elements: {} }), picked: '', loading: false, editable: true, width: 420, maxLength: 300,
  })
  const emit = defineEmits<{ 'pick': [group: string], 'update:design': [design: Record<string, any>] }>()

  const line = ref<Line | null>(null)
  const typing = ref(false)
  const draft = ref('')
  const editor = ref<HTMLInputElement | HTMLTextAreaElement | null>(null)

  const groups = computed(() => props.catalog?.groups ?? [])
  function groupLabel (key: string) {
    return groups.value.find((g: any) => g.key === key)?.label ?? key
  }
  function elementInfo (key: string) {
    return (props.catalog?.elements ?? []).find((e: any) => e.key === key) ?? null
  }
  const lineInfo = computed(() => (line.value ? elementInfo(line.value.key) : null))
  const partBand = computed(() => (props.editable && props.picked ? props.marks.find(m => m.group === props.picked) ?? null : null))

  // A redrawn receipt moves its lines: the picked one follows by its key.
  watch(() => props.lines, next => {
    if (!line.value) {return}
    const same = next.find(l => l.key === line.value!.key)
    if (same) {line.value = same} else {close()}
  })

  // Parts and lines sit where the till drew them, as shares of the picture.
  function bandStyle (m: Band) {
    const h = props.height || 1
    return { top: `${(m.top / h) * 100}%`, height: `${((m.bottom - m.top) / h) * 100}%` }
  }
  function lineStyle (l: Line) {
    const style: Record<string, string> = bandStyle(l)
    if (l.left && l.right) {
      const w = props.imageWidth || 576
      style.left = `${(l.left / w) * 100 - 0.5}%`
      style.width = `${((l.right - l.left) / w) * 100 + 1}%`
      style.right = 'auto'
    }
    return style
  }
  function toolsStyle (b: Band) {
    const h = props.height || 1
    return { top: `calc(${(b.bottom / h) * 100}% + 4px)` }
  }
  function inputStyle (l: Line) {
    const scale = props.width / (props.imageWidth || 576)
    return { fontSize: `${Math.max(10, l.size_px * scale)}px`, textAlign: (l.align || 'center') as 'left' | 'center' | 'right' }
  }
  function multiline (key: string) {
    return key === 'policy_text' || key === 'referral_note'
  }

  // The words a line starts from when typed over: the design's own, else the
  // default with its placeholders ({percent}), else what it prints.
  function wordsOf (l: Line) {
    const own = props.design.elements?.[l.key]?.text
    if (own) {return own}
    const info = elementInfo(l.key)
    if (info?.default && /\{(percent|currency)\}/.test(info.default)) {return info.default}
    return l.text
  }

  function pickPart (group: string) {
    line.value = null
    typing.value = false
    emit('pick', group)
  }

  async function pickLine (l: Line) {
    emit('pick', l.group)
    line.value = l
    const info = elementInfo(l.key)
    typing.value = Boolean(info && props.catalog && can(props.catalog, info, 'text'))
    if (!typing.value) {return}
    draft.value = wordsOf(l)
    await nextTick()
    editor.value?.focus()
    editor.value?.select()
  }

  function commit () {
    const l = line.value
    if (!l || !typing.value) {return}
    typing.value = false
    const text = draft.value.trim()
    if (text === wordsOf(l).trim()) {return}
    // The default typed back is no change at all.
    const value = text === elementInfo(l.key)?.default ? '' : text
    emit('update:design', setElement(props.design, l.key, { text: value }))
  }

  function cancelTyping () {
    typing.value = false
  }

  function close () {
    line.value = null
    typing.value = false
    emit('pick', '')
  }
</script>

<style scoped>
.rlive {
  position: relative;
  max-width: 100%;
}
/* The receipt is white paper in either app theme. */
.rlive-paper {
  position: relative;
  background: #fff;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.25);
}
.rlive-img {
  display: block;
  width: 100%;
  height: auto;
}
.rlive-part,
.rlive-line {
  position: absolute;
  left: 0;
  right: 0;
  padding: 0;
  border: 0;
  background: transparent;
  cursor: pointer;
}
.rlive-part:disabled {
  cursor: default;
}
.rlive-part:not(:disabled):hover,
.rlive-part--picked {
  outline: 2px solid rgb(var(--v-theme-primary));
  outline-offset: -2px;
}
/* A line's words, inset from the part's edges so the part stays clickable
   around them; a column title only over its own words. */
.rlive-line {
  left: 6%;
  right: 6%;
  cursor: text;
}
.rlive-line:hover,
.rlive-line--picked {
  background: rgba(var(--v-theme-primary), 0.14);
}
.rlive-label {
  position: absolute;
  top: 0;
  left: calc(100% + 8px);
  display: none;
  padding: 0 4px;
  font-size: 11px;
  white-space: nowrap;
  color: rgb(var(--v-theme-primary));
}
.rlive-part:not(:disabled):hover .rlive-label,
.rlive-part--picked .rlive-label {
  display: block;
}
.rlive-edit {
  position: absolute;
  left: 0;
  right: 0;
  min-height: 28px;
  z-index: 2;
}
.rlive-input {
  display: block;
  width: 100%;
  height: 100%;
  min-height: 28px;
  padding: 0 6%;
  border: 0;
  outline: 2px solid rgb(var(--v-theme-primary));
  resize: none;
  background: #fffbe6;
  color: #000;
  font-family: 'Cascadia Mono', 'SFMono-Regular', Menlo, Consolas, monospace;
  font-weight: 700;
}
.rlive-tools {
  position: absolute;
  left: -8px;
  right: -8px;
  z-index: 3;
}
.rlive-empty {
  padding: 48px 16px;
  text-align: center;
  color: rgba(var(--v-theme-on-surface), 0.6);
}
.rlive-loading {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
}
</style>
