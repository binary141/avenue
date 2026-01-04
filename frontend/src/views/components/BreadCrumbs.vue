<template>
  <li v-for="x in props.breadcrumbs"><a :href=x>My Text</a></li>
</template>

<script setup lang="ts">
import type { PropType } from 'vue';

const props = defineProps({
breadcrumbs: Array,
})
</script>

<style scoped>
.app-button {
    width: auto;
    background-color: var(--primary);
    color: var(--text);
    border-radius: 8px;
    box-shadow: 2px 2px 4px rgba(53, 53, 53, 0.13);
}
.app-button:hover {
    background-color: var(--primary-hover);
}
.app-button:active {
    background-color: var(--primary-active);
}
</style>
