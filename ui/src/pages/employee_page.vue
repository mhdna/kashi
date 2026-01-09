<script setup lang="ts">
import InitialsAvatar from "@/components/InitialsAvatar.vue";
import Heatmap from "@/components/Stats/Heatmap.vue";
import PercentageCircle from "@/components/Stats/PercentageCircle.vue";
</script>

<template>
  <v-card>
    <div class="d-flex">
      <InitialsAvatar />
      <div>
        <v-card-title>Test Employee</v-card-title>
        <v-card-text> This is the employee page. </v-card-text>
      </div>
    </div>
    <v-progress-circular
      :model-value="70"
      :rotate="360"
      :size="100"
      :width="15"
      color="teal"
    >
      <template v-slot:default> {{ 70 }} % </template>
    </v-progress-circular>
    <Heatmap />
  </v-card>
</template>
