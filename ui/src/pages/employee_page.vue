<script setup lang="ts">
import Heatmap from "@/components/Stats/Heatmap.vue";
import PercentageCircle from "@/components/Stats/PercentageCircle.vue";
</script>

<template>
  <v-card>
    <div class="d-flex">
      <v-avatar size="150">
        <v-img
          src="https://cdn.vuetifyjs.com/images/profiles/marcus.jpg"
          cover
        ></v-img>
      </v-avatar>
      <div>
        <v-card-title>Test Employee</v-card-title>
        <v-card-text> This is the employee page. </v-card-text>
      </div>
    </div>
    <PercentageCircle :size="120" :stroke-width="8" :percentage="75" />
    <Heatmap />
  </v-card>
</template>
