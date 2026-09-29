<template>
  <v-app>
    <v-main>
      <router-view />
    </v-main>
    <AppNotifications />
  </v-app>
</template>

<script setup>
  import AppNotifications from '@/components/AppNotifications.vue'
</script>
