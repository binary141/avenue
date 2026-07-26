<template>
  <AppModal
    :show="visible"
    :title="options.title || 'Are you sure?'"
    width="24rem"
    @close="resolveConfirm(false)"
  >
    <p class="text-sm" style="color: var(--text-secondary);">{{ options.message }}</p>

    <template #footer>
      <AppButton variant="secondary" size="sm" @click="resolveConfirm(false)">
        {{ options.cancelLabel || 'Cancel' }}
      </AppButton>
      <AppButton :variant="options.danger ? 'danger' : 'primary'" size="sm" @click="resolveConfirm(true)">
        {{ options.confirmLabel || 'Confirm' }}
      </AppButton>
    </template>
  </AppModal>
</template>

<script setup lang="ts">
import AppModal from './AppModal.vue'
import AppButton from './AppButton.vue'
import { useConfirmDialogState, resolveConfirm } from '@/composables/confirm'

const { visible, options } = useConfirmDialogState()
</script>
