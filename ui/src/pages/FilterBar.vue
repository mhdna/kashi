<script setup>
const menu = ref(false)
const date = ref([new Date(), new Date()])
const formattedDate = computed(() => {
    if (!date.value || date.value.length === 0) return ""

    if (date.value.length === 1) {
        return new Date(date.value[0]).toLocaleDateString()
    }

    return `${new Date(date.value[0]).toLocaleDateString()} → ${new Date(date.value[1]).toLocaleDateString()}`
})
</script>
<template>
    <v-sheet width="100%" style="height: 60px;" class="d-flex align-center ">
        <!-- color="surface-light" -->
        <v-row class="d-flex align-center ">
            <v-spacer />
            <!-- <div class="me-2 mt-4"> -->
            <!--     Filters: -->
            <!-- </div> -->
            <v-select class="mt-10" label="Branch" :items="['Branch 1', 'Branch 2', 'Branch 3', 'Branch 4']"
                max-width="150" density="compact" variant="outlined"></v-select>
            <v-menu v-model="menu" :close-on-content-click="false" location="end">
                <template v-slot:activator="{ props }">
                    <v-btn variant="outlined" v-bind="props" class="mt-4 py-4 pt-5 me-6" width="200">
                        {{ formattedDate }}
                    </v-btn>
                </template>

                <v-card min-width="300">
                    <v-date-picker v-model="date" multiple="range"></v-date-picker>
                    <v-card-actions>
                        <v-spacer></v-spacer>
                        <v-btn variant="text" @click="menu = false">
                            Cancel
                        </v-btn>
                        <v-btn color="primary" variant="text" @click="menu = false">
                            Save
                        </v-btn>
                    </v-card-actions>
                </v-card>
            </v-menu>
        </v-row>
    </v-sheet>
</template>