<template>
  <form @submit.prevent="submit">
    <v-row>
      <v-col cols="8">
        <div class="w-100">
          <v-text-field density="compact" v-model="code.value.value" :counter="20"
            :error-messages="name.errorMessage.value" label="Code"></v-text-field>
          <v-text-field density="compact" v-model="name.value.value" :counter="100"
            :error-messages="name.errorMessage.value" label="Name"></v-text-field>
          <v-textarea no-resize v-model="description.value.value" :counter="100"
            :error-messages="name.errorMessage.value" label="Description"></v-textarea>

          <v-select density="compact" v-model="select.value.value" :error-messages="select.errorMessage.value"
            :items="items" label="Kind"></v-select>
          <v-select density="compact" v-model="select.value.value" :error-messages="select.errorMessage.value"
            :items="items" label="Category"></v-select>
          <v-select density="compact" v-model="select.value.value" :error-messages="select.errorMessage.value"
            :items="items" label="SubCategory"></v-select>
          <v-select density="compact" v-model="select.value.value" :error-messages="select.errorMessage.value"
            :items="items" label="Unit"></v-select>

          <div class="d-flex">
            <v-checkbox class="me-8" width="110" density="compact" v-model="hasColors.value.value"
              :error-messages="hasColors.errorMessage.value" label="Has Colors" type="checkbox" value="1"></v-checkbox>
            <v-select density="compact" multiple clearable v-model="selectedColors.value.value"
              :error-messages="selectedColors.errorMessage.value" :items="colors" label="Select Colors"></v-select>
          </div>

          <div class="d-flex">
            <v-checkbox density="compact" width="110" class="me-8" v-model="hasSizes.value.value"
              :error-messages="hasSizes.errorMessage.value" label="Has Sizes" type="checkbox" value="1"></v-checkbox>
            <v-select density="compact" multiple clearable v-model="selectedSizes.value.value"
              :error-messages="selectedSizes.errorMessage.value" :items="sizes" label="Select Sizes"></v-select>
          </div>
        </div>
      </v-col>
      <v-col class="4">
        <v-card flat height="650" width="100%" class="pa-5">
          <FileUploadCard />
        </v-card>
      </v-col>
    </v-row>
    <v-row>
      <v-col>
        <v-btn class="w-100" @click="handleReset" variant="solid"> clear </v-btn>
      </v-col>
      <v-col>
        <v-btn class="me-4 w-100" type="submit" variant="solid"> Add Product </v-btn>
      </v-col>
    </v-row>
  </form>
</template>
<script setup>
import { ref } from "vue";
import { useField, useForm } from "vee-validate";
import FileUploadCard from "../Cards/FileUploadCard.vue";

const { handleSubmit, handleReset } = useForm({
  validationSchema: {
    name(value) {
      if (value?.length >= 2) return true;

      return "Name needs to be at least 2 characters.";
    },
    phone(value) {
      if (/^[0-9-]{7,}$/.test(value)) return true;

      return "Phone number needs to be at least 7 digits.";
    },
    email(value) {
      if (/^[a-z.-]+@[a-z.-]+\.[a-z]+$/i.test(value)) return true;

      return "Must be a valid e-mail.";
    },
    select(value) {
      if (value) return true;

      return "Select an item.";
    },
    checkbox(value) {
      if (value === "1") return true;

      return "Must be checked.";
    },
  },
});
const name = useField("name");
const code = useField("email");
const description = useField("phone");
const select = useField("select");
const checkbox = useField("checkbox");
const selectedSizes = useField("selectedSizes");
const selectedColors = useField("selectedColors");
const hasColors = useField("hasColors");
const hasSizes = useField("hasSizes");

const items = ref(["Item 1", "Item 2", "Item 3", "Item 4"]);
const colors = ref(["Blue", "White", "Orange", "Yellow"]);
const sizes = ref(["XL", "L", "M", "S"]);

const submit = handleSubmit((values) => {
  alert(JSON.stringify(values, null, 2));
});
</script>
