<template>
  <v-main>
    <v-app-bar color="grey-darken-3" density="compact" :elevation="2">
      <template #prepend>
        <v-btn icon="mdi-menu" variant="text" @click.stop="toggleDrawer" />
      </template>

      <v-app-bar-title>
        <div class="d-flex align-center">
          <!--<img class="me-1" src="/everywear.png" style="height: 35px;">-->
          Kashi Cloud
          <span v-if="pageTitle" class="ms-2">- {{ pageTitle }}</span>
        </div>
      </v-app-bar-title>
      <!-- <SnackBar /> -->
      <template #append>
        <UpcomingEventsCard />
        <SyncCard />
        <ConnectedBranchesCard />
        <SyncingBranchesCard />
        <CommandPalette />
        <!-- <NotificationMenu class="me-4" /> -->
        <!-- <v-icon icon="mdi-translate" /> -->
        <ToggleTheme />
        <OnlineUsersStack />
        <v-btn icon="mdi-logout" variant="text" @click="handleLogout" />
      </template>
    </v-app-bar>

    <NavigationDrawer v-model="showDrawer" :mobile="mobile" :rail="isRail" />

    <!-- Kept outside <v-app-bar>: Vuetify transforms the app bar for its
         scroll behaviour, and a transformed ancestor would anchor this
         fixed reel to the bar (top, clipped) instead of the viewport. -->
    <BranchActivityToast />

    <div class="page-wrapper">
      <div v-if="isNavigating" class="loading-overlay">
        <div class="google-spinner">
          <div class="google-spinner-inner" />
        </div>
      </div>

      <Suspense>
        <template #default>
          <router-view />
        </template>

        <template #fallback>
          <v-container class="fill-height">
            <v-row align="center" justify="center">
              <v-progress-circular color="primary" indeterminate size="64" />
            </v-row>
          </v-container>
        </template>
      </Suspense>
    </div>
  </v-main>
  <!-- <AppFooter /> -->
  <AppNotifications />
</template>

<script lang="ts" setup>
import { onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useDisplay } from "vuetify";
import AppNotifications from "@/components/AppNotifications.vue";
import BranchActivityToast from "@/components/BranchActivityToast.vue";
import ToggleTheme from "@/components/Buttons/ToggleTheme.vue";
import CommandPalette from "@/components/CommandPalette.vue";
import ConnectedBranchesCard from "@/components/ConnectedBranchesCard.vue";
import OnlineUsersStack from "@/components/OnlineUsersStack.vue";
import SyncingBranchesCard from "@/components/SyncingBranchesCard.vue";
// import NotificationMenu from '@/components/Menus/NotificationMenu.vue'
import { useAuthStore } from "@/stores/auth";
import { useSettingsStore } from "@/stores/settings";

const auth = useAuthStore();
const settingsStore = useSettingsStore();
// Loads the global app settings (activity display prefs, financials high
// season months) once per session -- BranchActivityToast/SyncCard read
// settingsStore's activity fields reactively, so this can resolve after
// they mount without any special handling.
onMounted(() => {
  settingsStore.init();
  auth.refreshAccess();
});

function handleLogout() {
  auth.logout();
  router.push("/login");
}

const showDrawer = ref(false);
const isRail = ref(false);
const { mobile } = useDisplay();

function toggleDrawer() {
  if (mobile.value) {
    showDrawer.value = !showDrawer.value;
    isRail.value = false;
  } else {
    isRail.value = !isRail.value;
  }
}

const router = useRouter();
const route = useRoute();
const isNavigating = ref(false);
const pageTitle = ref(route.meta.title || "");
const navStart = 0;

router.beforeEach((to) => {
  pageTitle.value = to.meta.title || "";
});

router.afterEach(() => {
  isNavigating.value = false;
});
</script>

<style scoped>
/*@import url("https://fonts.googleapis.com/css2?family=Google+Sans:ital,opsz,wght@0,17..18,400..700;1,17..18,400..700&display=swap");*/

.page-wrapper {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 48px);
  overflow: auto;
  position: relative;
}

.loading-overlay {
  position: absolute;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: white;
}

.google-spinner {
  width: 50px;
  height: 50px;
  border-radius: 50%;
  background: conic-gradient(from 135deg, #4285f4 90deg, #e8eaed 90deg);
  animation: google-spin 1s linear infinite;
}

.google-spinner-inner {
  position: absolute;
  inset: 4px;
  background: white;
  border-radius: 50%;
}

@keyframes google-spin {
  0% {
    transform: rotate(0deg);
  }

  100% {
    transform: rotate(360deg);
  }
}
</style>
