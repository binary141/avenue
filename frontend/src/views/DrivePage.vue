<template>
  <div class="page gap-5">
    <h1>Drive</h1>
    <FileUploader :parent="currentFolderId" @upload="handleFileUpload" @error="handleUploadError" multiple=true />

    <div v-if="loading" class="flex flex-col align-center content-center gap-3">
      <SpinnerView />
      <p>Loading folder contents...</p>
    </div>

    <ErrorMessage v-if="error">{{ error }}</ErrorMessage>

    <div v-if="!loading && !error" class="folder-contents flex flex-col gap-4">
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

      <div v-if="files.length > 0" class="files-section">
        <h2>Files</h2>
        <div class="items-list flex flex-col gap-2">
          <div
            v-for="file in files"
            :key="file.id"
            class="file-item card flex flex-row align-center gap-3 p-3"
          >
            <span class="file-icon">📄</span>
            <span class="file-name" @click.prevent="download(file.id)" :title="file.name">
              {{ formatFileName(file.name) }}
            </span>
            <span class="file-size">{{ formatFileSize(file.file_size) }}</span>
            <span class="file-extension">{{ file.extension }}</span>
            <span class="file-delete" @click="deleteFile(file.id)">X</span>
          </div>
        </div>
      </div>

      <div v-if="!loading && folders.length === 0 && files.length === 0" class="empty-state">
        <p>This folder is empty.</p>
      </div>
    </div>
  </div>
</template>




<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import api from '@/utils/api';
import type { Folder, File, FolderContents } from '@/types/folder';
import SpinnerView from './components/SpinnerView.vue';
import ErrorMessage from './components/ErrorMessage.vue';
import FileUploader from '@/components/FileUploader.vue';

const route = useRoute();
const loading = ref(false);
const error = ref<string | undefined>();
const folders = ref<Folder[]>([]);
const files = ref<File[]>([]);
const currentFolderId = ref<string>('');

async function deleteFile(fileId: string): null{
  const response = await api({
    url: "v1/file/" + fileId,
    method: "DELETE",
  });

  refreshCurrentList();
}

function formatFileName(fileName: string): string{
  const maxNameLength = 35;

  if (fileName.length > maxNameLength) {
    return fileName.substring(0, maxNameLength) + "..."
  }

  return fileName
}

async function download(fileId: string): null{
    const response = await api({
        url: "v1/file/" + fileId,
        method: "GET",
        responseType: "blob",
    });

    if (response.status == 404) {
      console.log("file not found... not downloading");
      return
    } else if (response.status == 500) {
      console.log("Error downloading file: ", response)
      return
    }

    // Convert the stream to a Blob URL
    const blob = new Blob([response.body.blob]);
    const url = window.URL.createObjectURL(blob);

    // Trigger download
    const a = document.createElement("a");
    a.href = url;

    // Optional: use the filename from headers:
    const disposition = response.headers["content-disposition"];
    let filename = "download.bin";

    if (disposition) {
      console.log("disposition: ", disposition)
        const match = disposition.match(/filename="?([^"]+)"?/);
        console.log(match)
        if (match) {
            filename = match[1];
        }
    }

    a.download = filename;
    a.click();

    // Cleanup
    window.URL.revokeObjectURL(url);
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}

async function loadFolderContents(folderId: string = '') {
  loading.value = true;
  error.value = undefined;

  try {
    // Note: The backend route uses :folderID but the handler expects fileID param
    // Using folderID as that's what the route defines
    const response = await api({
      url: folderId ?  `v1/folder/list/${folderId}`:`v1/folder/list/-1`,
      method: 'GET',
    });

    if (response.ok && response.body) {
      const contents = response.body as FolderContents;
      folders.value = contents.folders || [];
      files.value = contents.files || [];
    } else {
      error.value = response.body?.error || response.body?.message || 'Failed to load folder contents';
    }
  } catch (e) {
    error.value = 'An error occurred while loading folder contents';
    console.error(e);
  } finally {
    loading.value = false;
  }
}

function handleFileUpload() {
  // Reload folder contents after successful upload
  loadFolderContents(currentFolderId.value);
}

function handleUploadError(message: string) {
  error.value = message;
}

function refreshCurrentList() {
  // Get folderId from route params or query, default to empty string for root
  const folderId = (route.params.folderId as string) || (route.query.folderId as string) || '';
  currentFolderId.value = folderId;
  loadFolderContents(folderId);
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
</style>
