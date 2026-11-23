<template>
  <v-btn
    :aria-label="t('Language')"
    variant="text"
    @click="setLocale(locale === 'fr' ? 'en' : 'fr')"
  >
    {{ locale === 'fr' ? 'EN' : 'FR' }}
  </v-btn>
</template>

<script setup>
  import { locale, setLocale, t } from '@/i18n'
</script>
