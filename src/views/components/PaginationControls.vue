<template>
  <div v-if="totalPages > 1" class="flex flex-row items-center justify-center gap-3 pagination-controls">
    <AppButton
      class="px-3 py-1 text-sm"
      :disabled="page <= 1"
      @click="$emit('update:page', page - 1)"
    >
      Prev
    </AppButton>

    <span class="text-sm" style="color: var(--text-secondary);">
      Page {{ page }} of {{ totalPages }}
    </span>

    <AppButton
      class="px-3 py-1 text-sm"
      :disabled="page >= totalPages"
      @click="$emit('update:page', page + 1)"
    >
      Next
    </AppButton>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import AppButton from './AppButton.vue';

const props = defineProps<{
  page: number;
  limit: number;
  total: number;
}>();

defineEmits<{
  (e: 'update:page', page: number): void;
}>();

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / Math.max(1, props.limit))));
</script>
