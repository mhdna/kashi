<template>
  <div class="text-center">
    <v-btn color="red-darken-2" @click="snackbar = true"> {{ t('Open Snackbar') }} </v-btn>

    <v-snackbar v-model="snackbar" location="top right" multi-line>
      {{ text }}

      <template #actions>
        <v-btn color="red" variant="text" @click="snackbar = false">
          {{ t('Close') }}
        </v-btn>
      </template>
    </v-snackbar>
  </div>
</template>
<script>
  import { t } from '@/i18n'

  export default {
    data: () => ({
      snackbar: false,
      text: `I am a multi-line snackbar.\nI can have more than one line. This is another line that is quite long.`,
    }),
  }
</script>
