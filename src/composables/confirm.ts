import { ref } from 'vue'

export interface ConfirmOptions {
  title?: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  danger?: boolean
}

const visible = ref(false)
const options = ref<ConfirmOptions>({ message: '' })
let resolver: ((value: boolean) => void) | null = null

// Backs a single app-wide <ConfirmDialog/> (mounted once in App.vue) so
// every destructive action asks the same way instead of each view picking
// its own pattern (window.confirm, no confirmation at all, ad hoc modals).
export function useConfirmDialogState() {
  return { visible, options }
}

export function confirm(opts: ConfirmOptions): Promise<boolean> {
  options.value = opts
  visible.value = true

  return new Promise<boolean>((resolve) => {
    resolver = resolve
  })
}

export function resolveConfirm(result: boolean) {
  visible.value = false
  resolver?.(result)
  resolver = null
}
