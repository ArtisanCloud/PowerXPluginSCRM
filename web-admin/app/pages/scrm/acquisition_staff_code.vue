<template>
  <UContainer class="acq-staff-page max-w-none py-10 space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="space-y-2">
        <h1 class="acq-page-title text-2xl font-semibold">员工活码</h1>
        <p class="acq-page-subtitle">独立域 V2：管理员工活码、成员映射与状态。</p>
      </div>
      <div class="flex items-center gap-2">
        <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="loadData">刷新</UButton>
        <UButton icon="i-heroicons-plus" color="primary" @click="createOpen = true">新建活码</UButton>
      </div>
    </div>

    <section class="acq-table-shell rounded-2xl border border-white/15 bg-[#0d1a34]/92 shadow-xl">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-white/10 px-5 py-4">
        <div class="flex items-center gap-3">
          <div class="text-base font-semibold text-white">活码列表</div>
          <UBadge color="neutral" variant="soft">{{ staffCodes.length }} 条</UBadge>
        </div>
        <div class="flex items-center gap-2">
          <UInput
            v-model="keyword"
            icon="i-heroicons-magnifying-glass"
            placeholder="搜索活动名/Code Key"
            class="w-56"
          />
        </div>
      </div>

      <div class="overflow-x-auto">
        <table class="min-w-full">
          <thead>
            <tr class="border-b border-white/10 text-left text-sm text-slate-300">
              <th class="acq-th px-6 py-3 font-medium">活动</th>
              <th class="acq-th px-6 py-3 font-medium">Code Key</th>
              <th class="acq-th px-6 py-3 font-medium">渠道</th>
              <th class="acq-th px-6 py-3 font-medium">应用</th>
              <th class="acq-th px-6 py-3 font-medium">状态</th>
              <th class="acq-th px-6 py-3 font-medium">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="6" class="acq-td px-6 py-12 text-center text-sm">加载中...</td>
            </tr>
            <tr v-else-if="filteredStaffCodes.length === 0">
              <td colspan="6" class="acq-empty px-6 py-12 text-center text-sm">暂无数据</td>
            </tr>
            <tr
              v-for="item in filteredStaffCodes"
              v-else
              :key="item.staff_code_uuid"
              class="border-b border-white/5 text-sm"
            >
              <td class="acq-td px-6 py-3">{{ item.activity_name }}</td>
              <td class="acq-td px-6 py-3">{{ item.code_key }}</td>
              <td class="acq-td px-6 py-3">{{ item.channel }}</td>
              <td class="acq-td px-6 py-3">{{ item.app_type }}</td>
              <td class="acq-td px-6 py-3">
                <UBadge :color="statusColor(item.status)" variant="soft">{{ item.status }}</UBadge>
              </td>
              <td class="acq-td px-6 py-3">
                <div class="flex items-center gap-2">
                  <UButton size="xs" variant="soft" @click="setStatus(item.staff_code_uuid, 'active')">启用</UButton>
                  <UButton size="xs" variant="soft" color="neutral" @click="setStatus(item.staff_code_uuid, 'disabled')">停用</UButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <UModal
      v-model:open="createOpen"
      fullscreen
      :overlay="true"
      :ui="{ overlay: 'bg-slate-950/90' }"
    >
      <template #body>
        <div class="acq-create-panel h-svh w-svw border-0 bg-[#0b1730] shadow-2xl">
          <div class="flex h-full flex-col">
            <div class="acq-create-header flex items-center justify-between border-b border-white/10 px-10 py-5">
              <div>
                <div class="text-lg font-semibold text-slate-100">新建员工活码</div>
                <div class="text-sm text-slate-300">基础设置 + 回复设置 + 实时手机预览</div>
              </div>
              <UButton icon="i-heroicons-x-mark" variant="ghost" color="neutral" @click="closeCreatePanel" />
            </div>

            <div class="acq-create-body min-h-0 flex-1 overflow-hidden">
              <div class="acq-create-main overflow-y-auto px-10 py-8">
                <div class="space-y-6">
                  <div class="rounded-2xl border border-white/10 bg-[#13264a] p-6 shadow-sm">
                    <div class="mb-4 text-base font-semibold text-slate-100">基础设置</div>
                    <div class="acq-setting-rows">
                      <div class="acq-setting-row">
                        <div class="acq-setting-label">活动名称</div>
                        <div class="acq-setting-content">
                          <UInput
                            v-model="form.activity_name"
                            class="w-full"
                            :ui="{ root: 'w-full' }"
                            placeholder="请输入活动名称"
                          />
                        </div>
                      </div>

                      <div class="acq-setting-row">
                        <div class="acq-setting-label">选择企业成员</div>
                        <div class="acq-setting-content">
                          <div class="flex flex-wrap items-center gap-3">
                            <UButton color="primary" variant="outline" icon="i-heroicons-plus">选择成员</UButton>
                            <span class="text-sm text-slate-200">已选择 <span class="text-rose-300">{{ selectedMemberCount }}</span> 人</span>
                          </div>
                          <UTextarea
                            v-model="form.member_uuids_text"
                            class="mt-3 w-full"
                            :ui="{ root: 'w-full' }"
                            :rows="2"
                            placeholder="member-uuid-1,member-uuid-2（需 org_sync confirmed）"
                          />
                        </div>
                      </div>

                      <div class="acq-setting-row">
                        <div class="acq-setting-label">企业微信标签</div>
                        <div class="acq-setting-content">
                          <div class="acq-tag-tip">可在标签管理页面新建标签及查看标签组</div>
                          <div class="acq-tag-panel mt-3">
                            <div class="text-sm text-slate-300">为本次活动的新粉丝打标签</div>
                            <div class="mt-3 flex flex-wrap items-center gap-3">
                              <UButton color="primary" variant="outline" icon="i-heroicons-plus">选择标签</UButton>
                              <UInput
                                v-model="form.corp_tag_ids_text"
                                class="min-w-[220px] flex-1"
                                :ui="{ root: 'w-full' }"
                                placeholder="tag_a,tag_b"
                              />
                            </div>
                          </div>
                        </div>
                      </div>

                      <div class="acq-setting-row">
                        <div class="acq-setting-label">客户备注</div>
                        <div class="acq-setting-content">
                          <div class="flex h-10 items-center gap-3 rounded-lg border border-white/10 bg-[#0f203f] px-3">
                            <USwitch v-model="form.new_customer_remark_enabled" />
                            <span class="text-sm text-slate-200">
                              {{ form.new_customer_remark_enabled ? "开启" : "关闭" }}
                            </span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class="rounded-2xl border border-white/10 bg-[#13264a] p-6 shadow-sm">
                    <div class="mb-4 text-base font-semibold text-slate-100">回复设置（预置）</div>
                    <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
                      <UFormField label="欢迎语设置" class="lg:col-span-2">
                        <URadioGroup
                          v-model="form.welcome_mode"
                          :items="welcomeModeOptions"
                          value-key="value"
                          label-key="label"
                        />
                      </UFormField>
                      <UFormField label="预设欢迎语内容" class="lg:col-span-2">
                        <UTextarea
                          v-model="form.welcome_text"
                          class="w-full"
                          :rows="3"
                          placeholder="欢迎添加，我们将尽快联系你"
                        />
                      </UFormField>
                    </div>
                  </div>
                </div>
              </div>

              <div class="acq-create-side overflow-y-auto px-10 py-8">
                <div class="mb-4 text-base font-semibold text-slate-100">页面预览</div>
                <div class="acq-preview-stage">
                  <div class="acq-phone-shell">
                    <div class="acq-phone-notch" />
                    <div class="acq-phone-screen">
                      <div class="flex items-center justify-between bg-[#111827] px-4 py-2 text-[11px] text-white">
                        <span>09:41</span>
                        <span>4G · 100%</span>
                      </div>
                      <div class="flex items-center gap-3 bg-[#1f2f55] px-4 py-3 text-white">
                        <div class="h-9 w-9 rounded-full bg-emerald-500/25" />
                        <div class="min-w-0">
                          <div class="truncate text-sm font-semibold">{{ form.activity_name || "员工活码会话" }}</div>
                          <div class="text-[11px] text-slate-200">已分配 {{ selectedMemberCount }} 位成员</div>
                        </div>
                      </div>
                      <div class="space-y-3 px-3 py-4">
                        <div class="max-w-[86%] rounded-2xl rounded-tl-sm bg-white px-3 py-2 text-xs text-slate-700 shadow">
                          您好，我是企业助手，已为您分配专属顾问。
                        </div>
                        <div class="ml-auto max-w-[86%] rounded-2xl rounded-tr-sm bg-[#0ea5e9] px-3 py-2 text-xs text-white shadow">
                          我刚扫码进来，想咨询下服务方案。
                        </div>
                        <div
                          v-if="form.welcome_mode === 'send'"
                          class="max-w-[86%] rounded-2xl rounded-tl-sm bg-white px-3 py-2 text-xs text-slate-700 shadow"
                        >
                          {{ form.welcome_text || "欢迎添加，我们将尽快联系你" }}
                        </div>
                        <div class="rounded-xl border border-dashed border-slate-300 bg-slate-50 px-3 py-2 text-[11px] text-slate-500">
                          标签：{{ previewTagText }} · 渠道码：系统自动生成
                        </div>
                      </div>
                      <div class="flex items-center gap-2 border-t border-slate-200 bg-white px-3 py-2">
                        <div class="h-8 flex-1 rounded-full bg-slate-100 px-3 leading-8 text-xs text-slate-400">请输入消息...</div>
                        <div class="h-8 w-8 rounded-full bg-emerald-500/20" />
                        <div class="h-8 w-8 rounded-full bg-sky-500/20" />
                      </div>
                      <div class="flex justify-center bg-white pb-2">
                        <div class="h-1 w-20 rounded-full bg-slate-300" />
                      </div>
                    </div>
                  </div>
                </div>
                <div class="mx-auto mt-3 w-[320px] rounded-xl border border-white/10 bg-[#102345] px-3 py-2 text-xs text-slate-300">
                  预览摘要：{{ form.welcome_mode === "send" ? "入会发送欢迎语" : "静默入会" }}，标签 {{ previewTagText }}
                </div>
              </div>
            </div>

            <div class="flex items-center justify-end gap-2 border-t border-white/10 px-10 py-5">
              <UButton variant="ghost" @click="closeCreatePanel">取消</UButton>
              <UButton color="primary" :loading="creating" @click="createStaffCode">创建</UButton>
            </div>
          </div>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { useAcquisitionService, type StaffLiveCodeRecord } from "~/composables/api/services/acquisition";

const toast = useToast();
const service = useAcquisitionService();
const loading = ref(false);
const creating = ref(false);
const createOpen = ref(false);
const staffCodes = ref<StaffLiveCodeRecord[]>([]);
const keyword = ref("");

const form = reactive({
  activity_name: "",
  member_uuids_text: "",
  corp_tag_ids_text: "",
  new_customer_remark_enabled: false,
  welcome_mode: "send" as "send" | "silent",
  welcome_text: "欢迎添加，我们将尽快联系你",
});

const welcomeModeOptions = [
  { label: "员工欢迎语", value: "send" },
  { label: "不发欢迎语", value: "silent" },
];

const selectedMemberCount = computed(() =>
  form.member_uuids_text
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean).length
);

const previewTagText = computed(() => {
  const tags = form.corp_tag_ids_text
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
  return tags.length > 0 ? tags.join(" / ") : "未设置";
});

const filteredStaffCodes = computed(() => {
  const q = keyword.value.trim().toLowerCase();
  if (!q) return staffCodes.value;
  return staffCodes.value.filter((item) =>
    [item.activity_name, item.code_key, item.channel, item.app_type]
      .filter(Boolean)
      .join(" ")
      .toLowerCase()
      .includes(q)
  );
});

const statusColor = (status: string) => {
  if (status === "active") return "success";
  if (status === "disabled") return "neutral";
  return "warning";
};

const loadData = async () => {
  loading.value = true;
  try {
    const resp = await service.listStaffCodes({ limit: 200 });
    staffCodes.value = (resp as any)?.data?.items || [];
  } catch (error: any) {
    toast.add({ title: "加载员工活码失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    loading.value = false;
  }
};

const setStatus = async (staffCodeUUID: string, status: "active" | "disabled") => {
  try {
    const resp = await service.updateStaffCodeStatus(staffCodeUUID, status);
    const updated = (resp as any)?.data;
    if (updated) {
      staffCodes.value = staffCodes.value.map((item) =>
        item.staff_code_uuid === staffCodeUUID ? { ...item, ...updated } : item
      );
    }
    toast.add({ title: "状态已更新", color: "success" });
  } catch (error: any) {
    toast.add({ title: "更新失败", description: error?.message || "unknown error", color: "error" });
  }
};

const createStaffCode = async () => {
  const memberUUIDs = form.member_uuids_text
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
  const corpTagIDs = form.corp_tag_ids_text
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
  if (!form.activity_name || memberUUIDs.length === 0) {
    toast.add({ title: "请填写完整信息", color: "warning" });
    return;
  }
  creating.value = true;
  try {
    await service.createStaffCode({
      channel: "wechat",
      app_type: "wecom",
      activity_name: form.activity_name,
      member_uuids: memberUUIDs,
      corp_tag_ids: corpTagIDs,
      new_customer_remark_enabled: form.new_customer_remark_enabled,
    });
    closeCreatePanel();
    toast.add({ title: "创建成功", color: "success" });
    await loadData();
  } catch (error: any) {
    toast.add({ title: "创建失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    creating.value = false;
  }
};

const resetCreateForm = () => {
  Object.assign(form, {
    activity_name: "",
    member_uuids_text: "",
    corp_tag_ids_text: "",
    new_customer_remark_enabled: false,
    welcome_mode: "send",
    welcome_text: "欢迎添加，我们将尽快联系你",
  });
};

const closeCreatePanel = () => {
  createOpen.value = false;
  resetCreateForm();
};

watch(createOpen, (open, prev) => {
  if (prev && !open) {
    resetCreateForm();
  }
});

onMounted(loadData);
</script>

<style scoped>
.acq-staff-page {
  position: relative;
}

.acq-staff-page::before {
  content: "";
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    radial-gradient(1200px 500px at 85% -120px, rgba(16, 185, 129, 0.14), transparent 55%),
    radial-gradient(900px 420px at -10% -160px, rgba(59, 130, 246, 0.1), transparent 55%);
}

.acq-create-panel {
  color: #e2e8f0;
  opacity: 1;
  position: relative;
  isolation: isolate;
}

.acq-create-header {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.2), rgba(16, 185, 129, 0.12));
}

.acq-create-body {
  display: block;
}

.acq-create-main {
  background: #0f1f3f;
  min-width: 0;
}

.acq-create-side {
  background:
    radial-gradient(520px 220px at 80% -40px, rgba(59, 130, 246, 0.22), transparent 60%),
    #12264b;
  min-width: 0;
}

.acq-preview-stage {
  display: flex;
  justify-content: center;
  padding: 4px 0;
}

.acq-phone-shell {
  position: relative;
  width: 320px;
  height: 640px;
  border-radius: 44px;
  padding: 10px;
  background: linear-gradient(160deg, #020617, #111827);
  box-shadow:
    0 24px 60px rgba(2, 6, 23, 0.6),
    inset 0 0 0 1px rgba(148, 163, 184, 0.25);
}

.acq-phone-notch {
  position: absolute;
  top: 10px;
  left: 50%;
  transform: translateX(-50%);
  width: 128px;
  height: 22px;
  border-radius: 0 0 14px 14px;
  background: #020617;
  z-index: 2;
}

.acq-phone-screen {
  height: 100%;
  overflow: hidden;
  border-radius: 34px;
  background: #f3f5f9;
}

.acq-setting-rows {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.acq-setting-row {
  display: grid;
  grid-template-columns: 168px minmax(0, 1fr);
  align-items: start;
  gap: 14px;
}

.acq-setting-label {
  padding-top: 8px;
  font-size: 15px;
  line-height: 22px;
  font-weight: 600;
  color: #cbd5e1;
}

.acq-setting-content {
  min-width: 0;
}

.acq-setting-content :deep(.w-full) {
  width: 100% !important;
}

.acq-setting-content :deep(input),
.acq-setting-content :deep(textarea) {
  width: 100% !important;
}

.acq-tag-tip {
  border: 1px solid rgba(251, 146, 60, 0.35);
  background: rgba(249, 115, 22, 0.12);
  border-radius: 10px;
  padding: 10px 12px;
  color: #fdba74;
  font-size: 13px;
}

.acq-tag-panel {
  border: 1px solid rgba(148, 163, 184, 0.3);
  background: rgba(148, 163, 184, 0.16);
  border-radius: 12px;
  padding: 14px;
}

.acq-phone-screen :is(.text-slate-900, .text-slate-800, .text-slate-700) {
  color: #334155 !important;
}

.acq-phone-screen :is(.text-slate-600, .text-slate-500, .text-slate-400) {
  color: #64748b !important;
}

.acq-phone-screen .bg-white {
  color: #334155 !important;
}

.acq-create-panel :deep(input),
.acq-create-panel :deep(textarea),
.acq-create-panel :deep(select) {
  background: #0b1730 !important;
  color: #f8fafc !important;
  border-color: rgba(148, 163, 184, 0.35) !important;
}

.acq-create-panel :deep(input::placeholder),
.acq-create-panel :deep(textarea::placeholder) {
  color: #94a3b8 !important;
}

.acq-create-panel :deep(label),
.acq-create-panel :deep(.text-gray-500),
.acq-create-panel :deep(.text-gray-400) {
  color: #cbd5e1 !important;
}

.acq-create-panel :deep(.text-slate-600),
.acq-create-panel :deep(.text-slate-700),
.acq-create-panel :deep(.text-slate-800),
.acq-create-panel :deep(.text-slate-900) {
  color: #e2e8f0 !important;
}

.acq-create-panel :deep(.ring-default),
.acq-create-panel :deep(.border-default) {
  border-color: rgba(148, 163, 184, 0.35) !important;
}

.acq-table-shell :deep(input) {
  background: rgba(15, 23, 42, 0.55) !important;
  color: #f8fafc !important;
  border-color: rgba(148, 163, 184, 0.35) !important;
}

.acq-table-shell :deep(input::placeholder) {
  color: #94a3b8 !important;
}

.acq-page-title {
  color: #f8fafc;
}

.acq-page-subtitle {
  color: #cbd5e1;
}

.acq-th {
  color: #cbd5e1 !important;
}

.acq-td {
  color: #f1f5f9 !important;
}

.acq-empty {
  color: #94a3b8 !important;
}

.acq-table-shell :deep(thead th) {
  color: #cbd5e1 !important;
}

.acq-table-shell :deep(tbody td) {
  color: #f1f5f9 !important;
}

@media (min-width: 1024px) {
  .acq-create-body {
    display: grid;
    grid-template-columns: minmax(0, 1.35fr) minmax(360px, 1fr);
  }

  .acq-create-main,
  .acq-create-side {
    height: 100%;
  }

  .acq-create-side {
    border-left: 1px solid rgba(148, 163, 184, 0.35);
  }
}

@media (max-width: 1400px) {
  .acq-phone-shell {
    width: 300px;
    height: 600px;
  }
}

@media (max-width: 1024px) {
  .acq-setting-row {
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .acq-setting-label {
    padding-top: 0;
  }
}

.acq-table-shell ::selection {
  background: rgba(59, 130, 246, 0.35);
  color: #ffffff;
}
</style>
