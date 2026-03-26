<template>
  <UContainer class="py-10 space-y-6">
    <div class="space-y-2">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">员工欢迎语</h1>
      <p class="text-gray-600 dark:text-gray-300">按员工活码配置欢迎语，支持结构化编辑并查看 JSON 预览。</p>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium">配置</span>
          <UButton variant="soft" icon="i-heroicons-arrow-path" :loading="loading" @click="loadStaffCodes">刷新活码</UButton>
        </div>
      </template>
      <div class="space-y-4">
        <UFormField label="员工活码">
          <USelectMenu
            v-model="selectedStaffCodeUUID"
            :items="staffCodeOptions"
            value-key="value"
            label-key="label"
            placeholder="请选择员工活码"
          />
        </UFormField>
        <UFormField label="欢迎模式">
          <URadioGroup v-model="welcomeMode" :items="welcomeModeOptions" value-key="value" label-key="label" />
        </UFormField>
        <UFormField label="内容块(JSON数组)">
          <UTextarea v-model="contentBlocksText" :rows="8" />
        </UFormField>
        <div class="flex items-center gap-2">
          <UButton color="primary" :loading="saving" :disabled="!selectedStaffCodeUUID" @click="saveConfig">保存配置</UButton>
          <UButton variant="soft" :loading="syncing" :disabled="!selectedStaffCodeUUID" @click="triggerSync">同步渠道</UButton>
          <UButton variant="soft" :loading="statusLoading" :disabled="!selectedStaffCodeUUID" @click="refreshStatus">刷新状态</UButton>
        </div>

        <div class="rounded border border-gray-200 p-3 text-sm text-gray-700 dark:border-gray-700 dark:text-gray-200">
          <div>同步状态：<strong>{{ syncStatus?.sync_status || '-' }}</strong></div>
          <div>最近错误：{{ syncStatus?.last_sync_error || '-' }}</div>
          <div>最近尝试：{{ syncStatus?.latest_attempt_no ?? '-' }}</div>
        </div>

        <UFormField label="Payload 预览(JSON)">
          <UTextarea :model-value="payloadPreviewText" :rows="8" readonly />
        </UFormField>
      </div>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
import { useAcquisitionService, type StaffWelcomeSyncStatus } from "~/composables/api/services/acquisition";

const toast = useToast();
const service = useAcquisitionService();

const loading = ref(false);
const saving = ref(false);
const syncing = ref(false);
const statusLoading = ref(false);
const selectedStaffCodeUUID = ref("");
const staffCodeOptions = ref<Array<{ label: string; value: string }>>([]);
const welcomeMode = ref<"send" | "silent">("send");
const welcomeModeOptions = [
  { label: "发送欢迎语", value: "send" },
  { label: "静默", value: "silent" },
];
const contentBlocksText = ref('[{"type":"text","text":"欢迎添加企业微信"}]');
const payloadPreviewText = ref("{}");
const syncStatus = ref<StaffWelcomeSyncStatus | null>(null);

const loadStaffCodes = async () => {
  loading.value = true;
  try {
    const resp = await service.listStaffCodes({ limit: 200 });
    const items = (resp as any)?.data?.items || [];
    staffCodeOptions.value = items.map((item: any) => ({
      value: item.staff_code_uuid,
      label: `${item.activity_name} (${item.code_key})`,
    }));
    if (!selectedStaffCodeUUID.value && staffCodeOptions.value.length > 0) {
      selectedStaffCodeUUID.value = staffCodeOptions.value[0].value;
    }
  } catch (error: any) {
    toast.add({ title: "加载员工活码失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    loading.value = false;
  }
};

const saveConfig = async () => {
  if (!selectedStaffCodeUUID.value) {
    toast.add({ title: "请先选择员工活码", color: "warning" });
    return;
  }
  let blocks: any[] = [];
  try {
    blocks = JSON.parse(contentBlocksText.value || "[]");
  } catch {
    toast.add({ title: "内容块 JSON 非法", color: "error" });
    return;
  }
  saving.value = true;
  try {
    const resp = await service.saveStaffWelcome(selectedStaffCodeUUID.value, {
      welcome_mode: welcomeMode.value,
      content_blocks: blocks,
    });
    const data = (resp as any)?.data;
    payloadPreviewText.value = JSON.stringify(data?.payload_preview || {}, null, 2);
    toast.add({ title: "欢迎语已保存", color: "success" });
  } catch (error: any) {
    toast.add({ title: "保存失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    saving.value = false;
  }
};

const triggerSync = async () => {
  if (!selectedStaffCodeUUID.value) return;
  syncing.value = true;
  try {
    const resp = await service.triggerStaffWelcomeSync(selectedStaffCodeUUID.value);
    const data = (resp as any)?.data;
    toast.add({ title: data?.message || "已触发同步", color: "info" });
    await refreshStatus();
  } catch (error: any) {
    toast.add({ title: "触发失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    syncing.value = false;
  }
};

const refreshStatus = async () => {
  if (!selectedStaffCodeUUID.value) return;
  statusLoading.value = true;
  try {
    const resp = await service.getStaffWelcomeSyncStatus(selectedStaffCodeUUID.value);
    syncStatus.value = (resp as any)?.data || null;
  } catch (error: any) {
    toast.add({ title: "加载状态失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    statusLoading.value = false;
  }
};

watch(selectedStaffCodeUUID, () => {
  syncStatus.value = null;
  payloadPreviewText.value = "{}";
});

onMounted(loadStaffCodes);
</script>
