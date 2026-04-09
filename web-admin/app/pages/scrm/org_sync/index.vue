<template>
  <UContainer class="py-6 space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-xl font-semibold">组织同步</h1>
        <p class="text-sm text-gray-600 dark:text-slate-300">查看组织来源部门、成员与主组织映射。</p>
      </div>
      <div class="flex items-center gap-2">
        <UButton
          variant="soft"
          :to="{ path: '/admin/iam/members', query: { tab: 'users' } }"
        >
          新增员工
        </UButton>
        <UButton color="primary" @click="goSync">进入同步页</UButton>
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-3">
      <UButton variant="soft" color="primary" :loading="loading" @click="refreshData">
        刷新组织数据
      </UButton>
      <span class="text-xs text-gray-600 dark:text-slate-300">默认使用当前租户默认渠道账号；同步动作请进入同步页。</span>
    </div>

    <div v-show="false" class="grid grid-cols-1 gap-6 lg:grid-cols-[280px_1fr]">
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

    <UCard>
      <div class="space-y-3">
        <div class="text-sm text-gray-700 dark:text-slate-200">
          同步动作统一在独立同步页执行；本页仅保留预览与快捷入口。
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <UButton variant="soft" color="primary" :loading="loading" @click="refreshData">刷新预览数据</UButton>
        </div>
      </div>
    </UCard>

    <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <span class="font-medium text-gray-700 dark:text-slate-200">本地组织预览</span>
            <UBadge variant="soft" color="neutral">{{ localDepartments.length }}</UBadge>
          </div>
        </template>
        <div v-if="localDepartments.length === 0" class="text-xs text-gray-600 dark:text-slate-300">
          暂无本地部门数据。
        </div>
        <ul v-else class="space-y-1 text-sm text-gray-700 dark:text-slate-200 max-h-[320px] overflow-auto">
          <li v-for="item in localDepartments" :key="item.id">{{ item.label }}</li>
        </ul>
      </UCard>
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <span class="font-medium text-gray-700 dark:text-slate-200">本地成员预览</span>
            <UBadge variant="soft" color="neutral">{{ mainOrgViewItems.length }}</UBadge>
          </div>
        </template>
        <div v-if="mainOrgViewItems.length === 0" class="text-xs text-gray-600 dark:text-slate-300">
          暂无本地成员数据。
        </div>
        <ul v-else class="space-y-1 text-sm text-gray-700 dark:text-slate-200 max-h-[320px] overflow-auto">
          <li v-for="item in mainOrgViewItems" :key="item.main_member_id">
            {{ item.main_member_name || item.main_member_id }}
          </li>
        </ul>
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
  type OrgSyncMainMemberView,
  type OrgSyncMappingSuggestions,
  type OrgSyncSourceMember,
  type OrgSyncSourceUnit,
  useOrgSyncService,
} from "~/composables/api/services/orgSync";
import { useDepartmentService } from "~/composables/api/services/departmentService";
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
const activeView = ref<"source" | "mapping">("source");
const currentMappingStep = ref<1 | 2 | 3>(1);
const loading = ref(false);
const loadingMembers = ref(false);
const loadingMainOrgView = ref(false);
const loadingSuggestions = ref(false);
const batchMappingLoading = ref(false);
const autoSyncLoading = ref(false);
const pushbackSyncLoading = ref(false);
const bidirectionalSyncLoading = ref(false);
const mappingModalOpen = ref(false);
const mappingSourceLoading = ref(false);
const confirmingMapping = ref(false);
const sourceMembers = ref<OrgSyncSourceMember[]>([]);
const sourceUnits = ref<OrgSyncSourceUnit[]>([]);
const localDepartments = ref<Array<{ id: string; label: string }>>([]);
const latestSuggestions = ref<OrgSyncMappingSuggestions>({
  unit_suggestions: [],
  member_suggestions: [],
});
const unitMappingOverrides = ref<Record<string, string>>({});
const confirmingUnitRowUUID = ref("");
const mainOrgViewItems = ref<OrgSyncMainMemberView[]>([]);
const mappingSourceMembers = ref<OrgSyncSourceMember[]>([]);
const mappingTargetMember = ref<OrgSyncMainMemberView | null>(null);
const selectedSourceMemberUUID = ref("");
const mappingSourceKeyword = ref("");
const mappingSuggestionLoading = ref(false);
const backendSuggestedMemberUUIDs = ref<Set<string>>(new Set());
const lastAutoSyncResult = ref<{
  executed_at: string;
  departments_created: number;
  members_created: number;
  unit_mappings: number;
  member_mappings: number;
} | null>(null);
const lastBatchMappingResult = ref<{
  executed_at: string;
  unit_suggestions: number;
  member_suggestions: number;
  unit_mappings: number;
  member_mappings: number;
  name_fallback_used: number;
} | null>(null);
const lastPushbackResult = ref<{
  executed_at: string;
  direction: string;
  mode: string;
  applied: number;
  conflicts: number;
} | null>(null);
const channelAccounts = ref<ChannelAccount[]>([]);
const channelSchema = ref<ChannelSchemaDocument | null>(null);
const selectedUnitUUID = ref("");
const expandedUnitUUIDs = ref<string[]>([]);
const memberKeyword = ref("");
const mainOrgKeyword = ref("");

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

const mappingSteps = [
  { value: 1 as const, label: "渠道补齐到本地" },
  { value: 2 as const, label: "批量确认映射" },
  { value: 3 as const, label: "人工处理剩余项" },
];

const mappingStepSummary = computed(() => {
  const current = mappingSteps.find((item) => item.value === currentMappingStep.value);
  if (!current) return "";
  return `当前：Step ${current.value} · ${current.label}`;
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

const unitMappingColumns = [
  { accessorKey: "source_unit", header: "来源部门" },
  { accessorKey: "suggested_main_unit", header: "建议本地部门" },
  { accessorKey: "main_unit_select", header: "选择本地部门" },
  { accessorKey: "actions", header: "操作" },
] as const;

const mainOrgColumns = [
  { accessorKey: "main_member_name", header: "本地员工" },
  { accessorKey: "mapping_status", header: "映射状态" },
  { accessorKey: "mapped_count", header: "来源成员数" },
  { accessorKey: "source_accounts", header: "来源账号" },
  { accessorKey: "actions", header: "操作" },
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
    const resp = await service.listSourceUnits("", undefined, selectedAccountUUID.value || undefined);
    sourceUnits.value = (resp as any)?.data?.items ?? [];
  } catch (err: any) {
    showToast("加载来源部门失败", "error", err?.message ?? "");
  }
};

const loadLocalDepartments = async () => {
  try {
    const service = useDepartmentService();
    const tree = await service.getDepartmentTree();
    const flattened: Array<{ id: string; label: string }> = [];
    const walk = (nodes: any[], chain: string[]) => {
      nodes.forEach((node) => {
        const id = String(node?.id || "").trim();
        const name = String(node?.name || "").trim();
        if (!id || !name) return;
        const nextChain = [...chain, name];
        flattened.push({
          id,
          label: nextChain.join(" / "),
        });
        if (Array.isArray(node?.children) && node.children.length > 0) {
          walk(node.children, nextChain);
        }
      });
    };
    walk(Array.isArray(tree) ? tree : [], []);
    localDepartments.value = flattened;
  } catch (err: any) {
    localDepartments.value = [];
    showToast("加载本地部门失败", "error", err?.message ?? "");
  }
};

const loadMappingSuggestionsPreview = async () => {
  if (!selectedAccountUUID.value) {
    latestSuggestions.value = { unit_suggestions: [], member_suggestions: [] };
    return;
  }
  loadingSuggestions.value = true;
  try {
    const service = useOrgSyncService();
    const resp = await service.getMappingSuggestions("", selectedAccountUUID.value || undefined);
    const data = (resp as any)?.data ?? {};
    latestSuggestions.value = {
      unit_suggestions: Array.isArray(data.unit_suggestions) ? data.unit_suggestions : [],
      member_suggestions: Array.isArray(data.member_suggestions) ? data.member_suggestions : [],
    };
  } catch (err: any) {
    latestSuggestions.value = { unit_suggestions: [], member_suggestions: [] };
    showToast("加载映射建议失败", "error", err?.message ?? "");
  } finally {
    loadingSuggestions.value = false;
  }
};

const resolveUnitTargetID = (row: { source_unit_uuid: string; suggested_main_unit_id: string }) => {
  const selected = String(unitMappingOverrides.value[row.source_unit_uuid] || "").trim();
  if (selected) return selected;
  return String(row.suggested_main_unit_id || "").trim();
};

const confirmSingleUnitMapping = async (row: {
  source_unit_uuid: string;
  suggested_main_unit_id: string;
}) => {
  const sourceUnitID = String(row.source_unit_uuid || "").trim();
  const mainUnitID = resolveUnitTargetID(row);
  if (!sourceUnitID || !mainUnitID) {
    showToast("请选择本地部门后再确认", "warning");
    return;
  }
  confirmingUnitRowUUID.value = sourceUnitID;
  try {
    const service = useOrgSyncService();
    await service.confirmMappings({
      unit_mappings: [{ source_unit_id: sourceUnitID, main_unit_id: mainUnitID }],
      member_mappings: [],
    });
    showToast("部门映射确认成功", "success");
    await Promise.all([loadMainOrgView(), loadMappingSuggestionsPreview()]);
  } catch (err: any) {
    showToast("部门映射确认失败", "error", err?.message ?? "");
  } finally {
    confirmingUnitRowUUID.value = "";
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
    const sourceAccountUUID = resolvedSourceAccountUUID.value;
    const scopedUnitUUIDs = sourceAccountUUID ? resolveSelectedUnitUUIDs() : [];
    const resp = await service.listSourceMembers(
      sourceAccountUUID || "",
      undefined,
      memberKeyword.value || undefined,
      selectedAccountUUID.value || undefined,
      undefined,
      scopedUnitUUIDs
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
    await loadSourceUnits();
    await Promise.all([loadSourceMembers(), loadMainOrgView(), loadMappingSuggestionsPreview()]);
  } finally {
    loading.value = false;
  }
};

const loadMainOrgView = async () => {
  loadingMainOrgView.value = true;
  try {
    const service = useOrgSyncService();
    const resp = await service.listMainOrgView(mainOrgKeyword.value || undefined);
    mainOrgViewItems.value = (resp as any)?.data?.items ?? [];
  } catch (err: any) {
    showToast("加载本地主组织映射失败", "error", err?.message ?? "");
  } finally {
    loadingMainOrgView.value = false;
  }
};

const filteredMainOrgViewItems = computed(() => {
  const accountUUID = (selectedAccountUUID.value || "").trim();
  if (!accountUUID) return mainOrgViewItems.value;
  return mainOrgViewItems.value.filter((item) => {
    const sourceAccounts = Array.isArray(item.source_accounts) ? item.source_accounts : [];
    if (sourceAccounts.length === 0) return true;
    return sourceAccounts.includes(accountUUID);
  });
});

const localDepartmentOptions = computed(() =>
  localDepartments.value.map((item) => ({ label: item.label, value: item.id }))
);

const sourceUnitNameByUUID = computed(() => {
  const map = new Map<string, string>();
  sourceUnits.value.forEach((unit) => {
    const key = String(unit.source_unit_uuid || "").trim();
    if (!key) return;
    const name = String(unit.name || "").trim() || String(unit.external_unit_id || "").trim() || key;
    map.set(key, name);
  });
  return map;
});

const localDepartmentNameByID = computed(() => {
  const map = new Map<string, string>();
  localDepartments.value.forEach((dept) => {
    const id = String(dept.id || "").trim();
    if (!id) return;
    map.set(id, dept.label);
  });
  return map;
});

const pendingUnitRows = computed(() => {
  return (latestSuggestions.value.unit_suggestions || []).map((item) => {
    const sourceUnitUUID = String(item.source_unit_uuid || "").trim();
    const suggestedMainUnitID = String(item.main_unit_id || "").trim();
    const sourceUnitName = sourceUnitNameByUUID.value.get(sourceUnitUUID) || sourceUnitUUID || "-";
    const suggestedMainUnitName = localDepartmentNameByID.value.get(suggestedMainUnitID) || suggestedMainUnitID || "";
    if (!unitMappingOverrides.value[sourceUnitUUID]) {
      unitMappingOverrides.value[sourceUnitUUID] = suggestedMainUnitID;
    }
    return {
      source_unit_uuid: sourceUnitUUID,
      source_unit_name: sourceUnitName,
      suggested_main_unit_id: suggestedMainUnitID,
      suggested_main_unit_name: suggestedMainUnitName,
    };
  });
});

const mappedMainMemberCount = computed(
  () => filteredMainOrgViewItems.value.filter((item) => item.mapping_status === "mapped").length
);
const unmappedMainMemberCount = computed(
  () => filteredMainOrgViewItems.value.length - mappedMainMemberCount.value
);

const selectedAccountLabel = computed(() => {
  if (!selectedAccountUUID.value) return "";
  return accountOptions.value.find((item) => item.value === selectedAccountUUID.value)?.label || "";
});

const resolvedSourceAccountUUID = computed(() => {
  const ids = new Set<string>();
  sourceUnits.value.forEach((item) => {
    const id = String(item.source_account_uuid || "").trim();
    if (id) ids.add(id);
  });
  if (ids.size === 1) return Array.from(ids)[0];
  return "";
});

const mappingSourceMemberOptions = computed(() =>
  mappingSourceMembers.value.map((member) => {
    const name = resolveMemberName(member);
    const phone = resolveMemberPhone(member);
    const email = resolveMemberEmail(member);
    const label = `${name}（${member.external_member_id}）`;
    const fullLabel = `${name}（${member.external_member_id}） · 手机:${phone} · 邮箱:${email}`;
    return {
      label,
      fullLabel,
      value: member.source_member_uuid,
    };
  })
);

type MappingRecommendedCandidate = {
  source_member_uuid: string;
  label: string;
  reason: string;
  score: number;
};

const mappingRecommendedCandidates = computed<MappingRecommendedCandidate[]>(() => {
  const targetName = normalizeName(mappingTargetMember.value?.main_member_name || "");
  if (!targetName) return [];
  const candidates: MappingRecommendedCandidate[] = [];
  for (const member of mappingSourceMembers.value) {
    const name = resolveMemberName(member);
    const normalizedName = normalizeName(name);
    if (!normalizedName) continue;
    const byBackend = backendSuggestedMemberUUIDs.value.has(member.source_member_uuid);
    const similarity = calculateNameSimilarity(targetName, normalizedName);
    const score = byBackend ? 98 : Math.round(similarity * 100);
    if (!byBackend && score < 45) continue;
    const reason = byBackend
      ? "系统已通过手机号/邮箱命中"
      : score >= 80
        ? "姓名高相似度"
        : "姓名弱相似度，建议人工确认";
    candidates.push({
      source_member_uuid: member.source_member_uuid,
      label: `${name}（${member.external_member_id}）`,
      reason,
      score,
    });
  }
  return candidates
    .sort((a, b) => b.score - a.score)
    .slice(0, 5);
});

const applyTopRecommendedCandidate = () => {
  const top = mappingRecommendedCandidates.value[0];
  if (!top) return;
  selectedSourceMemberUUID.value = top.source_member_uuid;
};

const normalizeName = (value: string) =>
  value
    .trim()
    .toLowerCase()
    .replace(/\s+/g, "");

const calculateNameSimilarity = (a: string, b: string) => {
  if (!a || !b) return 0;
  if (a === b) return 1;
  if (a.includes(b) || b.includes(a)) return 0.85;
  const setA = new Set(Array.from(a));
  const setB = new Set(Array.from(b));
  let inter = 0;
  for (const ch of setA) {
    if (setB.has(ch)) inter++;
  }
  const union = setA.size + setB.size - inter;
  if (union <= 0) return 0;
  return inter / union;
};

const loadBackendMappingSuggestions = async () => {
  backendSuggestedMemberUUIDs.value = new Set<string>();
  if (!selectedAccountUUID.value || !mappingTargetMember.value?.main_member_id) return;
  mappingSuggestionLoading.value = true;
  try {
    const service = useOrgSyncService();
    const resp = await service.getMappingSuggestions("", selectedAccountUUID.value || undefined);
    const suggestions = (resp as any)?.data?.member_suggestions ?? [];
    const currentMainMemberID = String(mappingTargetMember.value.main_member_id);
    const matches = suggestions
      .filter((item: any) => String(item.main_member_id || "") === currentMainMemberID)
      .map((item: any) => String(item.source_member_uuid || "").trim())
      .filter((id: string) => !!id);
    backendSuggestedMemberUUIDs.value = new Set<string>(matches);
  } catch {
    backendSuggestedMemberUUIDs.value = new Set<string>();
  } finally {
    mappingSuggestionLoading.value = false;
  }
};

const runBatchAutoMapping = async () => {
  if (!selectedAccountUUID.value) {
    showToast("请先选择渠道账号", "warning");
    return;
  }
  batchMappingLoading.value = true;
  try {
    const service = useOrgSyncService();
    const resp = await service.getMappingSuggestions("", selectedAccountUUID.value || undefined);
    const data = (resp as any)?.data ?? {};
    const unitSuggestions = Array.isArray(data.unit_suggestions) ? data.unit_suggestions : [];
    const memberSuggestions = Array.isArray(data.member_suggestions) ? data.member_suggestions : [];

    const payload = {
      unit_mappings: unitSuggestions
        .map((item: any) => ({
          source_unit_id: String(item?.source_unit_uuid || "").trim(),
          main_unit_id: String(item?.main_unit_id || "").trim(),
        }))
        .filter((item: { source_unit_id: string; main_unit_id: string }) => item.source_unit_id && item.main_unit_id),
      member_mappings: memberSuggestions
        .map((item: any) => ({
          source_member_id: String(item?.source_member_uuid || "").trim(),
          main_member_id: String(item?.main_member_id || "").trim(),
        }))
        .filter((item: { source_member_id: string; main_member_id: string }) => item.source_member_id && item.main_member_id),
    };

    let fallbackUsedCount = 0;
    // 后端建议为空时，使用前端保守姓名匹配兜底（仅高置信度，避免误绑）
    if (payload.member_mappings.length === 0) {
      const fallbackMappings = buildNameFallbackMappings();
      fallbackUsedCount = fallbackMappings.length;
      payload.member_mappings.push(...fallbackMappings);
    }

    if (payload.unit_mappings.length === 0 && payload.member_mappings.length === 0) {
      showToast("没有可批量确认的映射建议", "warning", "当前数据手机号/邮箱缺失且姓名未命中高置信规则");
      return;
    }

    const confirmResp = await service.confirmMappings(payload);
    const result = (confirmResp as any)?.data ?? {};
    const unitCount = Number(result.unit_mappings || 0);
    const memberCount = Number(result.member_mappings || 0);
    lastBatchMappingResult.value = {
      executed_at: new Date().toLocaleString(),
      unit_suggestions: unitSuggestions.length,
      member_suggestions: memberSuggestions.length,
      unit_mappings: unitCount,
      member_mappings: memberCount,
      name_fallback_used: fallbackUsedCount,
    };
    showToast("批量映射完成", "success", `部门 ${unitCount} 条，员工 ${memberCount} 条`);
    currentMappingStep.value = 3;
    await Promise.all([loadMainOrgView(), loadSourceMembers(), loadMappingSuggestionsPreview()]);
  } catch (err: any) {
    showToast("批量映射失败", "error", err?.message ?? "");
  } finally {
    batchMappingLoading.value = false;
  }
};

const runAutoSyncProvision = async () => {
  if (!selectedAccountUUID.value) {
    showToast("请先选择渠道账号", "warning");
    return;
  }
  autoSyncLoading.value = true;
  try {
    const service = useOrgSyncService();
    const resp = await service.autoSyncMappings(selectedAccountUUID.value);
    const result = (resp as any)?.data ?? {};
    const departmentsCreated = Number(result.departments_created || 0);
    const membersCreated = Number(result.members_created || 0);
    const unitMappings = Number(result.unit_mappings || 0);
    const memberMappings = Number(result.member_mappings || 0);
    lastAutoSyncResult.value = {
      executed_at: new Date().toLocaleString(),
      departments_created: departmentsCreated,
      members_created: membersCreated,
      unit_mappings: unitMappings,
      member_mappings: memberMappings,
    };
    showToast(
      "补齐同步完成",
      "success",
      `新增部门 ${departmentsCreated}，新增员工 ${membersCreated}，部门映射 ${unitMappings}，员工映射 ${memberMappings}`
    );
    currentMappingStep.value = 2;
    await Promise.all([loadMainOrgView(), loadSourceMembers(), loadMappingSuggestionsPreview()]);
  } catch (err: any) {
    showToast("补齐同步失败", "error", err?.message ?? "");
  } finally {
    autoSyncLoading.value = false;
  }
};

const runLocalToChannelPushback = async () => {
  const sourceAccountUUID = String(resolvedSourceAccountUUID.value || "").trim();
  if (!sourceAccountUUID) {
    showToast("缺少来源账号，无法执行本地推送", "warning", "请先完成一次渠道拉取并确认来源账号");
    return;
  }
  pushbackSyncLoading.value = true;
  try {
    const service = useOrgSyncService();
    const resp = await service.triggerPushSync(sourceAccountUUID, { changes: [] });
    const result = (resp as any)?.data ?? {};
    lastPushbackResult.value = {
      executed_at: new Date().toLocaleString(),
      direction: String(result.direction || "push"),
      mode: String(result.mode || "pushback"),
      applied: Number(result.applied || 0),
      conflicts: Number(result.conflicts || 0),
    };
    showToast(
      "本地推送完成",
      "success",
      `生效 ${lastPushbackResult.value.applied}，冲突 ${lastPushbackResult.value.conflicts}`
    );
    await Promise.all([loadSourceUnits(), loadSourceMembers(), loadMainOrgView(), loadMappingSuggestionsPreview()]);
  } catch (err: any) {
    showToast("本地推送失败", "error", err?.message ?? "");
  } finally {
    pushbackSyncLoading.value = false;
  }
};

const runBidirectionalSync = async () => {
  if (!selectedAccountUUID.value) {
    showToast("请先选择渠道账号", "warning");
    return;
  }
  if (bidirectionalSyncLoading.value) return;
  bidirectionalSyncLoading.value = true;
  try {
    // 1) 渠道 -> 本地补齐
    const service = useOrgSyncService();
    const autoResp = await service.autoSyncMappings(selectedAccountUUID.value);
    const autoResult = (autoResp as any)?.data ?? {};
    const departmentsCreated = Number(autoResult.departments_created || 0);
    const membersCreated = Number(autoResult.members_created || 0);
    const unitMappings = Number(autoResult.unit_mappings || 0);
    const memberMappings = Number(autoResult.member_mappings || 0);
    lastAutoSyncResult.value = {
      executed_at: new Date().toLocaleString(),
      departments_created: departmentsCreated,
      members_created: membersCreated,
      unit_mappings: unitMappings,
      member_mappings: memberMappings,
    };

    // 2) 刷新来源账号后执行本地 -> 渠道推送
    await loadSourceUnits();
    const sourceAccountUUID = String(resolvedSourceAccountUUID.value || "").trim();
    if (!sourceAccountUUID) {
      throw new Error("缺少来源账号，无法执行第二步推送");
    }
    const pushResp = await service.triggerPushSync(sourceAccountUUID, { changes: [] });
    const pushResult = (pushResp as any)?.data ?? {};
    lastPushbackResult.value = {
      executed_at: new Date().toLocaleString(),
      direction: String(pushResult.direction || "push"),
      mode: String(pushResult.mode || "pushback"),
      applied: Number(pushResult.applied || 0),
      conflicts: Number(pushResult.conflicts || 0),
    };

    currentMappingStep.value = 2;
    await Promise.all([loadSourceMembers(), loadMainOrgView(), loadMappingSuggestionsPreview()]);
    showToast(
      "双向同步完成",
      "success",
      `补齐: 部门+${departmentsCreated}/员工+${membersCreated}；推送: 生效 ${lastPushbackResult.value.applied}，冲突 ${lastPushbackResult.value.conflicts}`
    );
  } catch (err: any) {
    showToast("双向同步失败", "error", err?.message ?? "");
  } finally {
    bidirectionalSyncLoading.value = false;
  }
};

const buildNameFallbackMappings = (): Array<{ source_member_id: string; main_member_id: string }> => {
  const sourceByName = new Map<string, OrgSyncSourceMember[]>();
  for (const member of sourceMembers.value) {
    const key = normalizeName(resolveMemberName(member));
    if (!key) continue;
    const arr = sourceByName.get(key) || [];
    arr.push(member);
    sourceByName.set(key, arr);
  }

  const takenSourceIDs = new Set<string>();
  const mappings: Array<{ source_member_id: string; main_member_id: string }> = [];
  const targets = filteredMainOrgViewItems.value.filter((item) => (item.mapped_count || 0) === 0);

  for (const target of targets) {
    const targetName = normalizeName(target.main_member_name || "");
    if (!targetName) continue;

    // 规则1：姓名标准化后完全相等，且来源侧唯一
    const exactList = sourceByName.get(targetName) || [];
    if (exactList.length === 1) {
      const matched = exactList[0];
      if (!takenSourceIDs.has(matched.source_member_uuid)) {
        mappings.push({
          source_member_id: matched.source_member_uuid,
          main_member_id: String(target.main_member_id),
        });
        takenSourceIDs.add(matched.source_member_uuid);
        continue;
      }
    }

    // 规则2：高相似度唯一候选（>= 0.92）
    const scored = sourceMembers.value
      .map((member) => ({
        member,
        score: calculateNameSimilarity(targetName, normalizeName(resolveMemberName(member))),
      }))
      .filter((item) => item.score >= 0.92)
      .sort((a, b) => b.score - a.score);
    if (scored.length === 1) {
      const matched = scored[0].member;
      if (!takenSourceIDs.has(matched.source_member_uuid)) {
        mappings.push({
          source_member_id: matched.source_member_uuid,
          main_member_id: String(target.main_member_id),
        });
        takenSourceIDs.add(matched.source_member_uuid);
      }
    }
  }
  return mappings;
};

const goSync = async () => {
  const query: Record<string, string> = {};
  if (selectedAccountUUID.value) query.account_uuid = selectedAccountUUID.value;
  if (selectedChannel.value) query.channel_code = selectedChannel.value;
  if (selectedAppType.value) query.app_type = selectedAppType.value;
  await navigateTo({ path: "/scrm/org_sync/sync", query });
};

const openMappingModal = async (item: OrgSyncMainMemberView) => {
  if (!selectedAccountUUID.value) {
    showToast("请先选择渠道账号", "warning");
    return;
  }
  mappingTargetMember.value = item;
  selectedSourceMemberUUID.value = "";
  mappingSourceKeyword.value = item.main_member_name || "";
  mappingModalOpen.value = true;
  await Promise.all([loadMappingSourceMembers(), loadBackendMappingSuggestions()]);
};

const loadMappingSourceMembers = async () => {
  if (!selectedAccountUUID.value) {
    mappingSourceMembers.value = [];
    return;
  }
  mappingSourceLoading.value = true;
  try {
    const service = useOrgSyncService();
    const sourceAccountUUID = resolvedSourceAccountUUID.value;
    const resp = await service.listSourceMembers(
      sourceAccountUUID || "",
      undefined,
      mappingSourceKeyword.value || undefined,
      selectedAccountUUID.value || undefined
    );
    mappingSourceMembers.value = (resp as any)?.data?.items ?? [];
  } catch (err: any) {
    showToast("加载来源成员失败", "error", err?.message ?? "");
    mappingSourceMembers.value = [];
  } finally {
    mappingSourceLoading.value = false;
  }
};

const confirmMappingInModal = async () => {
  if (!mappingTargetMember.value?.main_member_id) {
    showToast("缺少本地员工信息", "error");
    return;
  }
  if (!selectedSourceMemberUUID.value) {
    showToast("请选择来源成员", "warning");
    return;
  }
  confirmingMapping.value = true;
  try {
    const service = useOrgSyncService();
    await service.confirmMappings({
      unit_mappings: [],
      member_mappings: [
        {
          source_member_id: selectedSourceMemberUUID.value,
          main_member_id: mappingTargetMember.value.main_member_id,
        },
      ],
    });
    showToast("映射确认成功", "success");
    mappingModalOpen.value = false;
    await Promise.all([loadMainOrgView(), loadSourceMembers()]);
  } catch (err: any) {
    showToast("映射确认失败", "error", err?.message ?? "");
  } finally {
    confirmingMapping.value = false;
  }
};

const mappingStatusMeta = (status?: string) => {
  return status === "mapped"
    ? { label: "已映射，可推送", color: "success" as const }
    : { label: "未映射，不可推送", color: "warning" as const };
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
  latestSuggestions.value = { unit_suggestions: [], member_suggestions: [] };
  unitMappingOverrides.value = {};
  lastPushbackResult.value = null;
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
  await Promise.all([loadChannelAccounts(), loadLocalDepartments()]);
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
