<template>
  <div class="fixed inset-0 bg-black/75 z-50 flex flex-col" @click.self="emit('close')">
    <!-- Header -->
    <div class="flex items-center justify-between px-4 py-3 bg-white border-b shadow-sm shrink-0">
      <span class="font-semibold text-gray-800 truncate max-w-[60%]">{{ file.name }}</span>
      <div class="flex items-center gap-3">
        <a
          :href="downloadUrl"
          :download="file.name"
          class="text-sm px-3 py-1.5 bg-blue-600 text-white rounded hover:bg-blue-700"
        >
          Download
        </a>
        <button
          @click="emit('close')"
          class="text-gray-500 hover:text-gray-800 text-2xl leading-none"
        >
          &times;
        </button>
      </div>
    </div>

    <!-- Content area -->
    <div class="flex-1 overflow-hidden flex items-center justify-center bg-gray-100">
      <div v-if="loading" class="flex flex-col items-center gap-3 text-gray-500">
        <SpinnerView />
        <span class="text-sm">Loading preview...</span>
      </div>

      <p v-else-if="loadError" class="text-red-600">{{ loadError }}</p>

      <!-- Image -->
      <img
        v-else-if="viewType === 'image'"
        :src="blobUrl"
        :alt="file.name"
        class="max-w-full max-h-full object-contain"
      />

      <!-- PDF -->
      <iframe
        v-else-if="viewType === 'pdf'"
        :src="blobUrl"
        class="w-full h-full border-0"
        title="PDF viewer"
      />

      <!-- Text -->
      <div v-else-if="viewType === 'text'" class="w-full h-full overflow-auto p-6">
        <pre class="text-sm text-gray-800 whitespace-pre-wrap break-words font-mono">{{ textContent }}</pre>
      </div>

      <!-- Unsupported -->
      <div v-else class="text-center space-y-4 p-8">
        <p class="text-gray-600">No preview available for this file type.</p>
        <a
          :href="downloadUrl"
          :download="file.name"
          class="inline-block px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
        >
          Download to view
        </a>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import type { File } from '@/types/folder';
import SpinnerView from '@/views/components/SpinnerView.vue';

const props = defineProps<{
  file: File;
  downloadUrl: string;
}>();

const emit = defineEmits<{ close: [] }>();

const IMAGE_EXTENSIONS = new Set([
  'jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico', 'tiff', 'tif', 'avif',
]);

const TEXT_EXTENSIONS = new Set([
  'txt', 'md', 'csv', 'json', 'xml', 'html', 'css', 'js', 'ts', 'jsx', 'tsx',
  'yaml', 'yml', 'toml', 'ini', 'log', 'sh', 'py', 'go', 'rs', 'java', 'c',
  'cpp', 'h', 'sql', 'env', 'gitignore',
]);

const MIME_BY_EXT: Record<string, string> = {
  jpg: 'image/jpeg',
  jpeg: 'image/jpeg',
  png: 'image/png',
  gif: 'image/gif',
  webp: 'image/webp',
  svg: 'image/svg+xml',
  bmp: 'image/bmp',
  ico: 'image/x-icon',
  tiff: 'image/tiff',
  tif: 'image/tiff',
  avif: 'image/avif',
  pdf: 'application/pdf',
};

const TEXT_SIZE_LIMIT = 5 * 1024 * 1024; // 5 MB

const loading = ref(false);
const loadError = ref('');
const blobUrl = ref('');
const textContent = ref('');

const ext = computed(() => props.file.extension?.toLowerCase() ?? '');

const viewType = computed(() => {
  if (IMAGE_EXTENSIONS.has(ext.value)) return 'image';
  if (ext.value === 'pdf') return 'pdf';
  if (TEXT_EXTENSIONS.has(ext.value) && props.file.file_size <= TEXT_SIZE_LIMIT) return 'text';
  return 'unsupported';
});

async function loadPreview() {
  if (viewType.value === 'unsupported') return;

  loading.value = true;
  loadError.value = '';

  try {
    const response = await fetch(props.downloadUrl);
    if (!response.ok) {
      loadError.value = 'Failed to load file preview.';
      return;
    }

    if (viewType.value === 'text') {
      textContent.value = await response.text();
    } else {
      const blob = await response.blob();
      const mimeType = MIME_BY_EXT[ext.value] ?? 'application/octet-stream';
      blobUrl.value = URL.createObjectURL(new Blob([blob], { type: mimeType }));
    }
  } catch {
    loadError.value = 'An error occurred while loading the preview.';
  } finally {
    loading.value = false;
  }
}

function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close');
}

onMounted(() => {
  loadPreview();
  document.addEventListener('keydown', onKeyDown);
});

onUnmounted(() => {
  if (blobUrl.value) URL.revokeObjectURL(blobUrl.value);
  document.removeEventListener('keydown', onKeyDown);
});
</script>
