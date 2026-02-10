<template>
  <UContainer class="py-6 space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-xl font-semibold">组织架构预览管理</h1>
        <p class="text-sm text-gray-600 dark:text-slate-300">查看组织来源部门、成员与主组织映射。</p>
      </div>
      <UButton color="primary" @click="goSync">进入同步页</UButton>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium text-gray-700 dark:text-slate-200">选择渠道账号</span>
          <UBadge variant="soft" color="primary">必选</UBadge>
        </div>
      </template>
      <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
        <UFormField label="渠道" required>
          <USelectMenu
            v-model="selectedChannel"
            :items="channelOptions"
            value-key="value"
            label-key="label"
            searchable
            placeholder="选择渠道"
            :portal="false"
            :ui="{ content: 'z-[200]' }"
          />
        </UFormField>
        <UFormField label="应用类型" required>
          <USelectMenu
            v-model="selectedAppType"
            :items="appTypeOptions"
            value-key="value"
            label-key="label"
            searchable
            placeholder="选择应用类型"
            :portal="false"
            :ui="{ content: 'z-[200]' }"
            :disabled="!selectedChannel"
          />
        </UFormField>
        <UFormField label="账号" required>
          <USelectMenu
            v-model="selectedAccountUUID"
            :items="accountOptions"
            value-key="value"
            label-key="label"
            option-attribute="fullLabel"
            searchable
            placeholder="选择账号"
            :portal="false"
            :ui="{ content: 'z-[200] w-72' }"
            :disabled="!selectedAppType"
          />
        </UFormField>
      </div>
      <div class="mt-4 flex flex-wrap items-center gap-3">
        <UButton variant="soft" color="primary" :loading="loading" @click="refreshData">
          刷新组织数据
        </UButton>
        <span class="text-xs text-gray-600 dark:text-slate-300">仅展示已同步数据；同步动作请进入同步页。</span>
      </div>
    </UCard>

    <div class="grid grid-cols-1 gap-6 lg:grid-cols-[280px_1fr]">
      <UCard class="h-full">
        <template #header>
          <div class="flex items-center justify-between">
            <span class="font-medium text-gray-700 dark:text-slate-200">来源部门</span>
            <UBadge variant="soft" color="neutral">{{ sourceUnits.length }}</UBadge>
          </div>
        </template>
        <div class="space-y-3">
          <div v-if="sourceUnitTree.length === 0" class="text-xs text-gray-600 dark:text-slate-300">
            暂无来源部门数据。
          </div>
          <UTree
            v-else
            :items="sourceUnitTree"
            :model-value="selectedUnitUUID"
            v-model:expanded="expandedUnitUUIDs"
            expanded-icon="i-heroicons-folder-open"
            collapsed-icon="i-heroicons-folder"
            class="org-sync-tree"
          >
            <template #item-leading="{ item, expanded }">
              <UIcon
                :name="item.hasChildren ? (expanded ? 'i-heroicons-folder-open' : 'i-heroicons-folder') : 'i-heroicons-document'"
                :class="[
                  'h-4 w-4',
                  item.hasChildren ? 'text-amber-500 dark:text-amber-300' : 'text-gray-400 dark:text-slate-200',
                ]"
              />
            </template>
            <template #item-label="{ item }">
              <button
                class="truncate text-left text-sm text-gray-700 dark:text-slate-100"
                @click.stop="handleNodeClick(item)"
              >
                {{ item.label }}
              </button>
            </template>
          </UTree>
        </div>
      </UCard>

      <UCard>
        <template #header>
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="flex items-center gap-2">
              <span class="font-medium text-gray-700 dark:text-slate-200">来源成员</span>
              <UBadge variant="soft" color="neutral">{{ sourceMembers.length }}</UBadge>
              <UBadge v-if="selectedUnitLabel" variant="soft" color="primary">{{ selectedUnitLabel }}</UBadge>
            </div>
            <div class="flex items-center gap-2">
              <UInput v-model="memberKeyword" placeholder="搜索成员" class="min-w-[200px]" />
              <UButton size="xs" variant="soft" color="primary" :loading="loadingMembers" @click="loadSourceMembers">
                搜索
              </UButton>
            </div>
          </div>
        </template>
        <UTable
          :columns="memberColumns"
          :data="sourceMembers"
          :loading="loadingMembers"
          :ui="{ table: 'min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-700' }"
        >
          <template #name-cell="{ row }">
            <span class="text-sm text-gray-700 dark:text-slate-200">{{ resolveMemberName(row.original) }}</span>
          </template>
          <template #position-cell="{ row }">
            <span class="text-sm text-gray-600 dark:text-slate-300">{{ resolveMemberPosition(row.original) }}</span>
          </template>
          <template #department-cell="{ row }">
            <span class="text-sm text-gray-600 dark:text-slate-300">{{ resolveMemberDepartment(row.original) }}</span>
          </template>
          <template #phone-cell="{ row }">
            <span class="text-sm text-gray-600 dark:text-slate-300">{{ resolveMemberPhone(row.original) }}</span>
          </template>
          <template #biz_mail-cell="{ row }">
            <span class="text-sm text-gray-600 dark:text-slate-300">{{ resolveMemberBizMail(row.original) }}</span>
          </template>
          <template #email-cell="{ row }">
            <span class="text-sm text-gray-600 dark:text-slate-300">{{ resolveMemberEmail(row.original) }}</span>
          </template>
          <template #status-cell="{ row }">
            <UBadge variant="soft" color="neutral">{{ row.original.status }}</UBadge>
          </template>
        </UTable>
        <div v-if="!loadingMembers && sourceMembers.length === 0" class="text-xs text-gray-600 dark:text-slate-300 mt-3">
          暂无来源成员。
        </div>
      </UCard>
    </div>

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
import { computed, onMounted, ref, watch } from "vue";
import ToastAlert from "~/components/ToastAlert.vue";
import {
  type OrgSyncSourceMember,
  type OrgSyncSourceUnit,
  useOrgSyncService,
} from "~/composables/api/services/orgSync";
import {
  type ChannelAccount,
  type ChannelSchemaDocument,
  useSocialChannelGovernanceService,
} from "~/composables/api/services/socialChannelGovernance";

definePageMeta({
  layout: "default",
});

const selectedChannel = ref("");
const selectedAppType = ref("");
const selectedAccountUUID = ref("");
const loading = ref(false);
const loadingMembers = ref(false);
const sourceMembers = ref<OrgSyncSourceMember[]>([]);
const sourceUnits = ref<OrgSyncSourceUnit[]>([]);
const channelAccounts = ref<ChannelAccount[]>([]);
const channelSchema = ref<ChannelSchemaDocument | null>(null);
const selectedUnitUUID = ref("");
const expandedUnitUUIDs = ref<string[]>([]);
const memberKeyword = ref("");

const toast = ref({
  visible: false,
  title: "",
  message: "",
  color: "primary" as
    | "primary"
    | "secondary"
    | "success"
    | "info"
    | "warning"
    | "error"
    | "neutral",
  duration: 3000,
});

const channelOptions = computed(() => {
  const set = new Set<string>();
  channelAccounts.value.forEach((acc) => set.add(acc.channel_code));
  return Array.from(set).map((value) => ({ label: channelLabel(value), value }));
});

const appTypeOptions = computed(() => {
  if (!selectedChannel.value) return [];
  const set = new Set<string>();
  channelAccounts.value
    .filter((acc) => acc.channel_code === selectedChannel.value)
    .forEach((acc) => set.add(acc.app_type));
  return Array.from(set).map((value) => ({
    label: appTypeLabel(selectedChannel.value, value),
    value,
  }));
});

const accountOptions = computed(() => {
  if (!selectedChannel.value || !selectedAppType.value) return [];
  return channelAccounts.value
    .filter(
      (acc) =>
        acc.channel_code === selectedChannel.value && acc.app_type === selectedAppType.value
    )
    .map((acc) => ({
      label: `${acc.display_name}（${acc.account_id}）${acc.org_sync_default ? " · 默认" : ""}`,
      fullLabel: `${acc.display_name}（${acc.account_id}）${acc.org_sync_default ? " · 默认" : ""}`,
      value: acc.account_uuid,
    }));
});

const memberColumns = [
  { accessorKey: "name", header: "姓名" },
  { accessorKey: "position", header: "职务" },
  { accessorKey: "department", header: "部门" },
  { accessorKey: "phone", header: "手机号" },
  { accessorKey: "biz_mail", header: "企业邮箱" },
  { accessorKey: "email", header: "邮箱" },
  { accessorKey: "status", header: "状态" },
] as const;

const showToast = (title: string, color: typeof toast.value.color, message = "") => {
  toast.value.title = title;
  toast.value.message = message;
  toast.value.color = color;
  toast.value.visible = true;
};

const channelLabel = (code: string) => {
  const match = channelSchema.value?.channels?.find((channel) => channel.code === code);
  return match?.label || code;
};

const appTypeLabel = (channelCode: string, appTypeCode: string) => {
  const channel = channelSchema.value?.channels?.find((item) => item.code === channelCode);
  const match = channel?.app_types?.find((app) => app.code === appTypeCode);
  return match?.label || appTypeCode;
};

const resolveMemberName = (member: OrgSyncSourceMember) => {
  const name = member.profile?.name || member.name || "";
  return name.trim() || member.external_member_id || "-";
};

const resolveMemberPhone = (member: OrgSyncSourceMember) => {
  return (member.profile?.phone || member.phone || "").trim() || "-";
};

const resolveMemberEmail = (member: OrgSyncSourceMember) => {
  return (member.profile?.email || member.email || "").trim() || "-";
};

const resolveMemberBizMail = (member: OrgSyncSourceMember) => {
  return (member.profile?.biz_mail || "").trim() || "-";
};

const resolveMemberPosition = (member: OrgSyncSourceMember) => {
  return (member.profile?.position || "").trim() || "-";
};

const resolveMemberDepartment = (member: OrgSyncSourceMember) => {
  if (selectedUnitLabel.value) {
    return selectedUnitLabel.value;
  }
  const mainDepartmentId = (member.profile?.main_department_id || "").trim();
  if (!mainDepartmentId) return "-";
  return sourceUnitNameByExternalId.value.get(mainDepartmentId) || "-";
};

const loadChannelSchema = async () => {
  try {
    const service = useSocialChannelGovernanceService();
    const resp = await service.getChannelSchema();
    channelSchema.value = (resp as any)?.data ?? null;
  } catch (err: any) {
    showToast("加载渠道字典失败", "error", err?.message ?? "");
    channelSchema.value = null;
  }
};

const loadChannelAccounts = async () => {
  try {
    const service = useSocialChannelGovernanceService();
    const resp = await service.listChannelAccounts();
    channelAccounts.value = (resp as any)?.data?.items ?? [];
    applyDefaultChannelSelection();
    applyDefaultAccountSelection();
  } catch (err: any) {
    showToast("加载渠道账号失败", "error", err?.message ?? "");
  }
};

const applyDefaultChannelSelection = () => {
  if (selectedChannel.value && selectedAppType.value) {
    return;
  }
  const hasWeCom = channelAccounts.value.some(
    (acc) => acc.channel_code === "wechat" && acc.app_type === "wecom"
  );
  if (hasWeCom) {
    selectedChannel.value = "wechat";
    selectedAppType.value = "wecom";
    return;
  }
  if (!selectedChannel.value && channelOptions.value.length > 0) {
    selectedChannel.value = channelOptions.value[0].value;
  }
  if (!selectedAppType.value && appTypeOptions.value.length > 0) {
    selectedAppType.value = appTypeOptions.value[0].value;
  }
};

const applyDefaultAccountSelection = () => {
  if (!selectedChannel.value || !selectedAppType.value) return;
  if (selectedAccountUUID.value) return;
  const defaultAccount = channelAccounts.value.find(
    (acc) =>
      acc.channel_code === selectedChannel.value &&
      acc.app_type === selectedAppType.value &&
      acc.org_sync_default
  );
  if (defaultAccount) {
    selectedAccountUUID.value = defaultAccount.account_uuid;
    return;
  }
  const firstAccount = channelAccounts.value.find(
    (acc) => acc.channel_code === selectedChannel.value && acc.app_type === selectedAppType.value
  );
  if (firstAccount) {
    selectedAccountUUID.value = firstAccount.account_uuid;
  }
};

const loadSourceUnits = async () => {
  if (!selectedAccountUUID.value) {
    sourceUnits.value = [];
    return;
  }
  try {
    const service = useOrgSyncService();
    const resp = await service.listSourceUnits(selectedAccountUUID.value);
    sourceUnits.value = (resp as any)?.data?.items ?? [];
  } catch (err: any) {
    showToast("加载来源部门失败", "error", err?.message ?? "");
  }
};

const loadSourceMembers = async () => {
  if (!selectedAccountUUID.value) {
    sourceMembers.value = [];
    return;
  }
  loadingMembers.value = true;
  try {
    const service = useOrgSyncService();
    const resp = await service.listSourceMembers(
      selectedAccountUUID.value,
      undefined,
      memberKeyword.value || undefined,
      undefined,
      undefined,
      resolveSelectedUnitUUIDs()
    );
    sourceMembers.value = (resp as any)?.data?.items ?? [];
  } catch (err: any) {
    showToast("加载来源成员失败", "error", err?.message ?? "");
  } finally {
    loadingMembers.value = false;
  }
};

const refreshData = async () => {
  loading.value = true;
  try {
    await Promise.all([loadSourceUnits(), loadSourceMembers()]);
  } finally {
    loading.value = false;
  }
};

const goSync = async () => {
  const query: Record<string, string> = {};
  if (selectedAccountUUID.value) query.account_uuid = selectedAccountUUID.value;
  if (selectedChannel.value) query.channel_code = selectedChannel.value;
  if (selectedAppType.value) query.app_type = selectedAppType.value;
  await navigateTo({ path: "/scrm/org_sync/sync", query });
};

const sourceUnitTree = computed(() => {
  const byExternal = new Map<string, any>();
  const nodes = new Map<string, any>();
  sourceUnits.value.forEach((unit) => {
    if (!unit.external_unit_id || !unit.source_unit_uuid) return;
    const label = (unit.name || "").trim() || (unit.external_unit_id || "").trim() || "未命名部门";
    const node = {
      id: unit.source_unit_uuid,
      value: unit.source_unit_uuid,
      externalId: unit.external_unit_id,
      parentExternalId: unit.parent_external_unit_id || undefined,
      label,
      order: typeof unit.order === "number" ? unit.order : 0,
      hasChildren: false,
      children: [] as any[],
    };
    nodes.set(unit.source_unit_uuid, node);
    byExternal.set(unit.external_unit_id, node);
  });
  const roots: any[] = [];
  nodes.forEach((node) => {
    if (node.parentExternalId && byExternal.has(node.parentExternalId)) {
      const parent = byExternal.get(node.parentExternalId);
      parent.children.push(node);
      parent.hasChildren = true;
    } else {
      roots.push(node);
    }
  });
  const sortNodes = (items: any[]) => {
    items.sort((a, b) => {
      if (a.order !== b.order) return b.order - a.order;
      return String(a.label).localeCompare(String(b.label), "zh");
    });
    items.forEach((item) => {
      if (item.children && item.children.length) {
        sortNodes(item.children);
      }
    });
  };
  sortNodes(roots);
  return roots;
});

const selectedUnitLabel = computed(() => {
  if (!selectedUnitUUID.value) return "";
  const unit = sourceUnits.value.find((item) => item.source_unit_uuid === selectedUnitUUID.value);
  const label = (unit?.name || "").trim() || (unit?.external_unit_id || "").trim();
  return label || "";
});

const sourceUnitNameByExternalId = computed(() => {
  const map = new Map<string, string>();
  sourceUnits.value.forEach((unit) => {
    const key = (unit.external_unit_id || "").trim();
    if (!key) return;
    const label = (unit.name || "").trim() || key;
    if (!map.has(key)) {
      map.set(key, label);
    }
  });
  return map;
});

const resolveSelectedUnitUUIDs = () => {
  if (!selectedUnitUUID.value) return [];
  const node = findTreeNode(selectedUnitUUID.value);
  if (!node) return [];
  if (!node.parentExternalId) {
    return [];
  }
  const ids: string[] = [];
  collectTreeNodeIDs(node, ids);
  return ids;
};

const findTreeNode = (id: string) => {
  const stack = [...sourceUnitTree.value];
  while (stack.length > 0) {
    const node = stack.pop();
    if (!node) continue;
    if (node.id === id) return node;
    if (node.children && node.children.length) {
      stack.push(...node.children);
    }
  }
  return null;
};

const collectTreeNodeIDs = (node: any, acc: string[]) => {
  if (!node?.id) return;
  acc.push(node.id);
  if (node.children && node.children.length) {
    node.children.forEach((child: any) => collectTreeNodeIDs(child, acc));
  }
};

const toggleExpanded = (id: string) => {
  const idx = expandedUnitUUIDs.value.indexOf(id);
  if (idx >= 0) {
    expandedUnitUUIDs.value.splice(idx, 1);
  } else {
    expandedUnitUUIDs.value.push(id);
  }
};

const handleNodeClick = (item: any) => {
  if (!item?.id) return;
  selectedUnitUUID.value = String(item.id);
  loadSourceMembers();
};

const collectExpandedNodes = (nodes: any[], acc: string[]) => {
  nodes.forEach((node) => {
    if (!node) return;
    if (node.hasChildren) {
      acc.push(node.id);
      if (node.children && node.children.length) {
        collectExpandedNodes(node.children, acc);
      }
    }
  });
};

watch(selectedChannel, () => {
  selectedAccountUUID.value = "";
  if (
    selectedChannel.value === "wechat" &&
    appTypeOptions.value.some((option) => option.value === "wecom")
  ) {
    selectedAppType.value = "wecom";
    applyDefaultAccountSelection();
    return;
  }
  if (!appTypeOptions.value.some((option) => option.value === selectedAppType.value)) {
    selectedAppType.value = "";
  }
});

watch(selectedAppType, () => {
  selectedAccountUUID.value = "";
  applyDefaultAccountSelection();
});

watch(selectedAccountUUID, () => {
  sourceUnits.value = [];
  sourceMembers.value = [];
  selectedUnitUUID.value = "";
  expandedUnitUUIDs.value = [];
  if (selectedAccountUUID.value) {
    refreshData();
  }
});

watch(
  sourceUnitTree,
  (tree) => {
    if (expandedUnitUUIDs.value.length > 0) return;
    if (!tree || tree.length === 0) return;
    const expanded: string[] = [];
    collectExpandedNodes(tree, expanded);
    expandedUnitUUIDs.value = expanded;
  },
  { deep: true }
);

onMounted(async () => {
  await loadChannelSchema();
  await loadChannelAccounts();
});
</script>

<style>
.org-sync-tree :deep(.u-tree-node) {
  padding-top: 0.25rem;
  padding-bottom: 0.25rem;
}
.org-sync-tree :deep(.u-tree-node-content) {
  padding: 0.25rem 0.5rem;
  border-radius: 0.5rem;
}
.org-sync-tree :deep(.u-tree-node-content:hover) {
  background-color: rgba(148, 163, 184, 0.12);
}
.org-sync-tree :deep(.u-tree-node-selected) {
  background-color: rgba(16, 185, 129, 0.16);
  color: #34d399;
}
.dark .org-sync-tree :deep(.u-tree-node-content:hover) {
  background-color: rgba(15, 23, 42, 0.6);
}
.dark .org-sync-tree :deep(.u-tree-node-selected) {
  background-color: rgba(16, 185, 129, 0.12);
  color: #34d399;
}
</style>
