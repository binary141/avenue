<template>
  <div class="page gap-5">
    <div class="flex items-center justify-between mb-4 w-full">
      <h1 class="text-center flex-1 text-2xl font-bold">Drive</h1>

      <AppButton
        @click="() => { show = true; $emit('close-menu')}"
        class="px-4 py-2 bg-blue-600 text-white rounded"
      >
        Create Folder
      </AppButton>
    </div>

    <!-- create folder dialog -->
    <div
      v-if="show"
      class="fixed inset-0 flex items-center justify-center bg-black/50 z-50"
    >
      <div class="share-modal p-6 rounded-lg w-96 shadow-lg">
        <h2 class="text-lg font-semibold mb-4">Create Folder</h2>

        <div>
          <label class="block text-sm font-medium mb-1 share-modal-subtext">Folder Name</label>
          <input
            ref="inputRef"
            v-model="folderName"
            type="text"
            class="w-full"
            @keyup.enter="createFolder"
          />
        </div>

        <div class="flex justify-end gap-3 mt-6">
          <AppButton
            @click="show = false"
            class="px-3 py-2 modal-secondary-button rounded"
          >
            Cancel
          </AppButton>

          <AppButton
            type="submit"
            @click="createFolder"
            class="px-3 py-2 bg-blue-600 text-white rounded"
          >
            Create
          </AppButton>
        </div>
      </div>
    </div>

    <FileUploader
      :parent="currentFolderId"
      @upload="refreshCurrentList"
      @error="handleUploadError"
      :multiple=true
      :maxSize=maxFileSize
      :disabled="usersStore.userData.data.quota !== 0 && usersStore.userData.data.spaceUsed >= usersStore.userData.data.quota" />

    <FileUsageBar
      :used="usersStore.userData.data.spaceUsed"
      :quota="usersStore.userData.data.quota"
    />

    <!-- Bulk Actions -->
    <div
      v-if="hasSelection"
      class="bulk-actions-bar flex items-center gap-3 mb-4 p-3 rounded"
    >
      <span class="text-sm share-modal-subtext">
        Selected:
        {{ selectedFolders.size }} folders,
        {{ selectedFiles.size }} files
      </span>

      <AppButton
        class="bg-blue-600 text-white px-3 py-1"
        :disabled="selectedFiles.size === 0"
        :title="selectedFolders.size > 0 ? 'Only selected files will be moved — folders can\'t be bulk-moved yet' : ''"
        @click="openBulkMoveModal"
      >
        Move
      </AppButton>

      <AppButton class="bg-red-600 text-white px-3 py-1" @click="bulkDelete">
        Delete
      </AppButton>

      <AppButton class="modal-secondary-button px-3 py-1" @click="clearSelection">
        Clear
      </AppButton>
    </div>

    <div v-if="loading" class="flex flex-col align-center content-center gap-3">
      <SpinnerView />
      <p>Loading folder contents...</p>
    </div>

    <ErrorMessage :msg=error @clear="error = ''"/>

    <div class="toolbar flex items-center justify-between gap-3 w-full flex-wrap">
      <BreadCrumbs :breadcrumbs=breadcrumbs />

      <div class="toolbar-controls flex items-center gap-2">
        <div class="search-input-wrap">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search this folder…"
            class="search-input"
          />
          <button
            v-if="searchQuery"
            type="button"
            class="search-clear-button"
            title="Clear search"
            @click="searchQuery = ''"
          >
            ✕
          </button>
        </div>
        <select v-model="sortKey" class="sort-select">
          <option value="name">Name</option>
          <option value="size">Size</option>
          <option value="date">Date</option>
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

    <div v-if="!loading" class="folder-contents flex flex-col gap-4">
      <!-- Folders Section -->
      <div v-if="filteredFolders.length > 0" class="folders-section">
        <h2>Folders</h2>
        <div class="items-list flex flex-col gap-2">
          <div
            v-for="(folder, index) in filteredFolders"
            :key="folder.uuid"
            class="folder-item card flex flex-row items-center gap-3 p-3"
            :class="{ 'item--selected': selectedFolders.has(folder.uuid) }"
          >
            <!-- Checkbox -->
            <input
              type="checkbox"
              :checked="selectedFolders.has(folder.uuid)"
              @click.stop="onFolderCheckboxClick(folder.uuid, index, $event)"
            />

            <!-- Folder clickable name -->
            <span
              class="folder-info flex-1 flex items-center gap-2 cursor-pointer min-w-0"
              @click="changeFolder(folder.uuid)"
            >
              <span class="folder-icon">📁</span>
              <span class="folder-name">{{ folder.name }}</span>
              <span v-if="sharedFolderCounts[folder.uuid]" class="shared-badge" title="Shared">
                🔗 {{ sharedFolderCounts[folder.uuid] }}
              </span>
            </span>

            <!-- Actions -->
            <span class="row-actions flex items-center gap-1">
              <span class="row-menu-wrap">
                <span class="icon-btn" title="More" @click.stop="toggleRowMenu('folder-' + folder.uuid)">⋮</span>
                <div v-if="openMenuId === 'folder-' + folder.uuid" class="row-menu" @click.stop>
                  <button class="row-menu-item" @click="openFolderEditModal(folder); openMenuId = null">Rename</button>
                  <button v-if="folderSharingEnabled" class="row-menu-item" @click="openFolderShareModal(folder); openMenuId = null">Share</button>
                  <button class="row-menu-item row-menu-item--danger" @click="deleteFolder(folder.uuid); openMenuId = null">Delete</button>
                </div>
              </span>
            </span>
          </div>
        </div>
      </div>

      <!-- Files Section -->
      <div v-if="filteredFiles.length > 0" class="files-section">
        <h2>Files</h2>
        <div class="items-list flex flex-col gap-2">
          <div
            v-for="(file, index) in filteredFiles"
            :key="file.uuid"
            class="file-item card flex flex-row items-center gap-3 p-3"
            :class="{ 'item--selected': selectedFiles.has(file.uuid) }"
          >
            <!-- Checkbox -->
            <input
              type="checkbox"
              :checked="selectedFiles.has(file.uuid)"
              @click.stop="onFileCheckboxClick(file.uuid, index, $event)"
            />

            <img
              v-if="isImageFile(file)"
              :src="getDownloadURL(file.uuid)"
              class="file-thumb cursor-pointer"
              alt=""
              @click.stop="openFileViewer(file)"
            />
            <span v-else class="file-icon-badge cursor-pointer" @click.stop="openFileViewer(file)">
              <svg viewBox="0 0 24 24" class="file-icon-svg">
                <path d="M5 3a1 1 0 0 0-1 1v16a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1V8l-6-6H5z" fill="var(--gray-4)" stroke="var(--gray-5)" stroke-width="1"/>
                <path d="M14 2v5a1 1 0 0 0 1 1h5" fill="none" stroke="var(--gray-5)" stroke-width="1"/>
                <rect x="1.5" y="15" width="19" height="7" rx="1.5" :fill="fileBadgeColor(file)"/>
                <text x="11" y="20.3" font-size="6.2" font-weight="700" fill="#fff" text-anchor="middle" font-family="Inter, sans-serif">{{ fileBadgeLabel(file) }}</text>
              </svg>
            </span>

            <span class="file-name cursor-pointer flex-1 min-w-0" @click.stop="openFileViewer(file)">
              {{ formatFileName(file.name) }}
              <span v-if="sharedFileCounts[file.uuid]" class="shared-badge" title="Shared">🔗 {{ sharedFileCounts[file.uuid] }}</span>
            </span>
            <span class="file-size">{{ formatFileSize(file.file_size) }}</span>

            <span class="row-actions flex items-center gap-1">
              <span class="icon-btn" title="Download">
                <a @click.stop :href="getDownloadURL(file.uuid)" :download="file.name">⬇️</a>
              </span>

              <span class="row-menu-wrap">
                <span class="icon-btn" title="More" @click.stop="toggleRowMenu('file-' + file.uuid)">⋮</span>
                <div v-if="openMenuId === 'file-' + file.uuid" class="row-menu" @click.stop>
                  <button class="row-menu-item" @click="openFileEditModal(file); openMenuId = null">Rename</button>
                  <button class="row-menu-item" @click="openMoveModal(file); openMenuId = null">Move to…</button>
                  <button v-if="fileSharingEnabled" class="row-menu-item" @click="openShareModal(file); openMenuId = null">Share</button>
                  <button class="row-menu-item row-menu-item--danger" @click="deleteFile(file.uuid); openMenuId = null">Delete</button>
                </div>
              </span>
            </span>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-if="!loading && folders.length === 0 && files.length === 0" class="empty-state">
        <span class="empty-state-icon">🗂️</span>
        <p class="empty-state-title">This folder is empty</p>
        <p class="empty-state-hint">Drag files in above, or click the upload area to get started.</p>
      </div>

      <!-- Searching -->
      <div v-else-if="!loading && searching" class="empty-state">
        <SpinnerView />
      </div>

      <!-- No Search Results -->
      <div v-else-if="!loading && (folders.length > 0 || files.length > 0) && filteredFolders.length === 0 && filteredFiles.length === 0" class="empty-state">
        <span class="empty-state-icon">🔍</span>
        <p class="empty-state-title">No matches for "{{ searchQuery }}"</p>
      </div>
    </div>

    <!-- Rename File Modal -->
    <div
      v-if="editingFile"
      class="fixed inset-0 flex items-center justify-center bg-black/50 z-50"
    >
      <!-- Modal content -->
      <div class="bg-white rounded shadow-lg w-96 p-6 relative">
        <h3 class="text-lg font-bold mb-4 text-black">Rename File</h3>
        <input
          v-model="newFileName"
          class="border p-2 w-full mb-4 text-black"
          placeholder="Enter new name"
        />
        <div class="flex justify-end gap-2">
          <AppButton @click="closeFileModal">Cancel</AppButton>
          <AppButton @click="saveFileName">Save</AppButton>
        </div>
        <!-- Optional: Close button in corner -->
        <button
          @click="closeFileModal"
          class="absolute top-2 right-2 text-gray-500 hover:text-gray-700"
        >
          ✕
        </button>
      </div>
    </div>

    <!-- Rename Folder Modal -->
    <div
      v-if="editingFolder"
      class="fixed inset-0 flex items-center justify-center bg-black/50 z-50"
    >
      <!-- Modal content -->
      <div class="bg-white rounded shadow-lg w-96 p-6 relative">
        <h3 class="text-lg font-bold mb-4 text-black">Rename Folder</h3>
        <input
          v-model="newFolderName"
          class="border p-2 w-full mb-4 text-black"
          placeholder="Enter new name"
        />
        <div class="flex justify-end gap-2">
          <AppButton @click="closeFolderModal">Cancel</AppButton>
          <AppButton @click="saveFolderName">Save</AppButton>
        </div>
        <!-- Optional: Close button in corner -->
        <button
          @click="closeFolderModal"
          class="absolute top-2 right-2 text-gray-500 hover:text-gray-700"
        >
          ✕
        </button>
      </div>
    </div>

    <!-- Share File Modal -->
    <div
      v-if="sharingFile"
      class="fixed inset-0 overflow-y-auto bg-black/50 z-50"
    >
      <div class="flex min-h-full items-center justify-center p-4">
      <div class="share-modal rounded shadow-lg w-[520px] p-6 relative flex flex-col">
        <h3 class="text-lg font-bold mb-1">Share "{{ sharingFile.name }}"</h3>
        <p class="text-sm share-modal-subtext mb-4">Anyone with the link can view and download this file.</p>

        <!-- Loading existing links -->
        <div v-if="sharesLoading" class="flex justify-center py-6">
          <svg class="animate-spin h-6 w-6" style="color: var(--primary-active);" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z"/>
          </svg>
        </div>

        <template v-else>
          <!-- Active links list -->
          <div class="mb-4">
            <p class="text-sm font-semibold share-modal-subtext mb-2">
              Active links<span v-if="shareLinks.length"> ({{ shareLinks.length }})</span>
            </p>
            <div v-if="shareLinks.length === 0" class="text-sm share-modal-muted py-2 text-center">
              No active links for this file.
            </div>
            <div v-else class="flex flex-col gap-2">
              <div
                v-for="link in shareLinks"
                :key="link.token"
                class="share-link-row flex items-center gap-2 rounded px-3 py-2 text-sm"
              >
                <span class="flex-1 font-mono text-xs share-modal-subtext truncate">{{ shareLinkURL(link.token) }}</span>
                <span class="text-xs share-modal-muted whitespace-nowrap shrink-0">
                  {{ link.expires_at ? formatExpiry(link.expires_at) : 'Never expires' }}
                </span>
                <AppButton @click="copyShareToken(link.token)" class="px-2 py-1 bg-blue-600 text-white text-xs rounded shrink-0">
                  {{ shareTokenCopied[link.token] ? '✓' : 'Copy' }}
                </AppButton>
                <AppButton @click="revokeShareLink(link.token)" class="px-2 py-1 revoke-button text-xs rounded shrink-0">
                  Revoke
                </AppButton>
              </div>
            </div>
          </div>

          <hr class="mb-4 share-modal-divider"/>

          <!-- New link -->
          <p class="text-sm font-semibold share-modal-subtext mb-2">Create new link</p>
          <div class="mb-4">
            <label class="block text-xs font-medium share-modal-muted mb-1">Expires (optional)</label>
            <input
              v-model="shareExpiresAt"
              type="datetime-local"
              class="w-full text-sm"
            />
          </div>

          <div class="mb-4 flex items-center gap-2">
            <input
              id="shareRequireLogin"
              v-model="shareRequireLogin"
              type="checkbox"
              class="rounded"
            />
            <label for="shareRequireLogin" class="text-sm share-modal-subtext select-none cursor-pointer">Require login to access</label>
          </div>

          <div class="flex justify-end gap-2">
            <AppButton @click="closeShareModal" class="px-3 py-2 modal-secondary-button rounded text-sm">Close</AppButton>
            <AppButton @click="generateShareLink" :disabled="shareGenerating" class="px-3 py-2 bg-blue-600 text-white rounded text-sm">
              {{ shareGenerating ? 'Generating…' : 'Generate Link' }}
            </AppButton>
          </div>
        </template>

        <button @click="closeShareModal" class="absolute top-2 right-2 share-modal-muted">✕</button>
      </div>
      </div>
    </div>

    <!-- Share Folder Modal -->
    <div
      v-if="sharingFolder"
      class="fixed inset-0 overflow-y-auto bg-black/50 z-50"
    >
      <div class="flex min-h-full items-center justify-center p-4">
      <div class="share-modal rounded shadow-lg w-[520px] p-6 relative flex flex-col">
        <h3 class="text-lg font-bold mb-1">Share "{{ sharingFolder.name }}"</h3>
        <p class="text-sm share-modal-subtext mb-4">Anyone with the link can browse and download all files in this folder.</p>

        <div v-if="folderSharesLoading" class="flex justify-center py-6">
          <svg class="animate-spin h-6 w-6" style="color: var(--primary-active);" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z"/>
          </svg>
        </div>

        <template v-else>
          <div class="mb-4">
            <p class="text-sm font-semibold share-modal-subtext mb-2">
              Active links<span v-if="folderShareLinks.length"> ({{ folderShareLinks.length }})</span>
            </p>
            <div v-if="folderShareLinks.length === 0" class="text-sm share-modal-muted py-2 text-center">
              No active links for this folder.
            </div>
            <div v-else class="flex flex-col gap-2">
              <div
                v-for="link in folderShareLinks"
                :key="link.token"
                class="share-link-row flex items-center gap-2 rounded px-3 py-2 text-sm"
              >
                <span class="flex-1 font-mono text-xs share-modal-subtext truncate">{{ folderShareLinkURL(link.token) }}</span>
                <span class="text-xs share-modal-muted whitespace-nowrap shrink-0">
                  {{ link.expires_at ? formatExpiry(link.expires_at) : 'Never expires' }}
                </span>
                <AppButton @click="copyFolderShareToken(link.token)" class="px-2 py-1 bg-blue-600 text-white text-xs rounded shrink-0">
                  {{ folderShareTokenCopied[link.token] ? '✓' : 'Copy' }}
                </AppButton>
                <AppButton @click="revokeFolderShareLink(link.token)" class="px-2 py-1 revoke-button text-xs rounded shrink-0">
                  Revoke
                </AppButton>
              </div>
            </div>
          </div>

          <hr class="mb-4 share-modal-divider"/>

          <p class="text-sm font-semibold share-modal-subtext mb-2">Create new link</p>
          <div class="mb-4">
            <label class="block text-xs font-medium share-modal-muted mb-1">Expires (optional)</label>
            <input
              v-model="folderShareExpiresAt"
              type="datetime-local"
              class="w-full text-sm"
            />
          </div>

          <div class="mb-4 flex items-center gap-2">
            <input
              id="folderShareRequireLogin"
              v-model="folderShareRequireLogin"
              type="checkbox"
              class="rounded"
            />
            <label for="folderShareRequireLogin" class="text-sm share-modal-subtext select-none cursor-pointer">Require login to access</label>
          </div>

          <div class="mb-4 flex items-center gap-2">
            <input
              id="folderShareAllowUpload"
              v-model="folderShareAllowUpload"
              type="checkbox"
              class="rounded"
            />
            <label for="folderShareAllowUpload" class="text-sm share-modal-subtext select-none cursor-pointer">Allow file uploads</label>
          </div>

          <div v-if="folderShareAllowUpload" class="mb-4">
            <label class="block text-xs font-medium share-modal-muted mb-1">Max upload size in MB (0 = server default)</label>
            <input
              v-model.number="folderShareMaxFileSizeMB"
              type="number"
              min="0"
              placeholder="0"
              class="w-full text-sm"
            />
          </div>

          <div class="flex justify-end gap-2">
            <AppButton @click="closeFolderShareModal" class="px-3 py-2 modal-secondary-button rounded text-sm">Close</AppButton>
            <AppButton @click="generateFolderShareLink" :disabled="folderShareGenerating" class="px-3 py-2 bg-blue-600 text-white rounded text-sm">
              {{ folderShareGenerating ? 'Generating…' : 'Generate Link' }}
            </AppButton>
          </div>
        </template>

        <button @click="closeFolderShareModal" class="absolute top-2 right-2 share-modal-muted">✕</button>
      </div>
      </div>
    </div>

    <!-- Move File Modal -->
    <div
      v-if="movingFileIds.length > 0"
      class="fixed inset-0 flex items-center justify-center bg-black/50 z-50"
    >
      <div class="share-modal rounded-lg w-96 p-6 relative shadow-lg flex flex-col">
        <h3 class="text-lg font-bold mb-1">
          {{ movingSingleFileName ? `Move "${movingSingleFileName}"` : `Move ${movingFileIds.length} files` }}
        </h3>
        <p class="text-sm share-modal-subtext mb-3">Choose a destination folder.</p>

        <div class="move-picker-breadcrumbs flex items-center flex-wrap gap-1 mb-2 text-sm">
          <button class="move-picker-crumb" @click="loadMoveFolder('')">Home</button>
          <template v-for="bc in moveBreadcrumbs" :key="bc.folder_id">
            <span class="share-modal-muted">/</span>
            <button class="move-picker-crumb" @click="loadMoveFolder(bc.folder_id)">{{ bc.label }}</button>
          </template>
        </div>

        <div class="move-picker-list">
          <div v-if="moveLoading" class="flex justify-center py-6">
            <SpinnerView />
          </div>
          <template v-else>
            <div v-if="moveFolders.length === 0" class="text-sm share-modal-muted text-center py-4">
              No subfolders here.
            </div>
            <button
              v-for="folder in moveFolders"
              :key="folder.uuid"
              class="move-picker-item"
              @click="loadMoveFolder(folder.uuid)"
            >
              📁 {{ folder.name }}
            </button>
          </template>
        </div>

        <div class="flex justify-end gap-2 mt-4">
          <AppButton @click="closeMoveModal" class="px-3 py-2 modal-secondary-button rounded text-sm">Cancel</AppButton>
          <AppButton @click="confirmMove" :disabled="moving || moveLoading" class="px-3 py-2 bg-blue-600 text-white rounded text-sm">
            {{ moving ? 'Moving…' : 'Move Here' }}
          </AppButton>
        </div>

        <button @click="closeMoveModal" class="absolute top-2 right-2 share-modal-muted">✕</button>
      </div>
    </div>

    <!-- File Viewer -->
    <FileViewer
      v-if="viewingFile"
      :file="viewingFile"
      :download-url="getDownloadURL(viewingFile.uuid)"
      @close="closeFileViewer"
    />

  </div>
</template>

<script setup lang="ts">
import AppButton from './components/AppButton.vue'
import BreadCrumbs from './components/BreadCrumbs.vue'
import { ref, onMounted, onUnmounted, watchEffect, watch, nextTick, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import api from '@/utils/api';
import type { Breadcrumb, Folder, File, FolderContents } from '@/types/folder';
import SpinnerView from './components/SpinnerView.vue';
import ErrorMessage from './components/ErrorMessage.vue';
import FileUploader from '@/components/FileUploader.vue';
import FileUsageBar from '@/components/FileUsageBar.vue';
import FileViewer from '@/components/FileViewer.vue';
import { useUsersStore } from '../stores/users';

const route = useRoute();
const emit = defineEmits(['close-menu']);
const router = useRouter();
const loading = ref(false);
const serverMax = ref(0);
const fileSharingEnabled = computed(() => usersStore.fileSharingEnabled);
const folderSharingEnabled = computed(() => usersStore.folderSharingEnabled);
const maxFileSize = computed(() => {
  if (serverMax.value === 0) return 0;
  const { quota, spaceUsed } = usersStore.userData.data;
  if (quota === 0) return serverMax.value;
  const remaining = quota - spaceUsed;
  return remaining < serverMax.value ? remaining : serverMax.value;
});
const error = ref<string | undefined>();
const folders = ref<Folder[]>([]);
const files = ref<File[]>([]);
const breadcrumbs = ref<Breadcrumb[]>([]);
const show = ref<boolean>(false);
const inputRef = ref<HTMLInputElement | null>(null);
const currentFolderId = ref<string>('');
const usersStore = useUsersStore();

// ----- Modal State -----
const viewingFile = ref<File | null>(null);

function openFileViewer(file: File) {
  viewingFile.value = file;
  router.push({ query: { ...route.query, preview: file.uuid } });
}

function closeFileViewer() {
  router.back();
}

watch(
  () => route.query.preview,
  (previewId) => {
    if (!previewId) {
      viewingFile.value = null;
    } else if (previewId !== viewingFile.value?.uuid) {
      viewingFile.value = files.value.find(f => f.uuid === previewId) ?? null;
    }
  }
);

const editingFile = ref<File | null>(null);
const newFileName = ref('');

const editingFolder = ref<Folder | null>(null);
const newFolderName = ref('');

// ----- Move File State -----
const movingFileIds = ref<string[]>([]);
const movingSingleFileName = ref<string | null>(null);
const moveFolderId = ref('');
const moveFolders = ref<Folder[]>([]);
const moveBreadcrumbs = ref<Breadcrumb[]>([]);
const moveLoading = ref(false);
const moving = ref(false);

async function loadMoveFolder(folderId: string) {
  moveLoading.value = true;
  const response = await api({ url: `v1/folder/list/${folderId}`, method: 'GET' });
  moveLoading.value = false;

  if (response.ok && response.body) {
    moveFolderId.value = folderId;
    moveFolders.value = response.body.folders || [];
    moveBreadcrumbs.value = response.body.breadcrumbs || [];
  } else {
    error.value = response.body?.error || 'Failed to load folders';
  }
}

function openMoveModal(file: File) {
  movingFileIds.value = [file.uuid];
  movingSingleFileName.value = file.name;
  loadMoveFolder(currentFolderId.value);
}

function openBulkMoveModal() {
  if (selectedFiles.value.size === 0) return;
  movingFileIds.value = Array.from(selectedFiles.value);
  movingSingleFileName.value = null;
  loadMoveFolder(currentFolderId.value);
}

function closeMoveModal() {
  movingFileIds.value = [];
  movingSingleFileName.value = null;
  moveFolders.value = [];
  moveBreadcrumbs.value = [];
}

async function confirmMove() {
  if (movingFileIds.value.length === 0) return;
  moving.value = true;

  let failures = 0;
  for (const fileId of movingFileIds.value) {
    const response = await api({
      url: `v1/file/${fileId}/move`,
      method: 'PATCH',
      json: { parent: moveFolderId.value },
    });
    if (!response.ok) failures++;
  }

  moving.value = false;

  if (failures > 0) {
    error.value = `Failed to move ${failures} of ${movingFileIds.value.length} file(s)`;
  }

  clearSelection();
  closeMoveModal();
  refreshCurrentList();
}

// ----- Share State -----
interface ShareLinkItem {
  token: string;
  expires_at: string | null;
  created_at: string;
}

const sharingFile = ref<File | null>(null);
const shareExpiresAt = ref('');
const shareRequireLogin = ref(false);
const shareLinks = ref<ShareLinkItem[]>([]);
const sharesLoading = ref(false);
const shareGenerating = ref(false);
const shareTokenCopied = ref<Record<string, boolean>>({});

// file uuid -> count of active links (loaded once on mount)
const sharedFileCounts = ref<Record<string, number>>({});

// ----- Folder Share State -----
const sharingFolder = ref<Folder | null>(null);
const folderShareExpiresAt = ref('');
const folderShareRequireLogin = ref(false);
const folderShareAllowUpload = ref(false);
const folderShareMaxFileSizeMB = ref(0);
const folderShareLinks = ref<ShareLinkItem[]>([]);
const folderSharesLoading = ref(false);
const folderShareGenerating = ref(false);
const folderShareTokenCopied = ref<Record<string, boolean>>({});

// folder uuid -> count of active folder share links (loaded once on mount)
const sharedFolderCounts = ref<Record<string, number>>({});

const folderName = ref('');

// ----- Search / Sort -----
const searchQuery = ref('')
const sortKey = ref<'name' | 'size' | 'date'>('name')
const sortDir = ref<'asc' | 'desc'>('asc')

// Folder search stays client-side (folders aren't covered by the search API).
// File search hits the backend, which does a `name LIKE 'query%'` prefix match.
const fileSearchResults = ref<File[] | null>(null)
const searching = ref(false)
let searchDebounceHandle: ReturnType<typeof setTimeout> | null = null

async function runFileSearch(query: string, folderId: string) {
  searching.value = true
  const url = folderId
    ? `v1/folder/${folderId}/files/${encodeURIComponent(query)}`
    : `v1/folder/files/${encodeURIComponent(query)}`

  const response = await api({ url, method: 'GET' })
  searching.value = false
  fileSearchResults.value = response.ok && Array.isArray(response.body) ? response.body : []
}

watch([searchQuery, currentFolderId], ([query, folderId]) => {
  if (searchDebounceHandle) clearTimeout(searchDebounceHandle)

  const trimmed = query.trim()
  if (!trimmed) {
    fileSearchResults.value = null
    return
  }

  searchDebounceHandle = setTimeout(() => runFileSearch(trimmed, folderId), 250)
})

const filteredFolders = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  const list = q ? folders.value.filter(f => f.name.toLowerCase().includes(q)) : folders.value.slice()
  return list.sort((a, b) => a.name.localeCompare(b.name) * (sortDir.value === 'asc' ? 1 : -1))
})

const filteredFiles = computed(() => {
  const isSearching = searchQuery.value.trim().length > 0
  const list = (isSearching ? fileSearchResults.value ?? [] : files.value).slice()

  return list.sort((a, b) => {
    let cmp = 0
    if (sortKey.value === 'name') cmp = a.name.localeCompare(b.name)
    else if (sortKey.value === 'size') cmp = a.file_size - b.file_size
    else cmp = new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
    return cmp * (sortDir.value === 'asc' ? 1 : -1)
  })
})

// ----- Row "more" menu -----
const openMenuId = ref<string | null>(null)

function toggleRowMenu(id: string) {
  openMenuId.value = openMenuId.value === id ? null : id
}

function closeRowMenu() {
  openMenuId.value = null
}

// ----- File type badge -----
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

function fileBadgeColor(file: File): string {
  return FILE_BADGE_COLORS[normalizedExtension(file)] || '#767676'
}

function fileBadgeLabel(file: File): string {
  const ext = normalizedExtension(file)
  return (ext || 'file').slice(0, 4).toUpperCase()
}

const selectedFolders = ref<Set<string>>(new Set())
const selectedFiles = ref<Set<string>>(new Set())

const lastFolderIndex = ref<number | null>(null)
const lastFileIndex = ref<number | null>(null)

function selectFolderRange(start: number, end: number) {
  const list = folders.value.slice()
  const [from, to] = start < end ? [start, end] : [end, start]

  for (let i = from; i <= to; i++) {
    const folder = list[i]
    if (!folder) continue
    selectedFolders.value.add(folder.uuid)
  }
}

function selectFileRange(start: number, end: number) {
  const list = files.value.slice() // snapshot
  const [from, to] = start < end ? [start, end] : [end, start]

  for (let i = from; i <= to; i++) {
    const file = list[i]
    if (!file) continue
    selectedFiles.value.add(file.uuid)
  }
}

function onFolderCheckboxClick(
  folderId: string,
  index: number,
  event: MouseEvent
) {
  if (event.shiftKey && lastFolderIndex.value !== null) {
    selectFolderRange(lastFolderIndex.value, index)
  } else {
    if (selectedFolders.value.has(folderId)) {
      selectedFolders.value.delete(folderId)
    } else {
      selectedFolders.value.add(folderId)
    }
  }

  lastFolderIndex.value = index
}

function onFileCheckboxClick(
  fileId: string,
  index: number,
  event: MouseEvent
) {
  if (event.shiftKey && lastFileIndex.value !== null) {
    selectFileRange(lastFileIndex.value, index)
  } else {
    if (selectedFiles.value.has(fileId)) {
      selectedFiles.value.delete(fileId)
    } else {
      selectedFiles.value.add(fileId)
    }
  }

  lastFileIndex.value = index
}

const hasSelection = computed(
  () => selectedFolders.value.size > 0 || selectedFiles.value.size > 0
)

function clearSelection() {
  selectedFolders.value.clear()
  selectedFiles.value.clear()
}

async function bulkDelete() {
  // Delete files
  for (const fileId of selectedFiles.value) {
    await deleteFile(fileId)
  }

  // Delete folders
  for (const folderId of selectedFolders.value) {
    await deleteFolder(folderId)
  }

  clearSelection()
  refreshCurrentList()
}


watch(show, async (open) => {
  if (open) {
    await nextTick()
    inputRef.value?.focus()
  }
})

async function createFolder() {
  const folderId = currentFolderId.value

  await api({
    url: "v1/folder",
    method: "POST",
    json: {
      name: folderName.value,
      parent: folderId,
    }
  });

  refreshCurrentList();

  folderName.value = '';
  show.value = false;
}

function changeFolder(folderId: string) {
  router.push({ path: '/', query: { folderId: folderId }})
}

function getDownloadURL(fileId: string): string {
  const baseURL = import.meta.env.VITE_APP_API_URL || '';

  return `${baseURL}v1/file/${fileId}?token=${usersStore.token}`
}

// ----- File Operations -----
async function deleteFile(fileId: string) {
  await api({ url: "v1/file/" + fileId, method: "DELETE" })
  refreshCurrentList();
}

async function deleteFolder(folderId: string) {
  const response = await api({ url: "v1/folder/" + folderId, method: "DELETE" });

  if (response.status > 399) {
    if (response.body.error) {
      error.value = response.body.error
    }
  }

  refreshCurrentList();
}

function openFileEditModal(file: File) {
  editingFile.value = file
  newFileName.value = file.name
}

function openFolderEditModal(folder: Folder) {
  editingFolder.value = folder
  newFolderName.value = folder.name
}

function closeFileModal() {
  editingFile.value = null
  newFileName.value = ""
}

function closeFolderModal() {
  editingFolder.value = null
  newFolderName.value = ""
}

function shareLinkURL(token: string): string {
  return `${window.location.origin}/share/${token}`
}

function formatExpiry(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' })
}

async function loadAllShares() {
  const response = await api({ url: 'v1/shares', method: 'GET' })
  if (response.ok && Array.isArray(response.body)) {
    const counts: Record<string, number> = {}
    for (const link of response.body) {
      counts[link.file_id] = (counts[link.file_id] || 0) + 1
    }
    sharedFileCounts.value = counts
  }
}

async function loadAllFolderShares() {
  const response = await api({ url: 'v1/folder-shares', method: 'GET' })
  if (response.ok && Array.isArray(response.body)) {
    const counts: Record<string, number> = {}
    for (const link of response.body) {
      counts[link.folder_uuid] = (counts[link.folder_uuid] || 0) + 1
    }
    sharedFolderCounts.value = counts
  }
}

async function loadFolderShares(folderUUID: string) {
  folderSharesLoading.value = true
  const response = await api({ url: `v1/folder/${folderUUID}/shares`, method: 'GET' })
  folderSharesLoading.value = false
  if (response.ok && Array.isArray(response.body)) {
    folderShareLinks.value = response.body
  }
}

function openFolderShareModal(folder: Folder) {
  sharingFolder.value = folder
  folderShareExpiresAt.value = ''
  folderShareRequireLogin.value = false
  folderShareAllowUpload.value = false
  folderShareMaxFileSizeMB.value = 0
  folderShareLinks.value = []
  folderShareTokenCopied.value = {}
  loadFolderShares(folder.uuid)
}

function closeFolderShareModal() {
  sharingFolder.value = null
  folderShareLinks.value = []
  folderShareExpiresAt.value = ''
  folderShareRequireLogin.value = false
  folderShareAllowUpload.value = false
  folderShareMaxFileSizeMB.value = 0
  folderShareTokenCopied.value = {}
}

function folderShareLinkURL(token: string): string {
  return `${window.location.origin}/share/folder/${token}`
}

async function generateFolderShareLink() {
  if (!sharingFolder.value) return
  folderShareGenerating.value = true

  const body: { expires_at?: string; require_login?: boolean; allow_upload?: boolean; max_file_size?: number } = {}
  if (folderShareExpiresAt.value) {
    body.expires_at = new Date(folderShareExpiresAt.value).toISOString()
  }
  if (folderShareRequireLogin.value) {
    body.require_login = true
  }
  if (folderShareAllowUpload.value) {
    body.allow_upload = true
  }
  if (folderShareAllowUpload.value && folderShareMaxFileSizeMB.value > 0) {
    body.max_file_size = folderShareMaxFileSizeMB.value * 1024 * 1024
  }

  const response = await api({
    url: `v1/folder/${sharingFolder.value.uuid}/share`,
    method: 'POST',
    json: body,
  })

  folderShareGenerating.value = false

  if (response.ok && response.body?.token) {
    folderShareLinks.value.unshift({
      token: response.body.token,
      expires_at: response.body.expires_at ?? null,
      created_at: response.body.created_at,
    })
    folderShareExpiresAt.value = ''
    const folderUUID = sharingFolder.value.uuid
    sharedFolderCounts.value[folderUUID] = (sharedFolderCounts.value[folderUUID] || 0) + 1
  } else {
    error.value = response.body?.error || 'Failed to create share link'
  }
}

async function revokeFolderShareLink(token: string) {
  const response = await api({ url: `v1/share/folder/${token}`, method: 'DELETE' })
  if (response.ok) {
    folderShareLinks.value = folderShareLinks.value.filter(l => l.token !== token)
    if (sharingFolder.value) {
      const folderUUID = sharingFolder.value.uuid
      const current = sharedFolderCounts.value[folderUUID] || 0
      if (current <= 1) {
        delete sharedFolderCounts.value[folderUUID]
      } else {
        sharedFolderCounts.value[folderUUID] = current - 1
      }
    }
  } else {
    error.value = response.body?.error || 'Failed to revoke share link'
  }
}

async function copyFolderShareToken(token: string) {
  await navigator.clipboard.writeText(folderShareLinkURL(token))
  folderShareTokenCopied.value = { ...folderShareTokenCopied.value, [token]: true }
  setTimeout(() => {
    folderShareTokenCopied.value = { ...folderShareTokenCopied.value, [token]: false }
  }, 2000)
}

async function loadFileShares(fileId: string) {
  sharesLoading.value = true
  const response = await api({ url: `v1/file/${fileId}/shares`, method: 'GET' })
  sharesLoading.value = false
  if (response.ok && Array.isArray(response.body)) {
    shareLinks.value = response.body
  }
}

function openShareModal(file: File) {
  sharingFile.value = file
  shareExpiresAt.value = ''
  shareRequireLogin.value = false
  shareLinks.value = []
  shareTokenCopied.value = {}
  loadFileShares(file.uuid)
}

function closeShareModal() {
  sharingFile.value = null
  shareLinks.value = []
  shareExpiresAt.value = ''
  shareRequireLogin.value = false
  shareTokenCopied.value = {}
}

async function generateShareLink() {
  if (!sharingFile.value) return
  shareGenerating.value = true

  const body: { expires_at?: string; require_login?: boolean } = {}
  if (shareExpiresAt.value) {
    body.expires_at = new Date(shareExpiresAt.value).toISOString()
  }
  if (shareRequireLogin.value) {
    body.require_login = true
  }

  const response = await api({
    url: `v1/file/${sharingFile.value.uuid}/share`,
    method: 'POST',
    json: body,
  })

  shareGenerating.value = false

  if (response.ok && response.body?.token) {
    shareLinks.value.unshift({
      token: response.body.token,
      expires_at: response.body.expires_at ?? null,
      created_at: response.body.created_at,
    })
    shareExpiresAt.value = ''
    const fileId = sharingFile.value.uuid
    sharedFileCounts.value[fileId] = (sharedFileCounts.value[fileId] || 0) + 1
  } else {
    error.value = response.body?.error || 'Failed to create share link'
  }
}

async function revokeShareLink(token: string) {
  const response = await api({ url: `v1/share/${token}`, method: 'DELETE' })
  if (response.ok) {
    shareLinks.value = shareLinks.value.filter(l => l.token !== token)
    if (sharingFile.value) {
      const fileId = sharingFile.value.uuid
      const current = sharedFileCounts.value[fileId] || 0
      if (current <= 1) {
        delete sharedFileCounts.value[fileId]
      } else {
        sharedFileCounts.value[fileId] = current - 1
      }
    }
  } else {
    error.value = response.body?.error || 'Failed to revoke share link'
  }
}

async function copyShareToken(token: string) {
  await navigator.clipboard.writeText(shareLinkURL(token))
  shareTokenCopied.value = { ...shareTokenCopied.value, [token]: true }
  setTimeout(() => {
    shareTokenCopied.value = { ...shareTokenCopied.value, [token]: false }
  }, 2000)
}

async function saveFileName() {
  if (!editingFile.value) return
  // Call API to update the file name on the server
  const response = await api({
    url: `v1/file/${editingFile.value.uuid}/${newFileName.value}`,
    method: 'PATCH',
    json: { name: newFileName.value }
  })

  if (response.ok) {
    editingFile.value.name = newFileName.value
    closeFileModal()
  } else {
    console.error("Failed to rename file", response)
  }
}

async function saveFolderName() {
  if (!editingFolder.value) return
  // Call API to update the file name on the server
  const response = await api({
    url: `v1/folder/${editingFolder.value.uuid}/${newFolderName.value}`,
    method: 'PATCH',
    json: { name: newFolderName.value }
  })

  if (response.ok) {
    editingFolder.value.name = newFolderName.value
    closeFolderModal()
  } else {
    console.error("Failed to rename folder", response)
  }
}

function formatFileName(fileName: string): string {
  const maxNameLength = 35
  return fileName.length > maxNameLength ? fileName.substring(0, maxNameLength) + "..." : fileName
}

const IMAGE_EXTENSIONS = new Set(['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp'])

function normalizedExtension(file: File): string {
  return (file.extension || '').replace(/^\./, '').toLowerCase()
}

function isImageFile(file: File): boolean {
  return IMAGE_EXTENSIONS.has(normalizedExtension(file))
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

// ----- Folder Contents -----
async function loadFolderContents(folderId: string = '') {
  loading.value = true

  try {
    const response = await api({
      url: `v1/folder/list/${folderId}`,
      method: 'GET'
    })

    if (response.ok && response.body) {
      const contents = response.body as FolderContents
      folders.value = contents.folders || []
      files.value = contents.files || []
      breadcrumbs.value = contents.breadcrumbs || []
    } else {
      error.value = response.body?.error || response.body?.message || 'Failed to load folder contents'
    }
  } catch (e) {
    error.value = 'An error occurred while loading folder contents'
    console.error(e)
  } finally {
    loading.value = false
  }
}

function handleUploadError(message: string) {
  error.value = message
}

function refreshCurrentList() {
  emit("close-menu")
  // if we have a folderid in a redirect somewhere, then we need to override our current state
  const folderId = (route.params.folderId as string) || (route.query.folderId as string) || ''

  currentFolderId.value = folderId

  route.params.folderId = currentFolderId.value

  loadFolderContents(currentFolderId.value)

  usersStore.pullMe()
}

watchEffect(() => {
  refreshCurrentList()
})

async function getDashboardInfo() {
  const response = await api({
    url: "v1/dashboard",
    method: "GET",
  });

  if (!response.ok) {
    return;
  }

  // 200MiB
  const defaultSize = 209715200;
  serverMax.value = response.body.maxFileSize || defaultSize;
  usersStore.fileSharingEnabled = response.body.fileSharingEnabled !== false;
  usersStore.folderSharingEnabled = response.body.folderSharingEnabled !== false;
}

function handleDocumentClick() {
  closeRowMenu()
}

onMounted(async () => {
  refreshCurrentList();
  await getDashboardInfo();
  if (fileSharingEnabled.value) {
    loadAllShares();
  }
  if (folderSharingEnabled.value) {
    loadAllFolderShares();
  }
  document.addEventListener('click', handleDocumentClick);
});

onUnmounted(() => {
  document.removeEventListener('click', handleDocumentClick);
});
</script>

<style scoped>
.folder-contents {
  width: 100%;
  max-width: 800px;
}

.folders-section,
.files-section {
  width: 100%;
}

.folders-section h2,
.files-section h2 {
  font-size: 0.8rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-tertiary);
  margin: 0 0 0.5rem 0.25rem;
}

.items-list {
  width: 100%;
}

.folder-info {
  flex: 1;
}

.folder-item,
.file-item {
  cursor: pointer;
  transition: background-color 0.15s, box-shadow 0.15s;
}

.folder-item:hover,
.file-item:hover {
  background-color: var(--gray-3);
}

.item--selected {
  background-color: var(--gray-3);
  box-shadow: inset 0 0 0 1px var(--primary-active);
}

.folder-icon {
  font-size: 1.5em;
}

.file-thumb {
  width: 2em;
  height: 2em;
  object-fit: cover;
  border-radius: 4px;
  flex-shrink: 0;
}

.file-icon-badge {
  flex-shrink: 0;
  display: inline-flex;
}

.file-icon-svg {
  width: 2em;
  height: 2em;
}

.folder-name,
.file-name {
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-size {
  color: var(--text-secondary, #666);
  font-size: 0.85em;
  white-space: nowrap;
}

/* Row actions: quieter by default, fully visible on row hover/focus */
.row-actions {
  opacity: 0.5;
  transition: opacity 0.15s;
  flex-shrink: 0;
}

.folder-item:hover .row-actions,
.file-item:hover .row-actions,
.item--selected .row-actions {
  opacity: 1;
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.9em;
  height: 1.9em;
  border-radius: 6px;
  cursor: pointer;
  position: relative;
}

.icon-btn:hover {
  background-color: var(--gray-4);
}

.icon-btn a {
  display: inline-flex;
}

.row-menu-wrap {
  position: relative;
}

.row-menu {
  position: absolute;
  right: 0;
  top: calc(100% + 4px);
  background-color: var(--gray-2);
  border: 1px solid var(--gray-4);
  border-radius: 8px;
  padding: 4px;
  display: flex;
  flex-direction: column;
  min-width: 120px;
  z-index: 20;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.35);
}

.row-menu-item {
  text-align: left;
  padding: 0.4rem 0.6rem;
  border-radius: 6px;
  color: var(--text);
  font-size: 0.85rem;
  cursor: pointer;
}

.row-menu-item:hover {
  background-color: var(--gray-4);
}

.row-menu-item--danger {
  color: #e57373;
}

.toolbar {
  margin-bottom: -0.5rem;
}

.search-input-wrap {
  position: relative;
  display: inline-flex;
  align-items: center;
}

.search-input {
  height: 36px !important;
  width: 200px;
  font-size: 0.85rem !important;
  padding-right: 28px !important;
}

.search-clear-button {
  position: absolute;
  right: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.4em;
  height: 1.4em;
  border-radius: 999px;
  font-size: 0.7rem;
  color: var(--text-tertiary);
  cursor: pointer;
}

.search-clear-button:hover {
  background-color: var(--gray-4);
  color: var(--text);
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

.empty-state {
  text-align: center;
  padding: 3rem 1rem;
  color: var(--text-secondary, #666);
}

.empty-state-icon {
  font-size: 2.5rem;
  display: block;
  margin-bottom: 0.75rem;
  opacity: 0.8;
}

.empty-state-title {
  font-weight: 600;
  color: var(--text);
  margin: 0 0 0.25rem 0;
}

.empty-state-hint {
  font-size: 0.85rem;
  margin: 0;
}

.modal {
  z-index: 50;
}

.modal-content {
  max-width: 400px;
}

.share-modal {
  background-color: var(--gray-2);
  color: var(--text);
}

.share-modal-subtext {
  color: var(--text-secondary);
}

.share-modal-muted {
  color: var(--text-tertiary);
}

.share-modal-divider {
  border-color: var(--gray-4);
}

.share-link-row {
  background-color: var(--gray);
  border: 1px solid var(--gray-4);
}

.revoke-button {
  background-color: rgba(229, 115, 115, 0.15);
  color: #e57373;
}

.modal-secondary-button {
  background-color: var(--gray-4);
  color: var(--text);
}

.move-picker-breadcrumbs {
  max-height: 3.5rem;
  overflow-y: auto;
}

.move-picker-crumb {
  color: var(--primary-active);
  cursor: pointer;
  padding: 0.1rem 0.25rem;
  border-radius: 4px;
}

.move-picker-crumb:hover {
  background-color: var(--gray-4);
}

.move-picker-list {
  border: 1px solid var(--gray-4);
  border-radius: 8px;
  max-height: 220px;
  overflow-y: auto;
  padding: 4px;
}

.move-picker-item {
  display: block;
  width: 100%;
  text-align: left;
  padding: 0.5rem 0.6rem;
  border-radius: 6px;
  cursor: pointer;
  color: var(--text);
}

.move-picker-item:hover {
  background-color: var(--gray-4);
}

.bulk-actions-bar {
  background-color: var(--gray-2);
  border: 1px solid var(--gray-4);
}

.shared-badge {
  display: inline-flex;
  align-items: center;
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--primary-active);
  background-color: var(--gray-4);
  border-radius: 999px;
  padding: 0.1rem 0.45rem;
  margin-left: 0.4rem;
  vertical-align: middle;
  white-space: nowrap;
}
</style>

