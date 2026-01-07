<template>
  <v-avatar size="150">
    <v-img
      src="https://cdn.vuetifyjs.com/images/profiles/marcus.jpg"
      cover
    ></v-img>
  </v-avatar>
  <v-list-item
    class="text-white"
    :subtitle="title"
    :title="props.name"
  ></v-list-item>
</template>

<script setup>
const props = defineProps({
  name: String,
  title: String,
});
</script>
