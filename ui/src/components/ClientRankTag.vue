<template>
    <!-- TODO: color based on rank -->
    <div class="d-flex mt-2 mx-3 px-2" style="background-color: gold; height: 25px; width: 120px;">
        <v-icon icon="mdi-star" class="me-2" />
        <div>
            {{ props.rank }}
        </div>
    </div>
</template>

<script setup>
const props = defineProps({
    rank: {
        type: String,
        required: true
    }
})
</script>