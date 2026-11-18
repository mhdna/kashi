<!--
  The receipt as the till prints it, strip by strip from the till's own
  picture (utils/receiptTexts.js), with every part the branch can reword or
  leave out clickable in place: click a part to type over it, the eye to leave
  it out. A reworded part is drawn here in a close font; everything else is
  the till's picture.
-->
<template>
  <div class="rp" data-no-discard-guard :style="{ width: `${paperWidth}px` }">
    <template v-for="(strip, i) in strips" :key="`${strip.kind}-${strip.part ?? strip.section ?? ''}-${i}`">
      <div
        v-if="strip.kind === 'image'"
        class="rp-image"
        :style="sliceStyle(strip.top, strip.bottom)"
      />

      <div
        v-else-if="strip.kind === 'hidden'"
        class="rp-hidden"
        :data-test="`hidden-${strip.section}`"
      >
        <span>{{ sectionLabels[strip.section] }} left out</span>
        <v-btn
          v-if="editable"
          :aria-label="`Show ${sectionLabels[strip.section]}`"
          :data-test="`show-${strip.section}`"
          density="comfortable"
          prepend-icon="mdi-eye"
          size="small"
          text="Show"
          variant="text"
          @click="toggle(strip.section)"
        />
      </div>

      <div
        v-else
        class="rp-part"
        :class="{ 'rp-part--editable': editable, 'rp-part--editing': editing === strip.part }"
        :data-test="`part-${strip.part}`"
        :style="{ minHeight: `${(strip.bottom - strip.top) * scale}px` }"
      >
        <textarea
          v-if="editing === strip.part"
          ref="editor"
          v-model="draft"
          class="rp-editor"
          :data-test="`edit-${strip.part}`"
          :maxlength="catalogTexts.max_length"
          rows="1"
          :style="textStyle(strip)"
          @blur="commit"
          @keydown.enter.exact.prevent="commit"
          @keydown.esc.prevent="cancel"
        />
        <button
          v-else
          :aria-label="`Reword ${partLabel(strip.part)}`"
          class="rp-face"
          :disabled="!editable"
          type="button"
          @click="startEdit(strip)"
        >
          <div
            v-if="!strip.reworded"
            class="rp-image"
            :style="sliceStyle(strip.top, strip.bottom)"
          />
          <div v-else-if="strip.kind === 'socials'" class="rp-socials" :style="{ minHeight: `${(strip.bottom - strip.top) * scale}px` }">
            <span :style="textStyle(strip)">{{ strip.text }}</span>
            <span
              v-if="strip.icons"
              class="rp-icons"
              :style="iconsStyle(strip.icons)"
            />
          </div>
          <div v-else class="rp-text" :style="{ ...textStyle(strip), minHeight: `${(strip.bottom - strip.top) * scale}px` }">
            {{ strip.text }}
          </div>
        </button>

        <div v-if="editable && editing !== strip.part" class="rp-tools">
          <v-btn
            :aria-label="`Reword ${partLabel(strip.part)}`"
            density="comfortable"
            icon="mdi-pencil"
            size="x-small"
            variant="flat"
            @click="startEdit(strip)"
          />
          <v-btn
            v-if="strip.reworded"
            :aria-label="`Back to the default ${partLabel(strip.part)}`"
            :data-test="`reset-${strip.part}`"
            density="comfortable"
            icon="mdi-restore"
            size="x-small"
            variant="flat"
            @click="reset(strip.part)"
          />
          <v-btn
            :aria-label="`Leave out ${sectionLabels[strip.section]}`"
            :data-test="`hide-${strip.section}`"
            density="comfortable"
            icon="mdi-eye-off"
            size="x-small"
            variant="flat"
            @click="toggle(strip.section)"
          />
        </div>
        <div v-if="editing === strip.part && strip.part === 'referral_note'" class="rp-hint">
          {percent} prints as the reward, like {{ catalogTexts.referral_percent }}%.
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, ref } from 'vue'
  import { paperStrips, sectionLabels, textParts, wording } from '@/utils/receiptTexts'

  type Mark = { kind: string, part: string, top: number, bottom: number, left?: number, right?: number, size_px?: number, bold?: boolean, text?: string }
  type ModeInfo = { mode: string, file: string, width: number, height: number, parts: Mark[] }
  type Texts = Record<string, any>
  type CatalogTexts = { defaults: Record<string, string>, sections: string[], max_length: number, referral_percent: number, referral_code: string }
  type Strip = { kind: string, part?: string, section?: string, top: number, bottom: number, text?: string, sizePx?: number, bold?: boolean, reworded?: boolean, icons?: { top: number, bottom: number, left: number, right: number } }

  const props = withDefaults(defineProps<{
    modeInfo: ModeInfo
    texts: Texts
    catalogTexts: CatalogTexts
    editable?: boolean
    paperWidth?: number
    // Where the previews are served (public/receipt-previews).
    base?: string
  }>(), { editable: true, paperWidth: 360, base: '/receipt-previews/' })
  const emit = defineEmits<{ 'update:texts': [texts: Texts] }>()

  const strips = computed<Strip[]>(() => paperStrips(props.modeInfo, props.texts, props.catalogTexts))
  const scale = computed(() => props.paperWidth / (props.modeInfo.width || 576))
  const src = computed(() => `${props.base}${props.modeInfo.file}`)

  function sliceStyle (top: number, bottom: number) {
    const s = scale.value
    return {
      height: `${(bottom - top) * s}px`,
      backgroundImage: `url("${src.value}")`,
      backgroundSize: `${props.paperWidth}px auto`,
      backgroundPosition: `0 ${-top * s}px`,
    }
  }

  function iconsStyle (icons: { top: number, bottom: number, left: number, right: number }) {
    const s = scale.value
    return {
      width: `${(icons.right - icons.left) * s}px`,
      height: `${(icons.bottom - icons.top) * s}px`,
      backgroundImage: `url("${src.value}")`,
      backgroundSize: `${props.paperWidth}px auto`,
      backgroundPosition: `${-icons.left * s}px ${-icons.top * s}px`,
    }
  }

  function textStyle (strip: Strip) {
    return {
      fontSize: `${(strip.sizePx || 24) * scale.value}px`,
      fontWeight: strip.bold ? 700 : 400,
    }
  }

  function partLabel (part?: string) {
    return (textParts.find(p => p.part === part)?.label ?? 'this part').toLowerCase()
  }

  const editing = ref<string | null>(null)
  const draft = ref('')
  const editor = ref<HTMLTextAreaElement[] | null>(null)

  async function startEdit (strip: Strip) {
    if (!props.editable || !strip.part) return
    editing.value = strip.part
    draft.value = wording(props.texts, strip.part, props.catalogTexts.defaults)
    await nextTick()
    const el = editor.value?.[0]
    el?.focus()
    el?.select()
  }

  function commit () {
    const part = editing.value
    if (!part) return
    editing.value = null
    const text = draft.value.trim()
    // The default typed back is no rewording at all.
    const value = text === props.catalogTexts.defaults[part] ? '' : text
    if (value !== String(props.texts[part] ?? '').trim()) emit('update:texts', { ...props.texts, [part]: value })
  }

  function cancel () {
    editing.value = null
  }

  function reset (part?: string) {
    if (part) emit('update:texts', { ...props.texts, [part]: '' })
  }

  function toggle (section?: string) {
    if (!section) return
    const hide: string[] = props.texts.hide ?? []
    emit('update:texts', { ...props.texts, hide: hide.includes(section) ? hide.filter(h => h !== section) : [...hide, section] })
  }
</script>

<style scoped>
/* The receipt is white paper in either app theme. */
.rp {
  --rp-ink: #000;
  max-width: 100%;
  background: #fff;
  color: var(--rp-ink);
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.25);
  font-family: 'Cascadia Mono', 'SFMono-Regular', Menlo, Consolas, monospace;
}
.rp-image {
  width: 100%;
  background-repeat: no-repeat;
}
.rp-part {
  position: relative;
}
.rp-face {
  display: block;
  width: 100%;
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-align: center;
  cursor: default;
}
.rp-part--editable .rp-face {
  cursor: text;
}
.rp-part--editable:hover,
.rp-part--editing {
  outline: 2px solid rgb(var(--v-theme-primary));
  outline-offset: -2px;
}
.rp-text,
.rp-socials {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 7%;
  white-space: pre-wrap;
  line-height: 1.25;
}
.rp-socials {
  gap: 6px;
}
.rp-icons {
  flex: 0 0 auto;
  background-repeat: no-repeat;
}
.rp-tools {
  position: absolute;
  top: 50%;
  right: 4px;
  display: none;
  gap: 2px;
  transform: translateY(-50%);
}
.rp-part--editable:hover .rp-tools,
.rp-part--editable:focus-within .rp-tools {
  display: flex;
}
.rp-editor {
  display: block;
  width: 100%;
  min-height: 100%;
  padding: 4px 7%;
  border: 0;
  outline: none;
  resize: none;
  background: #fffbe6;
  color: var(--rp-ink);
  font: inherit;
  text-align: center;
  field-sizing: content;
}
.rp-hint {
  padding: 2px 8px 6px;
  font-family: Roboto, sans-serif;
  font-size: 12px;
  text-align: center;
  color: #555;
}
.rp-hidden {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 4px 8px;
  padding: 2px 4px 2px 10px;
  border: 1px dashed #999;
  border-radius: 4px;
  font-family: Roboto, sans-serif;
  font-size: 12px;
  color: #555;
}
</style>
