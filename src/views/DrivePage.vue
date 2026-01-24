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
      :disabled="usersStore.userData.data.spaceUsed >= usersStore.userData.data.quota" />

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
            :key="folder.folder_id"
            class="folder-item card flex flex-row items-center gap-3 p-3"
          >
            <!-- Checkbox -->
            <input
              type="checkbox"
              :checked="selectedFolders.has(folder.folder_id)"
              @click.stop="onFolderCheckboxClick(folder.folder_id, index, $event)"
            />

            <!-- Folder clickable name -->
            <span
              class="folder-info flex-1 flex items-center gap-2 cursor-pointer"
              @click="changeFolder(folder.folder_id)"
            >
              <span class="folder-icon">📁</span>
              <span class="folder-name">{{ folder.name }}</span>
            </span>

            <!-- Actions -->
            <span class="folder-actions flex items-center gap-2">
              <span class="file-edit cursor-pointer" @click.stop="openFolderEditModal(folder)">✏️</span>
              <span class="file-delete cursor-pointer" @click.stop="deleteFolder(folder.folder_id)">🗑️</span>
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
            :key="file.id"
            class="file-item card flex flex-row items-center gap-3 p-3"
          >
            <!-- Checkbox -->
            <input
              type="checkbox"
              :checked="selectedFiles.has(file.id)"
              @click.stop="onFileCheckboxClick(file.id, index, $event)"
            />

            <span class="file-icon">📄</span>
            <span class="file-name">{{ formatFileName(file.name) }}</span>
            <span class="file-size">{{ formatFileSize(file.file_size) }}</span>
            <span class="file-extension">{{ file.extension }}</span>

            <span class="file-edit cursor-pointer" @click.stop="openFileEditModal(file)">✏️</span>
            <span class="file-delete cursor-pointer" @click.stop="deleteFile(file.id)">🗑️</span>
            <span class="file-download">
              <a @click.stop :href="getDownloadURL(file.id)" :download="file.name">⬇️</a>
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
const maxFileSize = ref(0);
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
    selectedFolders.value.add(folder.folder_id)
  }
}

function selectFileRange(start: number, end: number) {
  const list = files.value.slice() // snapshot
  const [from, to] = start < end ? [start, end] : [end, start]

  for (let i = from; i <= to; i++) {
    const file = list[i]
    if (!file) continue
    selectedFiles.value.add(file.id)
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
    selectedFolders.value.has(folderId)
      ? selectedFolders.value.delete(folderId)
      : selectedFolders.value.add(folderId)
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
    selectedFiles.value.has(fileId)
      ? selectedFiles.value.delete(fileId)
      : selectedFiles.value.add(fileId)
  }

  lastFileIndex.value = index
}

const hasSelection = computed(
  () => selectedFolders.value.size > 0 || selectedFiles.value.size > 0
)

function toggleFolderSelection(id: string) {
  selectedFolders.value.has(id)
    ? selectedFolders.value.delete(id)
    : selectedFolders.value.add(id)
}

function toggleFileSelection(id: string) {
  selectedFiles.value.has(id)
    ? selectedFiles.value.delete(id)
    : selectedFiles.value.add(id)
}

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
  currentFolderId.value = folderId

  router.push({ path: '/', query: { folderId: folderId }})

  refreshCurrentList();
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
    closeFileModal()
  } else {
    console.error("Failed to rename file", response)
  }
}

async function saveFolderName() {
  if (!editingFolder.value) return
  // Call API to update the file name on the server
  const response = await api({
    url: `v1/folder/${editingFolder.value.folder_id}/${newFolderName.value}`,
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

  maxFileSize.value = defaultSize;

  if (response.body.maxFileSize) {
    maxFileSize.value = response.body.maxFileSize;
  }
}

onMounted(() => {
  refreshCurrentList();

  getDashboardInfo();
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
</style>

