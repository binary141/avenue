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

const apiRoot = import.meta.env.VITE_APP_API_URL || '';
const token = computed(() => route.params.token as string);
const subFolderUUID = computed(() => route.query.sub as string | undefined);

function authHeaders(): Record<string, string> {
  return usersStore.token ? { Authorization: `Token ${usersStore.token}` } : {};
}

function fileDownloadURL(fileUUID: string): string {
  const base = `${apiRoot}share/folder/${token.value}/file/${fileUUID}`;
  return usersStore.token ? `${base}?token=${usersStore.token}` : base;
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

async function fetchContents() {
  loading.value = true;
  notFound.value = false;
  loginRequired.value = false;

  const sub = subFolderUUID.value;
  const url = sub
    ? `${apiRoot}share/folder/${token.value}/browse/${sub}`
    : `${apiRoot}share/folder/${token.value}`;

  try {
    const res = await fetch(url, { headers: authHeaders() });
    if (res.status === 401) {
      loginRequired.value = true;
    } else if (!res.ok) {
      notFound.value = true;
    } else {
      contents.value = await res.json();
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
