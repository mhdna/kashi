<template>
  <div v-if="ordered.length > 0" class="online-stack d-flex align-center mx-2">
    <v-avatar
      v-for="u in visible"
      :key="u.id"
      class="online-stack__avatar text-white font-weight-medium"
      :class="{ 'online-stack__avatar--me': u.id === auth.user?.id }"
      :color="colorFor(u)"
      :size="u.id === auth.user?.id ? 26 : 30"
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
  import { colorForUser, initialsOf, useUserAvatar } from '@/composables/useUserAvatar'
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
  // everyone else in the server's stable (ID) order. You come from the auth
  // store rather than the server's list: you only appear there once this
  // tab's admin socket has connected, which the first fetch usually beats,
  // so waiting on it left the stack empty until the next presence push.
  const ordered = computed(() => {
    const me = auth.user
    const others = users.value.filter(u => u.id !== me?.id)
    return me?.id ? [me, ...others] : others
  })
  const visible = computed(() => ordered.value.slice(0, maxVisible.value))
  const overflow = computed(() => ordered.value.slice(maxVisible.value))

  // Everyone in the colour they picked in Settings (sent with presence);
  // the signed-in user's own comes from their user, so a change shows here
  // before the next presence push.
  function colorFor (u) {
    return u.id === auth.user?.id ? ownColor.value : colorForUser(u)
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

  /* You: a smaller avatar so the dark gap and a yellow ring fit in the
     same 34px as everyone else (26 + 2 gap + 2 ring on each side). You're
     always first and overlap into the others like any neighbour, but drawn
     on top of them so your ring stays whole. */
  .online-stack__avatar--me {
    position: relative;
    z-index: 1;
    box-shadow: 0 0 0 2px #FFC107;
    margin: 2px;
    margin-inline-end: 0;
  }

  .online-stack__more {
    background: rgb(var(--v-theme-surface-variant));
    color: rgb(var(--v-theme-on-surface-variant));
  }
</style>
