<template>
  <UContainer class="py-10 space-y-6">
    <div class="flex items-start justify-between gap-3">
      <div class="space-y-2">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">企业客户标签</h1>
        <p class="text-gray-600 dark:text-gray-200">以标签清单为主视图，支持管理客户-标签关系，并通过同步中心任务回写企业微信。</p>
      </div>
      <div class="flex items-center gap-2">
        <UButton size="sm" color="primary" variant="soft" @click="openCreateGroupModal">
          新建标签组
        </UButton>
        <UButton size="sm" variant="soft" :loading="loading" @click="refreshAll">
          刷新数据
        </UButton>
        <UButton
          size="sm"
          color="primary"
          :loading="submitLoading"
          :disabled="pendingTotal === 0 || !activeTagAccountUUID"
          @click="submitPushbackJob"
        >
          提交回写任务
        </UButton>
        <UButton size="sm" color="primary" variant="soft" to="/scrm/tag_sync_center">
          前往标签同步中心
        </UButton>
      </div>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <span class="font-medium text-gray-800 dark:text-gray-100">标签列表（企业客户）</span>
          <div class="flex items-center gap-2">
            <UBadge variant="soft" color="primary">关系视图</UBadge>
            <UBadge variant="soft" color="warning">待回写 {{ pendingTotal }}</UBadge>
            <UBadge variant="soft" color="info">{{ `关系 ${relationRows.length}` }}</UBadge>
          </div>
        </div>
      </template>

      <div v-if="pendingTotal > 0" class="mb-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800 dark:border-amber-700/40 dark:bg-amber-950/20 dark:text-amber-200">
        <div class="flex items-center justify-between gap-2">
          <span>存在 {{ pendingTotal }} 条待回写变更（关系 {{ pendingOperations.length }} / 标签 {{ pendingTagOperations.length }}）。</span>
          <UButton size="xs" color="neutral" variant="soft" @click="clearPendingOperations">清空待回写</UButton>
        </div>
        <div v-if="pendingTagOperationSummaries.length > 0" class="mt-2 rounded border border-amber-200/80 bg-white/70 p-2 text-[11px] text-gray-800 dark:border-amber-700/40 dark:bg-gray-900/40 dark:text-gray-100">
          <div class="mb-1 font-medium">标签操作明细</div>
          <div class="max-h-32 space-y-1 overflow-auto">
            <div
              v-for="(item, idx) in pendingTagOperationSummaries"
              :key="`pending-tag-op:${idx}`"
              class="rounded border border-gray-200/80 bg-white px-2 py-1 dark:border-gray-700 dark:bg-gray-900/70"
            >
              {{ item }}
            </div>
          </div>
        </div>
      </div>

      <div class="space-y-3">
        <div class="flex items-center justify-between gap-2 rounded border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-700 dark:border-gray-700 dark:bg-gray-900/60 dark:text-gray-200">
          <span>关系总览：{{ relationRows.length }} 条（筛选后 {{ relationRowsFiltered.length }} 条）</span>
          <div class="flex items-center gap-2">
            <UBadge variant="soft" color="warning">失效关系 {{ danglingRelationRows.length }}</UBadge>
            <UButton
              size="xs"
              color="warning"
              variant="soft"
              :disabled="danglingRelationRows.length === 0"
              @click="queueDanglingRemovals"
            >
              清理失效关系到待回写
            </UButton>
          </div>
        </div>
        <div class="grid grid-cols-1 gap-2 md:grid-cols-3">
          <UFormField label="关键词">
            <UInput v-model="relationKeyword" placeholder="标签名/客户/external_userid/userid" />
          </UFormField>
          <div class="flex items-end">
            <UButton size="sm" variant="soft" color="neutral" @click="() => { relationKeyword = ''; }">清空筛选</UButton>
          </div>
        </div>
        <div v-if="relationRows.length === 0" class="py-6 text-center text-sm text-gray-500 dark:text-gray-300">
          暂无标签关系数据，请先执行标签拉取与客户同步。
        </div>
        <div v-else-if="relationRowsFiltered.length === 0" class="py-6 text-center text-sm text-gray-500 dark:text-gray-300">
          当前筛选下无关系记录。
        </div>
        <div v-else class="grid grid-cols-1 gap-3 lg:grid-cols-12">
          <div class="lg:col-span-4 rounded border border-gray-200 dark:border-gray-700">
            <div class="border-b border-gray-200 px-3 py-2 text-xs text-gray-500 dark:border-gray-700 dark:text-gray-300">
              标签组（{{ relationGroupSections.length }}）/ 标签（{{ relationGroups.length }}）
            </div>
            <div class="max-h-[520px] overflow-auto p-2 space-y-2">
              <div v-for="section in relationGroupSections" :key="`section:${section.groupName}`" class="rounded border border-gray-200 dark:border-gray-700">
                <div class="flex items-center justify-between gap-2 border-b border-gray-200 px-3 py-2 text-xs text-gray-500 dark:border-gray-700 dark:text-gray-300">
                  <span class="truncate">{{ section.groupName }}</span>
                  <div class="flex items-center gap-2">
                    <UBadge size="xs" variant="soft" color="neutral">{{ section.tags.length }}</UBadge>
                    <UButton
                      size="2xs"
                      variant="soft"
                      color="neutral"
                      :disabled="!section.groupID"
                      @click="openRenameGroup(section)"
                    >
                      改组名
                    </UButton>
                  </div>
                </div>
                <div class="space-y-1 p-2">
                  <button
                    v-for="group in section.tags"
                    :key="group.tagID"
                    type="button"
                    class="w-full rounded border px-3 py-2 text-left text-xs transition"
                    :class="[
                      selectedRelationTagID === `${group.groupName}::${group.tagID}`
                        ? 'border-primary-500 bg-primary-50/70 dark:bg-primary-950/20'
                        : 'border-gray-200 hover:border-primary-300 dark:border-gray-700',
                      group.isDangling ? 'ring-1 ring-amber-400/50' : '',
                    ]"
                    @click="selectedRelationTagID = `${group.groupName}::${group.tagID}`"
                  >
                    <div class="flex items-center justify-between gap-2">
                      <span class="font-medium text-gray-900 dark:text-white truncate">{{ group.tagLabel }}</span>
                      <UBadge size="xs" variant="soft" color="neutral">{{ group.members.length }}</UBadge>
                    </div>
                    <div class="mt-1 flex items-center gap-1">
                      <UBadge size="xs" variant="soft" :color="group.isDangling ? 'warning' : 'primary'">{{ group.tagID }}</UBadge>
                      <UBadge v-if="group.isDangling" size="xs" color="warning" variant="solid">失效</UBadge>
                    </div>
                  </button>
                </div>
              </div>
            </div>
          </div>
          <div class="lg:col-span-8 rounded border border-gray-200 dark:border-gray-700">
            <div class="flex items-center justify-between gap-2 border-b border-gray-200 px-3 py-2 dark:border-gray-700">
              <div class="flex items-center gap-2 text-xs">
                <span class="font-medium text-gray-900 dark:text-white">{{ selectedRelationGroup ? `${selectedRelationGroup.groupName} / ${selectedRelationGroup.tagLabel}` : "请选择左侧标签" }}</span>
                <UBadge v-if="selectedRelationGroup" size="xs" variant="soft" color="neutral">用户 {{ selectedRelationMembers.length }}</UBadge>
              </div>
              <div class="flex items-center gap-2">
                <UButton
                  size="xs"
                  variant="soft"
                  color="primary"
                  :disabled="!selectedRelationGroup || !allTagOptions.some((item) => item.tag_id === selectedRelationGroup?.tagID)"
                  @click="selectedRelationGroup && openTagCustomerEditor({ tagId: selectedRelationGroup.tagID, tagName: selectedRelationGroup.tagLabel, tagUuid: selectedRelationGroup.tagID })"
                >
                  管理该标签关系
                </UButton>
                <UButton
                  size="xs"
                  variant="soft"
                  color="neutral"
                  :disabled="!selectedRelationGroup || !allTagOptions.some((item) => item.tag_id === selectedRelationGroup?.tagID)"
                  @click="openRenameTag"
                >
                  改名
                </UButton>
                <UButton
                  size="xs"
                  variant="soft"
                  color="error"
                  :disabled="!selectedRelationGroup || !allTagOptions.some((item) => item.tag_id === selectedRelationGroup?.tagID)"
                  @click="queueDeleteTag"
                >
                  删除标签
                </UButton>
              </div>
            </div>
            <div class="max-h-[520px] overflow-auto p-3">
              <div v-if="!selectedRelationGroup" class="py-10 text-center text-sm text-gray-500 dark:text-gray-300">
                请在左侧选择一个标签查看关联用户
              </div>
              <div v-else-if="selectedRelationMembers.length === 0" class="py-10 text-center text-sm text-gray-500 dark:text-gray-300">
                当前标签暂无关联用户
              </div>
              <div v-else class="space-y-2">
                <div
                  v-for="member in selectedRelationMembers"
                  :key="member.key"
                  class="rounded border border-gray-200 px-3 py-2 text-xs dark:border-gray-700"
                >
                  <div class="flex items-center justify-between gap-2">
                    <span class="font-medium text-gray-900 dark:text-white">{{ member.displayName || "-" }}</span>
                    <UBadge size="xs" variant="soft" color="neutral">{{ member.userID }}</UBadge>
                  </div>
                  <div class="mt-1 text-gray-500 dark:text-gray-300">{{ member.externalUserID }}</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </UCard>

    <UModal v-model:open="customerTagEditor.open" title="编辑客户标签" :ui="{ content: 'max-w-2xl' }">
      <template #body>
        <div class="space-y-3">
          <div class="text-xs text-gray-500 dark:text-gray-300">
            客户：{{ customerTagEditor.displayName || "-" }} / 跟进人：{{ customerTagEditor.userid || "-" }}
          </div>
          <div class="max-h-80 overflow-auto rounded border border-gray-200 p-3 dark:border-gray-700">
            <div class="grid grid-cols-1 gap-2 md:grid-cols-2">
              <label v-for="tag in allTagOptions" :key="`editor:${tag.tag_id}`" class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
                <UCheckbox :model-value="customerTagEditor.selectedTagIDs.includes(tag.tag_id)" @update:model-value="(checked: boolean) => toggleEditorTag(tag.tag_id, checked)" />
                <span>{{ tag.group_name ? `${tag.group_name} / ` : "" }}{{ tag.tag_name || tag.tag_id }}</span>
              </label>
            </div>
          </div>
        </div>
      </template>
      <template #footer>
        <div class="ml-auto flex items-center gap-2">
          <UButton variant="ghost" @click="customerTagEditor.open = false">取消</UButton>
          <UButton color="primary" @click="saveCustomerTagEditor">保存到待回写</UButton>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="tagCustomerEditor.open" title="按标签批量管理客户" :ui="{ content: 'max-w-3xl' }">
      <template #body>
        <div class="space-y-3">
          <div class="text-xs text-gray-500 dark:text-gray-300">
            标签：{{ tagCustomerEditor.tagLabel || "-" }}
          </div>
          <div class="max-h-96 overflow-auto rounded border border-gray-200 p-3 dark:border-gray-700">
            <div class="space-y-2">
              <label
                v-for="candidate in followUserCandidates"
                :key="`candidate:${candidate.key}`"
                class="flex items-center justify-between gap-3 rounded border border-gray-100 px-3 py-2 text-sm dark:border-gray-800"
              >
                <div>
                  <div class="font-medium text-gray-900 dark:text-white">{{ candidate.display_name || "-" }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-300">{{ candidate.external_userid }} / {{ candidate.userid }}</div>
                </div>
                <UCheckbox :model-value="tagCustomerEditor.selectedKeys.includes(candidate.key)" @update:model-value="(checked: boolean) => toggleTagCustomerSelection(candidate.key, checked)" />
              </label>
            </div>
          </div>
        </div>
      </template>
      <template #footer>
        <div class="ml-auto flex items-center gap-2">
          <UButton variant="ghost" @click="tagCustomerEditor.open = false">取消</UButton>
          <UButton color="primary" @click="saveTagCustomerEditor">保存到待回写</UButton>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="tagRenameEditor.open" title="重命名标签" :ui="{ content: 'max-w-lg' }">
      <template #body>
        <div class="space-y-3">
          <div class="text-xs text-gray-500 dark:text-gray-300">
            标签组：{{ tagRenameEditor.groupName || "未分组" }}
          </div>
          <UFormField label="当前名称">
            <UInput :model-value="tagRenameEditor.currentName" readonly />
          </UFormField>
          <UFormField label="新名称" required>
            <UInput v-model="tagRenameEditor.nextName" maxlength="32" placeholder="请输入新标签名" />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="ml-auto flex items-center gap-2">
          <UButton variant="ghost" @click="tagRenameEditor.open = false">取消</UButton>
          <UButton color="primary" @click="saveRenameTag">保存到待回写</UButton>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="groupRenameEditor.open" title="重命名标签组" :ui="{ content: 'max-w-lg' }">
      <template #body>
        <div class="space-y-3">
          <UFormField label="当前名称">
            <UInput :model-value="groupRenameEditor.currentName" readonly />
          </UFormField>
          <UFormField label="新名称" required>
            <UInput v-model="groupRenameEditor.nextName" maxlength="32" placeholder="请输入新标签组名" />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="ml-auto flex items-center gap-2">
          <UButton variant="ghost" @click="groupRenameEditor.open = false">取消</UButton>
          <UButton color="primary" @click="saveRenameGroup">保存到待回写</UButton>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="groupCreateEditor.open" title="新建标签组" :ui="{ content: 'max-w-lg' }">
      <template #body>
        <div class="space-y-3">
          <UFormField label="标签组名称" required>
            <UInput v-model="groupCreateEditor.groupName" maxlength="32" placeholder="请输入标签组名称" />
          </UFormField>
          <UFormField label="首个标签名称" required>
            <UInput v-model="groupCreateEditor.tagName" maxlength="32" placeholder="请输入首个标签名称" />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="ml-auto flex items-center gap-2">
          <UButton variant="ghost" @click="groupCreateEditor.open = false">取消</UButton>
          <UButton color="primary" @click="saveCreateGroup">保存到待回写</UButton>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import {
  useSocialChannelGovernanceService,
  type FoundationCustomerTagOperation,
  type FoundationTagOperation,
} from "~/composables/api/services/socialChannelGovernance";
import { useGlobalLoadingAdapter } from "~/composables/useGlobalLoadingAdapter";

definePageMeta({ layout: "default" });

const toast = useToast();
const gl = useGlobalLoadingAdapter();
const PUSHBACK_DRAFT_KEY = "scrm:enterprise_tag_pushback_draft.v1";
const loading = ref(false);
const submitLoading = ref(false);
const tagAccounts = ref<any[]>([]);
const tagRecords = ref<any[]>([]);
const customerBindings = ref<any[]>([]);
const pendingOperations = ref<FoundationCustomerTagOperation[]>([]);
const pendingTagOperations = ref<FoundationTagOperation[]>([]);
const bindingsLoadedAccountUUID = ref("");
const initializing = ref(true);
let bindingInflight: Promise<void> | null = null;
const globalLoadingDepth = ref(0);

const beginGlobalLoading = (message = "加载中...") => {
  if (globalLoadingDepth.value === 0) {
    gl.setMessage(message);
    gl.show({ lock: true, minMs: 200 });
  } else {
    gl.setMessage(message);
  }
  globalLoadingDepth.value += 1;
};

const endGlobalLoading = () => {
  globalLoadingDepth.value = Math.max(0, globalLoadingDepth.value - 1);
  if (globalLoadingDepth.value === 0) {
    gl.hide();
    gl.unlock();
  }
};

const customerTagEditor = reactive({
  open: false,
  externalUserID: "",
  userid: "",
  displayName: "",
  baseTagIDs: [] as string[],
  selectedTagIDs: [] as string[],
});

const tagCustomerEditor = reactive({
  open: false,
  tagID: "",
  tagLabel: "",
  selectedKeys: [] as string[],
});
const tagRenameEditor = reactive({
  open: false,
  tagID: "",
  groupID: "",
  groupName: "",
  currentName: "",
  nextName: "",
});
const groupRenameEditor = reactive({
  open: false,
  groupID: "",
  currentName: "",
  nextName: "",
});
const groupCreateEditor = reactive({
  open: false,
  groupName: "",
  tagName: "",
});
const relationKeyword = ref("");
const selectedRelationTagID = ref("");
const pendingTotal = computed(() => pendingOperations.value.length + pendingTagOperations.value.length);
const pendingTagOperationSummaries = computed(() => {
  return (pendingTagOperations.value || []).map((item) => {
    const op = String(item?.operation || "").trim().toLowerCase();
    const tagName = String(item?.name || "").trim();
    const groupName = String(item?.group_name || "").trim();
    const groupID = String(item?.group_id || "").trim();
    const tagID = String(item?.tag_id || "").trim();
    if (op === "create") {
      return `新建标签组：${groupName || "(未命名)"} / 首标签：${tagName || "(未命名)"}`;
    }
    if (op === "rename_group") {
      return `重命名标签组：${groupID || "(无group_id)"} -> ${groupName || "(未命名)"}`;
    }
    if (op === "rename") {
      return `重命名标签：${tagID || "(无tag_id)"} -> ${tagName || "(未命名)"}`;
    }
    if (op === "delete") {
      return `删除标签：${tagID || "(无tag_id)"}${groupName ? `（${groupName}）` : ""}`;
    }
    return `${op || "unknown"}：${JSON.stringify(item)}`;
  });
});
const pendingTagRenameByID = computed(() => {
  const out = new Map<string, string>();
  (pendingTagOperations.value || []).forEach((item) => {
    const op = String(item?.operation || "").trim().toLowerCase();
    if (op !== "rename") return;
    const tagID = String(item?.tag_id || "").trim();
    const name = String(item?.name || "").trim();
    if (!tagID || !name) return;
    out.set(tagID, name);
  });
  return out;
});
const pendingGroupRenameByID = computed(() => {
  const out = new Map<string, string>();
  (pendingTagOperations.value || []).forEach((item) => {
    const op = String(item?.operation || "").trim().toLowerCase();
    if (op !== "rename_group") return;
    const groupID = String(item?.group_id || "").trim();
    const name = String(item?.group_name || item?.name || "").trim();
    if (!groupID || !name) return;
    out.set(groupID, name);
  });
  return out;
});

const activeTagAccount = computed(() => {
  const items = tagAccounts.value || [];
  if (items.length === 0) return null;
  return items.find((item) => Boolean(item?.org_sync_default)) || items[0] || null;
});

const activeTagAccountUUID = computed(() => String(activeTagAccount.value?.account_uuid || "").trim());

const tagMetaByID = computed(() => {
  const map = new Map<string, { tag_id: string; tag_name: string; group_id: string; group_name: string }>();
  (tagRecords.value || []).forEach((item: any) => {
    const tagID = String(item?.remote_tag_id || "").trim();
    if (!tagID) return;
    map.set(tagID, {
      tag_id: tagID,
      tag_name: String(item?.tag_name || tagID).trim(),
      group_id: String(item?.remote_group_id || "").trim(),
      group_name: String(item?.remote_group_name || "").trim(),
    });
  });
  (customerBindings.value || []).forEach((item: any) => {
    (item?.follow_users || []).forEach((fu: any) => {
      (fu?.tags || []).forEach((tag: any) => {
        const tagID = String(tag?.tag_id || "").trim();
        if (!tagID || map.has(tagID)) return;
        map.set(tagID, {
          tag_id: tagID,
          tag_name: String(tag?.tag_name || tagID).trim(),
          group_id: "",
          group_name: String(tag?.group_name || "").trim(),
        });
      });
    });
  });
  map.forEach((item, key) => {
    const nextTagName = pendingTagRenameByID.value.get(key);
    if (nextTagName) {
      item.tag_name = nextTagName;
    }
    const nextGroupName = pendingGroupRenameByID.value.get(String(item.group_id || "").trim());
    if (nextGroupName) {
      item.group_name = nextGroupName;
    }
  });
  return map;
});

const allTagOptions = computed(() => {
  return Array.from(tagMetaByID.value.values()).sort((a, b) => {
    const ga = String(a.group_name || "");
    const gb = String(b.group_name || "");
    if (ga !== gb) return ga.localeCompare(gb);
    return String(a.tag_name || "").localeCompare(String(b.tag_name || ""));
  });
});

const followUserCandidates = computed(() => {
  const out: Array<{ key: string; display_name: string; external_userid: string; userid: string; follow_user: any }> = [];
  (customerBindings.value || []).forEach((item: any) => {
    const externalUserID = String(item?.external_userid || "").trim();
    if (!externalUserID) return;
    (item?.follow_users || []).forEach((fu: any) => {
      const userID = String(fu?.userid || "").trim();
      if (!userID) return;
      out.push({
        key: buildOpKey(externalUserID, userID),
        display_name: String(item?.display_name || "").trim(),
        external_userid: externalUserID,
        userid: userID,
        follow_user: fu,
      });
    });
  });
  return out;
});

const relationRows = computed(() => {
  const rows: Array<{
    key: string;
    tagID: string;
    tagLabel: string;
    groupName: string;
    groupID: string;
    externalUserID: string;
    userID: string;
    displayName: string;
    isDangling: boolean;
  }> = [];
  (customerBindings.value || []).forEach((item: any) => {
    const externalUserID = String(item?.external_userid || "").trim();
    const displayName = String(item?.display_name || "").trim();
    if (!externalUserID) return;
    (item?.follow_users || []).forEach((fu: any) => {
      const userID = String(fu?.userid || "").trim();
      if (!userID) return;
      const tags = getDisplayTags(externalUserID, fu);
      tags.forEach((tag: any) => {
        const tagID = String(tag?.tag_id || "").trim();
        if (!tagID) return;
        const tagLabel = String(tag?.tag_name || tagID).trim();
        const groupName = String(tag?.group_name || "").trim() || "未分组";
        const groupID = String(tag?.group_id || "").trim();
        rows.push({
          key: `${externalUserID}:${userID}:${groupName}:${tagID}`,
          tagID,
          tagLabel,
          groupName,
          groupID,
          externalUserID,
          userID,
          displayName,
          isDangling: !tagMetaByID.value.has(tagID),
        });
      });
    });
  });
  return rows;
});

const danglingRelationRows = computed(() => relationRows.value.filter((row) => row.isDangling));

const relationRowsFiltered = computed(() => {
  const keyword = String(relationKeyword.value || "").trim().toLowerCase();
  return relationRows.value.filter((row) => {
    if (!keyword) return true;
    return [row.tagLabel, row.externalUserID, row.displayName, row.userID].join(" ").toLowerCase().includes(keyword);
  });
});

const relationGroups = computed(() => {
  const map = new Map<string, {
    tagID: string;
    tagLabel: string;
    groupName: string;
    groupID: string;
    isDangling: boolean;
    members: Array<{ key: string; externalUserID: string; displayName: string; userID: string }>;
  }>();
  (allTagOptions.value || []).forEach((tag) => {
    const tagID = String(tag?.tag_id || "").trim();
    if (!tagID) return;
    const groupName = String(tag?.group_name || "").trim() || "未分组";
    const mapKey = `${groupName}::${tagID}`;
    if (map.has(mapKey)) return;
    map.set(mapKey, {
      tagID,
      tagLabel: String(tag?.tag_name || tagID).trim(),
      groupName,
      groupID: String(tag?.group_id || "").trim(),
      isDangling: false,
      members: [],
    });
  });
  relationRowsFiltered.value.forEach((row) => {
    const mapKey = `${row.groupName}::${row.tagID}`;
    if (!map.has(mapKey)) {
      map.set(mapKey, {
        tagID: row.tagID,
        tagLabel: row.tagLabel || row.tagID,
        groupName: row.groupName || "未分组",
        groupID: row.groupID || "",
        isDangling: row.isDangling,
        members: [],
      });
    }
    const group = map.get(mapKey)!;
    group.isDangling = group.isDangling || row.isDangling;
    if (!group.groupID && row.groupID) {
      group.groupID = row.groupID;
    }
    group.members.push({
      key: row.key,
      externalUserID: row.externalUserID,
      displayName: row.displayName,
      userID: row.userID,
    });
  });
  return Array.from(map.values()).sort((a, b) =>
    `${a.groupName}/${a.tagLabel}`.localeCompare(`${b.groupName}/${b.tagLabel}`),
  );
});

const relationGroupSections = computed(() => {
  const map = new Map<string, Array<(typeof relationGroups.value)[number]>>();
  relationGroups.value.forEach((item) => {
    const key = String(item.groupName || "").trim() || "未分组";
    if (!map.has(key)) {
      map.set(key, []);
    }
    map.get(key)!.push(item);
  });
  return Array.from(map.entries())
    .map(([groupName, tags]) => ({
      groupName,
      groupID: tags.find((tag) => Boolean(tag.groupID))?.groupID
        || tags.map((tag) => String(tagMetaByID.value.get(tag.tagID)?.group_id || "").trim()).find(Boolean)
        || "",
      tags,
    }))
    .sort((a, b) => a.groupName.localeCompare(b.groupName));
});

const selectedRelationGroup = computed(() =>
  relationGroups.value.find((item) => `${item.groupName}::${item.tagID}` === selectedRelationTagID.value) || null,
);

const selectedRelationMembers = computed(() => selectedRelationGroup.value?.members || []);

const refreshTagAccounts = async () => {
  const service = useSocialChannelGovernanceService();
  const resp = await service.listChannelAccounts();
  const items = (((resp as any)?.data?.items || []) as any[]).filter((item) =>
    String(item?.channel_code || "").trim().toLowerCase() === "wechat"
    && String(item?.app_type || "").trim().toLowerCase() === "wecom"
    && String(item?.status || "").trim().toLowerCase() === "connected",
  );
  tagAccounts.value = items;
};

const refreshTagRecords = async () => {
  const service = useSocialChannelGovernanceService();
  beginGlobalLoading("加载标签数据...");
  try {
    const resp = await service.listFoundationTags({
      channel_account_uuid: activeTagAccountUUID.value || undefined,
      limit: 500,
    });
    tagRecords.value = ((resp as any)?.data?.items || []) as any[];
  } catch (err: any) {
    toast.add({ title: "加载标签列表失败", description: err?.message || "请稍后重试", color: "error" });
    tagRecords.value = [];
  } finally {
    endGlobalLoading();
  }
};

const refreshCustomerBindings = async () => {
  if (bindingInflight) {
    await bindingInflight;
    return;
  }
  const accountUUID = activeTagAccountUUID.value;
  if (!accountUUID) {
    customerBindings.value = [];
    bindingsLoadedAccountUUID.value = "";
    return;
  }
  bindingInflight = (async () => {
    const service = useSocialChannelGovernanceService();
    beginGlobalLoading("加载客户关系...");
    try {
      const resp = await service.listFoundationCustomerTagBindings({
        channel_account_uuid: accountUUID,
        limit: 80,
      });
      customerBindings.value = ((resp as any)?.data?.items || []) as any[];
      bindingsLoadedAccountUUID.value = accountUUID;
    } catch (err: any) {
      toast.add({ title: "加载客户标签绑定失败", description: err?.message || "请稍后重试", color: "error" });
      customerBindings.value = [];
    } finally {
      endGlobalLoading();
      bindingInflight = null;
    }
  })();
  await bindingInflight;
};

const ensureCustomerBindings = async (force = false) => {
  const accountUUID = activeTagAccountUUID.value;
  if (!accountUUID) {
    customerBindings.value = [];
    bindingsLoadedAccountUUID.value = "";
    return;
  }
  if (!force && bindingsLoadedAccountUUID.value === accountUUID && customerBindings.value.length > 0) {
    return;
  }
  await refreshCustomerBindings();
};

const refreshAll = async () => {
  loading.value = true;
  try {
    await refreshTagAccounts();
    await refreshTagRecords();
  } catch (err: any) {
    toast.add({ title: "加载失败", description: err?.message || "请稍后重试", color: "error" });
  } finally {
    loading.value = false;
    initializing.value = false;
  }
};

const buildOpKey = (externalUserID: string, userID: string) => `${String(externalUserID).trim()}::${String(userID).trim()}`;

const findPendingOperation = (externalUserID: string, userID: string) => {
  const key = buildOpKey(externalUserID, userID);
  return (pendingOperations.value || []).find((item) => buildOpKey(item.external_userid, item.userid) === key) || null;
};

const hasPendingFor = (externalUserID: string, userID: string) => Boolean(findPendingOperation(externalUserID, userID));

const resolveEffectiveTagIDs = (externalUserID: string, userID: string, baseTagIDs: string[]) => {
  const set = new Set((baseTagIDs || []).map((id) => String(id || "").trim()).filter(Boolean));
  const pending = findPendingOperation(externalUserID, userID);
  if (pending) {
    (pending.add_tag || []).forEach((tagID) => set.add(String(tagID || "").trim()));
    (pending.remove_tag || []).forEach((tagID) => set.delete(String(tagID || "").trim()));
  }
  return Array.from(set.values()).filter(Boolean);
};

const getDisplayTags = (externalUserID: string, followUser: any) => {
  const baseTags = ((followUser?.tags || []) as any[])
    .map((tag) => {
      const tagID = String(tag?.tag_id || "").trim();
      const meta = tagMetaByID.value.get(tagID);
      return {
        tag_id: tagID,
        tag_name: String(meta?.tag_name || tag?.tag_name || "").trim(),
        group_id: String(meta?.group_id || "").trim(),
        group_name: String(meta?.group_name || tag?.group_name || "").trim(),
      };
    })
    .filter((tag) => Boolean(tag.tag_id));
  const baseMap = new Map(baseTags.map((tag) => [tag.tag_id, tag]));
  const userID = String(followUser?.userid || "").trim();
  const effectiveTagIDs = resolveEffectiveTagIDs(externalUserID, userID, baseTags.map((tag) => tag.tag_id));
  return effectiveTagIDs.map((tagID) => {
    const normalizedTagID = String(tagID || "").trim();
    return baseMap.get(normalizedTagID) || tagMetaByID.value.get(normalizedTagID) || {
      tag_id: normalizedTagID,
      tag_name: normalizedTagID,
      group_id: "",
      group_name: "",
    };
  });
};

const upsertPendingOperation = (externalUserID: string, userID: string, addTagIDs: string[], removeTagIDs: string[]) => {
  const normalizedExternalUserID = String(externalUserID || "").trim();
  const normalizedUserID = String(userID || "").trim();
  if (!normalizedExternalUserID || !normalizedUserID) return;

  const idx = (pendingOperations.value || []).findIndex((item) =>
    buildOpKey(item.external_userid, item.userid) === buildOpKey(normalizedExternalUserID, normalizedUserID),
  );

  const addSet = new Set<string>();
  const removeSet = new Set<string>();
  if (idx >= 0) {
    (pendingOperations.value[idx]?.add_tag || []).forEach((tagID) => addSet.add(String(tagID || "").trim()));
    (pendingOperations.value[idx]?.remove_tag || []).forEach((tagID) => removeSet.add(String(tagID || "").trim()));
  }

  (addTagIDs || []).forEach((raw) => {
    const tagID = String(raw || "").trim();
    if (!tagID) return;
    removeSet.delete(tagID);
    addSet.add(tagID);
  });
  (removeTagIDs || []).forEach((raw) => {
    const tagID = String(raw || "").trim();
    if (!tagID) return;
    addSet.delete(tagID);
    removeSet.add(tagID);
  });

  const next: FoundationCustomerTagOperation = {
    external_userid: normalizedExternalUserID,
    userid: normalizedUserID,
    add_tag: Array.from(addSet.values()).filter(Boolean),
    remove_tag: Array.from(removeSet.values()).filter(Boolean),
  };

  if (next.add_tag.length === 0 && next.remove_tag.length === 0) {
    if (idx >= 0) pendingOperations.value.splice(idx, 1);
    return;
  }

  if (idx >= 0) {
    pendingOperations.value[idx] = next;
  } else {
    pendingOperations.value.unshift(next);
  }
};

const upsertPendingTagOperation = (operation: FoundationTagOperation) => {
  const op = String(operation?.operation || "").trim().toLowerCase() as FoundationTagOperation["operation"];
  if (op === "create" || op === "add") {
    const groupName = String(operation?.group_name || "").trim();
    const tagName = String(operation?.name || "").trim();
    if (!groupName || !tagName) return;
    const idx = pendingTagOperations.value.findIndex((item) =>
      String(item?.operation || "").trim().toLowerCase() === "create"
      && String(item?.group_name || "").trim() === groupName
      && String(item?.name || "").trim() === tagName,
    );
    const next: FoundationTagOperation = {
      operation: "create",
      tag_id: "",
      group_name: groupName,
      name: tagName,
    };
    if (idx >= 0) {
      pendingTagOperations.value[idx] = next;
    } else {
      pendingTagOperations.value.unshift(next);
    }
    return;
  }
  const groupID = String(operation?.group_id || "").trim();
  if (op === "rename_group") {
    const nextGroupName = String(operation?.group_name || operation?.name || "").trim();
    if (!groupID || !nextGroupName) return;
    const idx = pendingTagOperations.value.findIndex((item) =>
      String(item?.operation || "").trim().toLowerCase() === "rename_group"
      && String(item?.group_id || "").trim() === groupID,
    );
    const next: FoundationTagOperation = {
      operation: "rename_group",
      tag_id: "",
      group_id: groupID,
      group_name: nextGroupName,
    };
    if (idx >= 0) {
      pendingTagOperations.value[idx] = next;
    } else {
      pendingTagOperations.value.unshift(next);
    }
    return;
  }
  const tagID = String(operation?.tag_id || "").trim();
  if (!tagID || (op !== "rename" && op !== "delete")) return;
  const idx = pendingTagOperations.value.findIndex((item) => String(item?.tag_id || "").trim() === tagID);
  if (op === "delete") {
    const next: FoundationTagOperation = {
      operation: "delete",
      tag_id: tagID,
      group_id: String(operation?.group_id || "").trim() || undefined,
      group_name: String(operation?.group_name || "").trim() || undefined,
    };
    if (idx >= 0) {
      pendingTagOperations.value[idx] = next;
    } else {
      pendingTagOperations.value.unshift(next);
    }
    return;
  }
  const nextName = String(operation?.name || "").trim();
  if (!nextName) return;
  const next: FoundationTagOperation = {
    operation: "rename",
    tag_id: tagID,
    group_id: String(operation?.group_id || "").trim() || undefined,
    group_name: String(operation?.group_name || "").trim() || undefined,
    name: nextName,
  };
  if (idx >= 0 && pendingTagOperations.value[idx]?.operation === "delete") {
    return;
  }
  if (idx >= 0) {
    pendingTagOperations.value[idx] = next;
  } else {
    pendingTagOperations.value.unshift(next);
  }
};

const clearPendingOperations = () => {
  pendingOperations.value = [];
  pendingTagOperations.value = [];
};

const persistPushbackDraft = (accountUUID?: string) => {
  if (!process.client) return;
  const payload = {
    channel_account_uuid: String(accountUUID || activeTagAccountUUID.value || "").trim(),
    customer_tag_operations: pendingOperations.value || [],
    tag_operations: pendingTagOperations.value || [],
    updated_at: new Date().toISOString(),
  };
  if (
    !payload.channel_account_uuid
    || ((payload.customer_tag_operations || []).length === 0 && (payload.tag_operations || []).length === 0)
  ) {
    window.sessionStorage.removeItem(PUSHBACK_DRAFT_KEY);
    return;
  }
  window.sessionStorage.setItem(PUSHBACK_DRAFT_KEY, JSON.stringify(payload));
};

const restorePushbackDraft = () => {
  if (!process.client) return;
  const raw = window.sessionStorage.getItem(PUSHBACK_DRAFT_KEY);
  if (!raw) return;
  try {
    const parsed = JSON.parse(raw) as {
      customer_tag_operations?: FoundationCustomerTagOperation[];
      tag_operations?: FoundationTagOperation[];
    };
    const relationOps = Array.isArray(parsed?.customer_tag_operations) ? parsed.customer_tag_operations : [];
    const tagOps = Array.isArray(parsed?.tag_operations) ? parsed.tag_operations : [];
    if (pendingOperations.value.length === 0 && relationOps.length > 0) {
      pendingOperations.value = relationOps;
    }
    if (pendingTagOperations.value.length === 0 && tagOps.length > 0) {
      pendingTagOperations.value = tagOps;
    }
  } catch {
    // ignore broken draft payload
  }
};

const queueDanglingRemovals = () => {
  let count = 0;
  danglingRelationRows.value.forEach((row) => {
    upsertPendingOperation(row.externalUserID, row.userID, [], [row.tagID]);
    count++;
  });
  if (count > 0) {
    toast.add({ title: "已加入待回写", description: `已加入 ${count} 条失效关系清理`, color: "success" });
  } else {
    toast.add({ title: "没有可清理的失效关系", color: "warning" });
  }
};

const openCustomerTagEditor = (item: any, followUser: any) => {
  const externalUserID = String(item?.external_userid || "").trim();
  const userID = String(followUser?.userid || "").trim();
  if (!externalUserID || !userID) return;
  const baseTagIDs = ((followUser?.tags || []) as any[])
    .map((tag) => String(tag?.tag_id || "").trim())
    .filter(Boolean);
  customerTagEditor.externalUserID = externalUserID;
  customerTagEditor.userid = userID;
  customerTagEditor.displayName = String(item?.display_name || "").trim();
  customerTagEditor.baseTagIDs = baseTagIDs;
  customerTagEditor.selectedTagIDs = resolveEffectiveTagIDs(externalUserID, userID, baseTagIDs);
  customerTagEditor.open = true;
};

const toggleEditorTag = (tagID: string, checked: boolean) => {
  const id = String(tagID || "").trim();
  if (!id) return;
  const set = new Set(customerTagEditor.selectedTagIDs || []);
  if (checked) set.add(id);
  else set.delete(id);
  customerTagEditor.selectedTagIDs = Array.from(set.values());
};

const saveCustomerTagEditor = () => {
  const base = new Set((customerTagEditor.baseTagIDs || []).map((id) => String(id || "").trim()).filter(Boolean));
  const next = new Set((customerTagEditor.selectedTagIDs || []).map((id) => String(id || "").trim()).filter(Boolean));
  const add: string[] = [];
  const remove: string[] = [];
  next.forEach((tagID) => {
    if (!base.has(tagID)) add.push(tagID);
  });
  base.forEach((tagID) => {
    if (!next.has(tagID)) remove.push(tagID);
  });
  upsertPendingOperation(customerTagEditor.externalUserID, customerTagEditor.userid, add, remove);
  customerTagEditor.open = false;
};

const openTagCustomerEditor = async (tag: { tagId: string; tagName: string; tagUuid: string }) => {
  await ensureCustomerBindings(true);
  const tagID = String(tag?.tagId || "").trim();
  if (!tagID) return;
  tagCustomerEditor.tagID = tagID;
  tagCustomerEditor.tagLabel = String(tag?.tagName || tagID).trim();
  const selected = new Set<string>();
  followUserCandidates.value.forEach((candidate) => {
    const effectiveTagIDs = resolveEffectiveTagIDs(
      candidate.external_userid,
      candidate.userid,
      ((candidate.follow_user?.tags || []) as any[]).map((item) => String(item?.tag_id || "").trim()).filter(Boolean),
    );
    if (effectiveTagIDs.includes(tagID)) {
      selected.add(candidate.key);
    }
  });
  tagCustomerEditor.selectedKeys = Array.from(selected.values());
  tagCustomerEditor.open = true;
};

const toggleTagCustomerSelection = (key: string, checked: boolean) => {
  const normalized = String(key || "").trim();
  if (!normalized) return;
  const set = new Set(tagCustomerEditor.selectedKeys || []);
  if (checked) set.add(normalized);
  else set.delete(normalized);
  tagCustomerEditor.selectedKeys = Array.from(set.values());
};

const saveTagCustomerEditor = () => {
  const selected = new Set((tagCustomerEditor.selectedKeys || []).map((key) => String(key || "").trim()).filter(Boolean));
  const tagID = String(tagCustomerEditor.tagID || "").trim();
  if (!tagID) {
    tagCustomerEditor.open = false;
    return;
  }
  followUserCandidates.value.forEach((candidate) => {
    const baseTagIDs = ((candidate.follow_user?.tags || []) as any[])
      .map((tag) => String(tag?.tag_id || "").trim())
      .filter(Boolean);
    const effective = new Set(resolveEffectiveTagIDs(candidate.external_userid, candidate.userid, baseTagIDs));
    const shouldHave = selected.has(candidate.key);
    const has = effective.has(tagID);
    if (shouldHave && !has) {
      upsertPendingOperation(candidate.external_userid, candidate.userid, [tagID], []);
    }
    if (!shouldHave && has) {
      upsertPendingOperation(candidate.external_userid, candidate.userid, [], [tagID]);
    }
  });
  tagCustomerEditor.open = false;
};

const openRenameTag = () => {
  if (!selectedRelationGroup.value) return;
  const tag = selectedRelationGroup.value;
  tagRenameEditor.tagID = String(tag.tagID || "").trim();
  tagRenameEditor.groupID = "";
  tagRenameEditor.groupName = String(tag.groupName || "").trim();
  tagRenameEditor.currentName = String(tag.tagLabel || "").trim();
  tagRenameEditor.nextName = String(tag.tagLabel || "").trim();
  tagRenameEditor.open = true;
};

const openRenameGroup = (section: { groupName: string; groupID: string }) => {
  const groupID = String(section?.groupID || "").trim();
  if (!groupID) {
    toast.add({ title: "当前标签组缺少 group_id，暂无法改名", color: "warning" });
    return;
  }
  const groupName = String(section?.groupName || "").trim();
  groupRenameEditor.groupID = groupID;
  groupRenameEditor.currentName = groupName;
  groupRenameEditor.nextName = groupName;
  groupRenameEditor.open = true;
};

const openCreateGroupModal = () => {
  groupCreateEditor.groupName = "";
  groupCreateEditor.tagName = "";
  groupCreateEditor.open = true;
};

const saveRenameTag = () => {
  const tagID = String(tagRenameEditor.tagID || "").trim();
  const nextName = String(tagRenameEditor.nextName || "").trim();
  const currentName = String(tagRenameEditor.currentName || "").trim();
  if (!tagID) {
    tagRenameEditor.open = false;
    return;
  }
  if (!nextName) {
    toast.add({ title: "标签名称不能为空", color: "warning" });
    return;
  }
  if (nextName === currentName) {
    tagRenameEditor.open = false;
    return;
  }
  upsertPendingTagOperation({
    operation: "rename",
    tag_id: tagID,
    group_id: tagRenameEditor.groupID || undefined,
    group_name: tagRenameEditor.groupName || undefined,
    name: nextName,
  });
  tagRenameEditor.open = false;
  toast.add({ title: "标签改名已加入待回写", color: "success" });
};

const saveRenameGroup = () => {
  const groupID = String(groupRenameEditor.groupID || "").trim();
  const currentName = String(groupRenameEditor.currentName || "").trim();
  const nextName = String(groupRenameEditor.nextName || "").trim();
  if (!groupID) {
    groupRenameEditor.open = false;
    return;
  }
  if (!nextName) {
    toast.add({ title: "标签组名称不能为空", color: "warning" });
    return;
  }
  if (nextName === currentName) {
    groupRenameEditor.open = false;
    return;
  }
  upsertPendingTagOperation({
    operation: "rename_group",
    tag_id: "",
    group_id: groupID,
    group_name: nextName,
  });
  groupRenameEditor.open = false;
  toast.add({ title: "标签组改名已加入待回写", color: "success" });
};

const saveCreateGroup = () => {
  const groupName = String(groupCreateEditor.groupName || "").trim();
  const tagName = String(groupCreateEditor.tagName || "").trim();
  if (!groupName) {
    toast.add({ title: "标签组名称不能为空", color: "warning" });
    return;
  }
  if (!tagName) {
    toast.add({ title: "首个标签名称不能为空", color: "warning" });
    return;
  }
  upsertPendingTagOperation({
    operation: "create",
    tag_id: "",
    group_name: groupName,
    name: tagName,
  });
  groupCreateEditor.open = false;
  toast.add({ title: "新建标签组已加入待回写", color: "success" });
};

const queueDeleteTag = () => {
  if (!selectedRelationGroup.value) return;
  const tag = selectedRelationGroup.value;
  const tagID = String(tag.tagID || "").trim();
  if (!tagID) return;
  const confirmed = window.confirm(`确认删除标签“${tag.tagLabel}”？\n将同步清理该标签的所有客户关系，并在回写任务中推送到企业微信。`);
  if (!confirmed) return;
  relationRows.value
    .filter((row) => row.tagID === tagID)
    .forEach((row) => upsertPendingOperation(row.externalUserID, row.userID, [], [tagID]));
  upsertPendingTagOperation({
    operation: "delete",
    tag_id: tagID,
    group_name: String(tag.groupName || "").trim() || undefined,
  });
  toast.add({ title: "标签删除已加入待回写", description: "将先清理关系再删除标签", color: "success" });
};

const submitPushbackJob = async () => {
  if (submitLoading.value) return;
  const accountUUID = activeTagAccountUUID.value;
  if (!accountUUID) {
    toast.add({ title: "缺少可用渠道账号", color: "warning" });
    return;
  }
  if (pendingTotal.value === 0) {
    toast.add({ title: "暂无待回写变更", color: "warning" });
    return;
  }
  try {
    persistPushbackDraft(accountUUID);
    toast.add({ title: "已进入同步中心并开始回写", description: "若失败会保留本次待回写内容", color: "info" });
    if (process.client) {
      window.location.href = "/scrm/tag_sync_center?domain=tags&auto_pushback=1&source=enterprise";
      return;
    }
    await navigateTo("/scrm/tag_sync_center?domain=tags&auto_pushback=1&source=enterprise", { replace: true });
  } catch (err: any) {
    toast.add({ title: "提交回写任务失败", description: err?.message || "请稍后重试", color: "error" });
  }
};

onMounted(async () => {
  restorePushbackDraft();
  await refreshAll();
  await ensureCustomerBindings(true);
});

watch(
  () => activeTagAccountUUID.value,
  (value, oldValue) => {
    if (initializing.value) return;
    if (value === oldValue) return;
    bindingsLoadedAccountUUID.value = "";
    customerBindings.value = [];
    pendingOperations.value = [];
    pendingTagOperations.value = [];
    refreshTagRecords().catch(() => {});
    ensureCustomerBindings(true).catch(() => {});
  },
);

watch(
  () => relationGroups.value,
  (groups) => {
    if (groups.length === 0) {
      selectedRelationTagID.value = "";
      return;
    }
    if (!groups.some((item) => `${item.groupName}::${item.tagID}` === selectedRelationTagID.value)) {
      selectedRelationTagID.value = `${groups[0].groupName}::${groups[0].tagID}`;
    }
  },
  { immediate: true, deep: true },
);

watch(
  [pendingOperations, pendingTagOperations, activeTagAccountUUID],
  () => persistPushbackDraft(),
  { deep: true },
);
</script>
