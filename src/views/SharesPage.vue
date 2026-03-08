<template>
  <div class="page gap-5">
    <div class="flex items-center justify-between mb-4 w-full">
      <h1 class="text-center flex-1 text-2xl font-bold">Shared Links</h1>
    </div>

    <div v-if="loading" class="flex flex-col items-center gap-3">
      <SpinnerView />
      <p>Loading shared links…</p>
    </div>

    <ErrorMessage :msg="error" @clear="error = ''" />

    <div v-if="!loading" class="w-full" style="max-width: 800px;">
      <!-- Empty state -->
      <div v-if="shares.length === 0" class="card text-center py-10 flex flex-col items-center gap-3">
        <span style="font-size: 2.5rem;">🔗</span>
        <p class="font-semibold">No active shared links</p>
        <p class="text-sm" style="color: var(--text-secondary);">
          Go to Drive, click 🔗 on any file to share it.
        </p>
      </div>

      <!-- Shares table -->
      <div v-else class="flex flex-col gap-3">
        <div
          v-for="share in shares"
          :key="share.token"
          class="card flex flex-row items-center gap-3 p-4"
        >
          <span style="font-size: 1.4rem; flex-shrink: 0;">📄</span>

          <div class="flex-1 min-w-0">
            <p class="font-medium text-sm truncate">
              {{ share.file_name || '(deleted file)' }}
            </p>
            <p class="font-mono text-xs truncate" style="color: var(--text-secondary);">
              {{ shareLinkURL(share.token) }}
            </p>
          </div>

          <div class="text-xs shrink-0 text-right" style="color: var(--text-secondary);">
            <p>Created {{ formatDate(share.created_at) }}</p>
            <p>{{ share.expires_at ? 'Expires ' + formatDate(share.expires_at) : 'Never expires' }}</p>
          </div>

          <AppButton
            @click="copyLink(share.token)"
            class="px-3 py-1 bg-blue-600 text-white text-sm rounded shrink-0"
          >
            {{ copied[share.token] ? '✓ Copied' : 'Copy Link' }}
          </AppButton>

          <AppButton
            @click="revoke(share.token)"
            class="px-3 py-1 bg-red-100 text-red-600 text-sm rounded shrink-0"
          >
            Revoke
          </AppButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import api from '@/utils/api';
import AppButton from './components/AppButton.vue';
import SpinnerView from './components/SpinnerView.vue';
import ErrorMessage from './components/ErrorMessage.vue';

interface ShareWithFile {
  token: string;
  file_id: string;
  file_name: string;
  expires_at: string | null;
  created_at: string;
}

const loading = ref(true);
const error = ref('');
const shares = ref<ShareWithFile[]>([]);
const copied = ref<Record<string, boolean>>({});

function shareLinkURL(token: string): string {
  return `${window.location.origin}/share/${token}`;
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' });
}

async function loadShares() {
  loading.value = true;
  const response = await api({ url: 'v1/shares', method: 'GET' });
  loading.value = false;
  if (response.ok && Array.isArray(response.body)) {
    shares.value = response.body;
  } else {
    error.value = 'Failed to load shared links';
  }
}

async function copyLink(token: string) {
  await navigator.clipboard.writeText(shareLinkURL(token));
  copied.value = { ...copied.value, [token]: true };
  setTimeout(() => {
    copied.value = { ...copied.value, [token]: false };
  }, 2000);
}

async function revoke(token: string) {
  const response = await api({ url: `v1/share/${token}`, method: 'DELETE' });
  if (response.ok) {
    shares.value = shares.value.filter(s => s.token !== token);
  } else {
    error.value = 'Failed to revoke link';
  }
}

onMounted(loadShares);
</script>
