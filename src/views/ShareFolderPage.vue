<template>
  <div class="min-h-screen p-6" style="background: var(--gray, #2C2B2B);">
    <div class="mx-auto w-full max-w-2xl">

      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center gap-3 text-white pt-20">
        <svg class="animate-spin h-8 w-8 text-blue-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z"/>
        </svg>
        <p>Loading…</p>
      </div>

      <!-- Login required -->
      <div v-else-if="loginRequired" class="bg-white rounded-lg shadow-lg p-8 mt-10 text-center">
        <p class="text-4xl mb-3">🔐</p>
        <h1 class="text-xl font-bold text-gray-800 mb-2">Login required</h1>
        <p class="text-gray-500 text-sm mb-5">You need to be logged in to access this shared folder.</p>
        <a :href="`/login?redirect=${encodeURIComponent(route.fullPath)}`"
           class="inline-block px-4 py-2 rounded font-semibold text-white" style="background: #3A3F78;">
          Go to login
        </a>
      </div>

      <!-- Not found -->
      <div v-else-if="notFound" class="bg-white rounded-lg shadow-lg p-8 mt-10 text-center">
        <p class="text-4xl mb-3">🔒</p>
        <h1 class="text-xl font-bold text-gray-800 mb-2">Link not found</h1>
        <p class="text-gray-500 text-sm">This share link doesn't exist or has expired.</p>
      </div>

      <!-- Folder contents -->
      <div v-else-if="contents" class="flex flex-col gap-4">
        <!-- Header -->
        <div class="flex items-center gap-3 pt-4">
          <span class="text-4xl">📁</span>
          <h1 class="text-2xl font-bold text-white">{{ contents.folder_name }}</h1>
        </div>

        <!-- Breadcrumb -->
        <div v-if="breadcrumbs.length > 0" class="flex items-center gap-1 text-sm flex-wrap">
          <button @click="navigateTo(null)" class="text-blue-300 hover:underline">Root</button>
          <template v-for="(crumb, i) in breadcrumbs" :key="crumb.uuid">
            <span class="text-gray-400">/</span>
            <button
              v-if="i < breadcrumbs.length - 1"
              @click="navigateTo(crumb)"
              class="text-blue-300 hover:underline"
            >{{ crumb.name }}</button>
            <span v-else class="text-white font-medium">{{ crumb.name }}</span>
          </template>
        </div>

        <!-- Upload area -->
        <div v-if="contents.allow_upload" class="bg-white rounded-lg shadow-lg p-4 flex flex-col gap-3">
          <!-- Drop zone -->
          <div
            class="border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-all duration-200"
            :class="isDragging
              ? 'border-blue-500 bg-blue-100'
              : 'border-slate-300 bg-slate-50 hover:border-blue-500 hover:bg-blue-50'"
            @drop.prevent="handleDrop"
            @dragover.prevent="handleDragOver"
            @dragleave="handleDragLeave"
            @click="fileInputRef?.click()"
          >
            <input ref="fileInputRef" type="file" multiple class="hidden" @change="onFileSelected" />

            <div v-if="selectedFiles.length === 0" class="flex flex-col items-center gap-2">
              <svg class="w-12 h-12 text-slate-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"/>
              </svg>
              <p class="text-sm text-slate-600">
                <span class="text-blue-500 font-semibold">Click to upload</span> or drag and drop
              </p>
              <p class="text-xs text-slate-400">Any file type (Max {{ formatFileSize(maxUploadSize) }})</p>
            </div>

            <div v-else class="flex flex-col gap-2" @click.stop>
              <div
                v-for="(file, index) in selectedFiles"
                :key="index"
                class="flex items-center justify-between p-3 bg-white border border-slate-200 rounded-md"
              >
                <div class="flex items-center gap-3 flex-1 min-w-0">
                  <svg class="w-8 h-8 text-slate-500 shrink-0" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                  </svg>
                  <div class="flex-1 min-w-0 text-left">
                    <p class="text-sm font-medium text-slate-800 truncate">{{ file.name }}</p>
                    <p class="text-xs text-slate-500">{{ formatFileSize(file.size) }}</p>
                  </div>
                </div>
                <button v-if="!uploading" @click.stop="removeFile(index)" class="p-1 text-slate-500 hover:text-red-500 transition-colors">
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" class="w-5 h-5">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                  </svg>
                </button>
              </div>
            </div>
          </div>

          <!-- Progress bar -->
          <div v-if="uploading" class="relative w-full h-8 bg-slate-200 rounded overflow-hidden">
            <div class="absolute inset-y-0 left-0 bg-blue-500 transition-all duration-300" :style="{ width: `${uploadProgress}%` }"></div>
            <span class="absolute inset-0 flex items-center justify-center text-sm font-semibold text-slate-800">{{ uploadProgress }}%</span>
          </div>

          <!-- Upload button -->
          <button
            v-if="selectedFiles.length > 0 && !uploading"
            @click="uploadFiles"
            class="w-full py-3 text-white rounded text-sm font-semibold transition-colors"
            style="background-color: #3b82f6;"
            @mouseenter="$event.currentTarget.style.backgroundColor = '#2563eb'"
            @mouseleave="$event.currentTarget.style.backgroundColor = '#3b82f6'"
          >
            Upload {{ selectedFiles.length }} {{ selectedFiles.length === 1 ? 'file' : 'files' }}
          </button>

          <p v-if="uploadError" class="text-sm text-red-600">{{ uploadError }}</p>
          <p v-if="uploadSuccess" class="text-sm text-green-600">Files uploaded successfully.</p>
        </div>

        <!-- Contents card -->
        <div class="bg-white rounded-lg shadow-lg overflow-hidden">
          <!-- Empty state -->
          <div v-if="contents.folders.length === 0 && contents.files.length === 0"
               class="p-8 text-center text-gray-400">
            This folder is empty.
          </div>

          <div v-else class="divide-y divide-gray-100">
            <!-- Folders -->
            <div
              v-for="folder in contents.folders"
              :key="folder.uuid"
              class="flex items-center gap-3 px-4 py-3 hover:bg-gray-50 cursor-pointer"
              @click="enterFolder(folder)"
            >
              <span class="text-xl shrink-0">📁</span>
              <span class="flex-1 font-medium text-gray-800 truncate">{{ folder.name }}</span>
              <span class="text-gray-400 text-sm">›</span>
            </div>

            <!-- Files -->
            <div
              v-for="file in contents.files"
              :key="file.uuid"
              class="flex items-center gap-3 px-4 py-3 hover:bg-gray-50"
            >
              <span class="text-xl shrink-0">📄</span>
              <span class="flex-1 font-medium text-gray-800 truncate">{{ file.name }}</span>
              <span class="text-sm text-gray-400 shrink-0">{{ formatFileSize(file.file_size) }}</span>
              <button
                v-if="usersStore.token && file.created_by === currentUserId"
                @click.stop="deleteFile(file.uuid)"
                class="shrink-0 text-red-400 hover:text-red-600 transition-colors p-1"
                title="Delete"
              >🗑️</button>
              <a
                :href="fileDownloadURL(file.uuid)"
                class="shrink-0 px-3 py-1 rounded text-sm font-medium text-white"
                style="background: #3A3F78;"
                :download="file.name"
              >⬇️</a>
            </div>
          </div>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useUsersStore } from '../stores/users';
import type { File, Folder } from '@/types/folder';

interface FolderContents {
  folder_name: string;
  folder_uuid: string;
  files: File[];
  folders: Folder[];
  allow_upload: boolean;
  max_file_size: number;
}

interface Crumb {
  uuid: string;
  name: string;
}

const route = useRoute();
const router = useRouter();
const usersStore = useUsersStore();

const loading = ref(true);
const notFound = ref(false);
const loginRequired = ref(false);
const contents = ref<FolderContents | null>(null);
const breadcrumbs = ref<Crumb[]>([]);

const fileInputRef = ref<HTMLInputElement | null>(null);
const selectedFiles = ref<File_[]>([]);
const uploading = ref(false);
const uploadProgress = ref(0);
const uploadError = ref('');
const uploadSuccess = ref(false);
const maxUploadSize = ref(0);
const isDragging = ref(false);

// Use browser's native File type to avoid conflict with our File interface
type File_ = globalThis.File;

const apiRoot = import.meta.env.VITE_APP_API_URL || '';
const token = computed(() => route.params.token as string);
const subFolderUUID = computed(() => route.query.sub as string | undefined);
const currentUserId = computed(() => usersStore.userData.data.id);

function authHeaders(): Record<string, string> {
  return usersStore.token ? { Authorization: `Token ${usersStore.token}` } : {};
}

function fileDownloadURL(fileUUID: string): string {
  const base = `${apiRoot}api/share/folder/${token.value}/file/${fileUUID}`;
  return usersStore.token ? `${base}?token=${usersStore.token}` : base;
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

function validateAndAddFiles(list: FileList | File_[]) {
  uploadError.value = '';
  uploadSuccess.value = false;
  const oversized: string[] = [];
  const valid: File_[] = [];
  for (const file of Array.from(list)) {
    if (maxUploadSize.value > 0 && file.size > maxUploadSize.value) {
      oversized.push(file.name);
    } else {
      valid.push(file);
    }
  }
  if (oversized.length) {
    uploadError.value = `${oversized.map(n => `"${n}"`).join(', ')} exceed${oversized.length === 1 ? 's' : ''} the maximum size of ${formatFileSize(maxUploadSize.value)}`;
  }
  selectedFiles.value = [...selectedFiles.value, ...valid];
}

function onFileSelected(e: Event) {
  const input = e.target as HTMLInputElement;
  if (input.files?.length) validateAndAddFiles(input.files);
  input.value = '';
}

function handleDrop(e: DragEvent) {
  isDragging.value = false;
  if (e.dataTransfer?.files?.length) validateAndAddFiles(e.dataTransfer.files);
}

function handleDragOver() {
  isDragging.value = true;
}

function handleDragLeave() {
  isDragging.value = false;
}

function removeFile(index: number) {
  selectedFiles.value = selectedFiles.value.filter((_, i) => i !== index);
  uploadError.value = '';
  uploadSuccess.value = false;
}

async function uploadFiles() {
  if (selectedFiles.value.length === 0) return;
  uploading.value = true;
  uploadError.value = '';
  uploadSuccess.value = false;
  uploadProgress.value = 0;

  const sub = subFolderUUID.value;
  const url = sub
    ? `${apiRoot}api/share/folder/${token.value}/upload?folder=${sub}`
    : `${apiRoot}api/share/folder/${token.value}/upload`;

  const total = selectedFiles.value.length;
  let uploaded = 0;

  try {
    for (const file of [...selectedFiles.value]) {
      const formData = new FormData();
      formData.append('file', file);

      const res = await fetch(url, {
        method: 'POST',
        headers: authHeaders(),
        body: formData,
      });

      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        uploadError.value = `Failed to upload "${file.name}": ${body.error || body.message || 'Upload failed'}`;
        uploading.value = false;
        return;
      }

      selectedFiles.value = selectedFiles.value.filter(f => f !== file);
      uploaded++;
      uploadProgress.value = Math.round((uploaded / total) * 100);
    }

    uploadSuccess.value = true;
    if (fileInputRef.value) fileInputRef.value.value = '';
    await fetchContents();
  } catch {
    uploadError.value = 'Upload failed';
  } finally {
    uploading.value = false;
  }
}

async function deleteFile(fileUUID: string) {
  await fetch(`${apiRoot}v1/file/${fileUUID}`, {
    method: 'DELETE',
    headers: authHeaders(),
  });
  await fetchContents();
}

async function fetchContents() {
  loading.value = true;
  notFound.value = false;
  loginRequired.value = false;

  const sub = subFolderUUID.value;
  const url = sub
    ? `${apiRoot}api/share/folder/${token.value}/browse/${sub}`
    : `${apiRoot}api/share/folder/${token.value}`;

  try {
    const res = await fetch(url, { headers: authHeaders() });
    if (res.status === 401) {
      loginRequired.value = true;
    } else if (!res.ok) {
      notFound.value = true;
    } else {
      const data: FolderContents = await res.json();
      contents.value = data;
      if (data.max_file_size) maxUploadSize.value = data.max_file_size;
    }
  } catch {
    notFound.value = true;
  } finally {
    loading.value = false;
  }
}

function enterFolder(folder: Folder) {
  breadcrumbs.value = [...breadcrumbs.value, { uuid: folder.uuid, name: folder.name }];
  router.push({ query: { sub: folder.uuid } });
}

function navigateTo(crumb: Crumb | null) {
  if (!crumb) {
    breadcrumbs.value = [];
    router.push({ query: {} });
  } else {
    const idx = breadcrumbs.value.findIndex(c => c.uuid === crumb.uuid);
    breadcrumbs.value = breadcrumbs.value.slice(0, idx + 1);
    router.push({ query: { sub: crumb.uuid } });
  }
}

watch(subFolderUUID, fetchContents);
onMounted(fetchContents);
</script>
