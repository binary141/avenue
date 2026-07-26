<template>
  <div class="page gap-5">
    <div class="flex items-center justify-between mb-4 w-full">
      <h1 class="text-center flex-1 text-2xl font-bold">Trash</h1>

      <AppButton
        v-if="folders.length > 0 || files.length > 0"
        @click="confirmEmptyTrash"
        class="px-3 py-2 bg-red-600 text-white rounded text-sm"
        :disabled="emptying"
      >
        {{ emptying ? 'Emptying…' : 'Empty Trash' }}
      </AppButton>
    </div>

    <div v-if="!loading && (folders.length > 0 || files.length > 0)" class="toolbar-controls flex items-center gap-2 mb-2 flex-wrap" style="max-width: 800px; width: 100%; justify-content: space-between;">
      <div class="flex items-center gap-3">
        <label class="flex items-center gap-2 text-sm select-none cursor-pointer" style="color: var(--text-secondary);">
          <input type="checkbox" :checked="allSelected" @change="toggleSelectAll" />
          Select all
        </label>

        <p class="item-count-label text-sm" style="color: var(--text-secondary);">
          Showing {{ currentPageCount }} of {{ totalItemsCount }} items
        </p>
      </div>

      <div class="flex items-center gap-2">
        <select v-model="sortKey" class="sort-select">
          <option value="date">Date Deleted</option>
          <option value="name">Name</option>
          <option value="size">Size</option>
        </select>
        <button
          type="button"
          class="sort-dir-button"
          :title="sortDir === 'asc' ? 'Ascending' : 'Descending'"
          @click="sortDir = sortDir === 'asc' ? 'desc' : 'asc'"
        >
          {{ sortDir === 'asc' ? '↑' : '↓' }}
        </button>
      </div>
    </div>

    <!-- Bulk Actions -->
    <div
      v-if="hasSelection"
      class="bulk-actions-bar flex items-center gap-3 mb-2 p-3 rounded"
      style="max-width: 800px; width: 100%;"
    >
      <span class="text-sm" style="color: var(--text-secondary);">
        Selected:
        {{ selectedFolders.size }} folders,
        {{ selectedFiles.size }} files
      </span>

      <AppButton class="bg-blue-600 text-white px-3 py-1" @click="bulkRestore">
        Restore Selected
      </AppButton>

      <AppButton class="modal-secondary-button px-3 py-1" @click="clearSelection">
        Clear
      </AppButton>
    </div>

    <div v-if="loading" class="flex flex-col items-center gap-3">
      <SpinnerView />
      <p>Loading trash…</p>
    </div>

    <ErrorMessage :msg="error" @clear="error = ''" />

    <p
      v-if="!loading && retentionDays > 0 && (folders.length > 0 || files.length > 0)"
      class="text-sm text-center"
      style="color: var(--text-secondary); max-width: 800px;"
    >
      Items in the trash are automatically deleted forever after {{ retentionDays }} day{{ retentionDays === 1 ? '' : 's' }}.
    </p>

    <div v-if="!loading" class="w-full flex flex-col gap-6" style="max-width: 800px;">
      <!-- Empty state -->
      <div v-if="folders.length === 0 && files.length === 0" class="card text-center py-10 flex flex-col items-center gap-3">
        <span style="font-size: 2.5rem;">🗑️</span>
        <p class="font-semibold">Trash is empty</p>
        <p class="text-sm" style="color: var(--text-secondary);">
          Files and folders you delete from Drive will show up here until they're purged.
        </p>
      </div>

      <!-- Folders -->
      <div v-if="folders.length > 0" class="flex flex-col gap-3">
        <h2 class="text-sm font-semibold uppercase tracking-wide" style="color: var(--text-secondary);">Folders</h2>
        <div
          v-for="(folder, index) in folders"
          :key="folder.uuid"
          class="card flex flex-row items-center gap-3 p-4"
        >
          <input
            type="checkbox"
            :checked="selectedFolders.has(folder.uuid)"
            @click.stop="onFolderCheckboxClick(folder.uuid, index, $event)"
          />

          <span style="font-size: 1.4rem; flex-shrink: 0;">📁</span>

          <div class="flex-1 min-w-0">
            <p class="font-medium text-sm truncate">{{ folder.name }}</p>
            <p class="text-xs" style="color: var(--text-secondary);">
              Deleted {{ formatDate(folder.deleted_at) }}
            </p>
          </div>

          <AppButton
            @click="restoreFolder(folder.uuid)"
            class="px-3 py-1 bg-blue-600 text-white text-sm rounded shrink-0"
          >
            Restore
          </AppButton>

          <AppButton
            @click="confirmPurgeFolder(folder)"
            class="px-3 py-1 bg-red-100 text-red-600 text-sm rounded shrink-0"
          >
            Delete Forever
          </AppButton>
        </div>
      </div>

      <!-- Files -->
      <div v-if="files.length > 0" class="flex flex-col gap-3">
        <h2 class="text-sm font-semibold uppercase tracking-wide" style="color: var(--text-secondary);">Files</h2>
        <div
          v-for="(file, index) in files"
          :key="file.uuid"
          class="card flex flex-row items-center gap-3 p-4"
        >
          <input
            type="checkbox"
            :checked="selectedFiles.has(file.uuid)"
            @click.stop="onFileCheckboxClick(file.uuid, index, $event)"
          />

          <span class="file-icon-badge">
            <svg viewBox="0 0 24 24" class="file-icon-svg">
              <path d="M5 3a1 1 0 0 0-1 1v16a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1V8l-6-6H5z" fill="var(--gray-4)" stroke="var(--gray-5)" stroke-width="1"/>
              <path d="M14 2v5a1 1 0 0 0 1 1h5" fill="none" stroke="var(--gray-5)" stroke-width="1"/>
              <rect x="1.5" y="15" width="19" height="7" rx="1.5" :fill="fileBadgeColor(file)"/>
              <text x="11" y="20.3" font-size="6.2" font-weight="700" fill="#fff" text-anchor="middle" font-family="Inter, sans-serif">{{ fileBadgeLabel(file) }}</text>
            </svg>
          </span>

          <div class="flex-1 min-w-0">
            <p class="font-medium text-sm truncate">{{ file.name }}</p>
            <p class="text-xs" style="color: var(--text-secondary);">
              {{ formatFileSize(file.file_size) }} · Deleted {{ formatDate(file.deleted_at) }}
            </p>
          </div>

          <AppButton
            @click="restoreFile(file.uuid)"
            class="px-3 py-1 bg-blue-600 text-white text-sm rounded shrink-0"
          >
            Restore
          </AppButton>

          <AppButton
            @click="confirmPurgeFile(file)"
            class="px-3 py-1 bg-red-100 text-red-600 text-sm rounded shrink-0"
          >
            Delete Forever
          </AppButton>
        </div>
      </div>

      <div v-if="totalItemsCount > 0" class="flex flex-col items-center gap-2">
        <p class="item-count-label text-sm" style="color: var(--text-secondary);">
          Showing {{ currentPageCount }} of {{ totalItemsCount }} items
        </p>
        <PaginationControls
          :page="page"
          :limit="limit"
          :total="Math.max(totalFiles, totalFolders)"
          @update:page="changePage"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import api from '@/utils/api';
import AppButton from './components/AppButton.vue';
import SpinnerView from './components/SpinnerView.vue';
import ErrorMessage from './components/ErrorMessage.vue';
import PaginationControls from './components/PaginationControls.vue';

interface TrashedFolder {
  id: number;
  uuid: string;
  name: string;
  deleted_at: string;
}

interface TrashedFile {
  id: number;
  uuid: string;
  name: string;
  extension: string;
  file_size: number;
  deleted_at: string;
}

const FILE_BADGE_COLORS: Record<string, string> = {
  pdf: '#d64545',
  doc: '#3b6fd6', docx: '#3b6fd6', txt: '#3b6fd6', md: '#3b6fd6', rtf: '#3b6fd6',
  xls: '#2f9e5c', xlsx: '#2f9e5c', csv: '#2f9e5c',
  ppt: '#d6822f', pptx: '#d6822f',
  zip: '#a67c2e', tar: '#a67c2e', gz: '#a67c2e', rar: '#a67c2e', '7z': '#a67c2e',
  mp4: '#8a4fd6', mov: '#8a4fd6', avi: '#8a4fd6', mkv: '#8a4fd6', webm: '#8a4fd6',
  mp3: '#d6478f', wav: '#d6478f', flac: '#d6478f', ogg: '#d6478f',
  js: '#5a6fd6', ts: '#5a6fd6', py: '#5a6fd6', go: '#5a6fd6', java: '#5a6fd6', c: '#5a6fd6', cpp: '#5a6fd6', json: '#5a6fd6', html: '#5a6fd6', css: '#5a6fd6', vue: '#5a6fd6',
}

function normalizedExtension(file: TrashedFile): string {
  return (file.extension || '').replace(/^\./, '').toLowerCase()
}

function fileBadgeColor(file: TrashedFile): string {
  return FILE_BADGE_COLORS[normalizedExtension(file)] || '#767676'
}

function fileBadgeLabel(file: TrashedFile): string {
  const ext = normalizedExtension(file)
  return (ext || 'file').slice(0, 4).toUpperCase()
}

const loading = ref(true);
const error = ref('');
const folders = ref<TrashedFolder[]>([]);
const files = ref<TrashedFile[]>([]);
const emptying = ref(false);
const retentionDays = ref(0);
const page = ref(1);
const limit = ref(50);
const totalFiles = ref(0);
const totalFolders = ref(0);
const currentPageCount = computed(() => folders.value.length + files.value.length);
const totalItemsCount = computed(() => totalFiles.value + totalFolders.value);
const sortKey = ref<'name' | 'size' | 'date'>('date');
const sortDir = ref<'asc' | 'desc'>('desc');
const selectedFolders = ref<Set<string>>(new Set());
const selectedFiles = ref<Set<string>>(new Set());
const lastFolderIndex = ref<number | null>(null);
const lastFileIndex = ref<number | null>(null);

const hasSelection = computed(
  () => selectedFolders.value.size > 0 || selectedFiles.value.size > 0
);

const allSelected = computed(() => {
  const totalItems = folders.value.length + files.value.length;
  if (totalItems === 0) return false;
  return selectedFolders.value.size === folders.value.length
    && selectedFiles.value.size === files.value.length;
});

function selectFolderRange(start: number, end: number) {
  const [from, to] = start < end ? [start, end] : [end, start];
  for (let i = from; i <= to; i++) {
    const folder = folders.value[i];
    if (!folder) continue;
    selectedFolders.value.add(folder.uuid);
  }
}

function selectFileRange(start: number, end: number) {
  const [from, to] = start < end ? [start, end] : [end, start];
  for (let i = from; i <= to; i++) {
    const file = files.value[i];
    if (!file) continue;
    selectedFiles.value.add(file.uuid);
  }
}

function onFolderCheckboxClick(folderId: string, index: number, event: MouseEvent) {
  if (event.shiftKey && lastFolderIndex.value !== null) {
    selectFolderRange(lastFolderIndex.value, index);
  } else if (selectedFolders.value.has(folderId)) {
    selectedFolders.value.delete(folderId);
  } else {
    selectedFolders.value.add(folderId);
  }

  lastFolderIndex.value = index;
}

function onFileCheckboxClick(fileId: string, index: number, event: MouseEvent) {
  if (event.shiftKey && lastFileIndex.value !== null) {
    selectFileRange(lastFileIndex.value, index);
  } else if (selectedFiles.value.has(fileId)) {
    selectedFiles.value.delete(fileId);
  } else {
    selectedFiles.value.add(fileId);
  }

  lastFileIndex.value = index;
}

function toggleSelectAll() {
  if (allSelected.value) {
    clearSelection();
  } else {
    selectedFolders.value = new Set(folders.value.map(f => f.uuid));
    selectedFiles.value = new Set(files.value.map(f => f.uuid));
  }
}

function clearSelection() {
  selectedFolders.value = new Set();
  selectedFiles.value = new Set();
  lastFolderIndex.value = null;
  lastFileIndex.value = null;
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' });
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

async function loadTrash() {
  loading.value = true;
  const params = new URLSearchParams({
    page: String(page.value),
    limit: String(limit.value),
    sort: sortDir.value,
    sortBy: sortKey.value,
  });
  const response = await api({ url: `v1/trash?${params.toString()}`, method: 'GET' });
  loading.value = false;

  if (response.ok && response.body) {
    folders.value = response.body.folders || [];
    files.value = response.body.files || [];
    retentionDays.value = response.body.retentionDays || 0;
    page.value = response.body.page || 1;
    limit.value = response.body.limit || limit.value;
    totalFiles.value = response.body.totalFiles || 0;
    totalFolders.value = response.body.totalFolders || 0;
  } else {
    error.value = response.body?.error || 'Failed to load trash';
  }
}

function changePage(newPage: number) {
  page.value = newPage;
  clearSelection();
  loadTrash();
}

// Applies the totals from a restore response, and if that emptied the
// current page (and it isn't the first page), steps back a page and
// reloads — otherwise the pagination controls would either show a stale
// page count or strand the user on a now-empty page with no way back.
function applyRestoreTotals(body: { totalFiles?: number; totalFolders?: number }) {
  if (typeof body.totalFiles === 'number') totalFiles.value = body.totalFiles;
  if (typeof body.totalFolders === 'number') totalFolders.value = body.totalFolders;

  if (folders.value.length === 0 && files.value.length === 0 && page.value > 1) {
    page.value -= 1;
    loadTrash();
  }
}

async function restoreFolder(folderId: string) {
  const response = await api({ url: `v1/folder/${folderId}/restore`, method: 'PATCH' });
  if (response.ok) {
    folders.value = folders.value.filter(f => f.uuid !== folderId);
    applyRestoreTotals(response.body || {});
  } else {
    error.value = response.body?.error || 'Failed to restore folder';
  }
}

async function restoreFile(fileId: string) {
  const response = await api({ url: `v1/file/${fileId}/restore`, method: 'PATCH' });
  if (response.ok) {
    files.value = files.value.filter(f => f.uuid !== fileId);
    applyRestoreTotals(response.body || {});
  } else {
    error.value = response.body?.error || 'Failed to restore file';
  }
}

async function bulkRestore() {
  const fileIds = Array.from(selectedFiles.value);
  const folderIds = Array.from(selectedFolders.value);

  const response = await api({
    url: 'v1/files/bulk-restore',
    method: 'PATCH',
    json: { fileIds, folderIds },
  });

  if (response.ok) {
    folders.value = folders.value.filter(f => !folderIds.includes(f.uuid));
    files.value = files.value.filter(f => !fileIds.includes(f.uuid));
    applyRestoreTotals(response.body || {});
  } else {
    error.value = response.body?.error || 'Failed to restore selected items';
  }

  clearSelection();
}

async function confirmPurgeFolder(folder: TrashedFolder) {
  if (!window.confirm(`Permanently delete "${folder.name}" and everything inside it? This can't be undone.`)) return;

  const response = await api({ url: `v1/folder/${folder.uuid}/purge`, method: 'DELETE' });
  if (response.ok) {
    folders.value = folders.value.filter(f => f.uuid !== folder.uuid);
  } else {
    error.value = response.body?.error || 'Failed to permanently delete folder';
  }
}

async function confirmPurgeFile(file: TrashedFile) {
  if (!window.confirm(`Permanently delete "${file.name}"? This can't be undone.`)) return;

  const response = await api({ url: `v1/file/${file.uuid}/purge`, method: 'DELETE' });
  if (response.ok) {
    files.value = files.value.filter(f => f.uuid !== file.uuid);
  } else {
    error.value = response.body?.error || 'Failed to permanently delete file';
  }
}

async function confirmEmptyTrash() {
  if (!window.confirm('Permanently delete everything in the trash? This can\'t be undone.')) return;

  emptying.value = true;
  const response = await api({ url: 'v1/trash/empty', method: 'POST' });
  emptying.value = false;

  if (response.ok) {
    folders.value = [];
    files.value = [];
  } else {
    error.value = response.body?.error || 'Failed to empty trash';
  }
}

watch([sortKey, sortDir], () => {
  page.value = 1;
  clearSelection();
  loadTrash();
});

onMounted(loadTrash);
</script>

<style scoped>
.file-icon-badge {
  flex-shrink: 0;
  display: inline-flex;
}

.file-icon-svg {
  width: 2em;
  height: 2em;
}

.sort-select {
  height: 36px;
  background-color: var(--gray-4);
  color: var(--text);
  border: none;
  border-radius: 6px;
  padding: 0 0.5rem;
  font-size: 0.85rem;
  cursor: pointer;
}

.sort-dir-button {
  height: 36px;
  width: 36px;
  background-color: var(--gray-4);
  color: var(--text);
  border-radius: 6px;
  cursor: pointer;
}

.sort-dir-button:hover {
  background-color: var(--gray-5);
}

.bulk-actions-bar {
  background-color: var(--gray-2);
  border: 1px solid var(--gray-4);
}

.modal-secondary-button {
  background-color: var(--gray-4);
  color: var(--text);
}
</style>
