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

    <div v-if="!loading" class="w-full flex flex-col gap-6" style="max-width: 800px;">
      <!-- Empty state -->
      <div v-if="shares.length === 0 && folderShares.length === 0" class="card text-center py-10 flex flex-col items-center gap-3">
        <span style="font-size: 2.5rem;">🔗</span>
        <p class="font-semibold">No active shared links</p>
        <p class="text-sm" style="color: var(--text-secondary);">
          Go to Drive and click 🔗 on any file or folder to share it.
        </p>
      </div>

      <!-- File shares -->
      <div v-if="shares.length > 0" class="flex flex-col gap-3">
        <h2 class="text-sm font-semibold uppercase tracking-wide" style="color: var(--text-secondary);">Files</h2>
        <div
          v-for="share in shares"
          :key="share.token"
          class="card flex flex-row items-center gap-3 p-4"
        >
          <span style="font-size: 1.4rem; flex-shrink: 0;">📄</span>

          <span
            class="text-xs font-medium px-1.5 py-0.5 rounded shrink-0"
            :style="share.require_login
              ? 'background: #e8eaf6; color: #3A3F78;'
              : 'background: #e8f5e9; color: #2e7d32;'"
          >{{ share.require_login ? 'Internal' : 'Public' }}</span>

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
            <p>{{ share.last_accessed ? 'Last accessed ' + formatDate(share.last_accessed) : 'Never accessed' }}</p>
          </div>

          <AppButton
            @click="copyLink(share.token, 'file')"
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

      <!-- Folder shares -->
      <div v-if="folderShares.length > 0" class="flex flex-col gap-3">
        <h2 class="text-sm font-semibold uppercase tracking-wide" style="color: var(--text-secondary);">Folders</h2>
        <div
          v-for="share in folderShares"
          :key="share.token"
          class="card flex flex-row items-center gap-3 p-4"
        >
          <span style="font-size: 1.4rem; flex-shrink: 0;">📁</span>

          <span
            class="text-xs font-medium px-1.5 py-0.5 rounded shrink-0"
            :style="share.require_login
              ? 'background: #e8eaf6; color: #3A3F78;'
              : 'background: #e8f5e9; color: #2e7d32;'"
          >{{ share.require_login ? 'Internal' : 'Public' }}</span>

          <div class="flex-1 min-w-0">
            <p class="font-medium text-sm truncate">
              {{ share.folder_name || '(deleted folder)' }}
            </p>
            <p class="font-mono text-xs truncate" style="color: var(--text-secondary);">
              {{ folderShareLinkURL(share.token) }}
            </p>
          </div>

          <div class="text-xs shrink-0 text-right" style="color: var(--text-secondary);">
            <p>Created {{ formatDate(share.created_at) }}</p>
            <p>{{ share.expires_at ? 'Expires ' + formatDate(share.expires_at) : 'Never expires' }}</p>
            <p>{{ share.last_accessed ? 'Last accessed ' + formatDate(share.last_accessed) : 'Never accessed' }}</p>
          </div>

          <AppButton
            @click="copyLink(share.token, 'folder')"
            class="px-3 py-1 bg-blue-600 text-white text-sm rounded shrink-0"
          >
            {{ copied[share.token] ? '✓ Copied' : 'Copy Link' }}
          </AppButton>

          <AppButton
            @click="revokeFolderShare(share.token)"
            class="px-3 py-1 bg-red-100 text-red-600 text-sm rounded shrink-0"
          >
            Revoke
          </AppButton>
        </div>
      </div>

      <!-- Expired links -->
      <div v-if="expiredShares.length > 0 || expiredFolderShares.length > 0" class="flex flex-col gap-3">
        <h2 class="text-sm font-semibold uppercase tracking-wide" style="color: var(--text-secondary);">Expired</h2>

        <div
          v-for="share in expiredShares"
          :key="share.token"
          class="card flex flex-row items-center gap-3 p-4 opacity-60"
        >
          <span style="font-size: 1.4rem; flex-shrink: 0;">📄</span>

          <div class="flex-1 min-w-0">
            <p class="font-medium text-sm truncate">{{ share.file_name || '(deleted file)' }}</p>
            <p class="font-mono text-xs truncate" style="color: var(--text-secondary);">{{ shareLinkURL(share.token) }}</p>
          </div>

          <div class="text-xs shrink-0 text-right" style="color: var(--text-secondary);">
            <p>Created {{ formatDate(share.created_at) }}</p>
            <p>Expired {{ formatDate(share.expires_at!) }}</p>
            <p>{{ share.last_accessed ? 'Last accessed ' + formatDate(share.last_accessed) : 'Never accessed' }}</p>
          </div>

          <AppButton
            @click="revoke(share.token)"
            class="px-3 py-1 bg-red-100 text-red-600 text-sm rounded shrink-0"
          >
            Delete
          </AppButton>
        </div>

        <div
          v-for="share in expiredFolderShares"
          :key="share.token"
          class="card flex flex-row items-center gap-3 p-4 opacity-60"
        >
          <span style="font-size: 1.4rem; flex-shrink: 0;">📁</span>

          <div class="flex-1 min-w-0">
            <p class="font-medium text-sm truncate">{{ share.folder_name || '(deleted folder)' }}</p>
            <p class="font-mono text-xs truncate" style="color: var(--text-secondary);">{{ folderShareLinkURL(share.token) }}</p>
          </div>

          <div class="text-xs shrink-0 text-right" style="color: var(--text-secondary);">
            <p>Created {{ formatDate(share.created_at) }}</p>
            <p>Expired {{ formatDate(share.expires_at!) }}</p>
            <p>{{ share.last_accessed ? 'Last accessed ' + formatDate(share.last_accessed) : 'Never accessed' }}</p>
          </div>

          <AppButton
            @click="revokeFolderShare(share.token)"
            class="px-3 py-1 bg-red-100 text-red-600 text-sm rounded shrink-0"
          >
            Delete
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
  last_accessed: string | null;
  require_login: boolean;
}

interface FolderShareLink {
  token: string;
  folder_uuid: string;
  folder_name: string;
  expires_at: string | null;
  created_at: string;
  last_accessed: string | null;
  require_login: boolean;
}

const loading = ref(true);
const error = ref('');
const shares = ref<ShareWithFile[]>([]);
const folderShares = ref<FolderShareLink[]>([]);
const expiredShares = ref<ShareWithFile[]>([]);
const expiredFolderShares = ref<FolderShareLink[]>([]);
const copied = ref<Record<string, boolean>>({});

function shareLinkURL(token: string): string {
  return `${window.location.origin}/share/${token}`;
}

function folderShareLinkURL(token: string): string {
  return `${window.location.origin}/share/folder/${token}`;
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' });
}

async function loadShares() {
  loading.value = true;
  const [fileRes, folderRes, expiredFileRes, expiredFolderRes] = await Promise.all([
    api({ url: 'v1/shares', method: 'GET' }),
    api({ url: 'v1/folder-shares', method: 'GET' }),
    api({ url: 'v1/shares/expired', method: 'GET' }),
    api({ url: 'v1/folder-shares/expired', method: 'GET' }),
  ]);
  loading.value = false;
  if (fileRes.ok && Array.isArray(fileRes.body)) {
    shares.value = fileRes.body;
  } else {
    error.value = 'Failed to load file shares';
  }
  if (folderRes.ok && Array.isArray(folderRes.body)) {
    folderShares.value = folderRes.body;
  } else {
    error.value = error.value ? error.value + '; failed to load folder shares' : 'Failed to load folder shares';
  }
  if (expiredFileRes.ok && Array.isArray(expiredFileRes.body)) {
    expiredShares.value = expiredFileRes.body;
  }
  if (expiredFolderRes.ok && Array.isArray(expiredFolderRes.body)) {
    expiredFolderShares.value = expiredFolderRes.body;
  }
}

async function copyLink(token: string, type: 'file' | 'folder') {
  const url = type === 'folder' ? folderShareLinkURL(token) : shareLinkURL(token);
  await navigator.clipboard.writeText(url);
  copied.value = { ...copied.value, [token]: true };
  setTimeout(() => {
    copied.value = { ...copied.value, [token]: false };
  }, 2000);
}

async function revoke(token: string) {
  const response = await api({ url: `v1/share/${token}`, method: 'DELETE' });
  if (response.ok) {
    shares.value = shares.value.filter(s => s.token !== token);
    expiredShares.value = expiredShares.value.filter(s => s.token !== token);
  } else {
    error.value = 'Failed to revoke link';
  }
}

async function revokeFolderShare(token: string) {
  const response = await api({ url: `v1/share/folder/${token}`, method: 'DELETE' });
  if (response.ok) {
    folderShares.value = folderShares.value.filter(s => s.token !== token);
    expiredFolderShares.value = expiredFolderShares.value.filter(s => s.token !== token);
  } else {
    error.value = 'Failed to revoke folder share link';
  }
}

onMounted(loadShares);
</script>
