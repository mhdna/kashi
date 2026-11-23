<template>
  <!-- <v-col cols="12" sm="6" md="3"> -->
  <!-- :prefix="$t('search')" -->
  <v-autocomplete
    ref="searchBar"
    auto-select-first
    density="compact"
    flat
    hide-details
    item-title="title"
    item-value="route"
    :items="items"
    :no-data-text="t('No routes found')"
    prepend-inner-icon="mdi-magnify"
    single-line
    transition="slide-y-transition"
    variant="outlined"
    @update:model-value="goToRoute"
  >
    <template #label>
      <span>{{ t('Search') }} <v-code><b>C-K</b></v-code></span>
    </template>
    <template #item="{ props, item }">
      <v-list-item
        v-bind="props"
        :prepend-icon="item.raw.prependIcon"
        :subtitle="item.raw.subtitle"
      />
    </template>
  </v-autocomplete>
  <!-- </v-col> -->
</template>

<script setup lang="ts">
  import { computed, onMounted, onUnmounted, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { t } from '@/i18n'

  const router = useRouter()
  const searchBar = ref<HTMLInputElement | null>(null)
  const selectedRoute = ref<string>('')

  function goToRoute (route: string | null) {
    if (route) {
      router.push(route)
    }
    selectedRoute.value = '' // Reset selected route
    if (searchBar?.value) {
      searchBar.value.blur()
    }
  }

  // const selectFirstItem = () => {
  //   if (!selectedRoute.value && items.length > 0) {
  //     selectedRoute.value = items[0].route;
  //   }
  //   goToRoute(selectedRoute.value);
  // };

  function openSearchBar (event: KeyboardEvent) {
    if (event.ctrlKey && event.key === 'k') {
      event.preventDefault()
      if (searchBar?.value) {
        searchBar.value.focus()
      }
    }
  }

  onMounted(() => {
    window.addEventListener('keydown', openSearchBar)
  })

  onUnmounted(() => {
    window.removeEventListener('keydown', openSearchBar)
  })

  const items = computed(() => [
    // TODO do it right
    // TODO english autocomplete even if you're typing arabic
    // TODO icons not working
    {
      prependIcon: 'mdi-magnify',
      title: t('Overview'),
      subtitle: t('Safe Search'),
      route: '/',
    },
    {
      prependIcon: 'mdi-clock-outline',
      title: t('Screenshots'),
      subtitle: t('Safe Search'),
      route: '/screenshots',
    },
    {
      prependIcon: 'mdi-clock-outline',
      title: t('URL List'),
      subtitle: t('Safe Search'),
      route: '/proxy/urls',
    },
    {
      prependIcon: 'mdi-clock-outline',
      title: t('Phrases'),
      subtitle: t('Safe Search'),
      route: '/proxy/phrases',
    },
    {
      prependIcon: 'mdi-clock-outline',
      title: t('Redirects'),
      subtitle: t('Safe Search'),
      route: '/proxy/redirects',
    },
    {
      prependIcon: 'mdi-clock-outline',
      title: t('Safe Search'),
      subtitle: t('Safe Search'),
      route: '/proxy/safesearch',
    },
    {
      prependIcon: 'mdi-clock-outline',
      title: t('Safe Search'),
      subtitle: t('Safe Search'),
      route: '/proxy/safesearch',
    },
    {
      prependIcon: 'mdi-clock-outline',
      title: t('Safe Search'),
      subtitle: t('Safe Search'),
      route: '/proxy/safesearch',
    },
    {
      prependIcon: 'mdi-clock-outline',
      title: t('Safe Search'),
      subtitle: t('Safe Search'),
      route: '/proxy/safesearch',
    },
    {
      prependIcon: 'mdi-clock-outline',
      title: t('Safe Search'),
      subtitle: t('Safe Search'),
      route: '/proxy/safesearch',
    },
  ])
</script>
