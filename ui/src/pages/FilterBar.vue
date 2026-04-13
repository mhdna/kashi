<script setup>
const menu = ref(false)
const date = ref([new Date(), new Date()])
const formattedDate = computed(() => {
    if (!date.value || date.value.length === 0) return ""

    const format = (d) =>
        new Date(d).toLocaleDateString("en-GB")

    if (date.value.length === 1) {
        return new Date(date.value[0]).toLocaleDateString()
    }

    return `${new Date(date.value[0]).toLocaleDateString()} → ${new Date(date.value[1]).toLocaleDateString()}`
})
</script>
<template>
    <v-sheet color="surface-light" width="100%" style="height: 50px;" class="d-flex align-center pt-0">
        <v-row class="d-flex align-center ">
            <v-spacer />
            <!-- <div class="me-2 mt-4"> -->
            <!--     Filters: -->
            <!-- </div> -->
            <v-icon icon="mdi-arrow-left-thin" variant="flat" size="22" class="pt-1" />
            <v-menu v-model="menu" :close-on-content-click="false" location="end">
                <template v-slot:activator="{ props }">
                    <v-btn color="surface-light" variant="flat" v-bind="props" class="ma-0 py-4 pt-5" max-width="180">
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
            <v-icon icon="mdi-arrow-right-thin" variant="flat" size="22" class="pt-1" />
            <v-spacer />
            <v-select label="Branch" :items="['Branch 1', 'Branch 2', 'Branch 3', 'Branch 4']" max-width="150"
                density="compact" variant="flat" class="pt-6"></v-select>
        </v-row>
    </v-sheet>
</template>