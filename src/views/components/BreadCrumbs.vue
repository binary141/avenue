<template>
  <nav class="breadcrumbs-container" aria-label="Breadcrumb">
    <ul class="breadcrumbs">
      <li
        v-for="(breadcrumb, index) in breadcrumbs"
        :key="breadcrumb.folder_id"
        class="breadcrumb-item"
      >
        <span v-if="index === 0" class="breadcrumb-icon" aria-hidden="true">📁</span>

        <!-- Clickable breadcrumbs (not last) -->
        <button
          v-if="index < breadcrumbs.length - 1"
          class="breadcrumb-button"
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
          ›
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

.breadcrumb-icon {
  margin-right: 0.4rem;
  font-size: 0.95em;
}

/* Buttons (ancestor crumbs) */
.breadcrumb-button {
  background-color: transparent;
  color: var(--text-secondary);
  border: none;
  padding: 0.15rem 0.35rem;
  margin: -0.15rem -0.35rem;
  border-radius: 6px;
  font-size: inherit;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.15s, color 0.15s;
}

.breadcrumb-button:hover {
  background-color: var(--gray-4);
  color: var(--text);
  text-decoration: underline;
}

.breadcrumb-button:active {
  background-color: var(--gray-5);
}

/* Current breadcrumb */
.breadcrumb-current {
  padding: 0.15rem 0.35rem;
  font-weight: 700;
  color: var(--text);
  cursor: default;
}

/* Separator */
.breadcrumb-separator {
  margin: 0 0.35rem;
  color: var(--text-secondary);
  opacity: 0.6;
  user-select: none;
}
</style>
