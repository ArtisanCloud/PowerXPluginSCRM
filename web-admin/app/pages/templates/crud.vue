<template>
  <UContainer class="py-10 space-y-8">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ $t("templates.crud.title") }}
        </h1>
        <p class="text-gray-600 dark:text-gray-300">
          {{ $t("templates.crud.description") }}
        </p>
      </div>
      <UButton
        icon="i-heroicons-plus"
        color="primary"
        :disabled="isDelegatedReadOnly"
        @click="startCreate"
      >
        {{ $t("templates.crud.create") }}
      </UButton>
    </div>

    <UAlert
      v-if="isDelegatedReadOnly"
      color="info"
      variant="soft"
      icon="i-heroicons-information-circle"
    >
      <template #title>{{ $t("templates.crud.readonlyTitle") }}</template>
      <template #description>
        {{ $t("templates.crud.readonlyDescription") }}
      </template>
    </UAlert>

    <TemplateFormModal
      v-if="showFormModal"
      v-model="showFormModal"
      :title="modalTitle"
      :submit-label="submitLabel"
      :initial-value="formSnapshot"
      :loading="saving"
      @submit="handleSubmit"
    />

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium">{{ $t("templates.crud.listTitle") }}</span>
          <UBadge variant="soft" color="primary">{{ templates.length }}</UBadge>
        </div>
      </template>
      <UTable
        :columns="tableColumns"
        :data="templates"
        :loading="loading"
        :ui="{ table: 'min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-700' }"
      >
        <!-- 注意：v3 是 -cell，不是 -data；row.original 才是你的对象 -->
        <template #description-header="{ column }">
          <span class="block w-64">
            {{ column.columnDef.header }}
          </span>
        </template>
        <template #content-header="{ column }">
          <span class="block w-80">
            {{ column.columnDef.header }}
          </span>
        </template>
        <template #description-cell="{ row }">
          <div class="description-cell">
            {{ row.original.description }}
          </div>
        </template>
        <template #content-cell="{ row }">
          <div class="content-cell">
            {{ row.original.content }}
          </div>
        </template>
        <template #actions-cell="{ row }">
          <div class="flex gap-2">
            <UButton
              size="xs"
              variant="soft"
              icon="i-heroicons-pencil"
              :disabled="isDelegatedReadOnly"
              @click="startEdit(row.original)"
            >
              {{ $t('common.edit') }}
            </UButton>
            <UButton
              size="xs"
              variant="soft"
              color="error"
              icon="i-heroicons-trash"
              :disabled="isDelegatedReadOnly"
              @click="confirmDelete(row.original)"
            >
              {{ $t('common.delete') }}
            </UButton>
          </div>
        </template>
      </UTable>

    </UCard>

    <ConfirmDialog
      v-model="deleteDialog"
      :title="$t('templates.crud.deleteTitle')"
      :description="$t('templates.crud.deleteDescription')"
      :message="$t('templates.crud.deleteConfirm', { name: selectedTemplate?.name || '' })"
      confirm-color="error"
      :confirm-text="$t('common.delete')"
      :loading="deleting"
      @confirm="performDelete"
      @cancel="handleDeleteCancel"
    />

    <ToastAlert
      v-model="toast.visible"
      :title="toast.title"
      :message="toast.message"
      :color="toast.color"
      :duration="toast.duration"
    />
  </UContainer>
</template>

<script setup lang="ts">
import ConfirmDialog from "~/components/ConfirmDialog.vue"
import ToastAlert from "~/components/ToastAlert.vue"
import { useTemplateApi } from "~/composables/api/useTemplate"
import type { Template } from "~/composables/api/useTemplate"
import TemplateFormModal from "~/components/templates/TemplateFormModal.vue"
import { nextTick } from "vue"
import { useI18n } from "vue-i18n"

type TemplateFormState = {
  name: string
  description: string
  content: string
}

const {
  listTemplates,
  createTemplate: createTemplateApi,
  updateTemplate: updateTemplateApi,
  deleteTemplate: deleteTemplateApi,
} = useTemplateApi()

const templates = ref<Template[]>([])
const loading = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)
const showFormModal = ref(false)
const deleteDialog = ref(false)
const deleting = ref(false)
const selectedTemplate = ref<Template | null>(null)

type ToastColor = "primary" | "secondary" | "success" | "info" | "warning" | "error" | "neutral"

const toast = reactive({
  visible: false,
  title: "",
  message: "",
  color: "primary" as ToastColor,
  duration: 3000,
})

const { t } = useI18n()

const runtimeConfig = useRuntimeConfig()
const isDelegatedReadOnly = computed(() => {
  const value = runtimeConfig.public?.insidePowerX
  if (value === true) return true
  if (typeof value === "string") {
    return value === "true" || value === "1"
  }
  return false
})

const tableColumns = computed(() => [
  { accessorKey: 'name', header: t('templates.crud.fields.name') },
  { accessorKey: 'description', header: t('templates.crud.fields.description') },
  { accessorKey: 'content', header: t('templates.crud.fields.content') },
  { id: 'actions', header: '' },
] satisfies any)

const defaultFormValue = (): TemplateFormState => ({
  name: "",
  description: "",
  content: "",
})

const form = reactive<TemplateFormState>(defaultFormValue())

const makeLogHandlers = (action: string, context: Record<string, any> = {}) => ({
  onRequest({ request: _request, options: _options }: any) {
    // console.debug(`[templates/crud] ${action} request`, {
    //   baseURL: templateApiBase,
    //   request,
    //   options,
    //   context,
    // })
  },
  onResponse({ response: _response }: any) {
    // console.debug(`[templates/crud] ${action} response`, {
    //   status: response.status,
    //   data: response._data,
    //   headers: typeof response.headers?.get === "function"
    //     ? {
    //         "x-request-id": response.headers.get("x-request-id"),
    //       }
    //     : undefined,
    //   context,
    // })
  },
  onResponseError({ response }: any) {
    console.error(`[templates/crud] ${action} response error`, {
      status: response?.status,
      data: response?._data,
      context,
    })
  },
})

const modalTitle = computed(() =>
  editingId.value ? t("templates.crud.actions.update") : t("templates.crud.create")
)

const submitLabel = computed(() =>
  editingId.value ? t("templates.crud.actions.update") : t("templates.crud.actions.save")
)

const formSnapshot = computed(() => ({
  name: form.name,
  description: form.description,
  content: form.content,
}))

const fetchTemplates = async () => {
  loading.value = true
  try {
    const query = { page: 1, page_size: 50 }
    // console.debug('[templates/crud] fetching templates', {
    //   baseURL: templateApiBase,
    //   path: 'templates',
    //   query,
    // })

    const res = await listTemplates(query.page, query.page_size, "", makeLogHandlers("templates:list", { query }))
    // console.log(res?.success , res.data , Array.isArray(res.data.list),res)
    if (res?.success && res.data && Array.isArray(res.data.list)) {
      templates.value = res.data.list
      // console.debug('[templates/crud] templates loaded', {
      //   count: templates.value.length,
      // })
    } else {
      templates.value = []
      console.warn('[templates/crud] templates response unexpected', res)
    }
  } catch (error) {
    console.error("[templates/crud] Failed to load templates", error)
    templates.value = []
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  editingId.value = null
  Object.assign(form, defaultFormValue())
}

const openFormModal = () => {
  showFormModal.value = true
}

const closeFormModal = () => {
  showFormModal.value = false
}

const ensureWritable = () => {
  if (!isDelegatedReadOnly.value) {
    return true
  }
  showToast({
    title: t("templates.crud.readonlyTitle"),
    message: t("templates.crud.readonlyToast"),
    color: "warning",
    duration: 4500,
  })
  return false
}

const startCreate = () => {
  if (!ensureWritable()) return
  resetForm()
  openFormModal()
}

const startEdit = (tpl: Template) => {
  if (!ensureWritable()) return
  editingId.value = tpl.id
  Object.assign(form, {
    name: tpl.name,
    description: tpl.description,
    content: tpl.content,
  })
  openFormModal()
}

const handleSubmit = async (payload: { name: string; description: string; content: string }) => {
  if (isDelegatedReadOnly.value) {
    ensureWritable()
    return
  }
  if (!payload.name || !payload.description || !payload.content) {
    return
  }
  saving.value = true
  const isUpdate = Boolean(editingId.value)
  try {
    if (editingId.value) {
      const res = await updateTemplateApi(
        editingId.value,
        payload,
        makeLogHandlers("templates:update", { id: editingId.value, payload })
      )
      if (!res?.success) {
        throw new Error(res?.message || "Update template failed")
      }
    } else {
      const res = await createTemplateApi(
        payload,
        makeLogHandlers("templates:create", { payload })
      )
      if (!res?.success) {
        throw new Error(res?.message || "Create template failed")
      }
    }
    await fetchTemplates()
    closeFormModal()
    resetForm()
    showToast({
      title: isUpdate ? t("templates.crud.actions.update") : t("templates.crud.create"),
      message: isUpdate ? t("message.saveSuccess") : t("message.templateCreated"),
      color: "success",
    })
  } catch (error: any) {
    console.error("[templates/crud] Failed to save template", error)
    showToast({
      title: t("message.error"),
      message: error?.message || t("message.error"),
      color: "error",
      duration: 5000,
    })
  } finally {
    saving.value = false
  }
}

const confirmDelete = (tpl: Template) => {
  if (!ensureWritable()) return
  selectedTemplate.value = tpl
  deleteDialog.value = true
}

const performDelete = async () => {
  if (!selectedTemplate.value || deleting.value) return
  if (isDelegatedReadOnly.value) {
    ensureWritable()
    return
  }
  deleting.value = true
  try {
    const res = await deleteTemplateApi(
      selectedTemplate.value.id,
      makeLogHandlers("templates:delete", { id: selectedTemplate.value.id })
    )
    if (!res?.success) {
      throw new Error(res?.message || "Delete template failed")
    }
    await fetchTemplates()
    deleteDialog.value = false
    showToast({
      title: t("templates.crud.deleteTitle"),
      message: t("message.deleteSuccess"),
      color: "success",
    })
  } catch (error: any) {
    console.error("[templates/crud] Failed to delete template", error)
    showToast({
      title: t("message.error"),
      message: error?.message || t("message.error"),
      color: "error",
      duration: 5000,
    })
  } finally {
    deleting.value = false
  }
}

const handleDeleteCancel = () => {
  deleteDialog.value = false
}

watch(deleteDialog, (isOpen) => {
  if (!isOpen) {
    selectedTemplate.value = null
    deleting.value = false
  }
})

watch(isDelegatedReadOnly, (readonlyMode) => {
  if (!readonlyMode) {
    return
  }
  showFormModal.value = false
  deleteDialog.value = false
})

const normalizeToString = (value?: string | number | null) => {
  if (value === null || value === undefined) {
    return ""
  }
  return typeof value === "string" ? value : String(value)
}

const showToast = ({
  title,
  message,
  color = "primary",
  duration = 3000,
}: {
  title?: string
  message: string | number
  color?: ToastColor
  duration?: number
}) => {
  const normalizedTitle = normalizeToString(title)
  const normalizedMessage = normalizeToString(message)
  toast.title = normalizedTitle
  toast.message = normalizedMessage
  toast.color = color
  toast.duration = duration
  if (!normalizedTitle && !normalizedMessage) {
    toast.visible = false
    return
  }
  toast.visible = false
  nextTick(() => {
    toast.visible = true
  })
}

onMounted(() => {
  fetchTemplates()
})
</script>

<style scoped>
.description-cell,
.content-cell {
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.description-cell {
  max-width: 16rem;
}

.content-cell {
  max-width: 24rem;
}
</style>
