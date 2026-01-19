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
        <div class="flex items-center gap-2">
          <UIcon name="i-heroicons-user-plus" class="text-primary" />
          <span class="font-medium">分配与状态</span>
        </div>
      </template>
      <div v-if="lead" class="space-y-6">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <UFormField label="负责人">
            <USelect
              v-model="selectedOwner"
              :items="memberOptions"
              placeholder="选择负责人"
              class="w-full"
            />
          </UFormField>
          <UFormField label="分配原因">
            <UInput v-model="assignReason" placeholder="可选" />
          </UFormField>
        </div>
        <div class="flex flex-col gap-2 sm:flex-row sm:justify-end">
          <UButton
            color="primary"
            :loading="store.detailLoading"
            :disabled="!selectedOwner"
            @click="submitAssign"
          >
            确认分配
          </UButton>
        </div>
        <UDivider />
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <UFormField label="跟进状态">
            <USelect
              v-model="selectedStatus"
              :items="statusOptions"
              placeholder="选择状态"
              class="w-full"
            />
          </UFormField>
        </div>
        <div class="flex flex-col gap-2 sm:flex-row sm:justify-end">
          <UButton
            color="primary"
            variant="soft"
            :loading="store.detailLoading"
            :disabled="!selectedStatus"
            @click="submitStatus"
          >
            更新状态
          </UButton>
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
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter, useToast } from "#imports";
import { useLeadCaptureStore } from "~/stores/scrm/lead_capture/lead_store";
import { useMemberService } from "~/composables/api/services/memberService";
import type { Member } from "~/composables/api/services/memberService";

definePageMeta({
  layout: "default",
});

const route = useRoute();
const router = useRouter();
const toast = useToast();
const store = useLeadCaptureStore();
const memberService = useMemberService();

const leadId = computed(() => String(route.params.lead_id || ""));
const lead = computed(() => store.leadDetail);
const assignments = computed(() => store.assignments);
const statusHistory = computed(() => store.statusHistory);

const members = ref<Member[]>([]);
const selectedOwner = ref<string>("");
const assignReason = ref("");
const selectedStatus = ref<string>("");

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

const statusOptions = computed(() => {
  const current = lead.value?.status || "new";
  switch (current) {
    case "new":
      return [{ label: "已分配", value: "assigned" }];
    case "assigned":
      return [{ label: "跟进中", value: "in_progress" }];
    case "in_progress":
      return [
        { label: "已转化", value: "converted" },
        { label: "已关闭", value: "closed" },
      ];
    default:
      return [];
  }
});

const memberOptions = computed(() =>
  members.value.map((member) => ({
    label: member.display_name || member.email || member.username,
    value: String(member.member_id),
  }))
);

const memberLabel = (memberId?: string) => {
  if (!memberId) return "";
  const found = members.value.find((item) => String(item.member_id) === String(memberId));
  return found?.display_name || found?.email || found?.username || "";
};

const refreshLead = async () => {
  if (!leadId.value) {
    return;
  }
  await Promise.all([
    store.fetchLead(leadId.value),
    store.fetchAssignments(leadId.value),
    store.fetchStatusHistory(leadId.value),
  ]);
  selectedOwner.value = lead.value?.owner_user_uuid || "";
};

const backToList = () => {
  router.push("/scrm/lead_capture");
};

const submitAssign = async () => {
  if (!leadId.value || !selectedOwner.value) {
    return;
  }
  try {
    await store.assignLead(leadId.value, {
      owner_user_uuid: selectedOwner.value,
      reason: assignReason.value.trim() || undefined,
    });
    await store.fetchAssignments(leadId.value);
    await store.fetchStatusHistory(leadId.value);
    toast.add({ title: "负责人已更新", color: "success" });
  } catch (err: any) {
    toast.add({ title: err?.message ?? "分配失败", color: "error" });
  }
};

const submitStatus = async () => {
  if (!leadId.value || !selectedStatus.value) {
    return;
  }
  try {
    await store.updateLeadStatus(leadId.value, { status: selectedStatus.value });
    await store.fetchStatusHistory(leadId.value);
    toast.add({ title: "状态已更新", color: "success" });
    selectedStatus.value = "";
  } catch (err: any) {
    toast.add({ title: err?.message ?? "更新失败", color: "error" });
  }
};

const loadMembers = async () => {
  members.value = await memberService.listAll();
};

onMounted(async () => {
  await loadMembers();
  await refreshLead();
});
</script>
