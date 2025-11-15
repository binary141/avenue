<template>
  <div class="file-uploader">
    <div
      class="drop-zone"
      :class="{
        'drop-zone--dragging': isDragging,
        'drop-zone--disabled': disabled,
        'drop-zone--has-files': hasFiles
      }"
      @drop.prevent="handleDrop"
      @dragover.prevent="handleDragOver"
      @dragleave="handleDragLeave"
      @click="triggerFileInput"
    >
      <input
        ref="fileInput"
        type="file"
        :accept="accept"
        :multiple="multiple"
        :disabled="disabled"
        class="file-input"
        @change="handleFileInputChange"
      />
      
      <div v-if="!hasFiles" class="drop-zone__content">
        <svg
          class="drop-zone__icon"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
          />
        </svg>
        <p class="drop-zone__text">
          <span class="drop-zone__text--primary">Click to upload</span>
          or drag and drop
        </p>
        <p class="drop-zone__hint">
          {{ accept === '*' ? 'Any file type' : accept }} (Max {{ maxSize }}MB)
        </p>
      </div>
      
      <div v-else class="file-list">
        <div
          v-for="(file, index) in selectedFiles"
          :key="index"
          class="file-item"
        >
          <div class="file-item__info">
            <svg
              class="file-item__icon"
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
              />
            </svg>
            <div class="file-item__details">
              <p class="file-item__name">{{ file.name }}</p>
              <p class="file-item__size">{{ formatFileSize(file.size) }}</p>
            </div>
          </div>
          <button
            v-if="!isUploading"
            type="button"
            class="file-item__remove"
            @click.stop="removeFile(index)"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>
      </div>
    </div>
    
    <div v-if="isUploading" class="progress-bar">
      <div class="progress-bar__fill" :style="{ width: `${uploadProgress}%` }"></div>
      <span class="progress-bar__text">{{ uploadProgress }}%</span>
    </div>
    
    <button
      v-if="hasFiles && !isUploading"
      type="button"
      class="upload-button"
      :disabled="disabled"
      @click.stop="uploadFiles"
    >
      Upload {{ selectedFiles.length }} {{ selectedFiles.length === 1 ? 'file' : 'files' }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

interface Props {
  accept?: string
  maxSize?: number // in MB
  multiple?: boolean
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  accept: '*',
  maxSize: 10,
  multiple: false,
  disabled: false
})

const emit = defineEmits<{
  upload: [files: File[]]
  error: [message: string]
}>()

const isDragging = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const selectedFiles = ref<File[]>([])
const uploadProgress = ref<number>(0)
const isUploading = ref(false)

const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

const validateFiles = (files: File[]): boolean => {
  const maxSizeBytes = props.maxSize * 1024 * 1024
  
  for (const file of files) {
    if (file.size > maxSizeBytes) {
      emit('error', `File "${file.name}" exceeds maximum size of ${props.maxSize}MB`)
      return false
    }
  }
  
  return true
}

const handleFiles = (files: FileList | null) => {
  if (!files || files.length === 0) return
  
  const fileArray = Array.from(files)
  
  if (!props.multiple && fileArray.length > 1) {
    emit('error', 'Only one file can be uploaded at a time')
    return
  }
  
  if (!validateFiles(fileArray)) return
  
  selectedFiles.value = fileArray
}

const handleDrop = (e: DragEvent) => {
  isDragging.value = false
  if (props.disabled) return
  
  handleFiles(e.dataTransfer?.files || null)
}

const handleDragOver = (e: DragEvent) => {
  e.preventDefault()
  if (!props.disabled) {
    isDragging.value = true
  }
}

const handleDragLeave = () => {
  isDragging.value = false
}

const handleFileInputChange = (e: Event) => {
  const target = e.target as HTMLInputElement
  handleFiles(target.files)
}

const triggerFileInput = () => {
  if (!props.disabled) {
    fileInput.value?.click()
  }
}

const removeFile = (index: number) => {
  selectedFiles.value.splice(index, 1)
}

const uploadFiles = async () => {
  if (selectedFiles.value.length === 0) {
    emit('error', 'No files selected')
    return
  }
  
  isUploading.value = true
  uploadProgress.value = 0
  
  try {
    const formData = new FormData()
    selectedFiles.value.forEach((file) => {
      formData.append('files', file)
    })
    
    const xhr = new XMLHttpRequest()
    
    xhr.upload.addEventListener('progress', (e) => {
      if (e.lengthComputable) {
        uploadProgress.value = Math.round((e.loaded / e.total) * 100)
      }
    })
    
    xhr.addEventListener('load', () => {
      if (xhr.status === 200) {
        emit('upload', selectedFiles.value)
        selectedFiles.value = []
        uploadProgress.value = 0
      } else {
        emit('error', `Upload failed: ${xhr.statusText}`)
      }
      isUploading.value = false
    })
    
    xhr.addEventListener('error', () => {
      emit('error', 'Upload failed due to network error')
      isUploading.value = false
    })
    
    const token = localStorage.getItem('authToken') // Adjust based on your auth implementation
    xhr.open('POST', '/v1/file')
    xhr.setRequestHeader('Authorization', `Token ${token}`)
    xhr.send(formData)
    
  } catch (error) {
    emit('error', error instanceof Error ? error.message : 'Upload failed')
    isUploading.value = false
  }
}

const hasFiles = computed(() => selectedFiles.value.length > 0)
</script>

<style scoped>
.file-uploader {
  width: 100%;
}

.drop-zone {
  border: 2px dashed #cbd5e1;
  border-radius: 0.5rem;
  padding: 2rem;
  text-align: center;
  cursor: pointer;
  transition: all 0.2s ease;
  background-color: #f8fafc;
}

.drop-zone:hover:not(.drop-zone--disabled) {
  border-color: #3b82f6;
  background-color: #eff6ff;
}

.drop-zone--dragging {
  border-color: #3b82f6;
  background-color: #dbeafe;
}

.drop-zone--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.drop-zone--has-files {
  padding: 1rem;
}

.file-input {
  display: none;
}

.drop-zone__content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}

.drop-zone__icon {
  width: 3rem;
  height: 3rem;
  color: #64748b;
}

.drop-zone__text {
  font-size: 0.875rem;
  color: #475569;
  margin: 0;
}

.drop-zone__text--primary {
  color: #3b82f6;
  font-weight: 600;
}

.drop-zone__hint {
  font-size: 0.75rem;
  color: #94a3b8;
  margin: 0;
}

.file-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.file-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem;
  background-color: white;
  border: 1px solid #e2e8f0;
  border-radius: 0.375rem;
  transition: all 0.2s ease;
}

.file-item:hover {
  border-color: #cbd5e1;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1);
}

.file-item__info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex: 1;
  min-width: 0;
}

.file-item__icon {
  width: 2rem;
  height: 2rem;
  color: #64748b;
  flex-shrink: 0;
}

.file-item__details {
  flex: 1;
  min-width: 0;
  text-align: left;
}

.file-item__name {
  font-size: 0.875rem;
  font-weight: 500;
  color: #1e293b;
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-item__size {
  font-size: 0.75rem;
  color: #64748b;
  margin: 0;
}

.file-item__remove {
  padding: 0.25rem;
  background: none;
  border: none;
  cursor: pointer;
  color: #64748b;
  transition: color 0.2s ease;
  flex-shrink: 0;
}

.file-item__remove:hover {
  color: #ef4444;
}

.file-item__remove svg {
  width: 1.25rem;
  height: 1.25rem;
}

.progress-bar {
  position: relative;
  width: 100%;
  height: 2rem;
  background-color: #e2e8f0;
  border-radius: 0.375rem;
  overflow: hidden;
  margin-top: 1rem;
}

.progress-bar__fill {
  height: 100%;
  background-color: #3b82f6;
  transition: width 0.3s ease;
}

.progress-bar__text {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 0.875rem;
  font-weight: 600;
  color: #1e293b;
}

.upload-button {
  width: 100%;
  margin-top: 1rem;
  padding: 0.75rem 1.5rem;
  background-color: #3b82f6;
  color: white;
  border: none;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.upload-button:hover:not(:disabled) {
  background-color: #2563eb;
}

.upload-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>