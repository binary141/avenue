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
      class="fixed inset-0 flex items-center justify-center"
    >
      <div class="bg-white p-6 rounded-lg w-96">
        <h2 class="text-lg font-semibold mb-4 text-gray-700 ">Create Folder</h2>

        <div
          class="space-y-4 w-80 bg-white text-gray-700 p-6 rounded-lg shadow"
        >
          <div>
            <label class="block text-sm font-medium mb-1 text-gray-600">Folder Name</label>
            <input
              ref="inputRef"
              v-model="folderName"
              type="text"
              class="w-full border border-gray-300 rounded px-3 py-2 text-gray-700 bg-white"
            />
          </div>

          <div class="flex justify-end gap-3 mt-6">
            <AppButton
              @click="show = false"
              class="px-3 py-2 bg-gray-200 text-gray-700 rounded"
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
      class="flex items-center gap-3 mb-4 p-3 rounded bg-blue-50 border border-blue-200"
    >
      <span class="text-sm text-gray-700">
        Selected:
        {{ selectedFolders.size }} folders,
        {{ selectedFiles.size }} files
      </span>

      <AppButton class="bg-red-600 text-white px-3 py-1" @click="bulkDelete">
        Delete
      </AppButton>

      <AppButton class="bg-gray-200 px-3 py-1" @click="clearSelection">
        Clear
      </AppButton>
    </div>

    <div v-if="loading" class="flex flex-col align-center content-center gap-3">
      <SpinnerView />
      <p>Loading folder contents...</p>
    </div>

    <ErrorMessage :msg=error @clear="error = ''"/>

    <BreadCrumbs :breadcrumbs=breadcrumbs>A</BreadCrumbs>


    <div v-if="!loading" class="folder-contents flex flex-col gap-4">
      <!-- Folders Section -->
      <div v-if="folders.length > 0" class="folders-section">
        <h2>Folders</h2>
        <div class="items-list flex flex-col gap-2">
          <div
            v-for="(folder, index) in folders"
            :key="folder.uuid"
            class="folder-item card flex flex-row items-center gap-3 p-3"
          >
            <!-- Checkbox -->
            <input
              type="checkbox"
              :checked="selectedFolders.has(folder.uuid)"
              @click.stop="onFolderCheckboxClick(folder.uuid, index, $event)"
            />

            <!-- Folder clickable name -->
            <span
              class="folder-info flex-1 flex items-center gap-2 cursor-pointer"
              @click="changeFolder(folder.uuid)"
            >
              <span class="folder-icon">📁</span>
              <span class="folder-name">{{ folder.name }}</span>
            </span>

            <!-- Actions -->
            <span class="folder-actions flex items-center gap-2">
              <span class="file-edit cursor-pointer" @click.stop="openFolderEditModal(folder)">✏️</span>
              <span class="file-delete cursor-pointer" @click.stop="deleteFolder(folder.uuid)">🗑️</span>
              <span class="cursor-pointer inline-flex items-center gap-0.5" @click.stop="openFolderShareModal(folder)" title="Share folder">
                🔗<span v-if="sharedFolderCounts[folder.uuid]" class="share-count">{{ sharedFolderCounts[folder.uuid] }}</span>
              </span>
            </span>
          </div>
        </div>
      </div>

      <!-- Files Section -->
      <div v-if="files.length > 0" class="files-section">
        <h2>Files</h2>
        <div class="items-list flex flex-col gap-2">
          <div
            v-for="(file, index) in files"
            :key="file.uuid"
            class="file-item card flex flex-row items-center gap-3 p-3"
          >
            <!-- Checkbox -->
            <input
              type="checkbox"
              :checked="selectedFiles.has(file.uuid)"
              @click.stop="onFileCheckboxClick(file.uuid, index, $event)"
            />

            <span class="file-icon">📄</span>
            <span class="file-name">{{ formatFileName(file.name) }}</span>
            <span class="file-size">{{ formatFileSize(file.file_size) }}</span>
            <span class="file-extension">{{ file.extension }}</span>

            <span class="file-edit cursor-pointer" @click.stop="openFileEditModal(file)">✏️</span>
            <span class="file-delete cursor-pointer" @click.stop="deleteFile(file.uuid)">🗑️</span>
            <span class="file-download">
              <a @click.stop :href="getDownloadURL(file.uuid)" :download="file.name">⬇️</a>
            </span>
            <span class="cursor-pointer inline-flex items-center gap-0.5" @click.stop="openShareModal(file)" title="Share">
              🔗<span v-if="sharedFileCounts[file.uuid]" class="share-count">{{ sharedFileCounts[file.uuid] }}</span>
            </span>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-if="!loading && folders.length === 0 && files.length === 0" class="empty-state">
        <p>This folder is empty.</p>
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
      <div class="bg-white rounded shadow-lg w-[520px] p-6 relative flex flex-col">
        <h3 class="text-lg font-bold mb-1 text-black">Share "{{ sharingFile.name }}"</h3>
        <p class="text-sm text-gray-500 mb-4">Anyone with the link can view and download this file.</p>

        <!-- Loading existing links -->
        <div v-if="sharesLoading" class="flex justify-center py-6">
          <svg class="animate-spin h-6 w-6 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z"/>
          </svg>
        </div>

        <template v-else>
          <!-- Active links list -->
          <div class="mb-4">
            <p class="text-sm font-semibold text-gray-600 mb-2">
              Active links<span v-if="shareLinks.length"> ({{ shareLinks.length }})</span>
            </p>
            <div v-if="shareLinks.length === 0" class="text-sm text-gray-400 py-2 text-center">
              No active links for this file.
            </div>
            <div v-else class="flex flex-col gap-2">
              <div
                v-for="link in shareLinks"
                :key="link.token"
                class="flex items-center gap-2 bg-gray-50 border border-gray-200 rounded px-3 py-2 text-sm"
              >
                <span class="flex-1 font-mono text-xs text-gray-600 truncate">{{ shareLinkURL(link.token) }}</span>
                <span class="text-xs text-gray-400 whitespace-nowrap shrink-0">
                  {{ link.expires_at ? formatExpiry(link.expires_at) : 'Never expires' }}
                </span>
                <AppButton @click="copyShareToken(link.token)" class="px-2 py-1 bg-blue-600 text-white text-xs rounded shrink-0">
                  {{ shareTokenCopied[link.token] ? '✓' : 'Copy' }}
                </AppButton>
                <AppButton @click="revokeShareLink(link.token)" class="px-2 py-1 bg-red-100 text-red-600 text-xs rounded shrink-0">
                  Revoke
                </AppButton>
              </div>
            </div>
          </div>

          <hr class="mb-4 border-gray-200"/>

          <!-- New link -->
          <p class="text-sm font-semibold text-gray-600 mb-2">Create new link</p>
          <div class="mb-4">
            <label class="block text-xs font-medium text-gray-500 mb-1">Expires (optional)</label>
            <input
              v-model="shareExpiresAt"
              type="datetime-local"
              class="border border-gray-300 rounded px-3 py-2 w-full text-gray-700 bg-white text-sm"
            />
          </div>

          <div class="mb-4 flex items-center gap-2">
            <input
              id="shareRequireLogin"
              v-model="shareRequireLogin"
              type="checkbox"
              class="rounded"
            />
            <label for="shareRequireLogin" class="text-sm text-gray-600 select-none cursor-pointer">Require login to access</label>
          </div>

          <div class="flex justify-end gap-2">
            <AppButton @click="closeShareModal" class="px-3 py-2 bg-gray-200 text-gray-700 rounded text-sm">Close</AppButton>
            <AppButton @click="generateShareLink" :disabled="shareGenerating" class="px-3 py-2 bg-blue-600 text-white rounded text-sm">
              {{ shareGenerating ? 'Generating…' : 'Generate Link' }}
            </AppButton>
          </div>
        </template>

        <button @click="closeShareModal" class="absolute top-2 right-2 text-gray-500 hover:text-gray-700">✕</button>
      </div>
      </div>
    </div>

    <!-- Share Folder Modal -->
    <div
      v-if="sharingFolder"
      class="fixed inset-0 overflow-y-auto bg-black/50 z-50"
    >
      <div class="flex min-h-full items-center justify-center p-4">
      <div class="bg-white rounded shadow-lg w-[520px] p-6 relative flex flex-col">
        <h3 class="text-lg font-bold mb-1 text-black">Share "{{ sharingFolder.name }}"</h3>
        <p class="text-sm text-gray-500 mb-4">Anyone with the link can browse and download all files in this folder.</p>

        <div v-if="folderSharesLoading" class="flex justify-center py-6">
          <svg class="animate-spin h-6 w-6 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z"/>
          </svg>
        </div>

        <template v-else>
          <div class="mb-4">
            <p class="text-sm font-semibold text-gray-600 mb-2">
              Active links<span v-if="folderShareLinks.length"> ({{ folderShareLinks.length }})</span>
            </p>
            <div v-if="folderShareLinks.length === 0" class="text-sm text-gray-400 py-2 text-center">
              No active links for this folder.
            </div>
            <div v-else class="flex flex-col gap-2">
              <div
                v-for="link in folderShareLinks"
                :key="link.token"
                class="flex items-center gap-2 bg-gray-50 border border-gray-200 rounded px-3 py-2 text-sm"
              >
                <span class="flex-1 font-mono text-xs text-gray-600 truncate">{{ folderShareLinkURL(link.token) }}</span>
                <span class="text-xs text-gray-400 whitespace-nowrap shrink-0">
                  {{ link.expires_at ? formatExpiry(link.expires_at) : 'Never expires' }}
                </span>
                <AppButton @click="copyFolderShareToken(link.token)" class="px-2 py-1 bg-blue-600 text-white text-xs rounded shrink-0">
                  {{ folderShareTokenCopied[link.token] ? '✓' : 'Copy' }}
                </AppButton>
                <AppButton @click="revokeFolderShareLink(link.token)" class="px-2 py-1 bg-red-100 text-red-600 text-xs rounded shrink-0">
                  Revoke
                </AppButton>
              </div>
            </div>
          </div>

          <hr class="mb-4 border-gray-200"/>

          <p class="text-sm font-semibold text-gray-600 mb-2">Create new link</p>
          <div class="mb-4">
            <label class="block text-xs font-medium text-gray-500 mb-1">Expires (optional)</label>
            <input
              v-model="folderShareExpiresAt"
              type="datetime-local"
              class="border border-gray-300 rounded px-3 py-2 w-full text-gray-700 bg-white text-sm"
            />
          </div>

          <div class="mb-4 flex items-center gap-2">
            <input
              id="folderShareRequireLogin"
              v-model="folderShareRequireLogin"
              type="checkbox"
              class="rounded"
            />
            <label for="folderShareRequireLogin" class="text-sm text-gray-600 select-none cursor-pointer">Require login to access</label>
          </div>

          <div class="mb-4 flex items-center gap-2">
            <input
              id="folderShareAllowUpload"
              v-model="folderShareAllowUpload"
              type="checkbox"
              class="rounded"
            />
            <label for="folderShareAllowUpload" class="text-sm text-gray-600 select-none cursor-pointer">Allow file uploads</label>
          </div>

          <div v-if="folderShareAllowUpload" class="mb-4">
            <label class="block text-xs font-medium text-gray-500 mb-1">Max upload size in MB (0 = server default)</label>
            <input
              v-model.number="folderShareMaxFileSizeMB"
              type="number"
              min="0"
              placeholder="0"
              class="border border-gray-300 rounded px-3 py-2 w-full text-gray-700 bg-white text-sm"
            />
          </div>

          <div class="flex justify-end gap-2">
            <AppButton @click="closeFolderShareModal" class="px-3 py-2 bg-gray-200 text-gray-700 rounded text-sm">Close</AppButton>
            <AppButton @click="generateFolderShareLink" :disabled="folderShareGenerating" class="px-3 py-2 bg-blue-600 text-white rounded text-sm">
              {{ folderShareGenerating ? 'Generating…' : 'Generate Link' }}
            </AppButton>
          </div>
        </template>

        <button @click="closeFolderShareModal" class="absolute top-2 right-2 text-gray-500 hover:text-gray-700">✕</button>
      </div>
      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
import AppButton from './components/AppButton.vue'
import BreadCrumbs from './components/BreadCrumbs.vue'
import { ref, onMounted, watchEffect, watch, nextTick, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import api from '@/utils/api';
import type { Breadcrumb, Folder, File, FolderContents } from '@/types/folder';
import SpinnerView from './components/SpinnerView.vue';
import ErrorMessage from './components/ErrorMessage.vue';
import FileUploader from '@/components/FileUploader.vue';
import FileUsageBar from '@/components/FileUsageBar.vue';
import { useUsersStore } from '../stores/users';

const route = useRoute();
const emit = defineEmits(['close-menu']);
const router = useRouter();
const loading = ref(false);
const serverMax = ref(0);
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
const editingFile = ref<File | null>(null);
const newFileName = ref('');

const editingFolder = ref<Folder | null>(null);
const newFolderName = ref('');

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
}

onMounted(() => {
  refreshCurrentList();
  getDashboardInfo();
  loadAllShares();
  loadAllFolderShares();
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

.items-list {
  width: 100%;
}

.folder-info {
  flex: 1;
}

.folder-actions {
  display: flex;
  gap: 0.5rem;
}

.folder-item,
.file-item {
  cursor: pointer;
  transition: background-color 0.2s;
}

.folder-item:hover,
.file-item:hover {
  background-color: var(--background-hover, rgba(0, 0, 0, 0.05));
}

.folder-icon,
.file-icon {
  font-size: 1.5em;
}

.folder-name,
.file-name {
  flex: 1;
  font-weight: 500;
}

.file-size {
  color: var(--text-secondary, #666);
  font-size: 0.9em;
  text-transform: uppercase;
}

.file-extension {
  color: var(--text-secondary, #666);
  font-size: 0.9em;
  text-transform: uppercase;
}

.empty-state {
  text-align: center;
  padding: 2rem;
  color: var(--text-secondary, #666);
}

.modal {
  z-index: 50;
}

.modal-content {
  max-width: 400px;
}

.share-count {
  font-size: 0.65rem;
  font-weight: 700;
  color: #3A3F78;
  line-height: 1;
}
</style>

