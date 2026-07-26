<template>
  <div class="app-page gap-5">
    <div class="flex items-center justify-between mb-4 w-full">
      <h1 class="text-center flex-1 text-2xl font-bold">Trash</h1>

      <AppButton
        v-if="items.length > 0"
        @click="confirmEmptyTrash"
        variant="danger"
        size="sm"
        :disabled="emptying"
      >
        {{ emptying ? 'Emptying…' : 'Empty Trash' }}
      </AppButton>
    </div>

    <div v-if="!loading && items.length > 0" class="toolbar-controls flex items-center gap-2 mb-2 flex-wrap" style="max-width: 1000px; width: 100%; justify-content: space-between;">
      <div class="flex items-center gap-3">
        <label class="flex items-center gap-2 text-sm select-none cursor-pointer" style="color: var(--text-secondary);">
          <input type="checkbox" :checked="allSelected" @change="toggleSelectAll" />
          Select all
        </label>

        <p class="item-count-label text-sm" style="color: var(--text-secondary);">
          Showing {{ currentPageCount }} of {{ totalItemsCount }} items
        </p>
      </div>

    </div>

    <!-- Bulk Actions -->
    <div
      v-if="hasSelection"
      class="bulk-actions-bar flex items-center gap-3 mb-2 p-3 rounded"
      style="max-width: 1000px; width: 100%;"
    >
      <span class="text-sm" style="color: var(--text-secondary);">
        Selected:
        {{ selectedFolders.size }} folders,
        {{ selectedFiles.size }} files
      </span>

      <AppButton size="sm" @click="bulkRestore">
        Restore Selected
      </AppButton>

      <AppButton variant="secondary" size="sm" @click="clearSelection">
        Clear
      </AppButton>
    </div>

    <div v-if="loading" class="flex flex-col items-center gap-3">
      <SpinnerView />
      <p>Loading trash…</p>
    </div>

    <ErrorMessage :msg="error" @clear="error = ''" />

    <p
      v-if="!loading && retentionDays > 0 && items.length > 0"
      class="text-sm text-center"
      style="color: var(--text-secondary); max-width: 1000px;"
    >
      Items in the trash are automatically deleted forever after {{ retentionDays }} day{{ retentionDays === 1 ? '' : 's' }}.
    </p>

    <div v-if="!loading" class="w-full flex flex-col gap-6" style="max-width: 1000px;">
      <!-- Empty state -->
      <div v-if="items.length === 0" class="card text-center py-10 flex flex-col items-center gap-3">
        <span style="font-size: 2.5rem;">🗑️</span>
        <p class="font-semibold">Trash is empty</p>
        <p class="text-sm" style="color: var(--text-secondary);">
          Files and folders you delete from Drive will show up here until they're purged.
        </p>
      </div>

      <!-- Items -->
      <div v-else class="flex flex-col gap-3">
        <div class="items-header flex flex-row items-center gap-3 p-3">
          <span class="row-checkbox-spacer"></span>
          <span class="col-icon-spacer"></span>
          <span class="col-header col-name sortable-header" @click="toggleSort('name')">Name{{ sortIndicator('name') }}</span>
          <span class="col-header col-size sortable-header" @click="toggleSort('size')">Size{{ sortIndicator('size') }}</span>
          <span class="col-header col-modified sortable-header" @click="toggleSort('date')">Deleted{{ sortIndicator('date') }}</span>
          <span class="row-actions-spacer"></span>
        </div>
        <div
          v-for="(item, index) in items"
          :key="item.type + '-' + item.uuid"
          class="card flex flex-row items-center gap-3 p-4"
        >
          <input
            type="checkbox"
            class="row-checkbox"
            :checked="isItemSelected(item)"
            @click.stop="onItemCheckboxClick(item, index, $event)"
          />

          <span v-if="item.type === 'folder'" style="font-size: 1.4rem; flex-shrink: 0;">📁</span>
          <span v-else class="file-icon-badge">
            <svg viewBox="0 0 24 24" class="file-icon-svg">
              <path d="M5 3a1 1 0 0 0-1 1v16a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1V8l-6-6H5z" fill="var(--gray-4)" stroke="var(--gray-5)" stroke-width="1"/>
              <path d="M14 2v5a1 1 0 0 0 1 1h5" fill="none" stroke="var(--gray-5)" stroke-width="1"/>
              <rect x="1.5" y="15" width="19" height="7" rx="1.5" :fill="fileBadgeColor(item)"/>
              <text x="11" y="20.3" font-size="6.2" font-weight="700" fill="#fff" text-anchor="middle" font-family="Inter, sans-serif">{{ fileBadgeLabel(item) }}</text>
            </svg>
          </span>

          <p class="font-medium text-sm truncate flex-1 min-w-0">{{ item.name }}</p>
          <span class="file-size">{{ item.type === 'file' ? formatFileSize(item.file_size) : '—' }}</span>
          <span class="file-modified">{{ formatDate(item.deleted_at) }}</span>

          <AppButton
            @click="item.type === 'folder' ? restoreFolder(item.uuid) : restoreFile(item.uuid)"
            size="sm"
            class="shrink-0"
          >
            Restore
          </AppButton>

          <AppButton
            @click="item.type === 'folder' ? confirmPurgeFolder(item) : confirmPurgeFile(item)"
            variant="danger-subtle"
            size="sm"
            class="shrink-0"
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
          :total="total"
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
import type { FolderItem } from '@/types/folder';
import { confirm } from '@/composables/confirm';

const FILE_BADGE_COLORS: Record<string, string> = {
  pdf: '#d64545',
  doc: '#3b6fd6', docx: '#3b6fd6', txt: '#3b6fd6', md: '#3b6fd6', rtf: '#3b6fd6',
  xls: 'var(--success)', xlsx: 'var(--success)', csv: 'var(--success)',
  ppt: '#d6822f', pptx: '#d6822f',
  zip: '#a67c2e', tar: '#a67c2e', gz: '#a67c2e', rar: '#a67c2e', '7z': '#a67c2e',
  mp4: '#8a4fd6', mov: '#8a4fd6', avi: '#8a4fd6', mkv: '#8a4fd6', webm: '#8a4fd6',
  mp3: '#d6478f', wav: '#d6478f', flac: '#d6478f', ogg: '#d6478f',
  js: '#5a6fd6', ts: '#5a6fd6', py: '#5a6fd6', go: '#5a6fd6', java: '#5a6fd6', c: '#5a6fd6', cpp: '#5a6fd6', json: '#5a6fd6', html: '#5a6fd6', css: '#5a6fd6', vue: '#5a6fd6',
}

function normalizedExtension(file: FolderItem): string {
  return (file.extension || '').replace(/^\./, '').toLowerCase()
}

function fileBadgeColor(file: FolderItem): string {
  return FILE_BADGE_COLORS[normalizedExtension(file)] || '#767676'
}

function fileBadgeLabel(file: FolderItem): string {
  const ext = normalizedExtension(file)
  return (ext || 'file').slice(0, 4).toUpperCase()
}

const loading = ref(true);
const error = ref('');
const items = ref<FolderItem[]>([]);
const emptying = ref(false);
const retentionDays = ref(0);
const page = ref(1);
const limit = ref(50);
const total = ref(0);
const currentPageCount = computed(() => items.value.length);
const totalItemsCount = computed(() => total.value);
type SortKey = 'name' | 'size' | 'date'
const sortKey = ref<SortKey>('date');
const sortDir = ref<'asc' | 'desc'>('desc');

function toggleSort(column: SortKey) {
  if (sortKey.value === column) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortKey.value = column
    sortDir.value = 'asc'
  }
}

function sortIndicator(column: SortKey) {
  if (sortKey.value !== column) return ''
  return sortDir.value === 'asc' ? ' ↑' : ' ↓'
}
const selectedFolders = ref<Set<string>>(new Set());
const selectedFiles = ref<Set<string>>(new Set());
const lastItemIndex = ref<number | null>(null);

const hasSelection = computed(
  () => selectedFolders.value.size > 0 || selectedFiles.value.size > 0
);

function isItemSelected(item: FolderItem): boolean {
  return item.type === 'folder' ? selectedFolders.value.has(item.uuid) : selectedFiles.value.has(item.uuid);
}

const allSelected = computed(() => {
  if (items.value.length === 0) return false;
  return items.value.every(isItemSelected);
});

function selectItemRange(start: number, end: number) {
  const [from, to] = start < end ? [start, end] : [end, start];
  for (let i = from; i <= to; i++) {
    const item = items.value[i];
    if (!item) continue;
    if (item.type === 'folder') selectedFolders.value.add(item.uuid);
    else selectedFiles.value.add(item.uuid);
  }
}

function onItemCheckboxClick(item: FolderItem, index: number, event: MouseEvent) {
  if (event.shiftKey && lastItemIndex.value !== null) {
    selectItemRange(lastItemIndex.value, index);
  } else {
    const set = item.type === 'folder' ? selectedFolders.value : selectedFiles.value;
    if (set.has(item.uuid)) {
      set.delete(item.uuid);
    } else {
      set.add(item.uuid);
    }
  }

  lastItemIndex.value = index;
}

function toggleSelectAll() {
  if (allSelected.value) {
    clearSelection();
  } else {
    selectedFolders.value = new Set(items.value.filter(i => i.type === 'folder').map(i => i.uuid));
    selectedFiles.value = new Set(items.value.filter(i => i.type === 'file').map(i => i.uuid));
  }
}

function clearSelection() {
  selectedFolders.value = new Set();
  selectedFiles.value = new Set();
  lastItemIndex.value = null;
}

function formatDate(iso: string | undefined): string {
  if (!iso) return '';
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
    items.value = response.body.items || [];
    retentionDays.value = response.body.retentionDays || 0;
    page.value = response.body.page || 1;
    limit.value = response.body.limit || limit.value;
    total.value = response.body.total || 0;
  } else {
    error.value = response.body?.error || 'Failed to load trash';
  }
}

function changePage(newPage: number) {
  page.value = newPage;
  clearSelection();
  loadTrash();
}

// Applies the total from a restore response, and if that emptied the
// current page (and it isn't the first page), steps back a page and
// reloads — otherwise the pagination controls would either show a stale
// page count or strand the user on a now-empty page with no way back.
function applyRestoreTotal(body: { total?: number }) {
  if (typeof body.total === 'number') total.value = body.total;

  if (items.value.length === 0 && page.value > 1) {
    page.value -= 1;
    loadTrash();
  }
}

async function restoreFolder(folderId: string) {
  const response = await api({ url: `v1/folder/${folderId}/restore`, method: 'PATCH' });
  if (response.ok) {
    items.value = items.value.filter(i => !(i.type === 'folder' && i.uuid === folderId));
    applyRestoreTotal(response.body || {});
  } else {
    error.value = response.body?.error || 'Failed to restore folder';
  }
}

async function restoreFile(fileId: string) {
  const response = await api({ url: `v1/file/${fileId}/restore`, method: 'PATCH' });
  if (response.ok) {
    items.value = items.value.filter(i => !(i.type === 'file' && i.uuid === fileId));
    applyRestoreTotal(response.body || {});
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
    clearSelection();
    await loadTrash();
    if (items.value.length === 0 && page.value > 1) {
      page.value -= 1;
      await loadTrash();
    }
  } else {
    error.value = response.body?.error || 'Failed to restore selected items';
    clearSelection();
  }
}

async function confirmPurgeFolder(folder: FolderItem) {
  const ok = await confirm({
    title: 'Delete folder forever?',
    message: `Permanently delete "${folder.name}" and everything inside it? This can't be undone.`,
    confirmLabel: 'Delete Forever',
    danger: true,
  });
  if (!ok) return;

  const response = await api({ url: `v1/folder/${folder.uuid}/purge`, method: 'DELETE' });
  if (response.ok) {
    items.value = items.value.filter(i => !(i.type === 'folder' && i.uuid === folder.uuid));
  } else {
    error.value = response.body?.error || 'Failed to permanently delete folder';
  }
}

async function confirmPurgeFile(file: FolderItem) {
  const ok = await confirm({
    title: 'Delete file forever?',
    message: `Permanently delete "${file.name}"? This can't be undone.`,
    confirmLabel: 'Delete Forever',
    danger: true,
  });
  if (!ok) return;

  const response = await api({ url: `v1/file/${file.uuid}/purge`, method: 'DELETE' });
  if (response.ok) {
    items.value = items.value.filter(i => !(i.type === 'file' && i.uuid === file.uuid));
  } else {
    error.value = response.body?.error || 'Failed to permanently delete file';
  }
}

async function confirmEmptyTrash() {
  const ok = await confirm({
    title: 'Empty trash?',
    message: 'Permanently delete everything in the trash? This can\'t be undone.',
    confirmLabel: 'Empty Trash',
    danger: true,
  });
  if (!ok) return;

  emptying.value = true;
  const response = await api({ url: 'v1/trash/empty', method: 'POST' });
  emptying.value = false;

  if (response.ok) {
    items.value = [];
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
.row-checkbox {
  width: 1.25em;
  height: 1.25em;
  flex-shrink: 0;
  cursor: pointer;
}

.file-icon-badge {
  flex-shrink: 0;
  display: inline-flex;
}

.file-icon-svg {
  width: 2em;
  height: 2em;
}

.file-size {
  color: var(--text-secondary, #666);
  font-size: 0.85em;
  white-space: nowrap;
  width: 90px;
  flex-shrink: 0;
  text-align: right;
}

.file-modified {
  color: var(--text-secondary, #666);
  font-size: 0.85em;
  white-space: nowrap;
  width: 130px;
  flex-shrink: 0;
  text-align: right;
}

.items-header {
  width: 100%;
  font-size: 0.85em;
  font-weight: 600;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--gray-4);
}

.row-checkbox-spacer {
  width: 1.25em;
  flex-shrink: 0;
}

.col-icon-spacer {
  width: 2em;
  flex-shrink: 0;
}

.row-actions-spacer {
  width: 12em;
  flex-shrink: 0;
}

.col-name {
  flex: 1;
  min-width: 0;
}

.col-size {
  width: 90px;
  flex-shrink: 0;
  text-align: right;
}

.col-modified {
  width: 130px;
  flex-shrink: 0;
  text-align: right;
}

.sortable-header {
  cursor: pointer;
  user-select: none;
  white-space: nowrap;
}

.sortable-header:hover {
  color: var(--text-primary, inherit);
}

.bulk-actions-bar {
  background-color: var(--gray-2);
  border: 1px solid var(--gray-4);
}

</style>
