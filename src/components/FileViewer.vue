<template>
  <div class="viewer-backdrop fixed inset-0 z-50 flex flex-col" @click.self="emit('close')">
    <!-- Header -->
    <div class="viewer-header flex items-center justify-between px-4 py-3 shrink-0">
      <span class="viewer-filename truncate">{{ file.name }}</span>
      <div class="flex items-center gap-3">
        <a :href="downloadUrl" :download="file.name">
          <AppButton>Download</AppButton>
        </a>
        <button class="viewer-close" @click="emit('close')" aria-label="Close">&times;</button>
      </div>
    </div>

    <!-- Content area -->
    <div class="viewer-content flex-1 overflow-hidden flex items-center justify-center">
      <div v-if="loading" class="flex flex-col items-center gap-3">
        <SpinnerView :size="36" color="var(--text-secondary)" />
        <span class="viewer-loading-text">Loading preview...</span>
      </div>

      <p v-else-if="loadError" class="viewer-error">{{ loadError }}</p>

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
      <div v-else-if="viewType === 'text'" class="viewer-text-wrap w-full h-full overflow-auto p-6">
        <pre class="viewer-pre">{{ textContent }}</pre>
      </div>

      <!-- Unsupported -->
      <div v-else class="flex flex-col items-center gap-4">
        <p class="viewer-unsupported-msg">No preview available for this file type.</p>
        <a :href="downloadUrl" :download="file.name">
          <AppButton>Download to view</AppButton>
        </a>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import type { File } from '@/types/folder';
import SpinnerView from '@/views/components/SpinnerView.vue';
import AppButton from '@/views/components/AppButton.vue';

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

const TEXT_SIZE_LIMIT = 5 * 1024 * 1024;

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

<style scoped>
.viewer-backdrop {
  background-color: rgba(0, 0, 0, 0.85);
}

.viewer-header {
  background-color: var(--gray-2);
  border-bottom: 1px solid var(--gray-4);
}

.viewer-filename {
  font-family: "Inter", sans-serif;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
  max-width: 60%;
}

.viewer-close {
  font-size: 22px;
  line-height: 1;
  color: var(--text-secondary);
  transition: color 0.15s;
}
.viewer-close:hover {
  color: var(--text);
}

.viewer-content {
  background-color: var(--gray-3);
}

.viewer-loading-text,
.viewer-unsupported-msg {
  color: var(--text-secondary);
  font-family: "Inter", sans-serif;
  font-size: 14px;
}

.viewer-error {
  color: #e57373;
  font-family: "Inter", sans-serif;
  font-size: 14px;
}

.viewer-text-wrap {
  background-color: var(--gray-2);
}

.viewer-pre {
  font-family: "Courier New", Courier, monospace;
  font-size: 13px;
  color: var(--text);
  white-space: pre-wrap;
  word-break: break-words;
  margin: 0;
}
</style>
