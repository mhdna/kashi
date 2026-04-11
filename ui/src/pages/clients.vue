<template>
    <ServerSideTable :header="headers" :apiURL="apiURL" rootKey="clients" />
</template>

<script setup>
const apiURL = 'http://localhost:4123/clients'
const headers = ref([
    { title: 'ID', key: 'id', align: 'start' },
    { title: 'Name', key: 'name', align: 'start' },
    { title: 'Phone', key: 'phone', align: 'start' },
    { title: 'Loyalty Points', key: 'loyalty_points', align: 'end' },
    { title: 'Created At', key: 'created_at', align: 'end' },
])
</script>