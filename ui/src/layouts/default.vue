<template>
  <v-main>
    <v-app-bar :elevation="1" density="compact">
      <template v-slot:prepend>
        <v-app-bar-nav-icon> </v-app-bar-nav-icon>
      </template>

      <v-app-bar-title>
        <div class="d-flex align-center">
          Kashi
          <!-- <div style="font-family: serif; font-size: 20px">Marionette</div> -->
          <!-- <div style="font-size: 20px">Dashboard</div> -->

          <!--     <v-tabs v-if="tabs.length > 0"> -->
          <!--       <v-tab -->
          <!--         v-for="tab in tabs" -->
          <!--         :key="tab.to" -->
          <!--         :to="tab.to" -->
          <!--         :text="tab.text" -->
          <!--         :value="tab.to" -->
          <!--       ></v-tab> -->
          <!--     </v-tabs> -->
        </div>
      </v-app-bar-title>
      <SnackBar />
      <!-- <SearchInput /> -->
      <template v-slot:append>
        <NotificationMenu />
        <ToggleTheme />
        <div class="me-4"></div>
      </template>
    </v-app-bar>

    <NavigationDrawer />
    <v-container>
      <router-view />
    </v-container>
  </v-main>
  <!-- <AppFooter /> -->
</template>

<script lang="ts" setup>
import ToggleTheme from "@/components/Buttons/ToggleTheme.vue";
import NotificationMenu from "@/components/Menus/NotificationMenu.vue";

//
</script>
