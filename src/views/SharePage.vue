<template>
  <div class="min-h-screen flex items-center justify-center p-6" style="background: var(--gray, #2C2B2B);">
    <div class="bg-white rounded-lg shadow-lg w-full max-w-md p-8">

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center gap-3 text-gray-500">
        <svg class="animate-spin h-8 w-8 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z"/>
        </svg>
        <p>Loading…</p>
      </div>

      <!-- Error / expired -->
      <div v-else-if="notFound" class="text-center">
        <p class="text-4xl mb-3">🔒</p>
        <h1 class="text-xl font-bold text-gray-800 mb-2">Link not found</h1>
        <p class="text-gray-500 text-sm">This share link doesn't exist or has expired.</p>
      </div>

      <!-- File info -->
      <div v-else-if="meta" class="flex flex-col gap-5">
        <div class="flex items-center gap-3">
          <span class="text-4xl">📄</span>
          <div class="min-w-0">
            <h1 class="text-lg font-bold text-gray-800 break-words">{{ meta.file_name }}</h1>
            <p class="text-sm text-gray-500">{{ formatFileSize(meta.file_size) }} &middot; {{ meta.mime_type }}</p>
          </div>
        </div>

        <div v-if="meta.expires_at" class="text-xs text-gray-400">
          Expires {{ formatExpiry(meta.expires_at) }}
        </div>

        <a
          :href="downloadURL"
          class="block text-center px-4 py-3 rounded font-semibold text-white"
          style="background: #3A3F78;"
        >
          ⬇️ Download
        </a>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRoute } from 'vue-router';

interface ShareMeta {
  file_name: string;
  file_size: number;
  mime_type: string;
  expires_at: string | null;
  token: string;
}

const route = useRoute();
const loading = ref(true);
const notFound = ref(false);
const meta = ref<ShareMeta | null>(null);

const apiRoot = import.meta.env.VITE_APP_API_URL || '';

const downloadURL = computed(() =>
  meta.value ? `${apiRoot}share/${meta.value.token}/download` : ''
);

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

function formatExpiry(iso: string): string {
  return new Date(iso).toLocaleString();
}

onMounted(async () => {
  const token = route.params.token as string;
  try {
    const res = await fetch(`${apiRoot}share/${token}`);
    if (!res.ok) {
      notFound.value = true;
    } else {
      meta.value = await res.json();
    }
  } catch {
    notFound.value = true;
  } finally {
    loading.value = false;
  }
});
</script>
