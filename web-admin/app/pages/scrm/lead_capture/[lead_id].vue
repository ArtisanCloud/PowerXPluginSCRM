<template>
  <UContainer class="py-10 space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="space-y-2">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          线索详情
        </h1>
        <p class="text-gray-600 dark:text-gray-300">
          查看线索基础信息与来源信息。
        </p>
      </div>
      <div class="flex items-center gap-2">
        <UButton variant="soft" icon="i-heroicons-arrow-left" @click="backToList">
          返回列表
        </UButton>
        <UButton
          icon="i-heroicons-arrow-path"
          variant="soft"
          :loading="store.detailLoading"
          @click="refreshLead"
        >
          刷新
        </UButton>
      </div>
    </div>

    <UAlert
      v-if="store.error"
      color="warning"
      variant="soft"
      icon="i-heroicons-exclamation-triangle"
    >
      <template #title>线索详情不可用</template>
      <template #description>{{ store.error }}</template>
    </UAlert>

    <UCard>
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-user" class="text-primary" />
          <span class="font-medium">基础信息</span>
        </div>
      </template>

      <div v-if="lead" class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div class="space-y-3">
          <div>
            <div class="text-xs text-gray-500">姓名</div>
            <div class="text-sm text-gray-900 dark:text-white">
              {{ lead.display_name || '未填写' }}
            </div>
          </div>
          <div>
            <div class="text-xs text-gray-500">手机号</div>
            <div class="text-sm text-gray-900 dark:text-white">
              {{ lead.phone || '未填写' }}
            </div>
          </div>
          <div>
            <div class="text-xs text-gray-500">邮箱</div>
            <div class="text-sm text-gray-900 dark:text-white">
              {{ lead.email || '未填写' }}
            </div>
          </div>
        </div>
        <div class="space-y-3">
          <div>
            <div class="text-xs text-gray-500">状态</div>
            <UBadge :color="statusMeta(lead.status).color" variant="soft">
              {{ statusMeta(lead.status).label }}
            </UBadge>
          </div>
          <div>
            <div class="text-xs text-gray-500">负责人</div>
            <div class="text-sm text-gray-900 dark:text-white">
              {{ memberLabel(lead.owner_user_uuid) || lead.owner_user_uuid || '未分配' }}
            </div>
          </div>
          <div>
            <div class="text-xs text-gray-500">创建时间</div>
            <div class="text-sm text-gray-900 dark:text-white">
              {{ lead.created_at || '-' }}
            </div>
          </div>
        </div>
      </div>
      <div v-else class="py-6 text-center text-sm text-gray-500">
        暂无可用数据。
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-user-plus" class="text-primary" />
            <span class="font-medium">分配与状态</span>
          </div>
          <UButton
            color="primary"
            variant="soft"
            :disabled="!lead"
            @click="openAssignModal"
          >
            编辑分配
          </UButton>
        </div>
      </template>
      <div v-if="lead" class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <div class="text-xs text-gray-500">负责人</div>
          <div class="text-sm text-gray-900 dark:text-white">
            {{ memberLabel(lead.owner_user_uuid) || lead.owner_user_uuid || '未分配' }}
          </div>
        </div>
        <div>
          <div class="text-xs text-gray-500">当前状态</div>
          <UBadge :color="statusMeta(lead.status).color" variant="soft">
            {{ statusMeta(lead.status).label }}
          </UBadge>
        </div>
        <div>
          <div class="text-xs text-gray-500">分配原因</div>
          <div class="text-sm text-gray-900 dark:text-white">
            {{ latestAssignmentReason || '未填写' }}
          </div>
        </div>
      </div>
      <div v-else class="py-6 text-center text-sm text-gray-500">
        暂无可用数据。
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-link" class="text-primary" />
          <span class="font-medium">来源信息</span>
          <UBadge v-if="lead?.has_merge" color="warning" variant="soft">已合并</UBadge>
        </div>
      </template>
      <div v-if="lead" class="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div>
          <div class="text-xs text-gray-500">渠道</div>
          <div class="text-sm text-gray-900 dark:text-white">
            {{ lead.source_channel || '未知' }}
          </div>
        </div>
        <div>
          <div class="text-xs text-gray-500">应用类型</div>
          <div class="text-sm text-gray-900 dark:text-white">
            {{ lead.source_app_type || '未知' }}
          </div>
        </div>
        <div>
          <div class="text-xs text-gray-500">渠道账号 UUID</div>
          <div class="text-sm text-gray-900 dark:text-white">
            {{ lead.source_account_uuid || '未知' }}
          </div>
        </div>
      </div>
      <div v-else class="py-6 text-center text-sm text-gray-500">
        暂无可用数据。
      </div>
    </UCard>


    <UCard>
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-clock" class="text-primary" />
          <span class="font-medium">历史记录</span>
        </div>
      </template>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <div class="text-sm font-medium text-gray-900 dark:text-white mb-3">分配历史</div>
          <div v-if="assignments.length" class="space-y-3">
            <div
              v-for="item in assignments"
              :key="item.assignment_uuid"
              class="rounded-lg border border-gray-200 dark:border-gray-700 p-3"
            >
              <div class="text-sm text-gray-900 dark:text-white">
                {{ memberLabel(item.owner_user_uuid) || item.owner_user_uuid }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ item.created_at }}
              </div>
              <div v-if="item.reason" class="text-xs text-gray-500 dark:text-gray-400">
                {{ item.reason }}
              </div>
            </div>
          </div>
          <div v-else class="text-sm text-gray-500">暂无分配记录。</div>
        </div>
        <div>
          <div class="text-sm font-medium text-gray-900 dark:text-white mb-3">状态历史</div>
          <div v-if="statusHistory.length" class="space-y-3">
            <div
              v-for="item in statusHistory"
              :key="item.history_uuid"
              class="rounded-lg border border-gray-200 dark:border-gray-700 p-3"
            >
              <div class="text-sm text-gray-900 dark:text-white">
                {{ statusMeta(item.from_status).label }} → {{ statusMeta(item.to_status).label }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ item.changed_at }}
              </div>
            </div>
          </div>
          <div v-else class="text-sm text-gray-500">暂无状态记录。</div>
        </div>
      </div>
    </UCard>

    <ToastAlert
      v-model="toast.visible"
      :title="toast.title"
      :message="toast.message"
      :color="toast.color"
      :duration="toast.duration"
    />

    <UModal
      v-model:open="assignModalOpen"
      :prevent-close="true"
      :dismissible="false"
      :modal="true"
      :title="'分配与状态'"
      :description="'分配负责人并更新跟进状态。'"
      :ui="{ content: 'max-w-3xl w-full' }"
    >
      <template #body>
        <UForm :state="assignForm" class="space-y-4 p-4 sm:p-5">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <UFormField label="负责人" required>
              <USelectMenu
                v-model="assignForm.owner_user_uuid"
                v-model:search="memberSearch"
                :items="memberOptions"
                value-key="value"
                label-key="label"
                searchable
                placeholder="搜索并选择负责人"
                class="w-full"
                :portal="false"
                :ui="{ content: 'z-[200]' }"
              />
            </UFormField>
            <UFormField label="分配原因">
              <UInput v-model="assignForm.reason" placeholder="可选" />
            </UFormField>
          </div>
          <hr class="border-gray-200/60 dark:border-gray-700" />
          <div>
            <div class="text-sm text-gray-500">状态</div>
            <UBadge :color="assignStatusMeta.color" variant="soft">
              {{ assignStatusMeta.label }}
            </UBadge>
          </div>
        </UForm>
      </template>
      <template #footer>
        <div class="flex w-full flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-end sm:gap-3">
          <UButton color="neutral" variant="subtle" :disabled="store.detailLoading" @click="closeAssignModal">
            取消
          </UButton>
          <UButton color="primary" :loading="store.detailLoading" @click="submitAssignForm">
            保存
          </UButton>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "#imports";
import { useLeadCaptureStore } from "~/stores/scrm/lead_capture/lead_store";
import { useMemberService } from "~/composables/api/services/memberService";
import type { Member } from "~/composables/api/services/memberService";
import ToastAlert from "~/components/ToastAlert.vue";

definePageMeta({
  layout: "default",
});

const route = useRoute();
const router = useRouter();
const store = useLeadCaptureStore();
const memberService = useMemberService();

const leadId = computed(() => String(route.params.lead_id || ""));
const lead = computed(() => store.leadDetail);
const assignments = computed(() => store.assignments);
const statusHistory = computed(() => store.statusHistory);
const latestAssignmentReason = computed(() => assignments.value[0]?.reason || "");

const members = ref<Member[]>([]);
const selectedOwner = ref<string>("");
const memberSearch = ref("");
const assignModalOpen = ref(false);
const assignForm = reactive<{
  owner_user_uuid: any;
  reason: string;
}>({
  owner_user_uuid: "",
  reason: "",
});

type ToastColor =
  | "primary"
  | "secondary"
  | "success"
  | "info"
  | "warning"
  | "error"
  | "neutral";

const toast = reactive({
  visible: false,
  title: "",
  message: "",
  color: "primary" as ToastColor,
  duration: 3000,
});

const statusMeta = (status?: string) => {
  switch (status) {
    case "assigned":
      return { label: "已分配", color: "primary" };
    case "in_progress":
      return { label: "跟进中", color: "warning" };
    case "converted":
      return { label: "已转化", color: "success" };
    case "closed":
      return { label: "已关闭", color: "neutral" };
    case "new":
    default:
      return { label: "新线索", color: "info" };
  }
};

const assignStatusMeta = computed(() => {
  const hasOwner =
    !!(lead.value?.owner_user_uuid || "").trim() ||
    !!assignForm.owner_user_uuid.trim();
  return hasOwner
    ? { label: "已分配", color: "primary" }
    : { label: "未分配", color: "neutral" };
});

const resolveMemberValue = (member: Member) => {
  const candidate =
    (member as any).member_id ??
    (member as any).id ??
    (member as any).user_id ??
    "";
  return candidate ? String(candidate) : "";
};

const memberOptions = computed(() =>
  members.value
    .map((member) => ({
      label: member.display_name || member.email || member.username,
      value: resolveMemberValue(member),
    }))
    .filter((item) => item.value)
);

const memberLabel = (memberId?: string) => {
  if (!memberId) return "";
  const found = members.value.find(
    (item) => resolveMemberValue(item) === String(memberId)
  );
  return found?.display_name || found?.email || found?.username || "";
};

const refreshLead = async () => {
  if (!leadId.value) {
    return;
  }
  await store.fetchLead(leadId.value);
  await loadMembers(lead.value?.tenant_uuid);
  await Promise.all([
    store.fetchAssignments(leadId.value),
    store.fetchStatusHistory(leadId.value),
  ]);
  selectedOwner.value = lead.value?.owner_user_uuid || "";
};

const backToList = () => {
  router.push("/scrm/lead_capture");
};

const openAssignModal = () => {
  if (!lead.value) return;
  const rawOwner = lead.value.owner_user_uuid || "";
  assignForm.owner_user_uuid = /^\d+$/.test(rawOwner) ? rawOwner : "";
  assignForm.reason = latestAssignmentReason.value || "";
  memberSearch.value = "";
  assignModalOpen.value = true;
};

const closeAssignModal = () => {
  blurActiveElement();
  assignModalOpen.value = false;
};

const submitAssignForm = async () => {
  if (!leadId.value) {
    return;
  }
  try {
    if (assignForm.owner_user_uuid) {
      if (members.value.length === 0) {
        showToast("成员列表为空，请刷新后再试", "warning");
        return;
      }
      const resolvedOwner = resolveMemberId(assignForm.owner_user_uuid);
      if (!resolvedOwner) {
        showToast("负责人无效，请重新选择", "warning");
        return;
      }
      await store.assignLead(leadId.value, {
        owner_user_uuid: resolvedOwner,
        reason: assignForm.reason.trim() || undefined,
      });
      await store.fetchAssignments(leadId.value);
    }
    showToast("已保存", "success");
    closeAssignModal();
    await refreshLead();
  } catch (err: any) {
    showToast(err?.message ?? "保存失败", "error");
  }
};

const loadMembers = async (tenantUUID?: string) => {
  members.value = await memberService.listBound(tenantUUID);
};

onMounted(async () => {
  await loadMembers();
  await refreshLead();
});

const showToast = (message: string, color: ToastColor = "primary", title = "") => {
  toast.title = title || message;
  toast.message = title ? message : "";
  toast.color = color;
  toast.visible = true;
};

const resolveMemberId = (value: any) => {
  if (value === null || value === undefined) return "";
  if (typeof value === "number") return String(value);
  if (typeof value === "object") {
    const candidate = value.value ?? value.member_id ?? value.id;
    if (candidate !== undefined && candidate !== null) {
      return String(candidate);
    }
    value = value.label ?? "";
  }
  const normalized = String(value).trim().toLowerCase();
  if (!normalized) return "";
  if (/^\d+$/.test(normalized)) return normalized;
  const match = members.value.find((member) =>
    [member.display_name, member.email, member.username].some(
      (field) => field && field.trim().toLowerCase() === normalized
    )
  );
  return match ? resolveMemberValue(match) : "";
};

const blurActiveElement = () => {
  if (!process.client) return;
  const el = document.activeElement as HTMLElement | null;
  if (el && typeof el.blur === "function") {
    el.blur();
  }
};
</script>
