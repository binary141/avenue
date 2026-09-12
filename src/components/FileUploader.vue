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

      <div v-if="!hasFiles" class="drop-zone__row">
        <div class="drop-zone__content">
          <div class="drop-zone__icon-badge">
            <svg
              class="drop-zone__icon"
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              stroke="#fff"
              stroke-width="2.4"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M12 3v12" />
              <path d="M7 8l5-5 5 5" />
              <path d="M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2" />
            </svg>
          </div>
          <div class="drop-zone__copy">
            <p class="drop-zone__title">Upload files</p>
            <p class="drop-zone__hint">
              Drag and drop, or click to browse · Max {{ formattedSize }}{{ accept === '*' ? '' : ` (${accept})` }}
            </p>
          </div>
        </div>

        <template v-if="$slots.aside">
          <div class="drop-zone__divider"></div>
          <div class="drop-zone__aside" @click.stop>
            <slot name="aside" />
          </div>
        </template>
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
              <p class="file-item__name" :title="file.name">{{ file.name }}</p>
              <template v-if="fileStatus.has(fileKey(file))">
                <div class="file-item__progress-track">
                  <div class="file-item__progress-fill" :style="{ width: `${fileStatus.get(fileKey(file))!.progress}%` }"></div>
                </div>
                <p class="file-item__speed">{{ fileStatus.get(fileKey(file))!.progress }}% · {{ fileStatus.get(fileKey(file))!.speed }}</p>
              </template>
              <p v-else class="file-item__size">{{ formatFileSize(file.size) }}</p>
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

    <div v-if="isUploading && selectedFiles.length > 1" class="progress-bar">
      <div class="progress-bar__fill" :style="{ width: `${uploadProgress}%` }"></div>
      <span class="progress-bar__text">{{ uploadProgress }}% overall</span>
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

    <button
      v-if="isUploading"
      type="button"
      class="upload-button upload-button--cancel"
      @click.stop="cancelUpload"
    >
      Cancel upload
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watchEffect } from 'vue'
import api, { GLOBAL_HEADERS, type JsonValue } from '@/utils/api'
import type { FolderItem } from '@/types/folder'

interface Props {
  accept?: string
  maxSize?: number // in bytes
  chunkSize?: number // in bytes
  multiple?: boolean
  disabled?: boolean
  parent?: string // folder ID for parent folder
}

const props = withDefaults(defineProps<Props>(), {
  accept: '*',
  maxSize: 10_000_000,
  chunkSize: 16 * 1024 * 1024,
  multiple: false,
  disabled: false,
  parent: ''
})

// The upload endpoint responds with the folder's uuid as `parent` (not the
// numeric parent_id used by the listing endpoint's FolderItem shape), so the
// caller can tell whether the new file belongs in the currently-open folder.
const emit = defineEmits<{
  upload: [files: File[]]
  'file-uploaded': [file: FolderItem & { parent?: string }]
  error: [message: string]
}>()

const CONCURRENT_UPLOADS = 5
// Bounds how many chunk PUT requests are in flight across the WHOLE batch at
// once, shared by every file uploading concurrently, rather than a fixed
// per-file share. A file with fewer chunks left (or none) doesn't hog its
// slots — they're returned to the shared pool immediately for whichever
// other file has work — so the limit reflects real transport capacity (e.g.
// a browser's ~6 connection cap per origin under HTTP/1.1) instead of a
// guessed static split. A single big file uploading alone can still use the
// whole budget itself.
const MAX_CHUNKS_IN_FLIGHT = 6
const MAX_CHUNK_RETRIES = 5
const RESUME_STORAGE_PREFIX = 'avenue:upload:'

// Simple counting semaphore: acquire() resolves immediately while permits
// remain, otherwise queues the caller until a release() frees one up.
class Semaphore {
  private available: number
  private readonly waiters: Array<() => void> = []

  constructor(permits: number) {
    this.available = permits
  }

  acquire(): Promise<void> {
    if (this.available > 0) {
      this.available -= 1
      return Promise.resolve()
    }
    return new Promise((resolve) => this.waiters.push(resolve))
  }

  release(): void {
    const next = this.waiters.shift()
    if (next) {
      next()
    } else {
      this.available += 1
    }
  }
}

const isDragging = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const selectedFiles = ref<File[]>([])
const uploadProgress = ref<number>(0)
const isUploading = ref(false)
const formattedSize = ref('')

// Keyed by fileKey(file); present only while that file is actively uploading.
const fileStatus = ref<Map<string, { progress: number; speed: string }>>(new Map())
const activeXHRs = new Map<string, XMLHttpRequest>()
let cancelRequested = false

const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

watchEffect(() => {
  formattedSize.value = formatFileSize(props.maxSize);
});

const validateFiles = (files: File[]): boolean => {
  const maxSizeBytes = props.maxSize

  for (const file of files) {
    if (file.size > maxSizeBytes) {
      emit('error', `File "${file.name}" exceeds maximum size of ${formattedSize.value}`)
      return false
    }
  }

  return true
}

const fileKey = (file: File): string => `${file.name}:${file.size}:${file.lastModified}`

const handleFiles = (files: FileList | null) => {
  if (!files || files.length === 0) return

  const fileArray = Array.from(files)

  if (!props.multiple && fileArray.length > 1) {
    emit('error', 'Only one file can be uploaded at a time')
    return
  }

  if (!validateFiles(fileArray)) return

  if (!props.multiple) {
    selectedFiles.value = fileArray
    return
  }

  // Add to the existing staged selection instead of replacing it, so a
  // second drag-and-drop doesn't silently discard the first batch.
  const existingKeys = new Set(selectedFiles.value.map(fileKey))
  const newFiles = fileArray.filter(f => !existingKeys.has(fileKey(f)))
  selectedFiles.value = [...selectedFiles.value, ...newFiles]
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

// Resume state survives a page reload: keyed by fileKey() (name+size+mtime)
// so re-selecting the same file after a crash/reload picks the upload back
// up instead of restarting it. The File object itself can't be persisted —
// the browser only gives us a fresh one when the user re-selects it — so
// this is only consulted once handleFiles() sees a matching file again.
interface ResumeInfo { uploadId: string }

function loadResumeInfo(key: string): ResumeInfo | null {
  try {
    const raw = localStorage.getItem(RESUME_STORAGE_PREFIX + key)
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

function saveResumeInfo(key: string, info: ResumeInfo): void {
  try {
    localStorage.setItem(RESUME_STORAGE_PREFIX + key, JSON.stringify(info))
  } catch { /* localStorage unavailable (private mode, quota) — resume just won't survive a reload */ }
}

function clearResumeInfo(key: string): void {
  try {
    localStorage.removeItem(RESUME_STORAGE_PREFIX + key)
  } catch { /* ignore */ }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

// Hashes a chunk up front so its digest can ride along as a request header —
// the server verifies it against the bytes it actually receives, catching a
// corrupted chunk immediately (cheap resend of one chunk) instead of only
// surfacing it via the whole-file checksum after every chunk has landed.
async function sha256Hex(blob: Blob): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', await blob.arrayBuffer())
  return Array.from(new Uint8Array(digest)).map((b) => b.toString(16).padStart(2, '0')).join('')
}

// Sends one chunk via XHR (rather than fetch) so upload progress within a
// large chunk still drives the progress bar, not just whole-chunk jumps.
async function sendChunkXHR(uploadId: string, file: File, start: number, end: number, key: string, onProgress: (loaded: number) => void): Promise<{ status: number; body: JsonValue; aborted: boolean }> {
  const chunk = file.slice(start, end)
  const checksum = await sha256Hex(chunk)

  return new Promise((resolve) => {
    const xhr = new XMLHttpRequest()
    activeXHRs.set(key, xhr)

    xhr.upload.onprogress = (e) => {
      if (!e.lengthComputable) return
      onProgress(e.loaded)
    }

    xhr.onload = () => {
      activeXHRs.delete(key)
      let body: JsonValue = null
      try { body = JSON.parse(xhr.responseText) } catch { /* ignore */ }
      resolve({ status: xhr.status, body, aborted: false })
    }
    xhr.onerror = () => {
      activeXHRs.delete(key)
      resolve({ status: 0, body: null, aborted: false })
    }
    xhr.onabort = () => {
      activeXHRs.delete(key)
      resolve({ status: 0, body: null, aborted: true })
    }

    const apiRoot = import.meta.env.VITE_APP_API_URL || ''
    xhr.open('PUT', `${apiRoot}v1/upload/${uploadId}/chunk`)
    xhr.withCredentials = true
    xhr.setRequestHeader('Content-Range', `bytes ${start}-${end - 1}/${file.size}`)
    xhr.setRequestHeader('X-Chunk-Checksum', `sha256:${checksum}`)
    for (const [header, value] of Object.entries(GLOBAL_HEADERS)) {
      if (value !== undefined) xhr.setRequestHeader(header, value)
    }
    xhr.send(chunk)
  })
}

// Uploads one file end to end: resumes an existing session if the resume
// store has one for this file, otherwise starts a new one, then sends
// chunks — a few at a time in flight — until the server has all of it, then
// finalizes. Returns the created file record, or an error/aborted flag.
async function uploadOneFile(file: File, key: string, chunkSemaphore: Semaphore, onProgress: (pct: number, speedStr: string) => void): Promise<{ ok: boolean; aborted: boolean; body?: JsonValue; error?: string }> {
  let uploadId: string
  // Chunk indices (0-based, each props.chunkSize bytes except the last)
  // the server has already told us it has, so a resumed upload only
  // re-sends what's actually missing instead of just a contiguous prefix.
  let receivedChunkIndexes = new Set<number>()

  const resumeInfo = loadResumeInfo(key)
  if (resumeInfo) {
    const status = await api({ url: `v1/upload/${resumeInfo.uploadId}`, method: 'GET' })
    if (status.ok) {
      uploadId = resumeInfo.uploadId
      receivedChunkIndexes = chunkIndexesFromRanges(status.body.receivedRanges || [])
    } else {
      clearResumeInfo(key)
      const init = await api({ url: 'v1/upload', method: 'POST', json: { name: file.name, size: file.size, parent: props.parent || '' } })
      if (!init.ok) return { ok: false, aborted: false, error: init.body?.error || init.body?.message || 'Could not start upload' }
      uploadId = init.body.uploadId
      saveResumeInfo(key, { uploadId })
    }
  } else {
    const init = await api({ url: 'v1/upload', method: 'POST', json: { name: file.name, size: file.size, parent: props.parent || '' } })
    if (!init.ok) return { ok: false, aborted: false, error: init.body?.error || init.body?.message || 'Could not start upload' }
    uploadId = init.body.uploadId
    saveResumeInfo(key, { uploadId })
  }

  const chunkCount = Math.ceil(file.size / props.chunkSize)
  const chunkBounds = (idx: number): [number, number] => [idx * props.chunkSize, Math.min((idx + 1) * props.chunkSize, file.size)]

  // Bytes confirmed by the server (whole chunks) plus bytes reported
  // in-flight by each chunk's own XHR progress event, so the progress bar
  // and speed readout reflect every chunk currently uploading, not just one.
  let confirmedBytes = 0
  for (const idx of receivedChunkIndexes) {
    const [start, end] = chunkBounds(idx)
    confirmedBytes += end - start
  }
  const inFlightBytes = new Map<number, number>()
  const sessionStartBytes = confirmedBytes
  const startTime = Date.now()

  const reportProgress = () => {
    let loaded = confirmedBytes
    for (const bytes of inFlightBytes.values()) loaded += bytes
    const pct = Math.round((loaded / file.size) * 100)
    const elapsed = (Date.now() - startTime) / 1000
    const speedStr = elapsed > 0
      ? ((loaded - sessionStartBytes) / elapsed / (1024 * 1024)).toFixed(1) + ' MB/s'
      : ''
    onProgress(pct, speedStr)
  }

  let nextIdx = 0
  const claimNextChunk = (): number | null => {
    while (nextIdx < chunkCount && receivedChunkIndexes.has(nextIdx)) nextIdx += 1
    if (nextIdx >= chunkCount) return null
    return nextIdx++
  }

  let failure: string | null = null
  let aborted = false

  // Large files are sent in fixed-size chunks so a dropped connection loses
  // at most a few chunks instead of the whole transfer, and so a resumed
  // upload only has to re-send what the server never got. Actual network
  // concurrency is gated by chunkSemaphore, shared across every file in the
  // batch — these worker loops just claim chunks and queue on it, so one
  // file's upload isn't bottlenecked on round-trip latency per chunk, and a
  // file with nothing left to send never holds a slot idle.
  const chunkWorker = async () => {
    while (!failure && !aborted) {
      if (cancelRequested) { aborted = true; return }
      const idx = claimNextChunk()
      if (idx === null) return

      const [chunkStart, chunkEnd] = chunkBounds(idx)
      const chunkKey = `${key}:${idx}`

      let attempt = 0
      let done = false
      while (!done) {
        await chunkSemaphore.acquire()
        const result = await sendChunkXHR(uploadId, file, chunkStart, chunkEnd, chunkKey, (loaded) => {
          inFlightBytes.set(idx, loaded)
          reportProgress()
        })
        chunkSemaphore.release()

        if (result.aborted || cancelRequested) { aborted = true; return }

        if (result.status === 200) {
          inFlightBytes.delete(idx)
          confirmedBytes += chunkEnd - chunkStart
          reportProgress()
          done = true
        } else {
          inFlightBytes.delete(idx)
          attempt += 1
          if (attempt > MAX_CHUNK_RETRIES) {
            failure = result.body?.error || result.body?.message || 'Upload failed — you can retry without losing progress'
            return
          }
          await sleep(Math.min(500 * 2 ** attempt, 8000))
        }
      }
    }
  }

  const workerCount = Math.min(MAX_CHUNKS_IN_FLIGHT, chunkCount)
  await Promise.all(Array.from({ length: workerCount }, () => chunkWorker()))

  if (aborted || cancelRequested) return { ok: false, aborted: true }
  if (failure) return { ok: false, aborted: false, error: failure }

  const complete = await api({ url: `v1/upload/${uploadId}/complete`, method: 'POST' })
  if (!complete.ok) {
    return { ok: false, aborted: false, error: complete.body?.error || complete.body?.message || 'Could not finalize upload' }
  }

  clearResumeInfo(key)
  return { ok: true, aborted: false, body: complete.body }
}

// Converts the server's received byte ranges into chunk indexes, aligned to
// this upload's chunkSize. A range that doesn't start on a chunk boundary
// (e.g. chunkSize changed since the session started) is simply not counted
// as received — the chunk gets re-sent, which is always safe since chunk
// writes are idempotent.
function chunkIndexesFromRanges(ranges: { start: number; end: number }[]): Set<number> {
  const indexes = new Set<number>()
  for (const { start } of ranges) {
    if (start % props.chunkSize !== 0) continue
    indexes.add(start / props.chunkSize)
  }
  return indexes
}

const cancelUpload = () => {
  cancelRequested = true
  for (const xhr of activeXHRs.values()) {
    xhr.abort()
  }
}

const resetUploadState = () => {
  isUploading.value = false
  uploadProgress.value = 0
  fileStatus.value.clear()
  activeXHRs.clear()
}

const uploadFiles = async () => {
  if (selectedFiles.value.length === 0) {
    emit('error', 'No files selected')
    return
  }

  isUploading.value = true
  uploadProgress.value = 0
  cancelRequested = false

  const totalFiles = selectedFiles.value.length
  // Pending pool: files are pulled off the front by whichever worker frees
  // up first. Completed files are removed from selectedFiles as they
  // succeed, so a retry after a failure doesn't re-send finished uploads.
  const pending = [...selectedFiles.value]
  const uploaded: File[] = []
  let uploadError: string | null = null

  // One semaphore shared by every file's chunk workers this batch, so chunk
  // concurrency reflects real transport capacity instead of a fixed
  // per-file share.
  const chunkSemaphore = new Semaphore(MAX_CHUNKS_IN_FLIGHT)

  const worker = async () => {
    while (!cancelRequested && !uploadError) {
      const file = pending.shift()
      if (!file) return

      const key = fileKey(file)
      fileStatus.value.set(key, { progress: 0, speed: '' })

      const response = await uploadOneFile(file, key, chunkSemaphore, (pct, speedStr) => {
        fileStatus.value.set(key, { progress: pct, speed: speedStr })
      })

      fileStatus.value.delete(key)

      if (response.aborted || cancelRequested) {
        return
      }

      if (!response.ok) {
        uploadError = response.error || 'Upload failed'
        emit('error', `Failed to upload "${file.name}": ${uploadError}`)
        return
      }

      const idx = selectedFiles.value.indexOf(file)
      if (idx !== -1) selectedFiles.value.splice(idx, 1)
      uploaded.push(file)
      uploadProgress.value = Math.round((uploaded.length / totalFiles) * 100)

      // Let the parent add this file to the UI right away instead of
      // waiting for every file in the batch to finish uploading.
      if (response.body) {
        emit('file-uploaded', { ...(response.body as FolderItem & { parent?: string }), type: 'file' })
      }
    }
  }

  try {
    const workerCount = Math.min(CONCURRENT_UPLOADS, pending.length)
    await Promise.all(Array.from({ length: workerCount }, () => worker()))

    resetUploadState()
    if (uploaded.length > 0) {
      emit('upload', uploaded)
    }
  } catch (error) {
    resetUploadState()
    emit('error', error instanceof Error ? error.message : 'Upload failed')
  }
}

const hasFiles = computed(() => selectedFiles.value.length > 0)
</script>

<style scoped>
.file-uploader {
  width: 100%;
}

.drop-zone {
  border: 1px solid var(--gray-4);
  border-radius: 12px;
  padding: 16px 20px;
  cursor: pointer;
  transition: all 0.2s ease;
  background-color: var(--gray-2);
}

.drop-zone:hover:not(.drop-zone--disabled) {
  border-color: var(--accent);
}

.drop-zone--dragging {
  border-color: var(--accent);
  background-color: var(--gray-3);
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

.drop-zone__row {
  display: flex;
  align-items: center;
  gap: 20px;
}

.drop-zone__content {
  display: flex;
  align-items: center;
  gap: 14px;
  flex: 1;
  min-width: 0;
  padding: 8px 12px;
  border-radius: 9px;
  background: rgba(107, 111, 230, 0.12);
  border: 1px solid rgba(107, 111, 230, 0.4);
  transition: background-color 0.2s ease;
}

.drop-zone:hover:not(.drop-zone--disabled) .drop-zone__content {
  background: rgba(107, 111, 230, 0.2);
}

.drop-zone__divider {
  width: 1px;
  height: 40px;
  background: var(--gray-4);
  flex-shrink: 0;
}

.drop-zone__aside {
  min-width: 220px;
  cursor: default;
}

.drop-zone__icon-badge {
  width: 38px;
  height: 38px;
  border-radius: 9px;
  background: var(--accent);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.drop-zone__icon {
  width: 20px;
  height: 20px;
}

.drop-zone__copy {
  text-align: left;
}

.drop-zone__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  margin: 0;
}

.drop-zone__hint {
  font-size: 12px;
  color: var(--text-secondary);
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
  background-color: var(--gray-2);
  border: 1px solid var(--gray-4);
  border-radius: 0.375rem;
  transition: all 0.2s ease;
}

.file-item:hover {
  border-color: var(--gray-5);
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.3);
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
  color: var(--text-secondary);
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
  color: var(--text);
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-item__size {
  font-size: 0.75rem;
  color: var(--text-secondary);
  margin: 0;
}

.file-item__progress-track {
  height: 4px;
  background-color: var(--gray-4);
  border-radius: 2px;
  overflow: hidden;
  margin-top: 4px;
}

.file-item__progress-fill {
  height: 100%;
  background-color: var(--primary-active);
  transition: width 0.15s ease;
}

.file-item__speed {
  font-size: 0.7rem;
  color: var(--text-secondary);
  margin: 2px 0 0;
}

.file-item__remove,
.file-item__cancel {
  padding: 0.25rem;
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-secondary);
  transition: color 0.2s ease;
  flex-shrink: 0;
}

.file-item__remove:hover {
  color: #ef4444;
}

.file-item__cancel:hover {
  color: #ef4444;
}

.file-item__remove svg,
.file-item__cancel svg {
  width: 1.25rem;
  height: 1.25rem;
}

.progress-bar {
  position: relative;
  width: 100%;
  height: 2rem;
  background-color: var(--gray-4);
  border-radius: 0.375rem;
  overflow: hidden;
  margin-top: 1rem;
}

.progress-bar__fill {
  height: 100%;
  background-color: var(--primary-active);
  transition: width 0.3s ease;
}

.progress-bar__text {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text);
}

.upload-button {
  width: 100%;
  margin-top: 1rem;
  padding: 0.75rem 1.5rem;
  background-color: var(--primary);
  color: var(--text);
  border: none;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.upload-button:hover:not(:disabled) {
  background-color: var(--primary-hover);
}

.upload-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.upload-button--cancel {
  background-color: var(--gray-4);
}

.upload-button--cancel:hover {
  background-color: #ef4444;
}
</style>
