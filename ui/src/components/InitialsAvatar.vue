<template>
  <v-avatar color="blue-darken-4" class="text-white" size="180">
    <span class="text-h2"> {{ initials }} Z </span>
  </v-avatar>
</template>

<script setup>
import { defineProps } from "vue";

defineProps({
  initials: {
    type: String,
    required: true,
  },
});
</script>
