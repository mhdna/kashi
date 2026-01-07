<template>
  <v-main>
    <v-app-bar :elevation="1" density="compact">
      <template v-slot:prepend>
        <v-app-bar-nav-icon></v-app-bar-nav-icon>
      </template>

      <v-app-bar-title>Kashi</v-app-bar-title>
    </v-app-bar>

    <NavigationDrawer />
    <router-view />
  </v-main>
  <AppFooter />
</template>

<script lang="ts" setup>
//
</script>
