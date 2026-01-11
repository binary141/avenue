<template>
  <li v-for="breadcrumb in props.breadcrumbs">
    <button @click="handleClick(breadcrumb)">{{ breadcrumb.label }}</button>
  </li>
</template>

<script setup lang="ts">
import type { PropType } from 'vue';
import { useRouter } from 'vue-router';
import type { Breadcrumb } from '@/types/folder';

const router = useRouter();

const props = defineProps({
  breadcrumbs: Array as PropType<Breadcrumb[]>,
})

function handleClick(breadcrumb: Breadcrumb) {
  router.push({ path: '/', query: { folderId: breadcrumb.folder_id }})
}
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
