<template>
  <div v-if="users.length > 0" class="online-stack d-flex align-center mx-2">
    <v-avatar
      v-for="u in visible"
      :key="u.id"
      class="online-stack__avatar text-white font-weight-medium"
      :color="colorFor(u)"
      size="30"
      :style="u.id === auth.user?.id ? 'cursor: pointer' : ''"
      @click="u.id === auth.user?.id && router.push('/settings')"
    >
      <span class="text-caption">{{ initialsOf(u.name) }}</span>
      <v-tooltip activator="parent" location="bottom">
        {{ u.name }}<template v-if="u.id === auth.user?.id"> (you)</template>
      </v-tooltip>
    </v-avatar>

    <v-avatar
      v-if="overflow.length > 0"
      class="online-stack__avatar online-stack__more"
      size="30"
    >
      <span class="text-caption font-weight-medium">+{{ overflow.length }}</span>
      <v-tooltip activator="parent" location="bottom">
        <div v-for="u in overflow" :key="u.id">{{ u.name }}</div>
      </v-tooltip>
    </v-avatar>
  </div>
</template>

<script setup>
  import { computed, onMounted } from 'vue'
  import { useRouter } from 'vue-router'
  import { useDisplay } from 'vuetify'
  import { useOnlineUsers } from '@/composables/useOnlineUsers'
  import { hashColor, initialsOf, useUserAvatar } from '@/composables/useUserAvatar'
  import { useAuthStore } from '@/stores/auth'

  const auth = useAuthStore()
  const { users, start } = useOnlineUsers()
  const { color: ownColor } = useUserAvatar()
  const router = useRouter()
  const { mobile } = useDisplay()

  onMounted(start)

  // Fewer slots on a phone so the stack doesn't crowd the other app-bar
  // controls out.
  const maxVisible = computed(() => (mobile.value ? 2 : 4))

  // Show the signed-in user first so "you" is always in the same spot, then
  // everyone else in the server's stable (ID) order.
  const ordered = computed(() => {
    const meId = auth.user?.id
    return users.value.toSorted((a, b) => (b.id === meId) - (a.id === meId))
  })
  const visible = computed(() => ordered.value.slice(0, maxVisible.value))
  const overflow = computed(() => ordered.value.slice(maxVisible.value))

  // The current user's avatar honours the per-device colour override.
  function colorFor (u) {
    return u.id === auth.user?.id ? ownColor.value : hashColor(u.name)
  }
</script>

<style scoped>
  /* Overlap each avatar onto the previous one; the ring is the app bar's own
     colour (grey-darken-3), which reads as a cut-out between neighbours. */
  .online-stack__avatar {
    border: 2px solid #424242;
    box-sizing: content-box;
  }

  .online-stack__avatar + .online-stack__avatar {
    margin-inline-start: -10px;
  }

  .online-stack__more {
    background: rgb(var(--v-theme-surface-variant));
    color: rgb(var(--v-theme-on-surface-variant));
  }
</style>
