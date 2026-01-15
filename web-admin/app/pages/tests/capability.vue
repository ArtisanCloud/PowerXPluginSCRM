<script setup lang="ts">
import { computed, ref } from 'vue'
import { usePowerXCapability } from '~/composables/usePowerXCapability'
import { useCapabilityLab } from '~/composables/useCapabilityLab'

definePageMeta({
  title: 'Capability Invocation Playground'
})

type CapabilityMode = 'success' | 'fail' | 'mock'

const { invoke, loading, lastTraceId, lastError } = usePowerXCapability()
const lastStatus = ref('idle')
const lastPayload = ref<string>('{}')
const {
  invokeCapability: invokeCapabilityLab,
  result: labResult,
  errorMessage: labErrorMessage,
  lastTraceId: labTraceId
} = useCapabilityLab()
const localStatus = ref('idle')
const localPayloadPreview = ref('{}')

async function triggerInvocation(mode: CapabilityMode) {
  lastStatus.value = 'pending'
  lastPayload.value = '{}'
  try {
    const response = await invoke('com.corex.media.assets.manage', 'TestAction', { mode })
    lastStatus.value = response.status || 'ok'
    lastPayload.value = JSON.stringify(response.data ?? {}, null, 2)
  } catch (error: any) {
    lastStatus.value = 'error'
    lastPayload.value =
      error?.message || 'Capability invocation failed - see toast for traceId and reason.'
  }
}

async function triggerLocalDebug() {
  localStatus.value = 'pending'
  const payload = {
    method: 'POST',
    endpoint: '/tests/local-api',
    headers: {
      'X-Test-Header': 'local-debug'
    },
    query: {
      page: 1,
      tag: 'demo'
    },
    body: {
      message: 'hello from local debug'
    }
  }
  const payloadText = JSON.stringify(payload, null, 2)
  localPayloadPreview.value = payloadText
  try {
    await invokeCapabilityLab({
      capabilityId: 'com.demo.local',
      action: 'LocalCall',
      payload,
      payloadText,
      mode: 'local',
      preferredProtocol: 'rest'
    })
    localStatus.value = labResult.value?.status || 'completed'
  } catch {
    localStatus.value = 'error'
  }
}

const localTraceText = computed(() => labTraceId.value || 'pending')
const localErrorText = computed(() => labErrorMessage.value || 'none')
const localResponsePreview = computed(() =>
  labResult.value?.data ? JSON.stringify(labResult.value.data, null, 2) : '{}'
)
</script>

<template>
  <div class="capability-playground" data-testid="capability-playground">
    <div class="header">
      <h1>Capability Invocation Playground</h1>
      <p>使用内置 Bridge 验证不同响应状态，配合 Playwright 场景测试 Skeleton 行为。</p>
    </div>
    <div class="actions">
      <button
        data-testid="trigger-success"
        :disabled="loading"
        type="button"
        @click="triggerInvocation('success')"
      >
        触发成功响应
      </button>
      <button
        data-testid="trigger-fail"
        :disabled="loading"
        type="button"
        class="warn"
        @click="triggerInvocation('fail')"
      >
        触发失败响应
      </button>
      <button
        data-testid="trigger-mock"
        :disabled="loading"
        type="button"
        class="mock"
        @click="triggerInvocation('mock')"
      >
        触发 Mock 模式
      </button>
      <button
        data-testid="trigger-local-debug"
        :disabled="loading"
        type="button"
        class="local"
        @click="triggerLocalDebug"
      >
        本地调试（REST）
      </button>
    </div>

    <div class="state-grid">
      <div>
        <span>Trace ID</span>
        <strong data-testid="trace-output">{{ lastTraceId || 'pending' }}</strong>
      </div>
      <div>
        <span>状态</span>
        <strong data-testid="status-indicator">{{ lastStatus }}</strong>
      </div>
      <div>
        <span>错误</span>
        <strong data-testid="error-indicator">{{ lastError ? 'error' : 'none' }}</strong>
      </div>
    </div>

    <section class="payload">
      <header>
        <h2>响应数据</h2>
      </header>
      <pre data-testid="payload-viewer">{{ lastPayload }}</pre>
    </section>

    <section class="payload">
      <header>
        <h2>本地调试</h2>
      </header>
      <div class="local-state">
        <div>
          <span>Trace ID</span>
          <strong data-testid="local-trace">{{ localTraceText }}</strong>
        </div>
        <div>
          <span>状态</span>
          <strong data-testid="local-status">{{ localStatus }}</strong>
        </div>
        <div>
          <span>错误</span>
          <strong data-testid="local-error">{{ localErrorText }}</strong>
        </div>
      </div>
      <div class="local-panels">
        <div>
          <h3>请求 Payload</h3>
          <pre data-testid="local-request-preview">{{ localPayloadPreview }}</pre>
        </div>
        <div>
          <h3>响应数据</h3>
          <pre data-testid="local-response-preview">{{ localResponsePreview }}</pre>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.capability-playground {
  max-width: 960px;
  margin: 0 auto;
  padding: 2rem 1.5rem 3rem;
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.header h1 {
  font-size: 1.75rem;
  margin-bottom: 0.25rem;
}

.header p {
  color: #4b5563;
  font-size: 0.95rem;
}

.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.actions button {
  padding: 0.75rem 1.5rem;
  border-radius: 0.5rem;
  border: 1px solid #cbd5f5;
  background: #eef2ff;
  color: #1d4ed8;
  font-weight: 600;
  cursor: pointer;
}

.actions button.warn {
  border-color: #fecaca;
  background: #fee2e2;
  color: #b91c1c;
}

.actions button.mock {
  border-color: #fed7aa;
  background: #ffedd5;
  color: #c2410c;
}

.actions button.local {
  border-color: #c7f9cc;
  background: #ecfccb;
  color: #15803d;
}

.actions button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.state-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 1rem;
}

.state-grid div {
  padding: 1rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.75rem;
  background: #fff;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.state-grid span {
  font-size: 0.85rem;
  color: #6b7280;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.state-grid strong {
  font-size: 1rem;
  color: #111827;
  word-break: break-all;
}

.payload {
  border: 1px solid #e5e7eb;
  border-radius: 0.75rem;
  background: #f9fafb;
}

.payload header {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid #e5e7eb;
}

.payload pre {
  margin: 0;
  padding: 1rem;
  font-size: 0.9rem;
  overflow-x: auto;
  line-height: 1.4;
  color: #111827;
}

.local-state {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 0.75rem;
  padding: 1rem;
  border-bottom: 1px solid #e5e7eb;
  background: #fff;
}

.local-state div {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.local-state span {
  font-size: 0.8rem;
  color: #6b7280;
}

.local-state strong {
  font-size: 0.95rem;
  color: #111827;
  word-break: break-all;
}

.local-panels {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1rem;
  padding: 1rem;
}

.local-panels h3 {
  margin: 0 0 0.5rem;
  font-size: 0.95rem;
  color: #374151;
}

.local-panels pre {
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 0.5rem;
  padding: 0.75rem;
  min-height: 180px;
}
</style>
