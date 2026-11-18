<!--
  The receipt as the till prints it, drawn by kashi with the till's own code
  (/receipt_designer/preview). Every part is clickable (it opens in the panel
  beside it) and every line whose words can change is too: click its words and
  type over them right on the paper; Enter keeps them, Esc doesn't.
-->
<template>
  <div class="rlive" :style="{ width: `${width}px` }">
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
        @click="emit('pick', m.group)"
      >
        <!-- Outside the paper, so it never covers the receipt's words. -->
        <span class="rlive-label">{{ groupLabel(m.group) }}</span>
      </button>
      <template v-if="editable">
        <button
          v-for="(l, i) in lines"
          :key="`${l.key}-${i}`"
          :aria-label="`Type over ${l.text}`"
          class="rlive-line"
          :data-test="`words-${l.key}`"
          :style="bandStyle(l)"
          :title="`Click to change: ${l.text}`"
          type="button"
          @click="startEdit(l)"
        />
      </template>
      <div
        v-if="editing"
        class="rlive-edit"
        :style="bandStyle(editing)"
      >
        <textarea
          v-if="multiline(editing.key)"
          ref="editor"
          v-model="draft"
          class="rlive-input"
          data-test="edit-words"
          :maxlength="maxLength"
          rows="2"
          :style="inputStyle(editing)"
          @blur="commit"
          @keydown.enter.exact.prevent="commit"
          @keydown.esc.prevent="cancel"
        />
        <input
          v-else
          ref="editor"
          v-model="draft"
          class="rlive-input"
          data-test="edit-words"
          :maxlength="maxLength"
          :style="inputStyle(editing)"
          @blur="commit"
          @keydown.enter.prevent="commit"
          @keydown.esc.prevent="cancel"
        >
      </div>
    </div>
    <div v-else class="rlive-empty">Drawing the receipt</div>
    <v-progress-linear v-if="loading" class="rlive-loading" color="primary" indeterminate />
  </div>
</template>

<script setup lang="ts">
  import { nextTick, ref } from 'vue'

  type Band = { top: number, bottom: number }
  type Line = Band & { key: string, group: string, text: string, size_px: number, align: string }

  const props = withDefaults(defineProps<{
    image?: string
    height?: number
    // The picture's width in its own pixels, to size typed words like it.
    imageWidth?: number
    marks?: Array<Band & { group: string }>
    lines?: Line[]
    groups?: Array<{ key: string, label: string }>
    picked?: string
    loading?: boolean
    editable?: boolean
    width?: number
    maxLength?: number
    // The words a line starts from when typed over: the design's own, else
    // what it prints.
    wordsOf?: (key: string, printed: string) => string
  }>(), {
    image: '', height: 0, imageWidth: 576, marks: () => [], lines: () => [], groups: () => [],
    picked: '', loading: false, editable: true, width: 360, maxLength: 300,
    wordsOf: (_key: string, printed: string) => printed,
  })
  const emit = defineEmits<{ pick: [group: string], words: [key: string, text: string] }>()

  function groupLabel (key: string) {
    return props.groups.find(g => g.key === key)?.label ?? key
  }

  // Parts and lines sit where the till drew them, as a share of the
  // picture's height.
  function bandStyle (m: Band) {
    const h = props.height || 1
    return { top: `${(m.top / h) * 100}%`, height: `${((m.bottom - m.top) / h) * 100}%` }
  }

  function inputStyle (l: Line) {
    const scale = props.width / (props.imageWidth || 576)
    return { fontSize: `${Math.max(10, l.size_px * scale)}px`, textAlign: (l.align || 'center') as 'left' | 'center' | 'right' }
  }

  function multiline (key: string) {
    return key === 'policy_text' || key === 'referral_note'
  }

  const editing = ref<Line | null>(null)
  const draft = ref('')
  const editor = ref<HTMLInputElement | HTMLTextAreaElement | null>(null)

  async function startEdit (line: Line) {
    emit('pick', line.group)
    editing.value = line
    draft.value = props.wordsOf(line.key, line.text)
    await nextTick()
    editor.value?.focus()
    editor.value?.select()
  }

  function commit () {
    const line = editing.value
    if (!line) {return}
    editing.value = null
    const text = draft.value.trim()
    if (text !== props.wordsOf(line.key, line.text).trim()) {emit('words', line.key, text)}
  }

  function cancel () {
    editing.value = null
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
   around them. */
.rlive-line {
  left: 6%;
  right: 6%;
  cursor: text;
}
.rlive-line:hover {
  background: rgba(var(--v-theme-primary), 0.12);
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
