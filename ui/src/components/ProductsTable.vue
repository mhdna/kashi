<template>
    <ServerSideTable :header="headers" :apiURL="apiURL" rootKey="products" />
</template>

<script setup>
const apiURL = 'http://localhost:4123/products'
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
</script>