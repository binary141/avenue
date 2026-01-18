<template>
  <nav class="breadcrumbs-container" aria-label="Breadcrumb">
    <ul class="breadcrumbs">
      <li
        v-for="(breadcrumb, index) in breadcrumbs"
        :key="breadcrumb.folder_id"
        class="breadcrumb-item"
      >
        <!-- Clickable breadcrumbs (not last) -->
        <button
          v-if="index < breadcrumbs.length - 1"
          class="app-button breadcrumb-button"
          @click="handleClick(breadcrumb)"
        >
          {{ breadcrumb.label }}
        </button>

        <!-- Current folder (last breadcrumb) -->
        <span v-else class="breadcrumb-current">
          {{ breadcrumb.label }}
        </span>

        <!-- Separator -->
        <span
          v-if="index < breadcrumbs.length - 1"
          class="breadcrumb-separator"
          aria-hidden="true"
        >
          /
        </span>
      </li>
    </ul>
  </nav>
</template>

<script setup lang="ts">
import type { PropType } from 'vue'
import { useRouter } from 'vue-router'
import type { Breadcrumb } from '@/types/folder'

const router = useRouter()

defineProps({
  breadcrumbs: {
    type: Array as PropType<Breadcrumb[]>,
    required: true,
  },
})

function handleClick(breadcrumb: Breadcrumb) {
  router.push({
    path: '/',
    query: { folderId: breadcrumb.folder_id },
  })
}
</script>

<style scoped>
/* Container */
.breadcrumbs-container {
  width: 100%;
}

/* List */
.breadcrumbs {
  display: flex;
  align-items: center;
  flex-wrap: wrap;

  list-style: none;
  padding: 0;
  margin: 0;
}

/* List item */
.breadcrumb-item {
  display: flex;
  align-items: center;
}

/* Buttons */
.breadcrumb-button {
  background-color: transparent;
  color: var(--text);
  border: none;
  padding: 0.25rem 0.5rem;
  border-radius: 6px;
  cursor: pointer;
}

.breadcrumb-button:hover {
  background-color: var(--primary-hover);
}

.breadcrumb-button:active {
  background-color: var(--primary-active);
}

/* Current breadcrumb */
.breadcrumb-current {
  padding: 0.25rem 0.5rem;
  font-weight: 600;
  opacity: 0.85;
  cursor: default;
}

/* Separator */
.breadcrumb-separator {
  margin: 0 0.25rem;
  opacity: 0.5;
  user-select: none;
}
</style>

