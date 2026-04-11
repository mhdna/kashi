<template>
    <v-data-table-server v-model:items-per-page="itemsPerPage" :headers="headers" :items="serverItems"
        :items-length="totalItems" :loading="loading" :search="search" item-value="name"
        @update:options="loadItems"></v-data-table-server>
</template>
<script setup>
import { ref } from 'vue'

async function fetchProducts({ page, itemsPerPage, sortBy }) {
    const res = await fetch('http://localhost:4123/products')
    const data = await res.json()

    let items = data.products // <-- from your Go envelope

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
const headers = ref([
    { title: 'ID', key: 'id', align: 'start' },
    { title: 'Code', key: 'code', align: 'start' },
    { title: 'Name', key: 'name', align: 'start' },
    { title: 'Description', key: 'description', align: 'start' },
    { title: 'Active', key: 'is_active', align: 'center' },
    { title: 'Price', key: 'price', align: 'end' },
    { title: 'Discount', key: 'discount', align: 'end' },
    { title: 'Created At', key: 'created_at', align: 'end' },
])

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