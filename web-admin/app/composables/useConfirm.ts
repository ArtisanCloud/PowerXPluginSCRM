import { ref } from 'vue'

interface ConfirmOptions {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  type?: 'info' | 'warning' | 'error' | 'success'
}

export default function useConfirm() {
  const isOpen = ref(false)
  const options = ref<ConfirmOptions>({
    title: '',
    message: '',
    confirmText: 'Confirm',
    cancelText: 'Cancel',
    type: 'info'
  })
  
  let resolvePromise: (value: boolean) => void

  const open = (opts: ConfirmOptions): Promise<boolean> => {
    options.value = {
      ...options.value,
      ...opts
    }
    isOpen.value = true
    
    return new Promise<boolean>((resolve) => {
      resolvePromise = resolve
    })
  }

  const confirm = () => {
    isOpen.value = false
    resolvePromise(true)
  }

  const cancel = () => {
    isOpen.value = false
    resolvePromise(false)
  }

  return {
    isOpen,
    options,
    open,
    confirm,
    cancel
  }
}