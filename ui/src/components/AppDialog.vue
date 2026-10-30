<template>
  <v-dialog
    class="app-question-dialog"
    max-width="460"
    :model-value="dialogState.open"
    @update:model-value="(v: boolean) => !v && cancel()"
  >
    <v-card>
      <v-card-title v-if="dialogState.title" class="text-wrap">{{ dialogState.title }}</v-card-title>
      <v-card-text>
        <p v-if="dialogState.text" :class="{ 'mb-4': dialogState.kind === 'prompt' }">{{ dialogState.text }}</p>
        <v-text-field
          v-if="dialogState.kind === 'prompt'"
          v-model="dialogState.value"
          autofocus
          density="comfortable"
          :hint="dialogState.hint"
          :label="dialogState.label"
          :persistent-hint="!!dialogState.hint"
          :prefix="dialogState.prefix"
          :type="dialogState.type"
          variant="outlined"
          @focus="(e: FocusEvent) => (e.target as HTMLInputElement).select()"
          @keydown.enter="ok"
        />
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="cancel">Cancel</v-btn>
        <v-btn
          :autofocus="dialogState.kind === 'confirm'"
          :color="dialogState.danger ? 'error' : 'primary'"
          variant="flat"
          @click="ok"
        >{{ dialogState.okText }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
  import { closeDialog, dialogState } from '@/composables/useDialog'

  function ok () {
    closeDialog(dialogState.kind === 'confirm' ? true : dialogState.value)
  }
  function cancel () {
    closeDialog(dialogState.kind === 'confirm' ? false : null)
  }
</script>
