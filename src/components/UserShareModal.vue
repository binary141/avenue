<template>
  <AppModal :show="!!item" width="520px" @close="emit('close')">
    <template #header>
      <div>
        <h3 class="text-lg font-bold mb-1">Share "{{ item?.name }}" with people</h3>
        <p class="text-sm user-share-subtext">
          They need an account on this server.
          <template v-if="item?.type === 'folder'">Access includes everything inside the folder.</template>
        </p>
      </div>
    </template>

    <ErrorMessage :msg="error" @clear="error = ''" />

    <form class="flex items-center gap-2 mb-4" @submit.prevent="addShare">
      <input
        v-model="email"
        type="email"
        required
        placeholder="Email address"
        class="flex-1 user-share-input"
        aria-label="Email address"
      />
      <select v-model="role" class="user-share-select" aria-label="Role">
        <option value="viewer">Viewer</option>
        <option value="editor">Editor</option>
      </select>
      <AppButton type="submit" size="sm" :disabled="adding || !email.trim()">
        {{ adding ? 'Sharing…' : 'Share' }}
      </AppButton>
    </form>

    <hr class="mb-4 user-share-divider" />

    <div v-if="loading" class="flex justify-center py-6">
      <SpinnerView />
    </div>

    <template v-else>
      <p class="text-sm font-semibold user-share-subtext mb-2">
        People with access<span v-if="shares.length"> ({{ shares.length }})</span>
      </p>
      <div v-if="shares.length === 0" class="text-sm user-share-muted py-2 text-center">
        Not shared with anyone yet.
      </div>
      <div v-else class="flex flex-col gap-2">
        <div
          v-for="share in shares"
          :key="share.id"
          class="user-share-row flex items-center gap-2 rounded px-3 py-2 text-sm"
        >
          <div class="flex-1 min-w-0">
            <p class="truncate font-medium">{{ displayName(share) }}</p>
            <p v-if="displayName(share) !== share.grantee_email" class="truncate text-xs user-share-muted">
              {{ share.grantee_email }}
            </p>
          </div>
          <select
            :value="share.role"
            class="user-share-select shrink-0"
            :aria-label="`Role for ${share.grantee_email}`"
            @change="changeRole(share, ($event.target as HTMLSelectElement).value as ShareRole)"
          >
            <option value="viewer">Viewer</option>
            <option value="editor">Editor</option>
          </select>
          <AppButton variant="danger-subtle" size="sm" class="shrink-0" @click="removeShare(share)">
            Remove
          </AppButton>
        </div>
      </div>
    </template>

    <template #footer>
      <AppButton variant="secondary" size="sm" @click="emit('close')">Done</AppButton>
    </template>
  </AppModal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import api from '@/utils/api'
import AppButton from '@/views/components/AppButton.vue'
import AppModal from '@/views/components/AppModal.vue'
import ErrorMessage from '@/views/components/ErrorMessage.vue'
import SpinnerView from '@/views/components/SpinnerView.vue'
import type { FolderItem } from '@/types/folder'
import type { ShareRole, UserShare } from '@/types/sharing'

const props = defineProps<{
  // The file or folder to share; the modal is open while this is set.
  item: FolderItem | null
}>()

const emit = defineEmits<{
  close: []
  // How many people the item is now shared with, so the caller can keep any
  // badge in sync without refetching.
  changed: [count: number]
}>()

const shares = ref<UserShare[]>([])
const loading = ref(false)
const adding = ref(false)
const error = ref('')
const email = ref('')
const role = ref<ShareRole>('viewer')

function sharesURL(item: FolderItem): string {
  return `v1/${item.type}/${item.uuid}/user-shares`
}

function displayName(share: UserShare): string {
  const name = `${share.grantee_first_name} ${share.grantee_last_name}`.trim()
  return name || share.grantee_email
}

async function load() {
  if (!props.item) return
  loading.value = true
  const response = await api({ url: sharesURL(props.item), method: 'GET' })
  loading.value = false
  if (response.ok && Array.isArray(response.body)) {
    shares.value = response.body
  } else {
    error.value = response.body?.message || 'Failed to load who this is shared with'
  }
}

// Reset and reload whenever the modal is opened for a (different) item.
watch(
  () => props.item,
  (item) => {
    shares.value = []
    error.value = ''
    email.value = ''
    role.value = 'viewer'
    if (item) load()
  },
  { immediate: true },
)

async function addShare() {
  if (!props.item || !email.value.trim()) return
  adding.value = true
  error.value = ''
  const response = await api({
    url: sharesURL(props.item),
    method: 'POST',
    json: { email: email.value.trim(), role: role.value },
  })
  adding.value = false

  if (!response.ok) {
    error.value = response.body?.message || 'Failed to share'
    return
  }
  email.value = ''
  // Sharing again with someone who already has access just changes their
  // role, so reload rather than guess whether this added a row or updated one.
  await load()
  emit('changed', shares.value.length)
}

async function changeRole(share: UserShare, newRole: ShareRole) {
  const previous = share.role
  share.role = newRole
  const response = await api({
    url: `v1/user-shares/${share.id}`,
    method: 'PATCH',
    json: { role: newRole },
  })
  if (!response.ok) {
    share.role = previous
    error.value = response.body?.message || 'Failed to change role'
  }
}

async function removeShare(share: UserShare) {
  const response = await api({ url: `v1/user-shares/${share.id}`, method: 'DELETE' })
  if (!response.ok) {
    error.value = response.body?.message || 'Failed to remove access'
    return
  }
  shares.value = shares.value.filter((s) => s.id !== share.id)
  emit('changed', shares.value.length)
}
</script>

<style scoped>
.user-share-subtext {
  color: var(--text-secondary);
}

.user-share-muted {
  color: var(--text-tertiary);
}

.user-share-divider {
  border-color: var(--gray-4);
}

.user-share-row {
  background-color: var(--gray);
  border: 1px solid var(--gray-4);
}

.user-share-input {
  height: 36px;
  padding-block: 0;
  border: 1px solid var(--gray-5) !important;
}

.user-share-input:focus {
  border-color: var(--accent) !important;
  box-shadow: 0 0 0 3px rgba(107, 111, 230, 0.25);
}

.user-share-select {
  height: 36px;
  background-color: var(--gray-4);
  color: var(--text);
  border: none;
  border-radius: 6px;
  padding: 0 0.5rem;
  font-size: 0.85rem;
  cursor: pointer;
}
</style>
