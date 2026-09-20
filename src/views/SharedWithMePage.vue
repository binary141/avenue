<template>
  <div class="app-page gap-5">
    <div class="flex items-center justify-between mb-4 w-full">
      <h1 class="text-center flex-1 text-2xl font-bold">Shared with me</h1>
    </div>

    <div v-if="loading" class="flex flex-col items-center gap-3">
      <SpinnerView />
      <p>Loading shared items…</p>
    </div>

    <ErrorMessage :msg="error" @clear="error = ''" />

    <div v-if="!loading" class="w-full flex flex-col gap-3" style="max-width: 1000px;">
      <div v-if="items.length === 0" class="card text-center py-10 flex flex-col items-center gap-3">
        <span style="font-size: 2.5rem;">🤝</span>
        <p class="font-semibold">Nothing has been shared with you yet</p>
        <p class="text-sm" style="color: var(--text-secondary);">
          When someone shares a file or folder with you, it will show up here.
        </p>
      </div>

      <div
        v-for="item in items"
        :key="item.share_id"
        class="card flex flex-row items-center gap-3 p-4"
      >
        <span style="font-size: 1.4rem; flex-shrink: 0;">{{ item.type === 'folder' ? '📁' : '📄' }}</span>

        <div class="flex-1 min-w-0">
          <RouterLink
            v-if="item.type === 'folder'"
            :to="{ path: '/', query: { folderId: item.uuid } }"
            class="font-medium text-sm truncate block shared-name"
          >
            {{ item.name }}
          </RouterLink>
          <a
            v-else
            :href="downloadURL(item.uuid)"
            class="font-medium text-sm truncate block shared-name"
          >
            {{ item.name }}
          </a>
          <p class="text-xs truncate" style="color: var(--text-secondary);">
            Shared by {{ item.shared_by_email }} · {{ formatDate(item.shared_at) }}
          </p>
        </div>

        <span class="role-badge shrink-0" :class="`role-badge--${item.role}`">
          {{ item.role === 'editor' ? 'Can edit' : 'Can view' }}
        </span>

        <AppButton variant="danger-subtle" size="sm" class="shrink-0" @click="leave(item)">
          Remove
        </AppButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '@/utils/api'
import AppButton from './components/AppButton.vue'
import SpinnerView from './components/SpinnerView.vue'
import ErrorMessage from './components/ErrorMessage.vue'
import { confirm } from '@/composables/confirm'
import type { SharedWithMeItem } from '@/types/sharing'

const loading = ref(true)
const error = ref('')
const items = ref<SharedWithMeItem[]>([])

function downloadURL(fileId: string): string {
  const baseURL = import.meta.env.VITE_APP_API_URL || ''
  return `${baseURL}v1/file/${fileId}`
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' })
}

async function load() {
  loading.value = true
  const response = await api({ url: 'v1/shared-with-me', method: 'GET' })
  loading.value = false
  if (response.ok && Array.isArray(response.body)) {
    items.value = response.body
  } else {
    error.value = response.body?.message || 'Failed to load shared items'
  }
}

// Removing an item only drops your own access; the owner keeps the original.
async function leave(item: SharedWithMeItem) {
  const ok = await confirm({
    title: 'Remove from Shared with me?',
    message: `You'll lose access to "${item.name}" until ${item.shared_by_email} shares it with you again.`,
    confirmLabel: 'Remove',
    danger: true,
  })
  if (!ok) return

  const response = await api({ url: `v1/user-shares/${item.share_id}`, method: 'DELETE' })
  if (response.ok) {
    items.value = items.value.filter((i) => i.share_id !== item.share_id)
  } else {
    error.value = response.body?.message || 'Failed to remove shared item'
  }
}

onMounted(load)
</script>

<style scoped>
.shared-name {
  color: var(--text);
  text-decoration: none;
}

.shared-name:hover {
  text-decoration: underline;
}

.role-badge {
  font-size: 0.75rem;
  font-weight: 500;
  padding: 0.15rem 0.5rem;
  border-radius: 4px;
}

.role-badge--viewer {
  background: #e8eaf6;
  color: #3a3f78;
}

.role-badge--editor {
  background: #e8f5e9;
  color: #2e7d32;
}
</style>
