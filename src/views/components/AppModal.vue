<template>
  <Teleport to="body">
    <Transition name="modal-backdrop">
      <div
        v-if="show"
        class="modal-backdrop fixed inset-0 z-50 overflow-y-auto"
        @click.self="onBackdropClick"
      >
        <div class="modal-backdrop-inner flex min-h-full items-center justify-center p-4">
          <Transition name="modal-panel" appear>
            <div
              v-if="show"
              class="modal-panel rounded-lg relative flex flex-col"
              :style="{ width }"
            >
              <div v-if="title || $slots.header" class="modal-header flex items-center justify-between">
                <slot name="header">
                  <h3 class="text-lg font-bold">{{ title }}</h3>
                </slot>
              </div>

              <div class="modal-body">
                <slot />
              </div>

              <div v-if="$slots.footer" class="modal-footer flex justify-end gap-2">
                <slot name="footer" />
              </div>

              <button
                v-if="!hideClose"
                class="modal-close-btn absolute"
                aria-label="Close"
                @click="emit('close')"
              >
                ✕
              </button>
            </div>
          </Transition>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'

const props = withDefaults(defineProps<{
  show: boolean
  title?: string
  width?: string
  hideClose?: boolean
  closeOnBackdrop?: boolean
}>(), {
  title: '',
  width: '24rem',
  hideClose: false,
  closeOnBackdrop: true,
})

const emit = defineEmits<{ close: [] }>()

function onBackdropClick() {
  if (props.closeOnBackdrop) emit('close')
}

function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.show) emit('close')
}

onMounted(() => document.addEventListener('keydown', onKeyDown))
onUnmounted(() => document.removeEventListener('keydown', onKeyDown))
</script>

<style scoped>
.modal-backdrop {
  background-color: rgba(0, 0, 0, 0.5);
}

.modal-panel {
  background-color: var(--gray-2);
  color: var(--text);
  padding: var(--space-6);
  box-shadow: var(--shadow-lg);
}

.modal-header {
  margin-bottom: var(--space-4);
}

.modal-footer {
  margin-top: var(--space-6);
  gap: var(--space-2);
}

.modal-close-btn {
  top: var(--space-2);
  right: var(--space-2);
  color: var(--text-tertiary);
  font-size: 0.9rem;
  line-height: 1;
  padding: var(--space-1);
}

.modal-close-btn:hover {
  color: var(--text);
}

/* Backdrop fade */
.modal-backdrop-enter-active,
.modal-backdrop-leave-active {
  transition: opacity 0.15s ease;
}
.modal-backdrop-enter-from,
.modal-backdrop-leave-to {
  opacity: 0;
}

/* Panel fade + scale */
.modal-panel-enter-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.modal-panel-leave-active {
  transition: opacity 0.1s ease, transform 0.1s ease;
}
.modal-panel-enter-from,
.modal-panel-leave-to {
  opacity: 0;
  transform: scale(0.96) translateY(4px);
}
</style>
