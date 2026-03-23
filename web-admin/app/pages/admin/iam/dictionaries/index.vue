<template>
  <UContainer class="py-10 space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="space-y-2">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">数据字典</h1>
        <p class="text-gray-600 dark:text-gray-300">
          系统级字典配置。SCRM 线索来源会读取这里的数据。
        </p>
      </div>
      <div class="flex items-center gap-2">
        <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="refreshAll">
          刷新
        </UButton>
        <UButton color="primary" icon="i-heroicons-plus" @click="openCreate()">
          新增字典项
        </UButton>
      </div>
    </div>

    <UAlert
      v-if="error"
      color="warning"
      variant="soft"
      icon="i-heroicons-exclamation-triangle"
      :description="error"
    />

    <div class="grid grid-cols-1 gap-6">
      <UCard v-for="group in groupedItems" :key="group.namespace">
        <template #header>
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-book-open" class="text-primary" />
              <span class="font-medium text-gray-900 dark:text-gray-100">{{ namespaceLabel(group.namespace) }}</span>
              <UBadge size="xs" variant="soft">{{ group.total }}</UBadge>
            </div>
            <div class="flex items-center gap-2">
              <UButton size="xs" variant="soft" color="neutral" @click="toggleNamespaceExpanded(group.namespace)">
                {{ isNamespaceExpanded(group.namespace) ? "收起" : "展开" }}
              </UButton>
              <UButton size="xs" color="primary" @click="openCreate(group.namespace)">新增</UButton>
            </div>
          </div>
        </template>

        <div v-if="isNamespaceExpanded(group.namespace)">
          <UTable
            :columns="columns"
            :data="group.pagedItems"
            :loading="loading"
            :ui="{ td: 'text-gray-700 dark:text-gray-100', th: 'text-gray-500 dark:text-gray-300' }"
          >
            <template #label-cell="{ row }">
              <span class="font-medium text-gray-800 dark:text-gray-100">{{ row.original.label }}</span>
            </template>
            <template #code-cell="{ row }">
              <span class="text-gray-600 dark:text-gray-200">{{ row.original.code }}</span>
            </template>
            <template #namespace-cell="{ row }">
              <span class="text-gray-500 dark:text-gray-300">{{ row.original.namespace }}</span>
            </template>
            <template #enabled-cell="{ row }">
              <UBadge :color="row.original.enabled ? 'success' : 'neutral'" variant="soft">
                {{ row.original.enabled ? "启用" : "停用" }}
              </UBadge>
            </template>
            <template #actions-cell="{ row }">
              <div class="flex gap-2">
                <UButton size="xs" variant="soft" @click="openEdit(row.original)">编辑</UButton>
                <UButton size="xs" variant="soft" color="error" @click="removeItem(row.original)">删除</UButton>
              </div>
            </template>
          </UTable>

          <div class="mt-4 flex flex-wrap items-center justify-between gap-3">
            <div class="text-xs text-gray-500 dark:text-gray-300">
              第 {{ group.page }} / {{ group.totalPages }} 页，共 {{ group.total }} 条
            </div>
            <div class="flex items-center gap-2">
              <USelectMenu
                :model-value="group.pageSize"
                :items="pageSizeOptions"
                value-key="value"
                label-key="label"
                class="w-24"
                :portal="false"
                :ui="{ content: 'z-[200]' }"
                @update:model-value="(value) => onNamespacePageSizeChange(group.namespace, Number(value))"
              />
              <UButton
                size="xs"
                variant="soft"
                :disabled="group.page <= 1"
                @click="prevNamespacePage(group.namespace)"
              >
                上一页
              </UButton>
              <UButton
                size="xs"
                variant="soft"
                :disabled="group.page >= group.totalPages"
                @click="nextNamespacePage(group.namespace)"
              >
                下一页
              </UButton>
            </div>
          </div>
        </div>
      </UCard>

      <UCard v-if="groupedItems.length === 0">
        <div class="py-8 text-center text-sm text-gray-500 dark:text-gray-300">暂无字典项，点击“新增字典项”开始配置。</div>
      </UCard>
    </div>

    <UModal
      v-model:open="modalOpen"
      :prevent-close="true"
      :dismissible="false"
      :modal="true"
      :ui="{ content: 'max-w-lg w-full' }"
    >
      <template #title>{{ editItem ? "编辑字典项" : "新增字典项" }}</template>
      <template #body>
        <UForm :state="form" class="space-y-4 p-4">
          <UFormField label="命名空间" required>
            <UInput v-model="form.namespace" placeholder="例如：scrm.lead.traffic_platform" />
            <div class="mt-2 flex flex-wrap gap-2">
              <UButton
                v-for="ns in quickNamespaces"
                :key="ns"
                size="xs"
                variant="soft"
                color="neutral"
                @click="form.namespace = ns"
              >
                {{ ns }}
              </UButton>
            </div>
          </UFormField>
          <UFormField label="显示名称" required>
            <UInput v-model="form.label" placeholder="例如：抖音 / 小红书 / 视频号" />
          </UFormField>
          <UFormField label="编码" required>
            <UInput v-model="form.code" placeholder="例如：douyin / xiaohongshu / wechat_channels" />
          </UFormField>
          <UFormField label="排序">
            <UInput v-model.number="form.sort" type="number" />
          </UFormField>
          <UFormField label="状态">
            <USelectMenu
              v-model="form.enabled"
              :items="enabledOptions"
              value-key="value"
              label-key="label"
              class="w-full"
              :portal="false"
              :ui="{ content: 'z-[200]' }"
            />
          </UFormField>
          <UAlert
            v-if="formError"
            color="warning"
            variant="soft"
            icon="i-heroicons-exclamation-triangle"
            :description="formError"
          />
        </UForm>
      </template>
      <template #footer>
        <div class="flex gap-2 justify-end w-full">
          <UButton variant="soft" :disabled="saving" @click="closeModal">取消</UButton>
          <UButton color="primary" :loading="saving" @click="submit">保存</UButton>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import {
  RuntimeDictionaryNamespaces,
  useRuntimeDictionaryService,
  type RuntimeDictionaryItem,
} from "~/composables/api/services/runtimeDictionary";

definePageMeta({
  layout: "default",
});

const service = useRuntimeDictionaryService();
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const formError = ref("");
const modalOpen = ref(false);
const editItem = ref<RuntimeDictionaryItem | null>(null);
const items = ref<RuntimeDictionaryItem[]>([]);

const columns = [
  { accessorKey: "label", header: "名称" },
  { accessorKey: "code", header: "编码" },
  { accessorKey: "sort", header: "排序" },
  { accessorKey: "enabled", header: "状态" },
  { accessorKey: "actions", header: "操作" },
] satisfies any;

const enabledOptions = [
  { label: "启用", value: true },
  { label: "停用", value: false },
];

const form = reactive({
  namespace: RuntimeDictionaryNamespaces.leadTrafficPlatform,
  label: "",
  code: "",
  sort: 100,
  enabled: true,
});

const pageSizeOptions = [
  { label: "10/页", value: 10 },
  { label: "20/页", value: 20 },
  { label: "50/页", value: 50 },
];

const namespacePage = reactive<Record<string, number>>({});
const namespacePageSize = reactive<Record<string, number>>({});
const namespaceExpanded = reactive<Record<string, boolean>>({});

const quickNamespaces = computed(() => {
  const set = new Set<string>([
    RuntimeDictionaryNamespaces.leadTrafficPlatform,
    RuntimeDictionaryNamespaces.leadTrafficSource,
  ]);
  items.value.forEach((item) => set.add(item.namespace));
  return Array.from(set);
});

const groupedItems = computed(() => {
  const groups = new Map<string, RuntimeDictionaryItem[]>();
  items.value.forEach((item) => {
    const key = item.namespace || "default";
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key)?.push(item);
  });
  return Array.from(groups.entries())
    .map(([namespace, data]) => {
      const sortedItems = data.sort((a, b) => (a.sort || 100) - (b.sort || 100));
      const total = sortedItems.length;
      const pageSize = namespacePageSize[namespace] || 10;
      const totalPages = Math.max(1, Math.ceil(total / pageSize));
      const page = Math.min(namespacePage[namespace] || 1, totalPages);
      const start = (page - 1) * pageSize;
      return {
        namespace,
        total,
        page,
        pageSize,
        totalPages,
        pagedItems: sortedItems.slice(start, start + pageSize),
      };
    })
    .sort((a, b) => a.namespace.localeCompare(b.namespace));
});

const ensureNamespacePageState = (namespace: string) => {
  if (!namespacePage[namespace] || namespacePage[namespace] < 1) {
    namespacePage[namespace] = 1;
  }
  if (!namespacePageSize[namespace] || namespacePageSize[namespace] < 1) {
    namespacePageSize[namespace] = 10;
  }
  if (namespaceExpanded[namespace] === undefined) {
    namespaceExpanded[namespace] = true;
  }
};

const isNamespaceExpanded = (namespace: string) => {
  ensureNamespacePageState(namespace);
  return namespaceExpanded[namespace];
};

const toggleNamespaceExpanded = (namespace: string) => {
  ensureNamespacePageState(namespace);
  namespaceExpanded[namespace] = !namespaceExpanded[namespace];
};

const prevNamespacePage = (namespace: string) => {
  ensureNamespacePageState(namespace);
  namespacePage[namespace] = Math.max(1, namespacePage[namespace] - 1);
};

const nextNamespacePage = (namespace: string) => {
  ensureNamespacePageState(namespace);
  const group = groupedItems.value.find((item) => item.namespace === namespace);
  const totalPages = group?.totalPages || 1;
  namespacePage[namespace] = Math.min(totalPages, namespacePage[namespace] + 1);
};

const onNamespacePageSizeChange = (namespace: string, size: number) => {
  ensureNamespacePageState(namespace);
  namespacePageSize[namespace] = size > 0 ? size : 10;
  namespacePage[namespace] = 1;
};

const namespaceLabel = (namespace: string) => {
  if (namespace === RuntimeDictionaryNamespaces.leadTrafficPlatform) return "线索流量平台";
  if (namespace === RuntimeDictionaryNamespaces.leadTrafficSource) return "线索流量来源";
  return namespace;
};

const refreshAll = async () => {
  loading.value = true;
  error.value = "";
  try {
    const resp = await service.listDictionaries();
    items.value = ((resp as any)?.data?.items || []) as RuntimeDictionaryItem[];
    items.value.forEach((item) => {
      ensureNamespacePageState(item.namespace);
    });
  } catch (err: any) {
    error.value = err?.message || "加载字典失败";
  } finally {
    loading.value = false;
  }
};

const openCreate = (namespace?: string) => {
  editItem.value = null;
  form.namespace = namespace || RuntimeDictionaryNamespaces.leadTrafficPlatform;
  form.label = "";
  form.code = "";
  form.sort = 100;
  form.enabled = true;
  formError.value = "";
  modalOpen.value = true;
};

const openEdit = (item: RuntimeDictionaryItem) => {
  editItem.value = item;
  form.namespace = item.namespace;
  form.label = item.label;
  form.code = item.code;
  form.sort = item.sort || 100;
  form.enabled = !!item.enabled;
  formError.value = "";
  modalOpen.value = true;
};

const closeModal = () => {
  if (saving.value) return;
  modalOpen.value = false;
};

const submit = async () => {
  const namespace = form.namespace.trim().toLowerCase();
  if (!namespace || !form.label.trim() || !form.code.trim()) {
    formError.value = "命名空间、显示名称、编码为必填项";
    return;
  }
  saving.value = true;
  formError.value = "";
  try {
    if (editItem.value) {
      await service.updateDictionaryItem(editItem.value.item_id, {
        namespace,
        label: form.label.trim(),
        code: form.code.trim().toLowerCase(),
        sort: form.sort,
        enabled: form.enabled,
      });
    } else {
      await service.createDictionaryItem({
        namespace,
        label: form.label.trim(),
        code: form.code.trim().toLowerCase(),
        sort: form.sort,
        enabled: form.enabled,
      });
    }
    modalOpen.value = false;
    await refreshAll();
  } catch (err: any) {
    formError.value = err?.message || "保存失败";
  } finally {
    saving.value = false;
  }
};

const removeItem = async (item: RuntimeDictionaryItem) => {
  if (!process.client) return;
  if (!window.confirm("确认删除该字典项？")) return;
  loading.value = true;
  error.value = "";
  try {
    await service.deleteDictionaryItem(item.item_id, item.namespace);
    await refreshAll();
  } catch (err: any) {
    error.value = err?.message || "删除失败";
  } finally {
    loading.value = false;
  }
};

onMounted(async () => {
  await refreshAll();
});
</script>
