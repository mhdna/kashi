<template>
  <!-- <ProductsList /> -->
  <v-dialog max-width="920" max-height="780" v-model="dialog" persistent="true">
    <!-- <template v-slot:activator="{ props: activatorProps }"> -->
    <!-- </template> -->

    <template v-slot:default="{ isActive }">
      <v-card class="px-4">
        <v-card-title>
          <!-- <div class="d-flex justify-space-between"> -->
          <!-- <div> -->
          Create a New Product
          <!--   </div> -->
          <!--   <v-btn icon="mdi-close" size="sm" @click="dialogActive.value = false"></v-btn> -->
          <!-- </div> -->
          <v-spacer />
        </v-card-title>
        <NewProductForm />

        <v-card-actions>
          <v-spacer></v-spacer>

          <v-btn text="Close Dialog" @click="dialog = false"></v-btn>
        </v-card-actions>
      </v-card>
    </template>
  </v-dialog>


  <v-btn @click="dialog = true" text="Add a New Product"></v-btn>
  <ProductsTable />
</template>

<script setup>
import { ref } from 'vue';
const dialog = ref(false)
</script>