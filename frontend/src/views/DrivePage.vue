<template>
  <div class="page gap-5">
    <h1>Drive</h1>
    <FileUploader :parent="currentFolderId" @upload="handleFileUpload" @error="handleUploadError"
    multiple=true maxSize=10000 />

    <div v-if="loading" class="flex flex-col align-center content-center gap-3">
      <SpinnerView />
      <p>Loading folder contents...</p>
    </div>

    <ErrorMessage v-if="error">{{ error }}</ErrorMessage>

    <div v-if="!loading && !error" class="folder-contents flex flex-col gap-4">
      <!-- Folders Section -->
      <div v-if="folders.length > 0" class="folders-section">
        <h2>Folders</h2>
        <div class="items-list flex flex-col gap-2">
          <div
            v-for="folder in folders"
            :key="folder.folder_id"
            class="folder-item card flex flex-row align-center gap-3 p-3"
          >
            <span class="folder-icon">📁</span>
            <span class="folder-name">{{ folder.name }}</span>
          </div>
        </div>
      </div>

      <!-- Files Section -->
      <div v-if="files.length > 0" class="files-section">
        <h2>Files</h2>
        <div class="items-list flex flex-col gap-2">
          <div
            v-for="file in files"
            :key="file.id"
            class="file-item card flex flex-row align-center gap-3 p-3"
          >
            <span class="file-icon">📄</span>
            <span class="file-name" :title="file.name">{{ formatFileName(file.name) }}</span>
            <span class="file-size">{{ formatFileSize(file.file_size) }}</span>
            <span class="file-extension">{{ file.extension }}</span>
            <span class="file-delete" @click="deleteFile(file.id)">🗑️</span>

            <!-- action buttons -->
            <span class="file-edit cursor-pointer" @click="openEditModal(file)">✏️</span>
            <span class="file-download">
              <a :href="getDownloadURL(file.id)" :download="file.name">⬇️</a>
            </span>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-if="!loading && folders.length === 0 && files.length === 0" class="empty-state">
        <p>This folder is empty.</p>
      </div>
    </div>

    <!-- Rename Modal -->
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
          <AppButton @click="closeModal">Cancel</AppButton>
          <AppButton @click="saveFileName">Save</AppButton>
        </div>
        <!-- Optional: Close button in corner -->
        <button
          @click="closeModal"
          class="absolute top-2 right-2 text-gray-500 hover:text-gray-700"
        >
          ✕
        </button>
      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
import AppButton from './components/AppButton.vue'
import { ref, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import api from '@/utils/api';
import type { Folder, File, FolderContents } from '@/types/folder';
import SpinnerView from './components/SpinnerView.vue';
import ErrorMessage from './components/ErrorMessage.vue';
import FileUploader from '@/components/FileUploader.vue';
import { useUsersStore } from '../stores/users';

const route = useRoute();
const loading = ref(false);
const error = ref<string | undefined>();
const folders = ref<Folder[]>([]);
const files = ref<File[]>([]);
const currentFolderId = ref<string>('');
const usersStore = useUsersStore();

// ----- Modal State -----
const editingFile = ref<File | null>(null)
const newFileName = ref('')

function getDownloadURL(fileId: string): string {
  let baseURL = import.meta.env.VITE_APP_API_URL

  return `${baseURL}v1/file/${fileId}?token=${usersStore.token}`
}

// ----- File Operations -----
async function deleteFile(fileId: string) {
  await api({ url: "v1/file/" + fileId, method: "DELETE" })
  refreshCurrentList();
}

function openEditModal(file: File) {
  editingFile.value = file
  newFileName.value = file.name
}

function closeModal() {
  editingFile.value = null
  newFileName.value = ""
}

async function saveFileName() {
  if (!editingFile.value) return
  // Call API to update the file name on the server
  const response = await api({
    url: `v1/file/${editingFile.value.id}/${newFileName.value}`,
    method: 'PATCH',
    json: { name: newFileName.value }
  })

  if (response.ok) {
    editingFile.value.name = newFileName.value
    closeModal()
  } else {
    console.error("Failed to rename file", response)
  }
}

function formatFileName(fileName: string): string {
  const maxNameLength = 35
  return fileName.length > maxNameLength ? fileName.substring(0, maxNameLength) + "..." : fileName
}

async function download(fileId: string) {
  const response = await api({ url: "v1/file/" + fileId, method: "GET", responseType: "blob" })

  if (response.status === 404 || response.status === 500) {
    console.error("Error downloading file", response)
    return
  }

  const blob = new Blob([response.body.blob])
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url

  let filename = "download.bin"
  const disposition = response.headers["content-disposition"]
  if (disposition) {
    const match = disposition.match(/filename="?([^"]+)"?/)
    if (match) filename = match[1]
  }

  a.download = filename
  a.click()
  window.URL.revokeObjectURL(url)
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
  error.value = undefined

  try {
    const response = await api({
      url: folderId ? `v1/folder/list/${folderId}` : `v1/folder/list/-1`,
      method: 'GET'
    })

    if (response.ok && response.body) {
      const contents = response.body as FolderContents
      folders.value = contents.folders || []
      files.value = contents.files || []
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

function handleFileUpload() {
  loadFolderContents(currentFolderId.value)
}

function handleUploadError(message: string) {
  error.value = message
}

function refreshCurrentList() {
  const folderId = (route.params.folderId as string) || (route.query.folderId as string) || ''
  currentFolderId.value = folderId
  loadFolderContents(folderId)
}

onMounted(() => {
  refreshCurrentList();
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
</style>

