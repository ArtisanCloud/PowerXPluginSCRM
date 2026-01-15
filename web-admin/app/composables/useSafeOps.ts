import { ref } from 'vue'

import type { SafeOpPayload, JobRun } from '~/stores/dev-console/troubleshoot'
import { useDevConsoleTroubleshootStore } from '~/stores/dev-console/troubleshoot'

export function useSafeOps() {
  const store = useDevConsoleTroubleshootStore()
  const toast = useToast()
  const loading = ref(false)

  async function execute(payload: SafeOpPayload): Promise<JobRun | undefined> {
    loading.value = true
    try {
      const run = await store.executeSafeOp(payload)
      toast.add({ title: '操作已提交', color: 'green' })
      return run
    } catch (err: any) {
      toast.add({ title: '操作失败', description: err?.message ?? '请稍后重试', color: 'red' })
      throw err
    } finally {
      loading.value = false
    }
  }

  return { execute, loading }
}
