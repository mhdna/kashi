<template>
    <v-text-field v-model="search" label="Search" prepend-inner-icon="mdi-magnify" variant="outlined" hide-details
        single-line></v-text-field>
    <v-data-table-server v-model:items-per-page="itemsPerPage" :headers="headers" :items="serverItems"
        :items-length="totalItems" :loading="loading" :search="search" item-value="name"
        @update:options="loadItems"></v-data-table-server>
</template>
<script setup>
import { ref } from 'vue'

const props = defineProps({
    apiURL: {
        type: String,
        required: true,
    },
    headers: {
        type: Array,
        required: true,
    },
    rootKey: {
        type: String,
        required: true,
    }
})

async function fetchProducts({ page, itemsPerPage, sortBy }) {
    const res = await fetch(props.apiURL)
    const data = await res.json()

    let items = data[props.rootKey]

    // sorting
    if (sortBy.length) {
        const sortKey = sortBy[0].key
        const sortOrder = sortBy[0].order
        items = items.slice().sort((a, b) => {
            const aValue = a[sortKey]
            const bValue = b[sortKey]
            return sortOrder === 'desc'
                ? (bValue > aValue ? 1 : -1)
                : (aValue > bValue ? 1 : -1)
        })
    }

    // pagination
    const start = (page - 1) * itemsPerPage
    const end = start + itemsPerPage
    const paginated = items.slice(start, end)

    return {
        items: paginated,
        total: items.length,
    }
}
const itemsPerPage = ref(12)

const search = ref('')
const serverItems = ref([])
const loading = ref(true)
const totalItems = ref(0)
function loadItems({ page, itemsPerPage, sortBy }) {
    loading.value = true
    fetchProducts({ page, itemsPerPage, sortBy }).then(({ items, total }) => {
        serverItems.value = items
        totalItems.value = total
        loading.value = false
    })
}
</script>