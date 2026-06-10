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
        <UButton
          icon="i-heroicons-adjustments-horizontal"
          variant="soft"
          @click="syncCenterOpen = true"
        >
          同步中心
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
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="text-sm text-gray-600 dark:text-gray-300">
          同步中心已收纳为弹层，当前仅展示线索列表主视图。
        </div>
        <div class="flex items-center gap-2">
          <UBadge variant="soft" :color="syncTasks.some((item) => item.status === 'failed') ? 'error' : 'success'">
            同步任务 {{ syncTasks.length }} 条
          </UBadge>
          <UButton size="xs" variant="soft" :loading="syncLoading" @click="refreshSyncTasks">刷新</UButton>
          <UButton size="xs" color="primary" @click="syncCenterOpen = true">打开同步中心</UButton>
        </div>
      </div>
    </UCard>

    <UModal
      v-model:open="syncCenterOpen"
      :prevent-close="false"
      :dismissible="true"
      :modal="true"
      :ui="{ content: 'max-w-7xl w-full' }"
    >
      <template #title>同步中心</template>
      <template #description>统一管理渠道同步任务、线索回写策略与死信重放。</template>
      <template #body>
        <div class="p-1">
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
            <UButton
              size="xs"
              color="primary"
              :loading="syncSubmitting"
              :disabled="syncSubmitting"
              @click="triggerWeComSync"
            >
              触发同步
            </UButton>
          </div>
        </div>
      </template>
      <div class="space-y-3">
        <div class="grid grid-cols-1 gap-4 xl:grid-cols-12">
          <div class="xl:col-span-12">
            <div class="rounded-xl border border-gray-200/80 bg-gray-50/70 p-4 dark:border-gray-700/80 dark:bg-gray-900/40">
              <div class="mb-3 flex items-center justify-between">
                <div class="text-sm font-medium text-gray-800 dark:text-gray-100">
                  同步筛选
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  触发前选择同步动作，系统自动使用默认渠道账号
                </div>
              </div>
              <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
                <UFormField label="同步账号（系统默认）">
                  <UInput
                    :model-value="defaultSyncAccountLabel"
                    readonly
                    class="w-full"
                  />
                </UFormField>
                <UFormField label="同步动作">
                  <USelectMenu
                    v-model="syncTriggerAction"
                    :items="syncTriggerActionOptions"
                    value-key="value"
                    label-key="label"
                    class="w-full"
                    :portal="false"
                    :ui="{ content: 'z-[200]' }"
                  />
                </UFormField>
              </div>
              <div v-if="syncTriggerAction === 'push_leads'" class="mt-3 rounded-xl border border-gray-200/80 bg-white/80 p-3 dark:border-gray-700/80 dark:bg-gray-950/30">
                <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
                  <div class="text-sm font-medium text-gray-800 dark:text-gray-100">
                    回写线索选择（多选）
                  </div>
                  <div class="flex items-center gap-2">
                    <UButton size="xs" variant="soft" @click="selectAllFilteredLeadsForPush">
                      选择当前筛选全部
                    </UButton>
                    <UButton size="xs" variant="ghost" @click="clearSelectedLeadsForPush">
                      清空
                    </UButton>
                  </div>
                </div>
                <UFormField label="待回写线索">
                  <UInput
                    model-value="请在下方清单勾选要回写的线索（可多选）"
                    readonly
                    class="w-full"
                  />
                </UFormField>
                <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                  已选 {{ selectedLeadUUIDsForPush.length }} 条；只会推送你选中的线索，不会全量推送。
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ pushPreviewScopeHint }}
                </div>
                <div class="mt-3 grid grid-cols-1 gap-3 lg:grid-cols-2">
                  <div class="rounded-lg border border-emerald-300/50 bg-emerald-500/5 p-3">
                    <div class="mb-2 flex items-center justify-between">
                      <span class="text-xs font-semibold text-emerald-600 dark:text-emerald-300">可回写清单</span>
                      <UBadge size="xs" color="success" variant="soft">{{ pushLeadPassedRows.length }}</UBadge>
                    </div>
                    <UTable
                      :columns="pushLeadValidationColumns"
                      :data="pagedPushLeadPassedRows"
                      :ui="{ td: 'py-1.5 text-xs', th: 'py-1.5 text-xs' }"
                    >
                      <template #select-cell="{ row }">
                        <UCheckbox
                          :model-value="isLeadSelectedForPush(row.original.leadUUID)"
                          @update:model-value="setLeadSelectedForPush(row.original.leadUUID, $event)"
                        />
                      </template>
                      <template #result-cell>
                        <UBadge size="xs" color="success" variant="soft">通过</UBadge>
                      </template>
                      <template #name-cell="{ row }">
                        <div class="flex items-center gap-2">
                          <span>{{ row.original.name }}</span>
                          <UBadge
                            v-if="row.original.syncStatus === 'pending_push'"
                            size="xs"
                            color="warning"
                            variant="soft"
                          >
                            待回写
                          </UBadge>
                          <UBadge
                            v-else
                            size="xs"
                            color="neutral"
                            variant="soft"
                          >
                            未变更
                          </UBadge>
                        </div>
                      </template>
                    </UTable>
                    <div v-if="pushLeadPassedRows.length > 0" class="mt-2 flex items-center justify-end gap-2 text-xs text-gray-500 dark:text-gray-400">
                      <UButton size="2xs" variant="ghost" :disabled="pushLeadPassedPage <= 1" @click="pushLeadPassedPage = Math.max(1, pushLeadPassedPage - 1)">上一页</UButton>
                      <span>{{ pushLeadPassedPage }} / {{ pushLeadPassedTotalPages }}</span>
                      <UButton size="2xs" variant="ghost" :disabled="pushLeadPassedPage >= pushLeadPassedTotalPages" @click="pushLeadPassedPage = Math.min(pushLeadPassedTotalPages, pushLeadPassedPage + 1)">下一页</UButton>
                    </div>
                    <div v-if="pushLeadPassedRows.length === 0" class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                      暂无可回写线索。
                    </div>
                  </div>
                  <div class="rounded-lg border border-amber-300/50 bg-amber-500/5 p-3">
                    <div class="mb-2 flex items-center justify-between">
                      <span class="text-xs font-semibold text-amber-600 dark:text-amber-300">不可回写清单</span>
                      <UBadge size="xs" color="warning" variant="soft">{{ pushLeadRejectedRows.length }}</UBadge>
                    </div>
                    <UTable
                      :columns="pushLeadValidationColumns"
                      :data="pagedPushLeadRejectedRows"
                      :ui="{ td: 'py-1.5 text-xs', th: 'py-1.5 text-xs' }"
                    >
                      <template #select-cell="{ row }">
                        <div class="flex items-center">
                          <span class="text-xs text-gray-400">-</span>
                        </div>
                      </template>
                      <template #result-cell="{ row }">
                        <div
                          class="flex items-center gap-2"
                        >
                          <UBadge size="xs" color="error" variant="soft">{{ row.original.reason || "未通过" }}</UBadge>
                          <UButton
                            v-if="canJumpToLeadDetail(row.original)"
                            size="2xs"
                            color="warning"
                            variant="soft"
                            @click="tryOpenRejectedLeadDetail(row.original)"
                          >
                            {{ rejectedJumpLabel(row.original) }}
                          </UButton>
                        </div>
                      </template>
                      <template #name-cell="{ row }">
                        <div class="flex items-center gap-2">
                          <span>{{ row.original.name }}</span>
                        </div>
                      </template>
                      <template #contact-cell="{ row }">
                        <div>
                          {{ row.original.contact }}
                        </div>
                      </template>
                    </UTable>
                    <div v-if="pushLeadRejectedRows.length > 0" class="mt-2 flex items-center justify-end gap-2 text-xs text-gray-500 dark:text-gray-400">
                      <UButton size="2xs" variant="ghost" :disabled="pushLeadRejectedPage <= 1" @click="pushLeadRejectedPage = Math.max(1, pushLeadRejectedPage - 1)">上一页</UButton>
                      <span>{{ pushLeadRejectedPage }} / {{ pushLeadRejectedTotalPages }}</span>
                      <UButton size="2xs" variant="ghost" :disabled="pushLeadRejectedPage >= pushLeadRejectedTotalPages" @click="pushLeadRejectedPage = Math.min(pushLeadRejectedTotalPages, pushLeadRejectedPage + 1)">下一页</UButton>
                    </div>
                    <div v-if="pushLeadRejectedRows.length === 0" class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                      当前所选线索都可回写。
                    </div>
                  </div>
                </div>
              </div>
              <div
                class="mt-3 rounded-lg border border-dashed border-gray-300/80 bg-white/70 px-3 py-2 text-xs text-gray-600 dark:border-gray-700 dark:bg-gray-950/30 dark:text-gray-300"
              >
                最近一次账号解析来源：<span class="font-medium text-gray-800 dark:text-gray-100">{{ syncLastResolveSource || "未触发" }}</span>
                <span class="ml-2 text-[11px] text-gray-500 dark:text-gray-400">（触发同步时固定按系统默认账号解析）</span>
              </div>

              <div class="mt-3 rounded-lg border border-gray-200/80 bg-white/70 dark:border-gray-700 dark:bg-gray-950/30">
                <button
                  type="button"
                  class="flex w-full items-center justify-between px-3 py-2 text-left"
                  @click="dmRulePanelOpen = !dmRulePanelOpen"
                >
                  <div>
                    <div class="text-sm font-medium text-gray-900 dark:text-gray-100">
                      客户私信自动建线索（{{ currentSyncChannelTag }}）
                    </div>
                    <div class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                      默认收起，按需展开配置
                    </div>
                  </div>
                  <UIcon :name="dmRulePanelOpen ? 'i-heroicons-chevron-up' : 'i-heroicons-chevron-down'" class="text-gray-500" />
                </button>
                <div v-if="dmRulePanelOpen" class="border-t border-gray-200/80 px-3 py-3 dark:border-gray-700">
                  <div class="flex items-start justify-between gap-3">
                    <div class="text-xs leading-5 text-gray-600 dark:text-gray-300">
                      {{ isWeComDefaultSyncChannel
                        ? "关闭时进入待绑定池，开启后客户私信自动入池。"
                        : "当前默认渠道暂不支持该规则（仅 WeCom 已实现）。" }}
                    </div>
                    <USwitch
                      v-model="wecomCustomerDMAutoCreate"
                      :disabled="!isWeComDefaultSyncChannel"
                      :loading="wecomCustomerDMRuleLoading || wecomCustomerDMRuleSaving"
                    />
                  </div>
                  <div class="mt-3 flex justify-end">
                    <UButton
                      size="xs"
                      color="primary"
                      :disabled="!isWeComDefaultSyncChannel"
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
        </div>

        <div class="grid grid-cols-1 gap-4 xl:grid-cols-12">
          <div class="xl:col-span-12 rounded-xl border border-gray-200/80 bg-gray-50/70 p-4 dark:border-gray-700/80 dark:bg-gray-900/40">
            <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium text-gray-900 dark:text-gray-100">线索回写策略</span>
                <UBadge :color="writebackCapabilityColor(writebackPolicy?.capability_status)" variant="soft">
                  {{ writebackCapabilityLabel(writebackPolicy?.capability_status) }}
                </UBadge>
              </div>
              <div class="flex items-center gap-2">
                <UButton size="xs" variant="soft" :loading="writebackPolicyLoading" @click="loadWritebackPolicy">
                  刷新策略
                </UButton>
                <UButton size="xs" color="primary" @click="openWritebackPolicyDialog">
                  配置策略
                </UButton>
              </div>
            </div>
            <div class="space-y-2 text-xs text-gray-500 dark:text-gray-400">
              <div>覆盖模式：<span class="font-medium text-gray-800 dark:text-gray-200">{{ writebackOverwriteMode }}</span></div>
              <div>启用状态：<span class="font-medium text-gray-800 dark:text-gray-200">{{ writebackEnabled ? "已启用" : "未启用" }}</span></div>
              <div>白名单字段数：<span class="font-medium text-gray-800 dark:text-gray-200">{{ writebackWhitelistFields.length }}</span>，受保护字段数：<span class="font-medium text-gray-800 dark:text-gray-200">{{ writebackProtectedFields.length }}</span></div>
            </div>
          </div>
        </div>

        <UModal
          v-model:open="writebackPolicyDialogOpen"
          :prevent-close="false"
          :dismissible="true"
          :modal="true"
          :ui="{ content: 'max-w-4xl w-full' }"
        >
          <template #title>线索回写策略</template>
          <template #description>配置字段白名单与受保护字段，保存后生效。</template>
          <template #body>
            <div class="space-y-3 p-1">
              <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
                <UFormField label="覆盖模式">
                  <USelectMenu
                    v-model="writebackOverwriteMode"
                    :items="writebackOverwriteModeOptions"
                    value-key="value"
                    label-key="label"
                    class="w-full"
                    :portal="false"
                    :ui="{ content: 'z-[200]' }"
                  />
                </UFormField>
                <UFormField label="启用回写">
                  <div class="h-8 flex items-center">
                    <USwitch v-model="writebackEnabled" />
                  </div>
                </UFormField>
                <UFormField label="白名单字段（多选）" class="md:col-span-2">
                  <div class="grid grid-cols-2 gap-2 md:grid-cols-3">
                    <label
                      v-for="field in writebackFieldOptions"
                      :key="`whitelist-dialog-${field.value}`"
                      class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200"
                    >
                      <input
                        type="checkbox"
                        class="h-4 w-4"
                        :checked="writebackWhitelistFields.includes(field.value)"
                        @change="onWritebackCheckboxChange('whitelist', field.value, $event)"
                      />
                      <span>{{ field.label }}</span>
                    </label>
                  </div>
                </UFormField>
                <UFormField label="受保护字段（多选）" class="md:col-span-2">
                  <div class="grid grid-cols-2 gap-2 md:grid-cols-3">
                    <label
                      v-for="field in writebackFieldOptions"
                      :key="`protected-dialog-${field.value}`"
                      class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200"
                    >
                      <input
                        type="checkbox"
                        class="h-4 w-4"
                        :checked="writebackProtectedFields.includes(field.value)"
                        @change="onWritebackCheckboxChange('protected', field.value, $event)"
                      />
                      <span>{{ field.label }}</span>
                    </label>
                  </div>
                </UFormField>
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                提示：白名单外字段不会回写，受保护字段即使在白名单中也会被忽略。
              </div>
            </div>
          </template>
          <template #footer>
            <div class="flex items-center justify-end gap-2">
              <UButton variant="soft" @click="writebackPolicyDialogOpen = false">取消</UButton>
              <UButton color="primary" :loading="writebackPolicySaving" @click="saveWritebackPolicyFromDialog">
                保存策略
              </UButton>
            </div>
          </template>
        </UModal>

        <UCard>
          <template #header>
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium text-gray-900 dark:text-gray-100">同步任务队列</span>
                <UBadge variant="soft" color="info">{{ syncTasks.length }}</UBadge>
              </div>
              <div class="flex items-center gap-2">
                <UButton size="xs" variant="soft" :loading="syncLoading" @click="refreshSyncTasks">刷新任务</UButton>
                <UButton size="xs" color="warning" variant="soft" :loading="syncTaskClearing" @click="clearSyncTasks">清空任务</UButton>
                <UButton size="xs" variant="soft" @click="syncTaskPanelOpen = !syncTaskPanelOpen">
                  {{ syncTaskPanelOpen ? "收起" : "展开" }}
                </UButton>
              </div>
            </div>
          </template>
          <div v-if="syncTaskPanelOpen" class="space-y-3">
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
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium text-gray-900 dark:text-gray-100">回写死信队列</span>
                <UBadge variant="soft" color="warning">{{ writebackDeadLetters.length }}</UBadge>
              </div>
              <div class="flex items-center gap-2">
                <UButton size="xs" variant="soft" :loading="writebackDeadLettersLoading" @click="refreshWritebackDeadLetters">刷新死信</UButton>
                <UButton size="xs" variant="soft" @click="deadLetterPanelOpen = !deadLetterPanelOpen">
                  {{ deadLetterPanelOpen ? "收起" : "展开" }}
                </UButton>
              </div>
            </div>
          </template>
          <div v-if="deadLetterPanelOpen" class="space-y-3">
            <div v-if="writebackDeadLetters.length === 0" class="text-xs text-gray-500 dark:text-gray-400">
              暂无死信记录
            </div>
            <UTable v-else :columns="deadLetterColumns" :data="writebackDeadLetters" :loading="writebackDeadLettersLoading">
              <template #dead_letter_uuid-cell="{ row }">
                <span class="text-xs text-gray-300">{{ shortUUID(row.original.dead_letter_uuid) }}</span>
              </template>
              <template #replay_status-cell="{ row }">
                <UBadge variant="soft" :color="row.original.replay_status === 'done' ? 'success' : 'warning'">
                  {{ row.original.replay_status || "pending" }}
                </UBadge>
              </template>
              <template #error-cell="{ row }">
                <span class="text-xs text-amber-500">
                  {{ row.original.last_error_code || "-" }} / {{ row.original.last_error_message || "-" }}
                </span>
              </template>
              <template #actions-cell="{ row }">
                <UButton
                  size="xs"
                  variant="soft"
                  :loading="writebackReplayLoadingUUID === row.original.dead_letter_uuid"
                  :disabled="!row.original.dead_letter_uuid"
                  @click="replayWritebackDeadLetter(row.original.dead_letter_uuid)"
                >
                  重放
                </UButton>
              </template>
            </UTable>
          </div>
        </UCard>
      </div>
    </UCard>
        </div>
      </template>
    </UModal>

    <UCard v-if="showChannelCodeWelcomePanel">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-qr-code" class="text-primary" />
            <span class="font-medium text-gray-900 dark:text-gray-100">渠道码与欢迎语</span>
          </div>
          <UButton size="xs" variant="soft" :loading="channelCodeStore.loading" @click="refreshChannelCodePanel">
            刷新
          </UButton>
        </div>
      </template>
      <div class="space-y-4">
        <div class="grid grid-cols-1 gap-3 xl:grid-cols-12">
          <div class="xl:col-span-4 space-y-3">
            <UFormField label="选择渠道码">
              <USelectMenu
                v-model="selectedChannelCodeUUID"
                :items="channelCodeOptions"
                value-key="value"
                label-key="label"
                placeholder="请选择渠道码"
                class="w-full"
                :portal="false"
                :ui="{ content: 'z-[200]' }"
              />
            </UFormField>
            <div class="flex flex-wrap gap-2">
              <UButton
                size="xs"
                color="primary"
                :disabled="!selectedChannelCodeUUID"
                :loading="channelCodeStore.saving"
                @click="setSelectedChannelCodeStatus('active')"
              >
                启用
              </UButton>
              <UButton
                size="xs"
                variant="soft"
                :disabled="!selectedChannelCodeUUID"
                :loading="channelCodeStore.saving"
                @click="setSelectedChannelCodeStatus('disabled')"
              >
                停用
              </UButton>
              <UButton size="xs" variant="soft" @click="channelCodeCreateOpen = true">新建渠道码</UButton>
            </div>
            <div
              v-if="selectedChannelCode"
              class="rounded-lg border border-gray-200/80 bg-gray-50/70 px-3 py-2 text-xs text-gray-600 dark:border-gray-700/80 dark:bg-gray-900/40 dark:text-gray-300"
            >
              当前状态：<span class="font-medium">{{ selectedChannelCode.status }}</span>
            </div>
          </div>

          <div class="xl:col-span-8 space-y-3">
            <UFormField label="欢迎语内容（JSON）">
              <UTextarea
                v-model="channelCodeWelcomeContentText"
                :rows="5"
                placeholder='{"text":"欢迎添加企业微信"}'
              />
            </UFormField>
            <div class="flex items-center justify-between gap-3">
              <USwitch v-model="channelCodeWelcomeEnabled" label="启用欢迎语" />
              <div class="flex items-center gap-2">
                <UButton
                  size="sm"
                  color="primary"
                  :disabled="!selectedChannelCodeUUID"
                  :loading="channelCodeStore.saving"
                  @click="saveSelectedChannelCodeWelcomeConfig"
                >
                  保存欢迎语（不发布）
                </UButton>
                <UButton
                  size="sm"
                  variant="soft"
                  :disabled="!selectedChannelCodeUUID"
                  :loading="channelCodeWelcomeSyncLoading"
                  @click="triggerSelectedChannelCodeWelcomeSync"
                >
                  发布到渠道
                </UButton>
              </div>
            </div>
            <div
              class="rounded-lg border border-gray-200/80 bg-gray-50/70 px-3 py-2 text-xs text-gray-600 dark:border-gray-700/80 dark:bg-gray-900/40 dark:text-gray-300"
            >
              <div class="flex flex-wrap items-center gap-3">
                <span>同步状态：<span class="font-medium">{{ channelCodeWelcomeSyncStatus?.sync_status || "pending" }}</span></span>
                <span>最新尝试：<span class="font-medium">{{ channelCodeWelcomeSyncStatus?.latest_attempt_no ?? 0 }}</span></span>
                <span>最近同步：<span class="font-medium">{{ channelCodeWelcomeSyncStatus?.last_synced_at || "-" }}</span></span>
              </div>
              <div v-if="channelCodeWelcomeSyncStatus?.last_sync_error" class="mt-1 text-amber-600 dark:text-amber-300">
                错误信息：{{ channelCodeWelcomeSyncStatus.last_sync_error }}
              </div>
            </div>
            <div class="rounded-lg border border-gray-200/80 p-3 dark:border-gray-700/80">
              <div class="mb-2 text-xs text-gray-500 dark:text-gray-400">配置变更摘要（最近 20 条）</div>
              <div v-if="channelCodeStore.changeLogs.length === 0" class="text-xs text-gray-500 dark:text-gray-400">
                暂无变更记录
              </div>
              <div v-else class="space-y-2">
                <div
                  v-for="item in channelCodeStore.changeLogs"
                  :key="item.change_uuid"
                  class="rounded border border-gray-200/70 px-2 py-2 text-xs dark:border-gray-700/70"
                >
                  <div class="font-medium text-gray-800 dark:text-gray-100">{{ item.summary }}</div>
                  <div class="text-gray-500 dark:text-gray-400">v{{ item.version }} · {{ item.created_at || "-" }}</div>
                </div>
              </div>
            </div>

            <div class="rounded-lg border border-gray-200/80 p-3 dark:border-gray-700/80">
              <div class="mb-3 flex items-center justify-between">
                <div class="text-xs text-gray-500 dark:text-gray-400">事件入池与来源追溯</div>
                <UButton
                  size="xs"
                  variant="soft"
                  :disabled="!selectedChannelCodeUUID"
                  :loading="channelCodeEventsLoading"
                  @click="loadSelectedChannelCodeEvents"
                >
                  刷新事件
                </UButton>
              </div>
              <div class="mb-3 grid grid-cols-1 gap-2 sm:grid-cols-3">
                <div class="rounded border border-gray-200/70 px-2 py-2 text-xs dark:border-gray-700/70">
                  触达量：<span class="font-medium">{{ channelCodeEventStats.touch_total }}</span>
                </div>
                <div class="rounded border border-gray-200/70 px-2 py-2 text-xs dark:border-gray-700/70">
                  入池量：<span class="font-medium">{{ channelCodeEventStats.intake_total }}</span>
                </div>
                <div class="rounded border border-gray-200/70 px-2 py-2 text-xs dark:border-gray-700/70">
                  去重量：<span class="font-medium">{{ channelCodeEventStats.dedup_total }}</span>
                </div>
              </div>
              <div v-if="channelCodeEvents.length === 0" class="text-xs text-gray-500 dark:text-gray-400">
                暂无事件记录
              </div>
              <div v-else class="space-y-2">
                <div
                  v-for="event in channelCodeEvents"
                  :key="event.event_uuid"
                  class="rounded border border-gray-200/70 px-2 py-2 text-xs dark:border-gray-700/70"
                >
                  <div class="flex items-center justify-between gap-2">
                    <span class="font-medium text-gray-800 dark:text-gray-100">
                      {{ event.event_type }} · {{ event.external_event_id }}
                    </span>
                    <span class="text-gray-500 dark:text-gray-400">{{ event.occurred_at || "-" }}</span>
                  </div>
                  <div class="mt-1 text-gray-500 dark:text-gray-400">
                    event_uuid: {{ event.event_uuid }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-rectangle-stack" class="text-primary" />
            <span class="font-medium text-gray-900 dark:text-gray-100">线索列表</span>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <UBadge variant="soft" color="primary">{{ filteredLeads.length }}</UBadge>
            <UBadge v-if="selectedLeadUUIDsForAssign.length > 0" variant="soft" color="warning">
              已选 {{ selectedLeadUUIDsForAssign.length }} 条
            </UBadge>
            <UButton size="xs" variant="soft" :disabled="pagedLeads.length === 0" @click="toggleSelectAllCurrentPageForAssign">
              {{ allCurrentPageSelectedForAssign ? "取消当前页全选" : "全选当前页" }}
            </UButton>
            <UButton size="xs" variant="soft" :disabled="selectedLeadUUIDsForAssign.length === 0" @click="clearSelectedLeadsForAssign">
              清空勾选
            </UButton>
            <UButton size="xs" color="primary" :disabled="selectedLeadUUIDsForAssign.length === 0" @click="openBatchAssignModal">
              批量绑定负责人
            </UButton>
          </div>
        </div>
      </template>

      <UTable
        :columns="leadColumns"
        :data="pagedLeads"
        :loading="store.loading"
        :ui="{ table: 'min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-700' }"
      >
        <template #select-cell="{ row }">
          <input
            type="checkbox"
            class="h-4 w-4"
            :checked="isLeadSelectedForAssign(row.original.lead_uuid)"
            @change="setLeadSelectedForAssign(row.original.lead_uuid, $event)"
          />
        </template>
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
          <div class="space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <UBadge :color="statusMeta(row.original.status).color" variant="soft">
                {{ statusMeta(row.original.status).label }}
              </UBadge>
              <UBadge v-if="row.original.has_merge" color="warning" variant="soft">
                已合并
              </UBadge>
            </div>
            <div v-if="String(row.original.owner_user_uuid || '').trim()" class="text-xs text-gray-500 dark:text-gray-400">
              {{ resolveLeadOwnerDisplayName(row.original.owner_user_uuid) }}
            </div>
          </div>
        </template>
        <template #source-cell="{ row }">
          <div class="space-y-1 text-sm text-gray-600 dark:text-gray-300">
            <div class="flex items-center gap-2">
              <span>{{ row.original.source_channel || '未知渠道' }}</span>
              <UBadge size="xs" :color="leadOriginMeta(row.original).color" variant="soft">
                {{ leadOriginMeta(row.original).label }}
              </UBadge>
              <UBadge
                v-if="isChannelLead(row.original)"
                size="xs"
                :color="leadSyncStateMeta(row.original).color"
                variant="soft"
              >
                {{ leadSyncStateMeta(row.original).label }}
              </UBadge>
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              {{ row.original.source_app_type || '未知应用' }}
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              渠道账号：{{ resolveLeadSourceAccountLabel(row.original) }}
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              外部联系人ID：{{
                isChannelLead(row.original)
                  ? (leadSyncExternalInfo(row.original.lead_uuid).externalLeadId || '未记录（不可回写）')
                  : '本地线索不适用'
              }}
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              微信号：{{
                isChannelLead(row.original)
                  ? (leadSyncExternalInfo(row.original.lead_uuid).externalWechatId || '未记录（需重跑同步）')
                  : '本地线索不适用'
              }}
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              企微添加人：{{
                isChannelLead(row.original)
                  ? (String((row.original as any).wecom_adder_userid || '').trim() || '未记录（需重跑同步）')
                  : '本地线索不适用'
              }}
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              关系状态：{{
                isChannelLead(row.original)
                  ? relationStatusMeta(row.original.status).label
                  : '本地线索不适用'
              }}
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
      v-model:open="batchAssignModalOpen"
      :prevent-close="true"
      :dismissible="false"
      :modal="true"
      :ui="{ content: 'max-w-xl w-full' }"
    >
      <template #title>批量绑定负责人</template>
      <template #description>
        已选择 {{ selectedLeadUUIDsForAssign.length }} 条线索，设置后会逐条执行绑定。
      </template>
      <template #body>
        <UForm :state="batchAssignForm" class="space-y-4 p-4 sm:p-5">
          <UFormField label="负责人" required>
            <USelectMenu
              v-model="batchAssignOwnerOption"
              :items="createOwnerOptions"
              placeholder="请选择负责人"
              class="w-full"
              :portal="false"
              :ui="{ content: 'z-[200]' }"
            />
          </UFormField>
          <UFormField label="原因（可选）">
            <UInput v-model="batchAssignForm.reason" placeholder="例如：线索分配给销售 A 组" />
          </UFormField>
        </UForm>
      </template>
      <template #footer>
        <div class="flex flex-col gap-2 sm:flex-row sm:justify-end">
          <UButton color="neutral" variant="soft" :disabled="batchAssignSubmitting" @click="closeBatchAssignModal">
            取消
          </UButton>
          <UButton color="primary" :loading="batchAssignSubmitting" @click="submitBatchAssign">
            确认绑定
          </UButton>
        </div>
      </template>
    </UModal>

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

    <UModal v-model:open="channelCodeCreateOpen" :ui="{ content: 'max-w-2xl w-full' }">
      <template #title>新建渠道码</template>
      <template #description>创建后可为该渠道码配置专属欢迎语。</template>
      <template #body>
        <UForm :state="channelCodeCreateForm" class="space-y-3 p-4">
          <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
            <UFormField label="渠道" required>
              <UInput v-model="channelCodeCreateForm.channel" placeholder="wechat" />
            </UFormField>
            <UFormField label="应用类型" required>
              <UInput v-model="channelCodeCreateForm.app_type" placeholder="wecom" />
            </UFormField>
          </div>
          <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
            <UFormField label="渠道账号 UUID" required>
              <UInput v-model="channelCodeCreateForm.channel_account_uuid" placeholder="渠道账号 UUID" />
            </UFormField>
            <UFormField label="渠道码 Key" required>
              <UInput v-model="channelCodeCreateForm.code_key" placeholder="campaign-key" />
            </UFormField>
          </div>
          <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
            <UFormField label="显示名" required>
              <UInput v-model="channelCodeCreateForm.display_name" placeholder="显示名称" />
            </UFormField>
            <UFormField label="目标类型" required>
              <USelectMenu
                v-model="channelCodeCreateForm.target_type"
                :items="channelCodeTargetTypeOptions"
                value-key="value"
                label-key="label"
                class="w-full"
                :portal="false"
                :ui="{ content: 'z-[200]' }"
              />
            </UFormField>
          </div>
          <UFormField label="目标 ID" required>
            <UInput v-model="channelCodeCreateForm.target_id" placeholder="group-001" />
          </UFormField>
        </UForm>
      </template>
      <template #footer>
        <div class="flex items-center justify-end gap-2">
          <UButton variant="soft" @click="channelCodeCreateOpen = false">取消</UButton>
          <UButton color="primary" :loading="channelCodeStore.saving" @click="submitChannelCodeCreate">创建</UButton>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onActivated, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { useRouter } from "#imports";
import type { LeadCreatePayload } from "~/types/lead_capture/lead";
import { useLeadCaptureStore } from "~/stores/scrm/lead_capture/lead_store";
import { useLeadChannelCodeStore } from "~/stores/scrm/lead_capture/channel_code_store";
import { useUserStore } from "~/stores/user";
import ToastAlert from "~/components/ToastAlert.vue";
import {
  useLeadCaptureService,
  type ChannelCodeEventRecord,
  type ChannelCodeEventStats,
  type ChannelCodeWelcomeSyncStatus,
  type WeComWritebackPolicy,
  type WeComSyncTaskRecord,
  type WeComCustomerDMRule,
  type LeadBatchAssignResult,
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
import { useWsBusClient } from "~/composables/useWsBusClient";

definePageMeta({
  layout: "default",
});

const store = useLeadCaptureStore();
const channelCodeStore = useLeadChannelCodeStore();
const userStore = useUserStore();
const router = useRouter();
const leadCaptureService = useLeadCaptureService();
const runtimeDictionaryService = useRuntimeDictionaryService();
const socialChannelService = useSocialChannelGovernanceService();
const iamService = useIAMService();
const wsBus = useWsBusClient();

const ALL_OPTION_VALUE = "__all__";
const showChannelCodeWelcomePanel = false;

const searchText = ref("");
const statusFilter = ref<string>(ALL_OPTION_VALUE);
const channelFilter = ref<string>(ALL_OPTION_VALUE);
const appTypeFilter = ref<string>(ALL_OPTION_VALUE);
const createModalOpen = ref(false);
const creating = ref(false);
const createFormError = ref("");
const batchAssignModalOpen = ref(false);
const batchAssignSubmitting = ref(false);
const currentPage = ref(1);
const pageSize = ref(10);
const importModalOpen = ref(false);
const importing = ref(false);
const importFile = ref<File | null>(null);
const importFileName = ref("");
const importResult = ref<any | null>(null);
const importStep = ref(1);
const channelCodeCreateOpen = ref(false);
const selectedChannelCodeUUID = ref("");
const channelCodeWelcomeEnabled = ref(false);
const channelCodeWelcomeContentText = ref('{"text":"欢迎添加企业微信"}');
const channelCodeWelcomeSyncLoading = ref(false);
const channelCodeWelcomeSyncStatus = ref<ChannelCodeWelcomeSyncStatus | null>(null);
const channelCodeEventsLoading = ref(false);
const channelCodeEvents = ref<ChannelCodeEventRecord[]>([]);
const channelCodeEventStats = ref<ChannelCodeEventStats>({
  touch_total: 0,
  intake_total: 0,
  dedup_total: 0,
});
const syncLoading = ref(false);
const syncSubmitting = ref(false);
const syncTaskClearing = ref(false);
const syncCenterOpen = ref(false);
const syncTaskPanelOpen = ref(true);
const deadLetterPanelOpen = ref(false);
const dmRulePanelOpen = ref(false);
const syncTriggerAction = ref<"pull_external_contacts">("pull_external_contacts");
const selectedLeadUUIDsForAssign = ref<string[]>([]);
const selectedLeadUUIDsForPush = ref<string[]>([]);
const syncTasks = ref<WeComSyncTaskRecord[]>([]);
const syncTaskPage = ref(1);
const syncTaskPageSize = ref(5);
const pushLeadValidationPageSize = 10;
const pushLeadPassedPage = ref(1);
const pushLeadRejectedPage = ref(1);
const writebackPolicyLoading = ref(false);
const writebackPolicySaving = ref(false);
const writebackPolicyDialogOpen = ref(false);
const writebackDeadLettersLoading = ref(false);
const writebackReplayLoadingUUID = ref("");
const writebackPolicy = ref<WeComWritebackPolicy | null>(null);
const writebackEnabled = ref(true);
const writebackOverwriteMode = ref<"safe" | "force">("safe");
const writebackWhitelistFields = ref<string[]>([]);
const writebackProtectedFields = ref<string[]>([]);
const writebackDeadLetters = ref<any[]>([]);
const syncActivePollingTaskUUID = ref("");
const lastLeadAutoRefreshTaskSignature = ref("");
const lastActiveTaskToastSignature = ref("");
const syncLastResolveSource = ref("");
const wecomCustomerDMRuleLoading = ref(false);
const wecomCustomerDMRuleSaving = ref(false);
const wecomCustomerDMAutoCreate = ref(false);
const channelAccounts = ref<ChannelAccount[]>([]);
const iamMembers = ref<MemberRecord[]>([]);
const sourceCatalogs = ref<RuntimeDictionaryItem[]>([]);
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

const channelCodeCreateForm = reactive({
  channel: "wechat",
  app_type: "wecom",
  channel_account_uuid: "",
  code_key: "",
  display_name: "",
  target_type: "group" as "group" | "dm" | "entry",
  target_id: "",
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

const batchAssignForm = reactive({
  reason: "",
});

const batchAssignOwnerOption = ref<any>(null);

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
  { accessorKey: "select", header: "" },
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
  { label: "MQL", value: "mql" },
  { label: "SQL", value: "sql" },
  { label: "已转化", value: "converted" },
  { label: "已关闭", value: "closed" },
  { label: "已断开关系", value: "disconnected" },
];

const pageSizeOptions = [
  { label: "10/页", value: 10 },
  { label: "20/页", value: 20 },
  { label: "50/页", value: 50 },
];

const syncTriggerActionOptions = [
  { label: "拉取外部联系人到线索", value: "pull_external_contacts" },
];

const pushLeadSelectionOptions = computed(() =>
  filteredLeads.value.map((lead) => {
    const name = (lead.display_name || "").trim();
    const phone = (lead.phone || "").trim();
    const email = (lead.email || "").trim();
    const status = statusMeta(lead.status).label;
    const title = name || phone || email || lead.lead_uuid;
    const desc = [phone || "-", email || "-", status].join(" / ");
    return {
      value: lead.lead_uuid,
      label: `${title}（${desc}）`,
    };
  })
);

const syncTaskPageSizeOptions = [
  { label: "5/页", value: 5 },
  { label: "10/页", value: 10 },
  { label: "20/页", value: 20 },
];

const writebackOverwriteModeOptions = [
  { label: "安全覆盖（safe）", value: "safe" },
  { label: "强制覆盖（force）", value: "force" },
];

const writebackFieldOptions = [
  { label: "姓名（display_name）", value: "display_name" },
  { label: "手机号（phone）", value: "phone" },
  { label: "邮箱（email）", value: "email" },
  { label: "负责人（owner_user_uuid）", value: "owner_user_uuid" },
  { label: "状态（status）", value: "status" },
  { label: "线索UUID（lead_uuid）", value: "lead_uuid" },
  { label: "租户UUID（tenant_uuid）", value: "tenant_uuid" },
  { label: "来源账号（source_account_uuid）", value: "source_account_uuid" },
  { label: "来源渠道（source_channel）", value: "source_channel" },
  { label: "来源应用（source_app_type）", value: "source_app_type" },
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

const pushLeadValidationColumns = [
  { accessorKey: "select", header: "" },
  { accessorKey: "name", header: "线索" },
  { accessorKey: "contact", header: "联系方式" },
  { accessorKey: "result", header: "结果" },
] satisfies any;

const deadLetterColumns = [
  { accessorKey: "dead_letter_uuid", header: "死信 UUID" },
  { accessorKey: "domain", header: "域" },
  { accessorKey: "direction", header: "方向" },
  { accessorKey: "replay_status", header: "状态" },
  { accessorKey: "error", header: "错误" },
  { accessorKey: "actions", header: "操作" },
] satisfies any;

const channelCodeTargetTypeOptions = [
  { label: "群聊", value: "group" },
  { label: "私聊", value: "dm" },
  { label: "入口", value: "entry" },
];

const syncTaskTotalPages = computed(() => {
  const total = Math.ceil(syncTasks.value.length / syncTaskPageSize.value);
  return total > 0 ? total : 1;
});

const channelCodeOptions = computed(() => {
  return channelCodeStore.items.map((item) => ({
    value: item.code_uuid,
    label: `${item.display_name} (${item.code_key})`,
  }));
});

const selectedChannelCode = computed(() =>
  channelCodeStore.items.find((item) => item.code_uuid === selectedChannelCodeUUID.value) || null
);

const pagedSyncTasks = computed(() => {
  const start = (syncTaskPage.value - 1) * syncTaskPageSize.value;
  return syncTasks.value.slice(start, start + syncTaskPageSize.value);
});

type PushLeadValidationRow = {
  leadUUID: string;
  name: string;
  contact: string;
  pass: boolean;
  reason: string;
  syncStatus: "pending_push" | "synced" | "unsynced";
};

const writebackWhitelistSet = computed(() =>
  new Set(writebackWhitelistFields.value.map((item) => (item || "").trim()).filter(Boolean))
);

const writebackProtectedSet = computed(() =>
  new Set(writebackProtectedFields.value.map((item) => (item || "").trim()).filter(Boolean))
);

const selectedLeadsForPush = computed(() => {
  const picked = new Set(selectedLeadUUIDsForPush.value.map((v) => (v || "").trim()).filter(Boolean));
  return store.leads.filter((lead) => picked.has((lead.lead_uuid || "").trim()));
});

const leadExternalInfoMap = computed(() => {
  const map = new Map<string, { externalLeadId: string; externalWechatId: string; syncState: "synced" | "unsynced" }>();
  store.leads.forEach((lead) => {
    const leadUUID = (lead.lead_uuid || "").trim();
    if (!leadUUID) return;
    const externalLeadId = String((lead as any).external_userid || "").trim();
    const externalWechatId = String((lead as any).external_wechat_id || "").trim();
    const syncState = (String((lead as any).channel_sync_status || "").trim().toLowerCase() === "synced")
      ? "synced"
      : "unsynced";
    map.set(leadUUID, { externalLeadId, externalWechatId, syncState });
  });
  return map;
});

const previewLeadsForPush = computed(() => {
  const activeAccountUUID = String(defaultSyncAccount.value?.account_uuid || "").trim().toLowerCase();
  if (!activeAccountUUID) {
    return filteredLeads.value.filter((lead) => isChannelLead(lead));
  }
  return filteredLeads.value.filter((lead) => {
    if (!isChannelLead(lead)) return false;
    const leadAccountUUID = String((lead as any).source_account_uuid || "").trim().toLowerCase();
    return leadAccountUUID !== "" && leadAccountUUID === activeAccountUUID;
  });
});

const pushPreviewScopeHint = computed(() => {
  const account = defaultSyncAccount.value;
  if (!account) {
    return "当前预检范围：渠道线索（未识别默认账号，仅展示渠道线索）";
  }
  const label = account.display_name || account.account_id || account.account_uuid;
  return `当前预检范围：当前筛选结果中归属账号「${label}」的渠道线索（勾选仅用于确定最终推送集合）`;
});

const pushLeadValidationRows = computed<PushLeadValidationRow[]>(() => {
  const capability = (writebackPolicy.value?.capability_status || "").trim().toLowerCase();
  const writebackDisabled = !writebackEnabled.value;
  const activeAccountUUID = String(defaultSyncAccount.value?.account_uuid || "").trim().toLowerCase();
  return previewLeadsForPush.value.map((lead) => {
    const syncStatusRaw = String((lead as any).channel_sync_status || "").trim().toLowerCase();
    const syncStatus: "pending_push" | "synced" | "unsynced" =
      syncStatusRaw === "pending_push"
        ? "pending_push"
        : (syncStatusRaw === "synced" ? "synced" : "unsynced");
    const displayName =
      (lead.display_name || "").trim() ||
      (lead.phone || "").trim() ||
      (lead.email || "").trim() ||
      lead.lead_uuid;
    const contact = [
      (lead.phone || "").trim() || "-",
      (lead.email || "").trim() || "-",
    ].join(" / ");
    const leadAccountUUID = String((lead as any).source_account_uuid || "").trim().toLowerCase();
    if (activeAccountUUID && leadAccountUUID && leadAccountUUID !== activeAccountUUID) {
      return {
        leadUUID: lead.lead_uuid,
        name: displayName,
        contact,
        pass: false,
        reason: "线索归属其他渠道账号",
        syncStatus,
      };
    }
    if (capability === "not_supported") {
      return {
        leadUUID: lead.lead_uuid,
        name: displayName,
        contact,
        pass: false,
        reason: "当前渠道不支持回写",
        syncStatus,
      };
    }
    if (writebackDisabled) {
      return {
        leadUUID: lead.lead_uuid,
        name: displayName,
        contact,
        pass: false,
        reason: "回写策略未启用",
        syncStatus,
      };
    }
    const ext = leadSyncExternalInfo(lead.lead_uuid);
    if (!ext.externalLeadId) {
      return {
        leadUUID: lead.lead_uuid,
        name: displayName,
        contact,
        pass: false,
        reason: "缺少 external_userid",
        syncStatus,
      };
    }
    const ownerUserUUID = String((lead.owner_user_uuid || "")).trim();
    if (!ownerUserUUID) {
      return {
        leadUUID: lead.lead_uuid,
        name: displayName,
        contact,
        pass: false,
        reason: "缺少负责人（owner_user_uuid）",
        syncStatus,
      };
    }
    if (syncStatus !== "pending_push" && writebackOverwriteMode.value !== "force") {
      return {
        leadUUID: lead.lead_uuid,
        name: displayName,
        contact,
        pass: false,
        reason: "未发生变更（切换 force 可强制回写）",
        syncStatus,
      };
    }
    const fields = buildLeadWritebackFields(lead);
    const sanitized = applyWritebackPolicyToFields(fields);
    if (Object.keys(sanitized).length === 0) {
      return {
        leadUUID: lead.lead_uuid,
        name: displayName,
        contact,
        pass: false,
        reason: "字段被策略过滤",
        syncStatus,
      };
    }
    return {
      leadUUID: lead.lead_uuid,
      name: displayName,
      contact,
      pass: true,
      reason: "",
      syncStatus,
    };
  });
});

const pushLeadRowPriority = (row: PushLeadValidationRow) => {
  if (row.syncStatus === "pending_push") return 0;
  if (row.syncStatus === "unsynced") return 1;
  return 2;
};

const comparePushLeadRows = (a: PushLeadValidationRow, b: PushLeadValidationRow) => {
  const pa = pushLeadRowPriority(a);
  const pb = pushLeadRowPriority(b);
  if (pa !== pb) return pa - pb;
  const byName = a.name.localeCompare(b.name, "zh-Hans-CN");
  if (byName !== 0) return byName;
  return a.leadUUID.localeCompare(b.leadUUID);
};

const pushLeadPassedRows = computed(() =>
  pushLeadValidationRows.value.filter((row) => row.pass).slice().sort(comparePushLeadRows)
);
const pushLeadRejectedRows = computed(() =>
  pushLeadValidationRows.value.filter((row) => !row.pass).slice().sort(comparePushLeadRows)
);
const pushLeadPassedTotalPages = computed(() => Math.max(1, Math.ceil(pushLeadPassedRows.value.length / pushLeadValidationPageSize)));
const pushLeadRejectedTotalPages = computed(() => Math.max(1, Math.ceil(pushLeadRejectedRows.value.length / pushLeadValidationPageSize)));
const pagedPushLeadPassedRows = computed(() => {
  const start = (pushLeadPassedPage.value - 1) * pushLeadValidationPageSize;
  return pushLeadPassedRows.value.slice(start, start + pushLeadValidationPageSize);
});
const pagedPushLeadRejectedRows = computed(() => {
  const start = (pushLeadRejectedPage.value - 1) * pushLeadValidationPageSize;
  return pushLeadRejectedRows.value.slice(start, start + pushLeadValidationPageSize);
});
const canTriggerPushLeadSync = computed(() => pushLeadPassedRows.value.length > 0);

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

const defaultSyncAccount = computed(() => {
  const activeWeComAccounts = channelAccounts.value.filter((account) => {
    const channel = (account.channel_code || "").trim().toLowerCase();
    const appType = (account.app_type || "").trim().toLowerCase();
    const status = (account.status || "").trim().toLowerCase();
    return channel === "wechat" && (appType === "wecom" || appType === "openwork") && status !== "disabled";
  });
  if (!activeWeComAccounts.length) return null;
  const preferred = activeWeComAccounts.find((account) => !!account.org_sync_default);
  return preferred || activeWeComAccounts[0];
});

const defaultSyncAccountLabel = computed(() => {
  const account = defaultSyncAccount.value;
  if (!account) return "未找到默认渠道账号（将由服务端兜底解析）";
  return `${account.display_name || account.account_id} (${account.channel_code}/${account.app_type})`;
});

const isWeComDefaultSyncChannel = computed(() => {
  const account = defaultSyncAccount.value;
  if (!account) return false;
  const channel = (account.channel_code || "").trim().toLowerCase();
  const appType = (account.app_type || "").trim().toLowerCase();
  return channel === "wechat" && (appType === "wecom" || appType === "openwork");
});

const currentSyncChannelTag = computed(() => {
  const account = defaultSyncAccount.value;
  if (!account) return "渠道未识别";
  return `${account.channel_code}/${account.app_type}`;
});

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
  iamMembers.value
    .map((member) => {
      const memberID = String((member as any).member_id ?? member.id ?? "").trim();
      return {
        label: `${member.display_name} (${member.email || member.username || memberID || "-"})`,
        value: memberID,
      };
    })
    .filter((item) => item.value !== "")
);

const iamMemberNameMap = computed(() => {
  const map = new Map<string, string>();
  iamMembers.value.forEach((member) => {
    const id = String((member as any).member_id ?? member.id ?? "").trim();
    const name = String(member.display_name || member.username || member.email || "").trim();
    if (!id || !name) return;
    map.set(id, name);
  });
  return map;
});

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

const allCurrentPageSelectedForAssign = computed(() => {
  if (pagedLeads.value.length === 0) return false;
  return pagedLeads.value.every((lead) =>
    selectedLeadUUIDsForAssign.value.includes(String(lead.lead_uuid || "").trim())
  );
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
    case "mql":
      return { label: "MQL", color: "warning" };
    case "sql":
      return { label: "SQL", color: "success" };
    case "converted":
      return { label: "已转化", color: "success" };
    case "closed":
      return { label: "已关闭", color: "neutral" };
    case "disconnected":
      return { label: "已断开关系", color: "error" };
    case "new":
    default:
      return { label: "新线索", color: "info" };
  }
};

const relationStatusMeta = (status?: string) => {
  if (String(status || "").trim().toLowerCase() === "disconnected") {
    return { label: "disconnected（已断开）", color: "error" };
  }
  return { label: "connected（已关联）", color: "success" };
};

const leadSyncExternalInfo = (_leadId?: string) => {
  const leadId = (_leadId || "").trim();
  if (!leadId) {
    return {
      externalLeadId: "",
      externalWechatId: "",
      syncState: "unsynced" as const,
    };
  }
  return (
    leadExternalInfoMap.value.get(leadId) || {
      externalLeadId: "",
      externalWechatId: "",
      syncState: "unsynced" as const,
    }
  );
};

const leadOriginMeta = (lead?: any) => {
  const v = String(lead?.lead_origin_type || "").trim().toLowerCase();
  if (v === "channel") return { label: "渠道线索", color: "info" };
  return { label: "本地线索", color: "neutral" };
};

const isChannelLead = (lead?: any) =>
  String(lead?.lead_origin_type || "").trim().toLowerCase() === "channel";

const leadSyncStateMeta = (lead?: any) => {
  const v = String(lead?.channel_sync_status || "").trim().toLowerCase();
  if (v === "synced") return { label: "已同步", color: "success" };
  if (v === "pending_push") return { label: "待回写", color: "warning" };
  return {
    label: "未同步",
    color: "warning",
  };
};

const refreshLeads = async () => {
  await store.fetchLeads();
  pruneSelectedLeadsForAssign();
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

const isLeadSelectedForAssign = (leadUUID?: string) => {
  const id = String(leadUUID || "").trim();
  if (!id) return false;
  return selectedLeadUUIDsForAssign.value.includes(id);
};

const setLeadSelectedForAssign = (leadUUID?: string, checked?: unknown) => {
  const id = String(leadUUID || "").trim();
  if (!id) return;
  let value = Boolean(checked);
  if (checked && typeof checked === "object" && "target" in (checked as Record<string, unknown>)) {
    const target = (checked as Event).target as HTMLInputElement | null;
    value = !!target?.checked;
  }
  const next = new Set(selectedLeadUUIDsForAssign.value.map((v) => String(v || "").trim()).filter(Boolean));
  if (value) {
    next.add(id);
  } else {
    next.delete(id);
  }
  selectedLeadUUIDsForAssign.value = Array.from(next);
};

const toggleSelectAllCurrentPageForAssign = () => {
  const next = new Set(selectedLeadUUIDsForAssign.value.map((v) => String(v || "").trim()).filter(Boolean));
  if (allCurrentPageSelectedForAssign.value) {
    pagedLeads.value.forEach((lead) => next.delete(String(lead.lead_uuid || "").trim()));
  } else {
    pagedLeads.value.forEach((lead) => next.add(String(lead.lead_uuid || "").trim()));
  }
  selectedLeadUUIDsForAssign.value = Array.from(next);
};

const clearSelectedLeadsForAssign = () => {
  selectedLeadUUIDsForAssign.value = [];
};

const closeBatchAssignModal = () => {
  if (batchAssignSubmitting.value) return;
  blurActiveElement();
  batchAssignModalOpen.value = false;
};

const openBatchAssignModal = async () => {
  if (selectedLeadUUIDsForAssign.value.length === 0) {
    showToast("请先勾选至少一条线索", "warning");
    return;
  }
  if (iamMembers.value.length === 0) {
    await loadCreateLookupOptions();
  }
  batchAssignOwnerOption.value = null;
  batchAssignForm.reason = "";
  batchAssignModalOpen.value = true;
};

const resolveOwnerUserUUID = (raw: any): string => {
  if (raw === null || raw === undefined) return "";
  if (typeof raw === "string" || typeof raw === "number") {
    return String(raw).trim();
  }
  if (typeof raw === "object") {
    if ("value" in raw && raw.value !== undefined && raw.value !== null) {
      return String(raw.value).trim();
    }
    if ("member_id" in raw && raw.member_id !== undefined && raw.member_id !== null) {
      return String(raw.member_id).trim();
    }
    if ("id" in raw && raw.id !== undefined && raw.id !== null) {
      return String(raw.id).trim();
    }
  }
  return "";
};

const submitBatchAssign = async () => {
  const ownerUserUUID = resolveOwnerUserUUID(batchAssignOwnerOption.value);
  if (!ownerUserUUID) {
    showToast("请选择负责人", "warning");
    return;
  }
  const leadUUIDs = selectedLeadUUIDsForAssign.value.map((item) => String(item || "").trim()).filter(Boolean);
  if (leadUUIDs.length === 0) {
    showToast("请先勾选线索", "warning");
    return;
  }
  batchAssignSubmitting.value = true;
  try {
    const resp = await leadCaptureService.batchAssignLeads({
      lead_uuids: leadUUIDs,
      owner_user_uuid: ownerUserUUID,
      reason: String(batchAssignForm.reason || "").trim() || undefined,
    });
    const result = ((resp as any)?.data || null) as LeadBatchAssignResult | null;
    const successCount = Number(result?.success_count || 0);
    const failedCount = Number(result?.failed_count || 0);
    if (failedCount > 0) {
      const firstFailed = (result?.items || []).find((item) => !item.success);
      showToast(
        firstFailed?.error_message || `成功 ${successCount} 条，失败 ${failedCount} 条`,
        "warning",
        "批量绑定部分失败"
      );
    } else {
      showToast(`已成功绑定 ${successCount} 条线索`, "success");
    }
    await refreshLeads();
    clearSelectedLeadsForAssign();
    closeBatchAssignModal();
  } catch (err: any) {
    showToast(err?.message || "批量绑定负责人失败", "error");
  } finally {
    batchAssignSubmitting.value = false;
  }
};

const pruneSelectedLeadsForAssign = () => {
  if (selectedLeadUUIDsForAssign.value.length === 0) return;
  const current = new Set(store.leads.map((lead) => String(lead.lead_uuid || "").trim()).filter(Boolean));
  selectedLeadUUIDsForAssign.value = selectedLeadUUIDsForAssign.value
    .map((item) => String(item || "").trim())
    .filter((item) => item && current.has(item));
};

const clearSelectedLeadsForPush = () => {
  selectedLeadUUIDsForPush.value = [];
};

const selectAllFilteredLeadsForPush = () => {
  selectedLeadUUIDsForPush.value = filteredLeads.value.map((lead) => lead.lead_uuid);
};

const isLeadSelectedForPush = (leadUUID?: string) => {
  const id = (leadUUID || "").trim();
  if (!id) return false;
  return selectedLeadUUIDsForPush.value.includes(id);
};

const setLeadSelectedForPush = (leadUUID?: string, checked?: unknown) => {
  const id = (leadUUID || "").trim();
  if (!id) return;
  const next = new Set(selectedLeadUUIDsForPush.value.map((v) => (v || "").trim()).filter(Boolean));
  if (Boolean(checked)) {
    next.add(id);
  } else {
    next.delete(id);
  }
  selectedLeadUUIDsForPush.value = Array.from(next);
};

const normalizeRejectReason = (row?: PushLeadValidationRow) => String(row?.reason || "").trim();

const isNeedEditExternalContactReason = (reason: string) =>
  reason.includes("未发生变更") || reason.includes("external_userid") || reason.includes("未绑定客户");

const isOwnerBindingReason = (reason: string) =>
  reason.includes("负责人") || reason.includes("owner_user_uuid");

const canJumpToLeadDetail = (row?: PushLeadValidationRow) => {
  if (!row) return false;
  const reason = normalizeRejectReason(row);
  const leadUUID = String((row as any).leadUUID || (row as any).lead_uuid || "").trim();
  return (
    (
      isOwnerBindingReason(reason) ||
      isNeedEditExternalContactReason(reason)
    ) &&
    !!leadUUID
  );
};

const rejectedJumpLabel = (row?: PushLeadValidationRow) => {
  const reason = normalizeRejectReason(row);
  if (isNeedEditExternalContactReason(reason)) return "修改";
  if (isOwnerBindingReason(reason)) return "去组织绑定";
  return "去修改";
};

const tryOpenRejectedLeadDetail = (row?: PushLeadValidationRow) => {
  if (!canJumpToLeadDetail(row)) return;
  const reason = normalizeRejectReason(row);
  const leadUUID = String((row as any)?.leadUUID || (row as any)?.lead_uuid || "").trim();
  if (isOwnerBindingReason(reason)) {
    router.push("/scrm/org_sync");
    return;
  }
  if (leadUUID) openDetail(leadUUID);
};

const buildLeadWritebackFields = (lead: any) => ({
  lead_uuid: lead.lead_uuid,
  display_name: (lead.display_name || "").trim(),
  phone: (lead.phone || "").trim(),
  email: (lead.email || "").trim(),
  status: (lead.status || "").trim(),
  owner_user_uuid: (lead.owner_user_uuid || "").trim(),
  source_channel: (lead.source_channel || "").trim(),
  source_app_type: (lead.source_app_type || "").trim(),
  source_account_uuid: (lead.source_account_uuid || "").trim(),
});

const applyWritebackPolicyToFields = (fields: Record<string, any>) => {
  const output: Record<string, any> = {};
  Object.entries(fields || {}).forEach(([key, value]) => {
    const cleanKey = (key || "").trim();
    if (!cleanKey) return;
    if (writebackWhitelistSet.value.size > 0 && !writebackWhitelistSet.value.has(cleanKey)) return;
    if (writebackProtectedSet.value.has(cleanKey)) return;
    output[cleanKey] = value;
  });
  return output;
};

const buildLeadWritebackPayload = () => {
  const passed = new Set(pushLeadPassedRows.value.map((row) => row.leadUUID));
  const targets = selectedLeadsForPush.value.filter((lead) => passed.has((lead.lead_uuid || "").trim()));
  return targets.map((lead) => {
    const ext = leadSyncExternalInfo((lead.lead_uuid || "").trim());
    const externalUserID =
      String((lead as any).external_userid || "").trim() ||
      String(ext.externalLeadId || "").trim();
    const rawFields = buildLeadWritebackFields(lead);
    const filteredFields = applyWritebackPolicyToFields(rawFields);
    if (externalUserID) {
      filteredFields.external_userid = externalUserID;
    }
    return {
      lead_uuid: lead.lead_uuid,
      external_userid: externalUserID || undefined,
      phone: (lead.phone || "").trim() || undefined,
      idempotency_hint: `lead:${lead.lead_uuid}:${lead.updated_at || lead.created_at || ""}`,
      fields: filteredFields,
    };
  });
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

const resolveLeadSourceAccountLabel = (lead?: any) => {
  const uuid = String(lead?.source_account_uuid || "").trim();
  if (!uuid) return "未记录";
  const label = syncAccountLabelMap.value.get(uuid.toLowerCase());
  if (label) return `${label} · ${shortUUID(uuid)}`;
  return shortUUID(uuid);
};

const resolveLeadOwnerDisplayName = (ownerUserUUID?: string) => {
  const id = String(ownerUserUUID || "").trim();
  if (!id) return "-";
  return iamMemberNameMap.value.get(id) || id;
};

const toggleWritebackSelection = (scope: "whitelist" | "protected", field: string, checked: boolean) => {
  const target = scope === "whitelist" ? writebackWhitelistFields : writebackProtectedFields;
  const set = new Set(target.value.map((item) => item.trim()).filter(Boolean));
  if (checked) {
    set.add(field);
  } else {
    set.delete(field);
  }
  target.value = Array.from(set);
};

const onWritebackCheckboxChange = (
  scope: "whitelist" | "protected",
  field: string,
  event: Event
) => {
  const checked = !!(event.target as HTMLInputElement | null)?.checked;
  toggleWritebackSelection(scope, field, checked);
};

const writebackCapabilityLabel = (status?: string) => {
  switch ((status || "").trim()) {
    case "supported":
      return "已支持";
    case "partial":
      return "部分支持";
    case "not_supported":
      return "暂不支持";
    case "planned":
      return "规划中";
    default:
      return "未知";
  }
};

const writebackCapabilityColor = (status?: string) => {
  switch ((status || "").trim()) {
    case "supported":
      return "success";
    case "partial":
      return "warning";
    case "not_supported":
      return "neutral";
    case "planned":
      return "info";
    default:
      return "neutral";
  }
};

const loadWritebackPolicy = async () => {
  writebackPolicyLoading.value = true;
  try {
    const resp = await leadCaptureService.getWeComWritebackPolicy();
    const policy = ((resp as any)?.data || null) as WeComWritebackPolicy | null;
    writebackPolicy.value = policy;
    writebackEnabled.value = !!policy?.enabled;
    writebackOverwriteMode.value = (policy?.overwrite_mode || "safe") as "safe" | "force";
    writebackWhitelistFields.value = ((policy?.mapping_rules?.whitelist || []) as any[])
      .map((item) => String(item || "").trim())
      .filter(Boolean);
    writebackProtectedFields.value = ((policy?.protected_fields?.fields || []) as any[])
      .map((item) => String(item || "").trim())
      .filter(Boolean);
  } catch (err: any) {
    showToast(err?.message || "加载回写策略失败", "error");
  } finally {
    writebackPolicyLoading.value = false;
  }
};

const openWritebackPolicyDialog = async () => {
  await loadWritebackPolicy();
  writebackPolicyDialogOpen.value = true;
};

const saveWritebackPolicy = async () => {
  writebackPolicySaving.value = true;
  try {
    const resp = await leadCaptureService.updateWeComWritebackPolicy({
      enabled: !!writebackEnabled.value,
      overwrite_mode: writebackOverwriteMode.value,
      mapping_rules: { whitelist: writebackWhitelistFields.value },
      protected_fields: { fields: writebackProtectedFields.value },
    });
    writebackPolicy.value = ((resp as any)?.data || null) as WeComWritebackPolicy | null;
    showToast("回写策略已保存", "success");
  } catch (err: any) {
    showToast(err?.message || "保存回写策略失败", "error");
  } finally {
    writebackPolicySaving.value = false;
  }
};

const saveWritebackPolicyFromDialog = async () => {
  await saveWritebackPolicy();
  writebackPolicyDialogOpen.value = false;
};

const refreshWritebackDeadLetters = async () => {
  writebackDeadLettersLoading.value = true;
  try {
    const resp = await leadCaptureService.listWeComWritebackDeadLetters();
    writebackDeadLetters.value = ((resp as any)?.data?.items || []) as any[];
  } catch (err: any) {
    showToast(err?.message || "加载回写死信失败", "error");
  } finally {
    writebackDeadLettersLoading.value = false;
  }
};

const replayWritebackDeadLetter = async (deadLetterUUID?: string) => {
  const uuid = (deadLetterUUID || "").trim();
  if (!uuid) return;
  writebackReplayLoadingUUID.value = uuid;
  try {
    await leadCaptureService.replayWeComWritebackDeadLetter(uuid);
    showToast("死信重放已提交", "success");
    await refreshWritebackDeadLetters();
  } catch (err: any) {
    showToast(err?.message || "死信重放失败", "error");
  } finally {
    writebackReplayLoadingUUID.value = "";
  }
};

const refreshSyncTasks = async () => {
  syncLoading.value = true;
  try {
    const resp = await leadCaptureService.listWeComSyncTasks({
      limit: 100,
    });
    syncTasks.value = ((resp as any)?.data?.items || []) as WeComSyncTaskRecord[];
    if (syncTaskPage.value > syncTaskTotalPages.value) {
      syncTaskPage.value = syncTaskTotalPages.value;
    }
    const activeTaskUUID = (syncActivePollingTaskUUID.value || "").trim();
    if (activeTaskUUID) {
      const activeTask = syncTasks.value.find((item) => (item?.task_uuid || "").trim() === activeTaskUUID);
      if (activeTask && (activeTask.status === "success" || activeTask.status === "failed")) {
        const signature = `${activeTask.task_uuid}:${activeTask.finished_at || activeTask.status}`;
        if (activeTask.status === "success" && signature !== lastLeadAutoRefreshTaskSignature.value) {
          lastLeadAutoRefreshTaskSignature.value = signature;
          await refreshLeads();
        }
        if (signature !== lastActiveTaskToastSignature.value) {
          lastActiveTaskToastSignature.value = signature;
          showToast(
            activeTask.status === "success" ? "同步任务已完成" : `同步任务失败：${activeTask.error_message || "请查看任务详情"}`,
            activeTask.status === "success" ? "success" : "error"
          );
        }
        syncTaskPanelOpen.value = true;
        syncTaskPage.value = 1;
        syncActivePollingTaskUUID.value = "";
      } else if (!activeTask) {
        syncActivePollingTaskUUID.value = "";
      }
    } else {
      const latestFinishedSuccess = syncTasks.value
        .filter((item) => item?.status === "success" && !!item?.finished_at)
        .sort((a, b) => String(b.finished_at || "").localeCompare(String(a.finished_at || "")))[0];
      if (latestFinishedSuccess) {
        const signature = `${latestFinishedSuccess.task_uuid}:${latestFinishedSuccess.finished_at}`;
        if (signature !== lastLeadAutoRefreshTaskSignature.value) {
          lastLeadAutoRefreshTaskSignature.value = signature;
          await refreshLeads();
        }
      }
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
    const payload: Record<string, any> = {};
    payload.action = syncTriggerAction.value;
    payload.trace_id = `lead-sync-${Date.now()}`;
    const accountUUID = String(defaultSyncAccount.value?.account_uuid || "").trim();
    if (accountUUID) {
      payload.channel_account_uuid = accountUUID;
    }
    const appType = String(defaultSyncAccount.value?.app_type || "").trim().toLowerCase();
    if (appType === "wecom" || appType === "openwork") {
      payload.app_type = appType;
    }
    const resp = await leadCaptureService.triggerWeComSync(payload);
    const task = (resp as any)?.data as WeComSyncTaskRecord | undefined;
    if (task?.account_resolve_source) {
      syncLastResolveSource.value = task.account_resolve_source;
    }
    syncActivePollingTaskUUID.value = (task?.task_uuid || "").trim();
    syncTaskPanelOpen.value = true;
    syncTaskPage.value = 1;
    showToast("已触发渠道同步任务", "success");
    await refreshSyncTasks();
  } catch (err: any) {
    showToast(err?.message || "触发同步失败", "error");
    syncTaskPanelOpen.value = true;
    syncTaskPage.value = 1;
    await refreshSyncTasks();
  } finally {
    syncSubmitting.value = false;
  }
};

const clearSyncTasks = async () => {
  if (syncTaskClearing.value) return;
  if (!process.client) return;
  if (!window.confirm("确认清空当前租户的同步任务记录？此操作不可撤销。")) return;
  syncTaskClearing.value = true;
  try {
    const resp = await leadCaptureService.clearWeComSyncTasks();
    const deleted = Number((resp as any)?.data?.deleted || 0);
    syncTasks.value = [];
    syncTaskPage.value = 1;
    syncActivePollingTaskUUID.value = "";
    lastLeadAutoRefreshTaskSignature.value = "";
    lastActiveTaskToastSignature.value = "";
    showToast(`已清空任务 ${deleted} 条`, "success");
    await refreshSyncTasks();
  } catch (err: any) {
    showToast(err?.message || "清空任务失败", "error");
  } finally {
    syncTaskClearing.value = false;
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
  const [accountResult, memberResult, catalogResult] = await Promise.allSettled([
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

  channelAccounts.value =
    accountResult.status === "fulfilled"
      ? (((accountResult.value as any)?.data?.items || []) as ChannelAccount[])
      : [];
  iamMembers.value =
    memberResult.status === "fulfilled"
      ? (((memberResult.value as any)?.data?.items || []) as MemberRecord[])
      : [];
  sourceCatalogs.value =
    catalogResult.status === "fulfilled"
      ? (((catalogResult.value as any)?.data?.items || []) as RuntimeDictionaryItem[])
      : [];
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

const refreshChannelCodePanel = async () => {
  await channelCodeStore.fetchChannelCodes();
  if (!selectedChannelCodeUUID.value && channelCodeStore.items.length > 0) {
    selectedChannelCodeUUID.value = channelCodeStore.items[0].code_uuid;
  }
  if (selectedChannelCodeUUID.value) {
    await channelCodeStore.fetchChangeLogs(selectedChannelCodeUUID.value);
    await loadSelectedChannelCodeWelcomeSyncStatus();
    await loadSelectedChannelCodeEvents();
  }
};

const loadSelectedChannelCodeWelcomeSyncStatus = async () => {
  if (!selectedChannelCodeUUID.value) {
    channelCodeWelcomeSyncStatus.value = null;
    return;
  }
  try {
    const resp = await leadCaptureService.getChannelCodeWelcomeSyncStatus(selectedChannelCodeUUID.value);
    channelCodeWelcomeSyncStatus.value = ((resp as any)?.data || null) as ChannelCodeWelcomeSyncStatus | null;
  } catch {
    channelCodeWelcomeSyncStatus.value = null;
  }
};

const loadSelectedChannelCodeEvents = async () => {
  if (!selectedChannelCodeUUID.value) {
    channelCodeEvents.value = [];
    channelCodeEventStats.value = { touch_total: 0, intake_total: 0, dedup_total: 0 };
    return;
  }
  channelCodeEventsLoading.value = true;
  try {
    const resp = await leadCaptureService.listChannelCodeEvents(selectedChannelCodeUUID.value, 50);
    channelCodeEvents.value = ((resp as any)?.data?.events || []) as ChannelCodeEventRecord[];
    channelCodeEventStats.value = ((resp as any)?.data?.stats || {
      touch_total: 0,
      intake_total: 0,
      dedup_total: 0,
    }) as ChannelCodeEventStats;
  } catch (err: any) {
    channelCodeEvents.value = [];
    channelCodeEventStats.value = { touch_total: 0, intake_total: 0, dedup_total: 0 };
    showToast(err?.message || "加载渠道码事件失败", "error");
  } finally {
    channelCodeEventsLoading.value = false;
  }
};

const submitChannelCodeCreate = async () => {
  try {
    const created = await channelCodeStore.createChannelCode({ ...channelCodeCreateForm });
    if (created?.code_uuid) {
      selectedChannelCodeUUID.value = created.code_uuid;
      channelCodeCreateOpen.value = false;
      await channelCodeStore.fetchChangeLogs(created.code_uuid);
    }
    showToast("渠道码已创建", "success");
  } catch (err: any) {
    showToast(err?.message || "创建渠道码失败", "error");
  }
};

const setSelectedChannelCodeStatus = async (status: "active" | "disabled") => {
  if (!selectedChannelCodeUUID.value) {
    showToast("请先选择渠道码", "warning");
    return;
  }
  try {
    await channelCodeStore.updateChannelCodeStatus(selectedChannelCodeUUID.value, status);
    showToast(status === "active" ? "渠道码已启用" : "渠道码已停用", "success");
  } catch (err: any) {
    showToast(err?.message || "更新渠道码状态失败", "error");
  }
};

const saveSelectedChannelCodeWelcomeConfig = async () => {
  if (!selectedChannelCodeUUID.value) {
    showToast("请先选择渠道码", "warning");
    return;
  }
  let parsed: Record<string, any> = {};
  try {
    parsed = JSON.parse(channelCodeWelcomeContentText.value || "{}");
  } catch (_err) {
    showToast("欢迎语内容必须是合法 JSON", "warning");
    return;
  }
  try {
    await channelCodeStore.saveWelcomeConfig(selectedChannelCodeUUID.value, {
      welcome_enabled: channelCodeWelcomeEnabled.value,
      message_content: parsed,
    });
    showToast("欢迎语已保存（待发布）", "success");
    await loadSelectedChannelCodeWelcomeSyncStatus();
  } catch (err: any) {
    showToast(err?.message || "保存欢迎语失败", "error");
  }
};

const triggerSelectedChannelCodeWelcomeSync = async () => {
  if (!selectedChannelCodeUUID.value) {
    showToast("请先选择渠道码", "warning");
    return;
  }
  channelCodeWelcomeSyncLoading.value = true;
  try {
    const resp = await leadCaptureService.triggerChannelCodeWelcomeSync(selectedChannelCodeUUID.value);
    const result = (resp as any)?.data || {};
    await loadSelectedChannelCodeWelcomeSyncStatus();
    if (result?.sync_status === "success") {
      showToast("欢迎语已同步成功", "success");
      return;
    }
    if (result?.sync_status === "manual_required") {
      showToast(`同步失败：${result?.error_code || "请人工重试"}`, "warning");
      return;
    }
    showToast("欢迎语同步任务已触发", "success");
  } catch (err: any) {
    showToast(err?.message || "发布欢迎语失败", "error");
  } finally {
    channelCodeWelcomeSyncLoading.value = false;
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

watch([syncTaskPageSize], () => {
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

watch(selectedChannelCodeUUID, async (value) => {
  channelCodeStore.setSelectedCode(value || "");
  if (!value) {
    channelCodeWelcomeSyncStatus.value = null;
    channelCodeEvents.value = [];
    channelCodeEventStats.value = { touch_total: 0, intake_total: 0, dedup_total: 0 };
    return;
  }
  await channelCodeStore.fetchChangeLogs(value);
  await loadSelectedChannelCodeWelcomeSyncStatus();
  await loadSelectedChannelCodeEvents();
});

watch(syncTriggerAction, () => {
  clearSelectedLeadsForPush();
});

watch([pushLeadPassedRows, pushLeadRejectedRows], () => {
  if (pushLeadPassedPage.value > pushLeadPassedTotalPages.value) {
    pushLeadPassedPage.value = pushLeadPassedTotalPages.value;
  }
  if (pushLeadRejectedPage.value > pushLeadRejectedTotalPages.value) {
    pushLeadRejectedPage.value = pushLeadRejectedTotalPages.value;
  }
});

watch([selectedLeadUUIDsForPush, statusFilter, channelFilter, appTypeFilter, searchText], () => {
  pushLeadPassedPage.value = 1;
  pushLeadRejectedPage.value = 1;
});

watch(syncCenterOpen, (open) => {
  if (open) return;
  syncActivePollingTaskUUID.value = "";
});

let wsUnsubscribe: (() => void) | null = null;
let wsRefreshTimer: ReturnType<typeof setTimeout> | null = null;
let wsRefreshInFlight = false;
let wsRefreshQueued = false;
const wsTopics = ["lead_sync.progress", "powerx.lead_sync.progress.v1"];

const scheduleRefreshSyncTasksByWs = () => {
  if (wsRefreshInFlight) {
    wsRefreshQueued = true;
    return;
  }
  if (wsRefreshTimer) {
    clearTimeout(wsRefreshTimer);
    wsRefreshTimer = null;
  }
  wsRefreshTimer = setTimeout(() => {
    wsRefreshTimer = null;
    wsRefreshInFlight = true;
    void refreshSyncTasks().finally(() => {
      wsRefreshInFlight = false;
      if (wsRefreshQueued) {
        wsRefreshQueued = false;
        scheduleRefreshSyncTasksByWs();
      }
    });
  }, 280);
};

const handleLeadSyncWsProgress = (payload: any) => {
  if (!payload || !syncCenterOpen.value) return;
  const taskUUID = String(payload.task_uuid || "").trim();
  if (!taskUUID) return;
  const activeTaskUUID = String(syncActivePollingTaskUUID.value || "").trim();
  if (activeTaskUUID && taskUUID !== activeTaskUUID) {
    return;
  }
  scheduleRefreshSyncTasksByWs();
};

const ensureWsSubscription = () => {
  if (wsUnsubscribe) return;
  const unsubscribers = wsTopics.map((topic) => wsBus.client.subscribe(topic, handleLeadSyncWsProgress));
  wsUnsubscribe = () => {
    unsubscribers.forEach((unsub) => unsub());
  };
};

onMounted(async () => {
  ensureWsSubscription();
  await loadCreateLookupOptions();
  await refreshLeads();
  await refreshSyncTasks();
  await loadWritebackPolicy();
  await refreshWritebackDeadLetters();
  await loadWeComCustomerDMRule();
  await refreshChannelCodePanel();
});

onActivated(() => {
  // 从详情返回时刷新，避免负责人刚改完列表仍显示旧状态
  void refreshLeads();
  void refreshSyncTasks();
});

onBeforeUnmount(() => {
  syncActivePollingTaskUUID.value = "";
  if (wsRefreshTimer) {
    clearTimeout(wsRefreshTimer);
    wsRefreshTimer = null;
  }
  wsRefreshInFlight = false;
  wsRefreshQueued = false;
  if (wsUnsubscribe) {
    wsUnsubscribe();
    wsUnsubscribe = null;
  }
});
</script>
