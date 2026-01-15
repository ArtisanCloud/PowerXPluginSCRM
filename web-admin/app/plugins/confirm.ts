import { defineNuxtPlugin } from '#app'
import useConfirm from '~/composables/useConfirm'

export default defineNuxtPlugin(() => {
  const confirmInstance = useConfirm()
  
  return {
    provide: {
      confirm: confirmInstance.open
    }
  }
})