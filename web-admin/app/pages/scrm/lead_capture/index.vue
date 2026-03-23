<template>
  <UContainer class="py-10 space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="space-y-2">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          线索管理
        </h1>
        <p class="text-gray-600 dark:text-gray-300">
          统一查看线索来源、状态与基础信息。
        </p>
      </div>
      <div class="flex items-center gap-2">
        <UButton
          icon="i-heroicons-adjustments-horizontal"
          variant="soft"
          to="/admin/iam/dictionaries"
        >
          来源配置
        </UButton>
        <UButton
          icon="i-heroicons-arrow-path"
          variant="soft"
          :loading="store.loading"
          @click="refreshLeads"
        >
          刷新
        </UButton>
        <UButton
          icon="i-heroicons-arrow-up-tray"
          variant="soft"
          @click="openImportModal"
        >
          批量导入
        </UButton>
        <UButton icon="i-heroicons-plus" color="primary" @click="openCreateModal">
          新建线索
        </UButton>
      </div>
    </div>

    <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
      <div class="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-end">
        <UFormField label="搜索" class="w-full sm:w-64">
          <UInput
            v-model="searchText"
            icon="i-heroicons-magnifying-glass"
            placeholder="搜索姓名/手机号/邮箱"
            class="w-full"
          />
        </UFormField>
        <UFormField label="状态" class="w-full sm:w-40">
          <USelectMenu
            v-model="statusFilter"
            :items="statusFilterOptions"
            value-key="value"
            label-key="label"
            placeholder="全部状态"
            class="w-full"
            :portal="false"
            :ui="{ content: 'z-[200]' }"
          />
        </UFormField>
        <UFormField label="流量平台" class="w-full sm:w-40">
          <USelectMenu
            v-model="channelFilter"
            :items="channelOptions"
            value-key="value"
            label-key="label"
            placeholder="全部渠道"
            class="w-full"
            :portal="false"
            :ui="{ content: 'z-[200]' }"
          />
        </UFormField>
        <UFormField label="流量来源" class="w-full sm:w-40">
          <USelectMenu
            v-model="appTypeFilter"
            :items="appTypeOptions"
            value-key="value"
            label-key="label"
            placeholder="全部应用"
            class="w-full"
            :portal="false"
            :ui="{ content: 'z-[200]' }"
          />
        </UFormField>
      </div>
      <div class="text-sm text-gray-500 dark:text-gray-400">
        共 {{ filteredLeads.length }} 条
      </div>
    </div>

    <UCard>
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-arrow-path-rounded-square" class="text-primary" />
            <span class="font-medium text-gray-900 dark:text-gray-100">渠道同步任务</span>
          </div>
          <div class="flex items-center gap-2">
            <UButton size="xs" variant="soft" :loading="syncLoading" @click="refreshSyncTasks">
              刷新任务
            </UButton>
            <UButton size="xs" color="primary" :loading="syncSubmitting" @click="triggerWeComSync">
              触发同步
            </UButton>
          </div>
        </div>
      </template>
      <div class="space-y-3">
        <div class="grid grid-cols-1 gap-4 xl:grid-cols-12">
          <div class="xl:col-span-8">
            <div class="rounded-xl border border-gray-200/80 bg-gray-50/70 p-4 dark:border-gray-700/80 dark:bg-gray-900/40">
              <div class="mb-3 flex items-center justify-between">
                <div class="text-sm font-medium text-gray-800 dark:text-gray-100">
                  同步筛选
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  触发同步前可切换账号与状态范围
                </div>
              </div>
              <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
                <UFormField label="同步账号（可选）">
                  <USelectMenu
                    v-model="syncAccountUUID"
                    :items="syncAccountOptions"
                    value-key="value"
                    label-key="label"
                    placeholder="不填走默认账号"
                    class="w-full"
                    :portal="false"
                    :ui="{ content: 'z-[200]' }"
                  />
                </UFormField>
                <UFormField label="状态筛选">
                  <USelectMenu
                    v-model="syncStatusFilter"
                    :items="syncStatusOptions"
                    value-key="value"
                    label-key="label"
                    class="w-full"
                    :portal="false"
                    :ui="{ content: 'z-[200]' }"
                  />
                </UFormField>
              </div>
              <div
                class="mt-3 rounded-lg border border-dashed border-gray-300/80 bg-white/70 px-3 py-2 text-xs text-gray-600 dark:border-gray-700 dark:bg-gray-950/30 dark:text-gray-300"
              >
                最近一次账号解析来源：<span class="font-medium text-gray-800 dark:text-gray-100">{{ syncLastResolveSource || "未触发" }}</span>
              </div>
            </div>
          </div>

          <div class="xl:col-span-4">
            <div class="h-full rounded-xl border border-primary/30 bg-primary/5 p-4 dark:bg-primary/10">
              <div class="space-y-3">
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">
                      客户私信自动建线索（WeCom）
                    </div>
                    <div class="mt-1 text-xs leading-5 text-gray-600 dark:text-gray-300">
                      关闭时进入待绑定池，开启后客户私信自动入池。
                    </div>
                  </div>
                  <USwitch
                    v-model="wecomCustomerDMAutoCreate"
                    :loading="wecomCustomerDMRuleLoading || wecomCustomerDMRuleSaving"
                  />
                </div>
                <div class="flex justify-end">
                  <UButton
                    size="xs"
                    color="primary"
                    :loading="wecomCustomerDMRuleSaving"
                    @click="saveWeComCustomerDMRule"
                  >
                    保存规则
                  </UButton>
                </div>
              </div>
            </div>
          </div>
        </div>

        <UTable :columns="syncTaskColumns" :data="pagedSyncTasks" :loading="syncLoading">
          <template #task_uuid-cell="{ row }">
            <button
              type="button"
              class="max-w-[180px] truncate text-left text-sm text-gray-300 hover:text-primary cursor-pointer"
              :title="row.original.task_uuid"
              @click="copyTaskUUID(row.original.task_uuid)"
            >
              {{ shortUUID(row.original.task_uuid) }}
            </button>
          </template>
          <template #channel_account_uuid-cell="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-200">
              {{ resolveSyncAccountLabel(row.original) }}
            </span>
          </template>
          <template #status-cell="{ row }">
            <UBadge :color="syncStatusMeta(row.original.status).color" variant="soft">
              {{ syncStatusMeta(row.original.status).label }}
            </UBadge>
          </template>
          <template #progress-cell="{ row }">
            <div class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-300">
              <div class="h-1.5 w-24 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                <div
                  class="h-full rounded-full bg-primary transition-all duration-300"
                  :style="{ width: `${syncProgressPercent(row.original)}%` }"
                />
              </div>
              <span>{{ syncProgressPercent(row.original) }}%</span>
            </div>
          </template>
          <template #stats-cell="{ row }">
            <div class="text-xs text-gray-500 dark:text-gray-400">
              总数 {{ row.original.stats_total || 0 }} / 新增 {{ row.original.stats_created || 0 }} /
              更新 {{ row.original.stats_updated || 0 }} / 合并 {{ row.original.stats_merged || 0 }}
            </div>
          </template>
          <template #error-cell="{ row }">
            <span class="text-xs text-amber-500">{{ row.original.error_message || "-" }}</span>
          </template>
        </UTable>
        <div class="flex flex-col gap-3 pt-2 sm:flex-row sm:items-center sm:justify-between">
          <div class="text-xs text-gray-500 dark:text-gray-400">
            任务第 {{ syncTaskPage }} / {{ syncTaskTotalPages }} 页（共 {{ syncTasks.length }} 条）
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <USelectMenu
              v-model="syncTaskPageSize"
              :items="syncTaskPageSizeOptions"
              value-key="value"
              label-key="label"
              class="w-24"
              :portal="false"
              :ui="{ content: 'z-[200]' }"
            />
            <UButton size="xs" variant="soft" :disabled="syncTaskPage <= 1" @click="syncTaskPrevPage">
              上一页
            </UButton>
            <UButton size="xs" variant="soft" :disabled="syncTaskPage >= syncTaskTotalPages" @click="syncTaskNextPage">
              下一页
            </UButton>
          </div>
        </div>
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-rectangle-stack" class="text-primary" />
            <span class="font-medium text-gray-900 dark:text-gray-100">线索列表</span>
          </div>
          <UBadge variant="soft" color="primary">{{ filteredLeads.length }}</UBadge>
        </div>
      </template>

      <UTable
        :columns="leadColumns"
        :data="pagedLeads"
        :loading="store.loading"
        :ui="{ table: 'min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-700' }"
      >
        <template #display_name-cell="{ row }">
          <div class="space-y-1">
            <div class="font-medium text-gray-900 dark:text-white">
              {{ row.original.display_name || row.original.phone || row.original.email || '未命名线索' }}
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              {{ row.original.lead_uuid }}
            </div>
          </div>
        </template>
        <template #contact-cell="{ row }">
          <div class="space-y-1 text-sm text-gray-600 dark:text-gray-300">
            <div>{{ row.original.phone || '暂无手机号' }}</div>
            <div>{{ row.original.email || '暂无邮箱' }}</div>
          </div>
        </template>
        <template #status-cell="{ row }">
          <div class="flex flex-wrap items-center gap-2">
            <UBadge :color="statusMeta(row.original.status).color" variant="soft">
              {{ statusMeta(row.original.status).label }}
            </UBadge>
            <UBadge v-if="row.original.has_merge" color="warning" variant="soft">
              已合并
            </UBadge>
          </div>
        </template>
        <template #source-cell="{ row }">
          <div class="space-y-1 text-sm text-gray-600 dark:text-gray-300">
            <div>{{ row.original.source_channel || '未知渠道' }}</div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              {{ row.original.source_app_type || '未知应用' }}
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              外部联系人ID：{{ leadSyncExternalInfo(row.original.lead_uuid).externalLeadId || '未记录（需重跑同步）' }}
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              微信号：{{ leadSyncExternalInfo(row.original.lead_uuid).externalWechatId || '未记录（需重跑同步）' }}
            </div>
          </div>
        </template>
        <template #actions-cell="{ row }">
          <div class="flex flex-wrap gap-2">
            <UButton size="xs" variant="soft" @click="openDetail(row.original.lead_uuid)">
              查看
            </UButton>
          </div>
        </template>
      </UTable>

      <div v-if="!store.loading && filteredLeads.length === 0" class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">
        暂无线索，先创建一条试试。
      </div>
    </UCard>

    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="text-sm text-gray-500 dark:text-gray-400">
        当前第 {{ currentPage }} / {{ totalPages }} 页
      </div>
      <div class="flex flex-wrap items-center gap-3">
        <USelectMenu
          v-model="pageSize"
          :items="pageSizeOptions"
          value-key="value"
          label-key="label"
          class="w-28"
          :portal="false"
          :ui="{ content: 'z-[200]' }"
        />
        <UButton variant="soft" :disabled="currentPage <= 1" @click="prevPage">上一页</UButton>
        <UButton variant="soft" :disabled="currentPage >= totalPages" @click="nextPage">下一页</UButton>
      </div>
    </div>

    <UModal
      v-model:open="createModalOpen"
      :prevent-close="true"
      :dismissible="false"
      :modal="true"
      :ui="{ content: 'max-w-3xl w-full' }"
    >
      <template #title>新建线索</template>
      <template #description>
        填写基础信息即可创建，姓名/手机号/邮箱至少填一项。
      </template>
      <template #body>
        <UForm :state="createForm" class="space-y-4 p-4 sm:p-5">
          <UFormField label="联系人信息（至少一项）" required>
            <p class="text-xs text-gray-500">姓名 / 手机号 / 邮箱 至少填写一项。</p>
          </UFormField>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <UFormField label="姓名" required>
              <UInput v-model="createForm.display_name" placeholder="线索姓名" />
            </UFormField>
            <UFormField label="手机号" required>
              <UInput v-model="createForm.phone" placeholder="手机号" />
            </UFormField>
          </div>
          <UFormField label="邮箱" required>
            <UInput v-model="createForm.email" placeholder="邮箱" />
          </UFormField>
          <UAlert
            v-if="createFormError"
            color="warning"
            variant="soft"
            icon="i-heroicons-exclamation-triangle"
            :description="createFormError"
          />
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <UFormField label="流量平台">
              <USelectMenu
                v-model="createForm.source_channel"
                :items="createSourceChannelOptions"
                value-key="value"
                label-key="label"
                placeholder="请选择来源渠道"
                class="w-full"
                :portal="false"
                :ui="{ content: 'z-[200]' }"
              />
            </UFormField>
            <UFormField label="流量来源">
              <USelectMenu
                v-model="createForm.source_app_type"
                :items="createSourceAppTypeOptions"
                value-key="value"
                label-key="label"
                placeholder="请选择应用类型"
                class="w-full"
                :portal="false"
                :ui="{ content: 'z-[200]' }"
              />
            </UFormField>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <UFormField label="关联系统渠道账号（高级）">
              <USelectMenu
                v-model="createForm.source_account_uuid"
                :items="createAccountOptions"
                value-key="value"
                label-key="label"
                placeholder="请选择系统内渠道账号"
                class="w-full"
                :portal="false"
                :ui="{ content: 'z-[200]' }"
              />
            </UFormField>
            <UFormField label="负责人（高级）">
              <USelectMenu
                v-model="createForm.owner_user_uuid"
                :items="createOwnerOptions"
                value-key="value"
                label-key="label"
                placeholder="请选择负责人（可选）"
                class="w-full"
                :portal="false"
                :ui="{ content: 'z-[200]' }"
              />
            </UFormField>
          </div>
        </UForm>
      </template>
      <template #footer>
        <div class="flex flex-col gap-2 sm:flex-row sm:justify-end">
          <UButton color="neutral" variant="soft" :disabled="creating" @click="closeCreateModal">
            取消
          </UButton>
          <UButton color="primary" :loading="creating" @click="submitCreate">
            创建
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="importModalOpen"
      :prevent-close="true"
      :dismissible="false"
      :modal="true"
      :title="'批量导入线索'"
      :description="importStepDescription"
      :ui="{ content: 'max-w-5xl w-full' }"
    >
      <template #body>
        <UForm :state="importForm" class="space-y-4 p-4 sm:p-5">
          <div class="flex flex-wrap items-center gap-3">
            <UButton variant="soft" @click="downloadTemplate">下载模板</UButton>
            <span class="text-sm text-gray-500">支持 .csv</span>
          </div>
          <UFormField label="导入文件" required>
            <input
              type="file"
              accept=".csv"
              class="block w-full text-sm text-gray-300 file:mr-4 file:rounded file:border-0 file:bg-slate-700 file:px-3 file:py-2 file:text-sm file:text-white hover:file:bg-slate-600"
              @change="onFileChange"
            />
            <p v-if="importFileName" class="mt-2 text-xs text-gray-500">
              已选择：{{ importFileName }}
            </p>
          </UFormField>

          <div v-if="importStep >= 2" class="space-y-3">
            <div class="text-sm text-gray-500">字段映射</div>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <UFormField
                v-for="field in importFields"
                :key="field.key"
                :label="field.label"
                :required="field.required"
              >
                <USelectMenu
                  v-model="mappingForm[field.key]"
                  :items="headerOptions"
                  value-key="value"
                  label-key="label"
                  placeholder="请选择列"
                  class="w-full"
                  :portal="false"
                  :ui="{ content: 'z-[200]' }"
                />
                <p v-if="mappingIssues[field.key]" class="mt-1 text-xs text-amber-500">
                  {{ mappingIssues[field.key] }}
                </p>
              </UFormField>
            </div>
            <div v-if="previewRows.length" class="text-xs text-gray-500">
              <div class="font-medium mb-2">原始文件预览（前 {{ previewRows.length }} 行）</div>
              <div class="overflow-auto border border-gray-700 rounded-lg">
                <table class="min-w-full text-xs text-gray-300">
                  <thead class="bg-slate-900/60">
                    <tr>
                      <th v-for="header in previewHeaders" :key="header" class="px-3 py-2 text-left">
                        {{ header }}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="(row, idx) in previewRows" :key="idx" class="border-t border-gray-700/60">
                      <td v-for="(cell, col) in row" :key="col" class="px-3 py-2">
                        {{ cell }}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
            <div v-if="mappedPreviewFields.length" class="text-xs text-gray-500">
              <div class="font-medium mb-2">映射预览（按当前字段映射展示）</div>
              <div class="overflow-auto border border-gray-700 rounded-lg">
                <table class="min-w-full text-xs text-gray-300">
                  <thead class="bg-slate-900/60">
                    <tr>
                      <th
                        v-for="field in mappedPreviewFields"
                        :key="field.key"
                        class="px-3 py-2 text-left"
                      >
                        {{ getMappedHeaderLabel(field) }}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="(row, idx) in mappedPreviewRows"
                      :key="idx"
                      class="border-t border-gray-700/60"
                    >
                      <td v-for="(cell, col) in row" :key="col" class="px-3 py-2">
                        {{ cell }}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>

          <UAlert
            v-if="importResult"
            color="success"
            variant="soft"
            icon="i-heroicons-check-circle"
          >
            <template #title>导入完成</template>
            <template #description>
              共 {{ importResult.total }} 条，成功 {{ importResult.success }} 条，失败 {{ importResult.failed }} 条。
            </template>
          </UAlert>
          <div v-if="importResult?.errors?.length" class="text-xs text-gray-500">
            <div v-for="item in importResult.errors" :key="item.row">
              第 {{ item.row }} 行：{{ item.reason }}
            </div>
          </div>
        </UForm>
      </template>
      <template #footer>
        <div class="flex w-full flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-between sm:gap-3">
          <div class="text-xs text-gray-500">步骤 {{ importStep }} / 3</div>
          <div class="flex flex-col-reverse gap-2 sm:flex-row sm:items-center sm:gap-3">
            <UButton color="neutral" variant="subtle" :disabled="importing" @click="closeImportModal">
              取消
            </UButton>
            <UButton
              v-if="importStep === 1"
              color="primary"
              :loading="importing"
              @click="submitPreview"
            >
              下一步
            </UButton>
            <UButton
              v-else-if="importStep === 2"
              color="primary"
              :loading="importing"
              @click="submitConfirm"
            >
              确认导入
            </UButton>
            <UButton
              v-else
              color="primary"
              @click="closeImportModal"
            >
              完成
            </UButton>
          </div>
        </div>
      </template>
    </UModal>

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
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { useRouter } from "#imports";
import type { LeadCreatePayload } from "~/types/lead_capture/lead";
import { useLeadCaptureStore } from "~/stores/scrm/lead_capture/lead_store";
import { useUserStore } from "~/stores/user";
import ToastAlert from "~/components/ToastAlert.vue";
import {
  useLeadCaptureService,
  type LeadActivityRecord,
  type WeComSyncTaskRecord,
  type WeComCustomerDMRule,
} from "~/composables/api/services/leadCapture";
import {
  RuntimeDictionaryNamespaces,
  useRuntimeDictionaryService,
  type RuntimeDictionaryItem,
} from "~/composables/api/services/runtimeDictionary";
import {
  useSocialChannelGovernanceService,
  type ChannelAccount,
} from "~/composables/api/services/socialChannelGovernance";
import {
  useIAMService,
  type MemberRecord,
} from "~/composables/api/services/iamService";

definePageMeta({
  layout: "default",
});

const store = useLeadCaptureStore();
const userStore = useUserStore();
const router = useRouter();
const leadCaptureService = useLeadCaptureService();
const runtimeDictionaryService = useRuntimeDictionaryService();
const socialChannelService = useSocialChannelGovernanceService();
const iamService = useIAMService();

const ALL_OPTION_VALUE = "__all__";

const searchText = ref("");
const statusFilter = ref<string>(ALL_OPTION_VALUE);
const channelFilter = ref<string>(ALL_OPTION_VALUE);
const appTypeFilter = ref<string>(ALL_OPTION_VALUE);
const createModalOpen = ref(false);
const creating = ref(false);
const createFormError = ref("");
const currentPage = ref(1);
const pageSize = ref(10);
const importModalOpen = ref(false);
const importing = ref(false);
const importFile = ref<File | null>(null);
const importFileName = ref("");
const importResult = ref<any | null>(null);
const importStep = ref(1);
const syncLoading = ref(false);
const syncSubmitting = ref(false);
const syncAccountUUID = ref("");
const syncStatusFilter = ref<string>(ALL_OPTION_VALUE);
const syncTasks = ref<WeComSyncTaskRecord[]>([]);
const syncTaskPage = ref(1);
const syncTaskPageSize = ref(5);
let syncPollTimer: ReturnType<typeof setTimeout> | null = null;
const lastLeadAutoRefreshTaskSignature = ref("");
const syncLastResolveSource = ref("");
const wecomCustomerDMRuleLoading = ref(false);
const wecomCustomerDMRuleSaving = ref(false);
const wecomCustomerDMAutoCreate = ref(false);
const channelAccounts = ref<ChannelAccount[]>([]);
const iamMembers = ref<MemberRecord[]>([]);
const sourceCatalogs = ref<RuntimeDictionaryItem[]>([]);
const leadSyncTraceMap = ref<Record<string, Record<string, any>>>({});
const leadSyncTraceLoadingSet = ref<Set<string>>(new Set());
const leadSyncTraceWarmRunning = ref(false);
const leadSyncTraceWarmQueued = ref(false);
const previewHeaders = ref<string[]>([]);
const previewRows = ref<string[][]>([]);
const mappingForm = reactive<Record<string, number>>({});
const importFields = [
  { key: "display_name", label: "姓名", required: true },
  { key: "phone", label: "手机号", required: false },
  { key: "email", label: "邮箱", required: false },
  { key: "source_channel", label: "渠道", required: false },
  { key: "source_app_type", label: "应用类型", required: false },
  { key: "source_account_uuid", label: "渠道账号 UUID", required: false },
  { key: "owner_user_uuid", label: "负责人", required: false },
];

const fieldSynonyms: Record<string, string[]> = {
  display_name: ["display_name", "name", "full_name", "姓名", "名字"],
  phone: ["phone", "mobile", "tel", "手机号", "电话"],
  email: ["email", "mail", "邮箱"],
  source_channel: ["source_channel", "channel", "来源渠道", "渠道"],
  source_app_type: ["source_app_type", "app_type", "apptype", "应用类型", "应用"],
  source_account_uuid: [
    "source_account_uuid",
    "account_uuid",
    "account_id",
    "accountid",
    "渠道账号",
    "账号",
  ],
  owner_user_uuid: ["owner_user_uuid", "owner", "assignee", "负责人", "负责人id", "用户"],
};

const importForm = reactive({
  file: null as File | null,
});

const createForm = reactive<LeadCreatePayload>({
  display_name: "",
  phone: "",
  email: "",
  source_channel: "",
  source_app_type: "",
  source_account_uuid: "",
  owner_user_uuid: "",
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
const lastToast = ref<{ key: string; at: number } | null>(null);
const TOAST_DEDUP_WINDOW_MS = 3000;

const leadColumns = [
  { accessorKey: "display_name", header: "线索" },
  { accessorKey: "contact", header: "联系方式" },
  { accessorKey: "status", header: "状态" },
  { accessorKey: "source", header: "来源" },
  { accessorKey: "actions", header: "操作" },
] satisfies any;

const statusFilterOptions = [
  { label: "全部", value: ALL_OPTION_VALUE },
  { label: "新线索", value: "new" },
  { label: "已分配", value: "assigned" },
  { label: "跟进中", value: "in_progress" },
  { label: "已转化", value: "converted" },
  { label: "已关闭", value: "closed" },
];

const pageSizeOptions = [
  { label: "10/页", value: 10 },
  { label: "20/页", value: 20 },
  { label: "50/页", value: 50 },
];

const syncStatusOptions = [
  { label: "全部状态", value: ALL_OPTION_VALUE },
  { label: "排队中", value: "queued" },
  { label: "执行中", value: "running" },
  { label: "成功", value: "success" },
  { label: "失败", value: "failed" },
];

const syncTaskPageSizeOptions = [
  { label: "5/页", value: 5 },
  { label: "10/页", value: 10 },
  { label: "20/页", value: 20 },
];

const syncTaskColumns = [
  { accessorKey: "task_uuid", header: "任务 UUID" },
  { accessorKey: "channel_account_uuid", header: "同步账号" },
  { accessorKey: "task_provider", header: "Provider" },
  { accessorKey: "status", header: "状态" },
  { accessorKey: "progress", header: "进度" },
  { accessorKey: "stats", header: "统计" },
  { accessorKey: "error", header: "错误" },
] satisfies any;

const syncTaskTotalPages = computed(() => {
  const total = Math.ceil(syncTasks.value.length / syncTaskPageSize.value);
  return total > 0 ? total : 1;
});

const pagedSyncTasks = computed(() => {
  const start = (syncTaskPage.value - 1) * syncTaskPageSize.value;
  return syncTasks.value.slice(start, start + syncTaskPageSize.value);
});

const syncTaskPrevPage = () => {
  syncTaskPage.value = Math.max(1, syncTaskPage.value - 1);
};

const syncTaskNextPage = () => {
  syncTaskPage.value = Math.min(syncTaskTotalPages.value, syncTaskPage.value + 1);
};

const channelOptions = computed(() => {
  const entries = new Set<string>();
  sourceCatalogs.value
    .filter((item) => item.namespace === RuntimeDictionaryNamespaces.leadTrafficPlatform && item.enabled)
    .forEach((item) => entries.add(item.code));
  store.leads.forEach((lead) => {
    if (lead.source_channel) entries.add(lead.source_channel);
  });
  const labelMap = new Map<string, string>();
  sourceCatalogs.value
    .filter((item) => item.namespace === RuntimeDictionaryNamespaces.leadTrafficPlatform)
    .forEach((item) => labelMap.set(item.code, item.label));
  return [
    { label: "全部", value: ALL_OPTION_VALUE },
    ...Array.from(entries).map((value) => ({ label: labelMap.get(value) || value, value })),
  ];
});

const appTypeOptions = computed(() => {
  const entries = new Set<string>();
  sourceCatalogs.value
    .filter((item) => item.namespace === RuntimeDictionaryNamespaces.leadTrafficSource && item.enabled)
    .forEach((item) => entries.add(item.code));
  store.leads.forEach((lead) => {
    if (lead.source_app_type) entries.add(lead.source_app_type);
  });
  const labelMap = new Map<string, string>();
  sourceCatalogs.value
    .filter((item) => item.namespace === RuntimeDictionaryNamespaces.leadTrafficSource)
    .forEach((item) => labelMap.set(item.code, item.label));
  return [
    { label: "全部", value: ALL_OPTION_VALUE },
    ...Array.from(entries).map((value) => ({ label: labelMap.get(value) || value, value })),
  ];
});

const createSourceChannelOptions = computed(() => {
  const items = sourceCatalogs.value
    .filter((item) => item.namespace === RuntimeDictionaryNamespaces.leadTrafficPlatform && item.enabled)
    .sort((a, b) => (a.sort || 100) - (b.sort || 100));
  return items.map((item) => ({ value: item.code, label: item.label }));
});

const createSourceAppTypeOptions = computed(() => {
  const items = sourceCatalogs.value
    .filter((item) => item.namespace === RuntimeDictionaryNamespaces.leadTrafficSource && item.enabled)
    .sort((a, b) => (a.sort || 100) - (b.sort || 100));
  return items.map((item) => ({ value: item.code, label: item.label }));
});

const createAccountOptions = computed(() => {
  const selectedChannel = createForm.source_channel?.trim().toLowerCase() || "";
  const selectedAppType = createForm.source_app_type?.trim().toLowerCase() || "";
  const filtered = channelAccounts.value.filter((account) => {
    const channel = (account.channel_code || "").trim().toLowerCase();
    const appType = (account.app_type || "").trim().toLowerCase();
    if (selectedChannel && channel !== selectedChannel) return false;
    if (selectedAppType && appType !== selectedAppType) return false;
    return true;
  });
  return filtered.map((account) => ({
    label: `${account.display_name || account.account_id} (${account.channel_code}/${account.app_type})`,
    value: account.account_uuid,
  }));
});

const syncAccountOptions = computed(() =>
  channelAccounts.value.map((account) => ({
    label: `${account.display_name || account.account_id} (${account.channel_code}/${account.app_type})`,
    value: account.account_uuid,
  }))
);

const syncAccountLabelMap = computed(() => {
  const map = new Map<string, string>();
  channelAccounts.value.forEach((account) => {
    const key = (account.account_uuid || "").trim().toLowerCase();
    if (!key) return;
    map.set(key, `${account.display_name || account.account_id} (${account.channel_code}/${account.app_type})`);
  });
  return map;
});

const createOwnerOptions = computed(() =>
  iamMembers.value.map((member) => ({
    label: `${member.display_name} (${member.email || member.username || member.member_id})`,
    value: String(member.member_id),
  }))
);

const filteredLeads = computed(() => {
  const keyword = searchText.value.trim().toLowerCase();
  return store.leads.filter((lead) => {
    if (statusFilter.value !== ALL_OPTION_VALUE && lead.status !== statusFilter.value) {
      return false;
    }
    if (channelFilter.value !== ALL_OPTION_VALUE && lead.source_channel !== channelFilter.value) {
      return false;
    }
    if (appTypeFilter.value !== ALL_OPTION_VALUE && lead.source_app_type !== appTypeFilter.value) {
      return false;
    }
    if (!keyword) {
      return true;
    }
    return [
      lead.display_name,
      lead.phone,
      lead.email,
      lead.source_channel,
      lead.source_app_type,
    ]
      .filter(Boolean)
      .some((value) => value!.toLowerCase().includes(keyword));
  });
});

const totalPages = computed(() => {
  const total = Math.ceil(filteredLeads.value.length / pageSize.value);
  return total > 0 ? total : 1;
});

const pagedLeads = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value;
  return filteredLeads.value.slice(start, start + pageSize.value);
});

const prevPage = () => {
  currentPage.value = Math.max(1, currentPage.value - 1);
};

const nextPage = () => {
  currentPage.value = Math.min(totalPages.value, currentPage.value + 1);
};

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

const parseISOTime = (value?: string): number => {
  if (!value) return 0;
  const ts = Date.parse(value);
  return Number.isNaN(ts) ? 0 : ts;
};

const pickLatestSyncTracePayload = (activities: LeadActivityRecord[]): Record<string, any> => {
  const traces = activities.filter((item) => item.activity_type === "sync_trace");
  if (!traces.length) return {};
  const sorted = traces.slice().sort((a, b) => parseISOTime(b.created_at) - parseISOTime(a.created_at));
  return (sorted[0]?.payload || {}) as Record<string, any>;
};

const ensureLeadSyncTrace = async (leadId?: string) => {
  const key = (leadId || "").trim();
  if (!key) return;
  if (leadSyncTraceMap.value[key]) return;
  if (leadSyncTraceLoadingSet.value.has(key)) return;
  leadSyncTraceLoadingSet.value.add(key);
  try {
    const resp = await leadCaptureService.listActivities(key);
    const items = (((resp as any)?.data?.items || []) as LeadActivityRecord[]);
    leadSyncTraceMap.value[key] = pickLatestSyncTracePayload(items);
  } catch {
    leadSyncTraceMap.value[key] = {};
  } finally {
    leadSyncTraceLoadingSet.value.delete(key);
  }
};

const warmPagedLeadSyncTrace = async () => {
  if (leadSyncTraceWarmRunning.value) {
    leadSyncTraceWarmQueued.value = true;
    return;
  }
  leadSyncTraceWarmRunning.value = true;
  try {
    const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));
    let guard = 0;
    while (guard < 5) {
      guard += 1;
      leadSyncTraceWarmQueued.value = false;
      const ids = pagedLeads.value
        .map((item) => item?.lead_uuid || "")
        .filter((id) => id && !leadSyncTraceMap.value[id] && !leadSyncTraceLoadingSet.value.has(id));
      if (!ids.length) {
        break;
      }
      for (const id of ids) {
        await ensureLeadSyncTrace(id);
        await sleep(120);
      }
      if (!leadSyncTraceWarmQueued.value) {
        break;
      }
    }
  } finally {
    leadSyncTraceWarmRunning.value = false;
    leadSyncTraceWarmQueued.value = false;
  }
};

const leadSyncExternalInfo = (leadId?: string) => {
  const payload = leadId ? (leadSyncTraceMap.value[leadId] || {}) : {};
  return {
    externalLeadId: String(payload?.external_lead_id || "").trim(),
    externalWechatId: String(payload?.external_wechat_id || "").trim(),
  };
};

const refreshLeads = async () => {
  await store.fetchLeads();
  leadSyncTraceMap.value = {};
  void warmPagedLeadSyncTrace();
  if (store.error) {
    showToast(store.error, "error", "线索列表加载失败");
  }
};

const syncStatusMeta = (status?: string) => {
  switch (status) {
    case "running":
      return { label: "执行中", color: "warning" };
    case "success":
      return { label: "成功", color: "success" };
    case "failed":
      return { label: "失败", color: "error" };
    case "queued":
    default:
      return { label: "排队中", color: "info" };
  }
};

const syncProgressPercent = (task?: WeComSyncTaskRecord) => {
  if (!task) return 0;
  const raw = Number(task.progress_percent || 0);
  if (Number.isFinite(raw)) {
    if (raw < 0) return 0;
    if (raw > 100) return 100;
    return raw;
  }
  if (task.status === "success") return 100;
  if (task.status === "running") return 5;
  return 0;
};

const shortUUID = (value?: string) => {
  const s = (value || "").trim();
  if (!s) return "-";
  if (s.length <= 12) return s;
  return `${s.slice(0, 8)}...${s.slice(-4)}`;
};

const copyTaskUUID = async (value?: string) => {
  const text = (value || "").trim();
  if (!text) return;
  try {
    if (!process.client || !navigator?.clipboard?.writeText) {
      throw new Error("clipboard unavailable");
    }
    await navigator.clipboard.writeText(text);
    showToast(`任务 UUID 已复制：${text}`, "success");
  } catch {
    showToast("复制失败，请手动复制", "warning");
  }
};

const resolveSyncAccountLabel = (task?: WeComSyncTaskRecord) => {
  const uuid = (task?.channel_account_uuid || "").trim();
  if (!uuid) return "-";
  return syncAccountLabelMap.value.get(uuid.toLowerCase()) || uuid;
};

const refreshSyncTasks = async () => {
  syncLoading.value = true;
  try {
    const resp = await leadCaptureService.listWeComSyncTasks({
      channel_account_uuid: syncAccountUUID.value.trim() || undefined,
      status: (syncStatusFilter.value === ALL_OPTION_VALUE ? undefined : syncStatusFilter.value) as any,
      limit: 100,
    });
    syncTasks.value = ((resp as any)?.data?.items || []) as WeComSyncTaskRecord[];
    if (syncTaskPage.value > syncTaskTotalPages.value) {
      syncTaskPage.value = syncTaskTotalPages.value;
    }
    const finishedSuccessTask = syncTasks.value.find((item) => item?.status === "success" && !!item?.finished_at);
    if (finishedSuccessTask) {
      const signature = `${finishedSuccessTask.task_uuid}:${finishedSuccessTask.finished_at}`;
      if (signature !== lastLeadAutoRefreshTaskSignature.value) {
        lastLeadAutoRefreshTaskSignature.value = signature;
        await refreshLeads();
      }
    }
    if (syncPollTimer) {
      clearTimeout(syncPollTimer);
      syncPollTimer = null;
    }
    const hasPendingTasks = syncTasks.value.some(
      (item) => item?.status === "queued" || item?.status === "running"
    );
    if (hasPendingTasks && process.client) {
      syncPollTimer = setTimeout(() => {
        void refreshSyncTasks();
      }, 2000);
    }
  } catch (err: any) {
    showToast(err?.message || "同步任务加载失败", "error");
  } finally {
    syncLoading.value = false;
  }
};

const triggerWeComSync = async () => {
  syncSubmitting.value = true;
  try {
    const payload: Record<string, string> = {};
    if (syncAccountUUID.value.trim()) {
      payload.channel_account_uuid = syncAccountUUID.value.trim();
    }
    payload.trace_id = `lead-sync-${Date.now()}`;
    const resp = await leadCaptureService.triggerWeComSync(payload);
    const task = (resp as any)?.data as WeComSyncTaskRecord | undefined;
    if (task?.account_resolve_source) {
      syncLastResolveSource.value = task.account_resolve_source;
    }
    showToast("已触发渠道同步任务", "success");
    await refreshSyncTasks();
  } catch (err: any) {
    showToast(err?.message || "触发同步失败", "error");
  } finally {
    syncSubmitting.value = false;
  }
};

const loadWeComCustomerDMRule = async () => {
  wecomCustomerDMRuleLoading.value = true;
  try {
    const resp = await leadCaptureService.getWeComCustomerDMRule();
    const rule = ((resp as any)?.data || null) as WeComCustomerDMRule | null;
    wecomCustomerDMAutoCreate.value = !!rule?.auto_create_lead_from_customer_dm;
  } catch (err: any) {
    showToast(err?.message || "加载渠道规则失败", "error");
  } finally {
    wecomCustomerDMRuleLoading.value = false;
  }
};

const saveWeComCustomerDMRule = async () => {
  wecomCustomerDMRuleSaving.value = true;
  try {
    const resp = await leadCaptureService.updateWeComCustomerDMRule({
      enabled: !!wecomCustomerDMAutoCreate.value,
    });
    const rule = ((resp as any)?.data || null) as WeComCustomerDMRule | null;
    wecomCustomerDMAutoCreate.value = !!rule?.auto_create_lead_from_customer_dm;
    showToast("渠道规则已保存", "success");
  } catch (err: any) {
    showToast(err?.message || "保存渠道规则失败", "error");
  } finally {
    wecomCustomerDMRuleSaving.value = false;
  }
};

const openDetail = (leadId: string) => {
  router.push(`/scrm/lead_capture/${leadId}`);
};

const loadCreateLookupOptions = async () => {
  try {
    const [accountResp, memberResp, catalogResp] = await Promise.all([
      socialChannelService.listChannelAccounts(),
      userStore.currentTenantUuid
        ? iamService.listMembers({
            tenantUuid: userStore.currentTenantUuid,
            page: 1,
            pageSize: 200,
          })
        : Promise.resolve({ data: { items: [] } } as any),
      runtimeDictionaryService.listDictionaries(),
    ]);
    channelAccounts.value = ((accountResp as any)?.data?.items || []) as ChannelAccount[];
    iamMembers.value = ((memberResp as any)?.data?.items || []) as MemberRecord[];
    sourceCatalogs.value = ((catalogResp as any)?.data?.items || []) as RuntimeDictionaryItem[];
  } catch {
    channelAccounts.value = [];
    iamMembers.value = [];
    sourceCatalogs.value = [];
  }
};

const openCreateModal = () => {
  createFormError.value = "";
  createModalOpen.value = true;
  void loadCreateLookupOptions();
};

const openImportModal = () => {
  importFile.value = null;
  importFileName.value = "";
  importResult.value = null;
  importStep.value = 1;
  previewHeaders.value = [];
  previewRows.value = [];
  Object.keys(mappingForm).forEach((key) => delete mappingForm[key]);
  importModalOpen.value = true;
};

const closeCreateModal = () => {
  if (creating.value) {
    return;
  }
  blurActiveElement();
  createModalOpen.value = false;
};

const closeImportModal = () => {
  if (importing.value) {
    return;
  }
  blurActiveElement();
  importModalOpen.value = false;
};

const resetCreateForm = () => {
  createForm.display_name = "";
  createForm.phone = "";
  createForm.email = "";
  createForm.source_channel = "";
  createForm.source_app_type = "";
  createForm.source_account_uuid = "";
  createForm.owner_user_uuid = "";
  createFormError.value = "";
};

const sanitizeCreatePayload = (): LeadCreatePayload => {
  const payload: LeadCreatePayload = {};
  const displayName = createForm.display_name?.trim();
  const phone = createForm.phone?.trim();
  const email = createForm.email?.trim();
  const sourceChannel = createForm.source_channel?.trim();
  const sourceAppType = createForm.source_app_type?.trim();
  const sourceAccountUUID = createForm.source_account_uuid?.trim();
  const ownerUserUUID = createForm.owner_user_uuid?.trim();

  if (displayName) payload.display_name = displayName;
  if (phone) payload.phone = phone;
  if (email) payload.email = email;
  if (sourceChannel) payload.source_channel = sourceChannel;
  if (sourceAppType) payload.source_app_type = sourceAppType;
  if (sourceAccountUUID) payload.source_account_uuid = sourceAccountUUID;
  if (ownerUserUUID) payload.owner_user_uuid = ownerUserUUID;
  return payload;
};

const validateCreateForm = (): boolean => {
  const hasContact =
    !!createForm.display_name?.trim() ||
    !!createForm.phone?.trim() ||
    !!createForm.email?.trim();
  if (!hasContact) {
    createFormError.value = "姓名 / 手机号 / 邮箱至少填写一项。";
    return false;
  }
  createFormError.value = "";
  return true;
};

const submitCreate = async () => {
  if (!validateCreateForm()) {
    showToast("请至少填写姓名、手机号或邮箱之一", "warning");
    return;
  }
  creating.value = true;
  try {
    const created = await store.createLead(sanitizeCreatePayload());
    if (created?.lead_uuid) {
      showToast("线索已创建", "success");
      blurActiveElement();
      createModalOpen.value = false;
      resetCreateForm();
      openDetail(created.lead_uuid);
    }
  } catch (err: any) {
    showToast(err?.message ?? "创建失败", "error");
  } finally {
    creating.value = false;
  }
};

const onFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0] ?? null;
  importFile.value = file;
  importFileName.value = file?.name ?? "";
  importForm.file = file;
  importStep.value = 1;
  previewHeaders.value = [];
  previewRows.value = [];
  Object.keys(mappingForm).forEach((key) => delete mappingForm[key]);
};

const downloadTemplate = () => {
  if (!process.client) return;
  const header = [
    "display_name",
    "phone",
    "email",
    "source_channel",
    "source_app_type",
    "source_account_uuid",
    "owner_user_uuid",
  ];
  const sample = [
    "示例线索",
    "13800000000",
    "demo@example.com",
    "wechat",
    "wecom",
    "",
    "",
  ];
  const content = `${header.join(",")}\n${sample.join(",")}\n`;
  const blob = new Blob([content], { type: "text/csv;charset=utf-8;" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = "lead_import_template.csv";
  link.click();
  URL.revokeObjectURL(url);
};


const headerOptions = computed(() =>
  previewHeaders.value.map((header, idx) => ({
    label: header,
    value: idx,
  }))
);

const importStepDescription = computed(() => {
  if (importStep.value === 1) return "上传文件后解析字段。";
  if (importStep.value === 2) return "确认字段映射后导入。";
  return "导入已完成。";
});

const normalizeHeader = (value: string) =>
  value
    .toLowerCase()
    .replace(/[^a-z0-9\u4e00-\u9fa5]+/g, "")
    .trim();

const mappingIssues = computed(() => {
  const usage = new Map<number, string[]>();
  const issues: Record<string, string> = {};
  importFields.forEach((field) => {
    const idx = mappingForm[field.key];
    if (idx === undefined || idx === null || idx < 0) return;
    if (!usage.has(idx)) usage.set(idx, []);
    usage.get(idx)?.push(field.label);
  });
  importFields.forEach((field) => {
    const idx = mappingForm[field.key];
    if (idx === undefined || idx === null || idx < 0) return;
    const header = previewHeaders.value[idx] ?? "";
    const normalized = normalizeHeader(header);
    const synonyms = fieldSynonyms[field.key] || [];
    const matched = synonyms.some((item) => normalized.includes(normalizeHeader(item)));
    const duplicated = (usage.get(idx)?.length ?? 0) > 1;
    const hints: string[] = [];
    if (!matched) hints.push("列名可能不匹配");
    if (duplicated) hints.push("已重复映射");
    if (hints.length) issues[field.key] = hints.join("；");
  });
  return issues;
});

const mappedPreviewFields = computed(() =>
  importFields.filter((field) => {
    const idx = mappingForm[field.key];
    return idx !== undefined && idx !== null && idx >= 0;
  })
);

const mappedPreviewRows = computed(() =>
  previewRows.value.map((row) =>
    mappedPreviewFields.value.map((field) => {
      const idx = mappingForm[field.key];
      if (idx === undefined || idx === null || idx < 0) return "";
      return row[idx] ?? "";
    })
  )
);

const getMappedHeaderLabel = (field: { key: string; label: string }) => {
  const idx = mappingForm[field.key];
  if (idx === undefined || idx === null || idx < 0) return field.label;
  const header = previewHeaders.value[idx];
  return header ? `${field.label}（${header}）` : field.label;
};

const submitPreview = async () => {
  if (!importFile.value) {
    showToast("请先选择 CSV 文件", "warning");
    return;
  }
  importing.value = true;
  try {
    const service = useLeadCaptureService();
    const resp = await service.previewImport(importFile.value);
    const payload = (resp as any)?.data ?? {};
    previewHeaders.value = payload.headers || [];
    previewRows.value = payload.sample_rows || [];
    const suggested = payload.suggested_mappings || {};
    importFields.forEach((field) => {
      if (suggested[field.key] !== undefined && suggested[field.key] >= 0) {
        mappingForm[field.key] = suggested[field.key];
      }
    });
    importStep.value = 2;
  } catch (err: any) {
    showToast(err?.message ?? "解析失败", "error");
  } finally {
    importing.value = false;
  }
};

const buildMappingPayload = () => {
  const mapping: Record<string, number> = {};
  importFields.forEach((field) => {
    const idx = mappingForm[field.key];
    if (idx !== undefined && idx !== null && idx >= 0) {
      mapping[field.key] = idx;
    } else {
      mapping[field.key] = -1;
    }
  });
  return mapping;
};

const submitConfirm = async () => {
  if (!importFile.value) {
    showToast("请先选择 CSV 文件", "warning");
    return;
  }
  const mapping = buildMappingPayload();
  if (mapping.display_name < 0 && mapping.phone < 0 && mapping.email < 0) {
    showToast("姓名/手机号/邮箱至少映射一项", "warning");
    return;
  }
  importing.value = true;
  try {
    const service = useLeadCaptureService();
    const resp = await service.confirmImport(importFile.value, mapping);
    importResult.value = (resp as any)?.data ?? null;
    importStep.value = 3;
    showToast("导入完成", "success");
    await refreshLeads();
  } catch (err: any) {
    showToast(err?.message ?? "导入失败", "error");
  } finally {
    importing.value = false;
  }
};

const showToast = (message: string, color: ToastColor = "primary", title = "") => {
  const resolvedTitle = title || message;
  const resolvedMessage = title ? message : "";
  const key = `${color}|${resolvedTitle}|${resolvedMessage}`;
  const now = Date.now();
  if (lastToast.value && lastToast.value.key === key && now - lastToast.value.at < TOAST_DEDUP_WINDOW_MS) {
    return;
  }
  lastToast.value = { key, at: now };

  toast.title = resolvedTitle;
  toast.message = resolvedMessage;
  toast.color = color;
  toast.visible = true;
};

const blurActiveElement = () => {
  if (!process.client) return;
  const el = document.activeElement as HTMLElement | null;
  if (el && typeof el.blur === "function") {
    el.blur();
  }
};

watch([statusFilter, channelFilter, appTypeFilter, searchText], () => {
  currentPage.value = 1;
});

watch([syncAccountUUID, syncStatusFilter, syncTaskPageSize], () => {
  syncTaskPage.value = 1;
});

watch([currentPage, pageSize], () => {
  if (currentPage.value > totalPages.value) {
    currentPage.value = totalPages.value;
  }
});

watch([syncTaskPage, syncTaskPageSize], () => {
  if (syncTaskPage.value > syncTaskTotalPages.value) {
    syncTaskPage.value = syncTaskTotalPages.value;
  }
});

watch(
  () => pagedLeads.value.map((item) => item.lead_uuid).join(","),
  () => {
    void warmPagedLeadSyncTrace();
  },
  { immediate: true }
);

onMounted(async () => {
  await loadCreateLookupOptions();
  await refreshLeads();
  await refreshSyncTasks();
  await loadWeComCustomerDMRule();
});

onBeforeUnmount(() => {
  if (syncPollTimer) {
    clearTimeout(syncPollTimer);
    syncPollTimer = null;
  }
});
</script>
