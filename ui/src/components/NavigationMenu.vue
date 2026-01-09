<!-- TODO disgusting code -->
<template>
  <v-list-group :id="item.id" v-if="item.subItems" :fluid="isRail">
    <!-- :ripple="false" -->
    <template v-slot:activator="{ props }">
      <v-list-item
        v-bind="props"
        :prepend-icon="item.icon"
        :title="item.text"
      />
    </template>
    <NavigationMenu
      v-for="(subItem, i) in item.subItems"
      :key="i"
      :id="subItem.id"
      :item="subItem"
    />
    <!-- :ripple="false" -->
  </v-list-group>

  <!-- TODO: color="primary" -->
  <v-list-item
    v-else
    :id="item.id"
    :value="item"
    :prepend-icon="item.icon"
    :title="item.text"
    :to="item.to"
  >
    <!-- :ripple="false" -->
    <!-- TODO change these -->
    <template v-if="item.text === 'Requests'" v-slot:append>
      <v-badge color="red-darken-4" content="6" inline></v-badge>
    </template>
    <template v-if="item.text === 'Live Chat'" v-slot:append>
      <v-badge color="green-darken-4" content="6" inline></v-badge>
    </template>
  </v-list-item>
</template>

<script setup>
const props = defineProps({
  item: String,
  // isRail: Boolean,
});
</script>
