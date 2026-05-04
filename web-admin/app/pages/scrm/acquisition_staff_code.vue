<template>
  <UContainer class="acq-staff-page max-w-none py-10 space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="space-y-2">
        <h1 class="acq-page-title text-2xl font-semibold">员工活码</h1>
        <p class="acq-page-subtitle">管理员工活码、成员绑定与欢迎语配置。</p>
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
            placeholder="搜索活动名称"
            class="w-56"
          />
        </div>
      </div>

      <div class="overflow-x-auto">
        <table class="min-w-full">
          <thead>
            <tr class="border-b border-white/10 text-left text-sm text-slate-300">
              <th class="acq-th px-6 py-3 font-medium">活动</th>
              <th class="acq-th px-6 py-3 font-medium">二维码</th>
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
              <td class="acq-td px-6 py-3">
                <div v-if="item.qr_code" class="flex items-center gap-2">
                  <UButton size="xs" variant="soft" @click="previewQRCode(item.qr_code)">查看二维码</UButton>
                  <UButton size="xs" variant="soft" @click="copyQRCode(item.qr_code)">复制链接</UButton>
                  <UButton size="xs" variant="soft" @click="downloadQRCode(item.qr_code, item.config_id)">下载</UButton>
                </div>
                <span v-else class="text-xs text-slate-400">未生成</span>
              </td>
              <td class="acq-td px-6 py-3">{{ item.channel }}</td>
              <td class="acq-td px-6 py-3">{{ item.app_type }}</td>
              <td class="acq-td px-6 py-3">
                <UBadge :color="statusColor(item.status)" variant="soft"><span class="normal-case">{{ item.status }}</span></UBadge>
              </td>
              <td class="acq-td px-6 py-3">
                <div class="flex items-center gap-2">
                  <UButton size="xs" variant="soft" color="primary" @click="openEditPanel(item)">编辑</UButton>
                  <UButton size="xs" variant="soft" @click="setStatus(item.staff_code_uuid, 'active')">启用</UButton>
                  <UButton size="xs" variant="soft" color="error" @click="deleteStaffCode(item)">删除</UButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <UModal
      v-model:open="qrPreviewOpen"
      title="员工活码二维码"
      :ui="{ content: 'max-w-md' }"
    >
      <template #body>
        <div class="space-y-3">
          <img
            v-if="qrPreviewImageURL && !qrPreviewImageError"
            :src="qrPreviewImageURL"
            alt="员工活码二维码"
            class="w-full rounded border border-white/15"
            @error="qrPreviewImageError = true"
          />
          <div v-if="qrPreviewImageError" class="rounded border border-amber-300/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-200">
            图片预览失败，请使用“复制链接”或“下载”打开原图。
          </div>
          <div class="text-xs text-slate-300 break-all">{{ qrPreviewURL }}</div>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" @click="qrPreviewOpen = false">关闭</UButton>
          <UButton color="primary" variant="soft" @click="copyQRCode(qrPreviewURL)">复制链接</UButton>
          <UButton color="primary" @click="downloadQRCode(qrPreviewURL)">下载</UButton>
        </div>
      </template>
    </UModal>

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
                <div class="text-lg font-semibold text-slate-100">员工活码设置</div>
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
                            <UButton color="primary" variant="outline" icon="i-heroicons-plus" @click="openMemberPicker">选择成员</UButton>
                            <span class="text-sm text-slate-200">已选择 <span class="text-rose-300">{{ selectedMemberCount }}</span> 人</span>
                          </div>
                          <div class="mt-3 rounded-xl border border-white/10 bg-[#0b1730] p-3">
                            <div v-if="selectedMembers.length === 0" class="text-sm text-slate-400">请选择已完成 org_sync 映射的成员</div>
                            <div v-else class="flex flex-wrap gap-2">
                              <UBadge
                                v-for="item in selectedMembers"
                                :key="item.uuid"
                                color="primary"
                                variant="soft"
                                class="cursor-pointer"
                                @click="removeSelectedMember(item.uuid)"
                              >
                                <span class="normal-case">{{ item.display_name }} · {{ item.username || item.email || item.uuid }}</span>
                              </UBadge>
                            </div>
                          </div>
                        </div>
                      </div>

                      <div class="acq-setting-row">
                        <div class="acq-setting-label">企业微信标签</div>
                        <div class="acq-setting-content">
                          <div class="acq-tag-tip">可在标签管理页面新建标签及查看标签组</div>
                          <div class="acq-tag-panel mt-3">
                            <div class="text-sm text-slate-300">为本次活动的新粉丝打标签</div>
                            <div class="mt-3 flex flex-wrap items-center gap-3">
                              <UButton color="primary" variant="outline" icon="i-heroicons-plus" @click="openTagPicker">选择标签</UButton>
                              <div class="min-w-[220px] flex-1 rounded-lg border border-white/10 bg-[#0b1730] p-2">
                                <div v-if="selectedTags.length === 0" class="text-sm text-slate-400">未选择标签</div>
                                <div v-else class="flex flex-wrap gap-2">
                                  <UBadge
                                    v-for="item in selectedTags"
                                    :key="item.id"
                                    color="neutral"
                                    variant="soft"
                                    class="cursor-pointer"
                                    @click="removeSelectedTag(item.id)"
                                  >
                                    <span class="normal-case">{{ item.group_name ? `${item.group_name}: ` : "" }}{{ resolveTagLabel(item.id, item.label) }}</span>
                                  </UBadge>
                                </div>
                              </div>
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
                              {{ form.new_customer_remark_enabled ? "开启后可为客户昵称加备注，便于查看客户来源" : "关闭" }}
                            </span>
                          </div>
                          <div v-if="form.new_customer_remark_enabled" class="mt-3 rounded-xl border border-white/10 bg-[#0f203f] p-4">
                            <div class="mb-3 flex items-center gap-6">
                              <span class="text-sm font-medium text-slate-200">备注位置</span>
                              <URadioGroup
                                v-model="form.customer_remark_position"
                                :items="customerRemarkPositionOptions"
                                value-key="value"
                                label-key="label"
                                orientation="horizontal"
                              />
                            </div>
                            <div class="mb-3 rounded-lg border border-white/10 bg-[#0b1730] p-2">
                              <div class="flex items-center overflow-hidden rounded-md border border-white/10">
                                <UInput
                                  v-model="form.customer_remark_prefix"
                                  class="flex-1"
                                  :ui="{ root: 'w-full' }"
                                  placeholder="请输入备注前缀，如：亲爱的"
                                />
                                <div class="shrink-0 border-l border-white/10 px-3 py-2 text-sm text-slate-400">——客户昵称</div>
                              </div>
                            </div>
                            <div class="rounded-lg border border-white/10 bg-[#12274c] px-3 py-3 text-sm text-slate-200">
                              {{ customerRemarkPreviewText }}
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class="rounded-2xl border border-white/10 bg-[#13264a] p-6 shadow-sm">
                    <div class="mb-4 text-base font-semibold text-slate-100">回复设置</div>
                    <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
                      <UFormField label="欢迎语设置" class="lg:col-span-2">
                        <URadioGroup
                          v-model="form.welcome_mode"
                          :items="welcomeModeOptions"
                          value-key="value"
                          label-key="label"
                        />
                      </UFormField>
                      <UFormField label="回复内容" class="lg:col-span-2">
                        <div class="rounded-xl border border-white/10 bg-[#0f203f] p-3">
                          <div class="mb-3 space-y-3">
                            <div
                              v-for="(block, idx) in replyBlocks"
                              :key="`editor-${block.type}-${idx}`"
                              class="rounded-lg border border-white/10 bg-[#0b1730] p-3"
                            >
                              <div class="mb-2 flex items-center justify-end">
                                <UButton size="xs" color="neutral" variant="ghost" @click="removeReplyBlock(idx)">
                                  删除
                                </UButton>
                              </div>
                              <div v-if="block.type === 'text'">
                                <div class="acq-text-editor-head">
                                  <button type="button" class="acq-tool-btn" title="插入表情" @click.stop="toggleEmojiPicker(idx)">
                                    <UIcon name="i-heroicons-face-smile" />
                                  </button>
                                  <button type="button" class="acq-tool-btn border-l border-white/10" title="插入粉丝昵称" @click="appendToTextBlock(idx, '{粉丝昵称}')">
                                    <UIcon name="i-heroicons-user" />
                                  </button>
                                </div>
                                <div v-if="emojiPickerIndex === idx" class="acq-emoji-panel mt-2">
                                  <button
                                    v-for="emoji in emojiPalette"
                                    :key="emoji"
                                    type="button"
                                    class="acq-emoji-item"
                                    @click="pickEmoji(idx, emoji)"
                                  >
                                    {{ emoji }}
                                  </button>
                                </div>
                                <UTextarea v-model="block.content" class="mt-3 w-full" :rows="4" placeholder="请输入欢迎语内容" />
                              </div>
                              <div v-else-if="block.type === 'image'" class="grid grid-cols-1 gap-2 lg:grid-cols-2">
                                <UInput v-model="block.media_id" placeholder="media_id（可选）" />
                                <UInput v-model="block.url" placeholder="图片 URL" />
                              </div>
                              <div v-else-if="block.type === 'link'" class="grid grid-cols-1 gap-2">
                                <UInput v-model="block.title" placeholder="链接标题" />
                                <UInput v-model="block.url" placeholder="链接 URL" />
                                <UInput v-model="block.desc" placeholder="链接描述（可选）" />
                              </div>
                              <div v-else-if="block.type === 'mini_program'" class="grid grid-cols-1 gap-2">
                                <UInput v-model="block.title" placeholder="小程序标题" />
                                <UInput v-model="block.appid" placeholder="小程序 AppID" />
                                <UInput v-model="block.page" placeholder="页面路径，如 pages/index/index" />
                              </div>
                            </div>
                          </div>
                          <div class="flex flex-wrap items-center gap-2">
                            <UPopover v-model:open="replyTypeMenuOpen" :ui="{ content: 'z-[1200]' }">
                              <UButton
                                id="reply-type-anchor"
                                color="primary"
                                variant="outline"
                                icon="i-heroicons-plus"
                                @click.stop="replyTypeMenuOpen = true"
                              >
                                添加回复内容
                              </UButton>
                              <template #content>
                                <div class="acq-reply-menu w-44 p-1">
                                  <button
                                    v-for="item in replyTypeOptions"
                                    :key="item.value"
                                    type="button"
                                    class="acq-reply-menu-item w-full rounded-md px-3 py-2 text-left text-sm"
                                    @click="appendReplyBlock(item.value)"
                                  >
                                    {{ item.label }}
                                  </button>
                                </div>
                              </template>
                            </UPopover>
                            <UButton color="neutral" variant="outline">选择雷达</UButton>
                          </div>
                        </div>
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
                        <div class="acq-chat-msg acq-chat-msg-left">
                          您好，我是企业助手，已为您分配专属顾问。
                        </div>
                        <div class="acq-chat-msg acq-chat-msg-right">
                          我刚扫码进来，想咨询下服务方案。
                        </div>
                        <div
                          v-if="form.welcome_mode === 'send'"
                          class="acq-chat-msg acq-chat-msg-left"
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

            <div class="border-t border-white/10 px-10 py-3">
              <div class="flex flex-wrap items-center gap-3 text-xs text-slate-300">
                <span class="rounded-md border border-white/10 bg-[#0f203f] px-2 py-1">
                  草稿UUID：{{ editingStaffCodeUUID || "未创建" }}
                </span>
                <span class="rounded-md border border-white/10 bg-[#0f203f] px-2 py-1">
                  同步状态：{{ welcomeSyncStatus?.sync_status || "pending" }}
                </span>
                <span v-if="welcomeSyncStatus?.last_synced_at" class="rounded-md border border-white/10 bg-[#0f203f] px-2 py-1">
                  最近同步：{{ welcomeSyncStatus?.last_synced_at }}
                </span>
                <span v-if="welcomeSyncStatus?.last_sync_error" class="rounded-md border border-rose-300/30 bg-rose-500/10 px-2 py-1 text-rose-200">
                  错误：{{ welcomeSyncStatus?.last_sync_error }}
                </span>
              </div>
            </div>

            <div class="flex items-center justify-end gap-2 border-t border-white/10 px-10 py-5">
              <UButton variant="ghost" @click="closeCreatePanel">取消</UButton>
              <UButton color="primary" :loading="submitting" @click="submitStaffConfig">提交</UButton>
            </div>
          </div>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="memberPickerOpen" :ui="{ content: 'max-w-3xl' }">
      <template #header>
        <div class="text-base font-semibold">选择企业成员（仅 org_sync 已绑定）</div>
      </template>
      <template #body>
        <div class="space-y-3">
          <UInput
            v-model="memberKeyword"
            icon="i-heroicons-magnifying-glass"
            placeholder="搜索姓名/用户名/邮箱"
            @update:model-value="debouncedLoadMembers"
          />
          <div class="max-h-[420px] overflow-auto rounded-lg border border-white/10">
            <div v-if="memberLoading" class="p-4 text-sm text-slate-400">加载中...</div>
            <div
              v-for="item in memberCandidates"
              v-else
              :key="resolveMemberUUID(item) || String(item.id)"
              class="flex cursor-pointer items-center justify-between border-b border-white/5 px-3 py-2 last:border-b-0 hover:bg-white/5"
              role="button"
              tabindex="0"
              @click="setMemberChecked(item, !memberCheckedSet.has(resolveMemberUUID(item)))"
              @keydown.enter.prevent="setMemberChecked(item, !memberCheckedSet.has(resolveMemberUUID(item)))"
              @keydown.space.prevent="setMemberChecked(item, !memberCheckedSet.has(resolveMemberUUID(item)))"
            >
              <div class="min-w-0">
                <div class="truncate text-sm text-slate-100">{{ item.display_name }}</div>
                <div class="truncate text-xs text-slate-400">{{ item.username || item.email || resolveMemberUUID(item) }}</div>
              </div>
              <UCheckbox
                :model-value="memberCheckedSet.has(resolveMemberUUID(item))"
                @update:model-value="(checked) => setMemberChecked(item, Boolean(checked))"
                @click.stop
              />
            </div>
            <div v-if="!memberLoading && memberCandidates.length === 0" class="p-4 text-sm text-slate-400">暂无可选成员</div>
          </div>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full items-center justify-between">
          <div class="text-sm text-slate-400">已选择 {{ selectedMembers.length }} 人</div>
          <div class="flex items-center gap-2">
            <UButton variant="ghost" @click="memberPickerOpen = false">关闭</UButton>
            <UButton color="primary" @click="memberPickerOpen = false">确定</UButton>
          </div>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="tagPickerOpen" :ui="{ content: 'max-w-3xl' }">
      <template #header>
        <div class="text-base font-semibold">选择企业微信标签</div>
      </template>
      <template #body>
        <div class="space-y-3">
          <UInput
            v-model="tagKeyword"
            icon="i-heroicons-magnifying-glass"
            placeholder="搜索标签组/标签名"
            @update:model-value="loadTagCandidates"
          />
          <div class="max-h-[420px] overflow-auto rounded-lg border border-white/10">
            <div v-if="tagLoading" class="p-4 text-sm text-slate-400">加载中...</div>
            <div
              v-for="item in tagCandidates"
              v-else
              :key="item.remote_tag_id"
              class="flex cursor-pointer items-center justify-between border-b border-white/5 px-3 py-2 last:border-b-0 hover:bg-white/5"
              role="button"
              tabindex="0"
              @click="setTagChecked(item, !tagCheckedNormalizedSet.has(normalizeRemoteID(String(item.remote_tag_id || ''))))"
              @keydown.enter.prevent="setTagChecked(item, !tagCheckedNormalizedSet.has(normalizeRemoteID(String(item.remote_tag_id || ''))))"
              @keydown.space.prevent="setTagChecked(item, !tagCheckedNormalizedSet.has(normalizeRemoteID(String(item.remote_tag_id || ''))))"
            >
              <div class="min-w-0">
                <div class="truncate text-sm text-slate-100">{{ item.tag_name }}</div>
                <div class="truncate text-xs text-slate-400">{{ item.remote_group_name || "未分组" }}</div>
              </div>
              <UCheckbox
                :model-value="tagCheckedNormalizedSet.has(normalizeRemoteID(String(item.remote_tag_id || '')))"
                @update:model-value="(checked) => setTagChecked(item, Boolean(checked))"
                @click.stop
              />
            </div>
            <div v-if="!tagLoading && tagCandidates.length === 0" class="p-4 text-sm text-slate-400">暂无可选标签</div>
          </div>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full items-center justify-between">
          <div class="text-sm text-slate-400">已选择 {{ selectedTags.length }} 个标签</div>
          <div class="flex items-center gap-2">
            <UButton variant="ghost" @click="tagPickerOpen = false">关闭</UButton>
            <UButton color="primary" @click="tagPickerOpen = false">确定</UButton>
          </div>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { useAcquisitionService, type StaffLiveCodeRecord } from "~/composables/api/services/acquisition";
import { useMemberService, type Member } from "~/composables/api/services/memberService";
import { useSocialChannelGovernanceService, type FoundationTagRecord } from "~/composables/api/services/socialChannelGovernance";

const toast = useToast();
const service = useAcquisitionService();
const memberService = useMemberService();
const socialService = useSocialChannelGovernanceService();
const loading = ref(false);
const createOpen = ref(false);
const memberPickerOpen = ref(false);
const tagPickerOpen = ref(false);
const memberLoading = ref(false);
const tagLoading = ref(false);
const staffCodes = ref<StaffLiveCodeRecord[]>([]);
const keyword = ref("");
const memberKeyword = ref("");
const tagKeyword = ref("");
const memberCandidates = ref<Member[]>([]);
const tagCandidates = ref<FoundationTagRecord[]>([]);
const selectedMembers = ref<Array<{ uuid: string; display_name: string; username?: string; email?: string }>>([]);
const selectedTags = ref<Array<{ id: string; label: string; group_name?: string }>>([]);
const defaultChannelAccountUUID = ref("");
const formChannelAccountUUID = ref("");
const editingStaffCodeUUID = ref("");
const welcomeSyncStatus = ref<{ sync_status: string; last_sync_error?: string; last_synced_at?: string; latest_attempt_no?: number } | null>(null);
const submitting = ref(false);
const qrPreviewOpen = ref(false);
const qrPreviewURL = ref("");
const qrPreviewImageError = ref(false);
const normalizeQRCodeURL = (value: string) => String(value || "").replace(/\s+/g, "").trim();
const qrPreviewImageURL = computed(() => normalizeQRCodeURL(qrPreviewURL.value));
const replyTypeMenuOpen = ref(false);
const emojiPickerIndex = ref(-1);
type ReplyBlock =
  | { type: "text"; content: string }
  | { type: "image"; media_id: string; url: string }
  | { type: "link"; title: string; url: string; desc: string }
  | { type: "mini_program"; appid: string; page: string; title: string };

const createReplyBlock = (type: "text" | "image" | "link" | "mini_program"): ReplyBlock => {
  if (type === "text") return { type, content: "欢迎添加，我们将尽快联系你" };
  if (type === "image") return { type, media_id: "", url: "" };
  if (type === "link") return { type, title: "链接标题", url: "", desc: "" };
  return { type, appid: "", page: "", title: "" };
};

const replyBlocks = ref<ReplyBlock[]>([createReplyBlock("text")]);
const emojiPalette = ["😀", "😁", "😂", "🤣", "😊", "😍", "😘", "😎", "😏", "😇", "🤝", "👏", "👍", "🙏", "🎉", "❤️"];

const form = reactive({
  activity_name: "",
  member_uuids: [] as string[],
  corp_tag_ids_text: "",
  new_customer_remark_enabled: false,
  customer_remark_position: "prefix" as "prefix" | "suffix",
  customer_remark_prefix: "亲爱的",
  welcome_mode: "send" as "send" | "silent",
  welcome_text: "欢迎添加，我们将尽快联系你",
});

const welcomeModeOptions = [
  { label: "员工欢迎语", value: "send" },
  { label: "不发欢迎语", value: "silent" },
];

const customerRemarkPositionOptions = [
  { label: "备注在昵称前", value: "prefix" },
  { label: "备注在昵称后", value: "suffix" },
];

const selectedMemberCount = computed(() => form.member_uuids.length);
const memberCheckedSet = computed(() => new Set(form.member_uuids));
const normalizeRemoteID = (value: string) => String(value || "").trim().toLowerCase();

const resolveMemberUUID = (item: Partial<Member> & Record<string, any>): string => {
  return String(item?.id || "").trim();
};

const previewTagText = computed(() => {
  const tags = selectedTags.value.map((item) => resolveTagLabel(item.id, item.label));
  return tags.length > 0 ? tags.join(" / ") : "未设置";
});

const resolveTagLabel = (id: string, fallback?: string) => {
  const key = String(id || "").trim();
  if (!key) return String(fallback || "").trim();
  const normalizedKey = normalizeRemoteID(key);
  const hit = tagCandidates.value.find((item) => normalizeRemoteID(String(item.remote_tag_id || "")) === normalizedKey);
  const resolved = String(hit?.tag_name || "").trim();
  if (resolved) return resolved;
  const cleanFallback = String(fallback || "").trim();
  if (cleanFallback && cleanFallback !== key) return cleanFallback;
  return `标签已失效(${key})`;
};

const hydrateSelectedTagLabels = async () => {
  if (selectedTags.value.length === 0) return;
  if (tagCandidates.value.length === 0) {
    await loadTagCandidates();
  }
  selectedTags.value = selectedTags.value.map((item) => {
    const normalizedID = normalizeRemoteID(String(item.id || ""));
    const hit = tagCandidates.value.find((row) => normalizeRemoteID(String(row.remote_tag_id || "")) === normalizedID);
    if (!hit) return item;
    return {
      ...item,
      id: String(hit.remote_tag_id || item.id || "").trim(),
      label: String(hit.tag_name || item.label || item.id),
      group_name: String(hit.remote_group_name || item.group_name || ""),
    };
  });
};

const customerRemarkPreviewText = computed(() => {
  const prefix = (form.customer_remark_prefix || "").trim() || "亲爱的";
  return form.customer_remark_position === "suffix"
    ? `客户昵称 —— ${prefix}`
    : `${prefix} —— 客户昵称`;
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

const previewQRCode = (url: string) => {
  const link = normalizeQRCodeURL(url);
  if (!link) return;
  qrPreviewURL.value = link;
  qrPreviewImageError.value = false;
  qrPreviewOpen.value = true;
};

const copyQRCode = async (url: string) => {
  const link = normalizeQRCodeURL(url);
  if (!link) return;
  try {
    await navigator.clipboard.writeText(link);
    toast.add({ title: "二维码链接已复制", color: "success" });
  } catch {
    toast.add({ title: "复制失败", color: "error" });
  }
};

const downloadQRCode = async (url: string, configID?: string) => {
  const link = normalizeQRCodeURL(url);
  if (!link) return;
  const filename = `${String(configID || "staff-code-qr").trim() || "staff-code-qr"}.png`;
  try {
    const resp = await fetch(link, { mode: "cors" });
    if (!resp.ok) throw new Error(`download failed: ${resp.status}`);
    const blob = await resp.blob();
    const blobURL = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = blobURL;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(blobURL);
    toast.add({ title: "二维码已下载", color: "success" });
  } catch {
    window.open(link, "_blank", "noopener,noreferrer");
    toast.add({ title: "已打开二维码链接，请在新页面手动保存", color: "warning" });
  }
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

const openEditPanel = async (item: StaffLiveCodeRecord) => {
  resetCreateForm();
  editingStaffCodeUUID.value = String(item.staff_code_uuid || "").trim();
  formChannelAccountUUID.value = String(item.channel_account_uuid || defaultChannelAccountUUID.value || "").trim();
  form.activity_name = String(item.activity_name || "");
  form.member_uuids = Array.isArray(item.member_uuids) ? item.member_uuids.map((v) => String(v).trim()).filter(Boolean) : [];
  form.new_customer_remark_enabled = Boolean(item.new_customer_remark_enabled);

  const staffTagIDs = Array.isArray(item.corp_tag_ids) ? item.corp_tag_ids.map((v) => String(v).trim()).filter(Boolean) : [];
  if (staffTagIDs.length > 0 && formChannelAccountUUID.value) {
    try {
      const resp = await socialService.listFoundationTags({
        channel_account_uuid: formChannelAccountUUID.value,
        limit: 500,
      });
      const items = ((resp as any)?.data?.items || []) as FoundationTagRecord[];
      const byID = new Map(items.map((t) => [normalizeRemoteID(String(t.remote_tag_id || "")), t]));
      selectedTags.value = staffTagIDs.map((id) => {
        const hit = byID.get(normalizeRemoteID(id));
        return {
          id,
          label: hit?.tag_name || id,
          group_name: hit?.remote_group_name || "",
        };
      });
      form.corp_tag_ids_text = staffTagIDs.join(",");
    } catch {
      selectedTags.value = staffTagIDs.map((id) => ({ id, label: id }));
      form.corp_tag_ids_text = staffTagIDs.join(",");
    }
  }

  if (form.member_uuids.length > 0) {
    try {
      const members = await memberService.getMemberList({ org_sync_bound: true, status: "active", page: 1, page_size: 500 });
      const byID = new Map(members.map((m: any) => [String(m.id), m]));
      selectedMembers.value = form.member_uuids.map((id) => {
        const hit: any = byID.get(String(id));
        return {
          uuid: String(id),
          display_name: String(hit?.display_name || hit?.username || hit?.email || id),
          username: hit?.username,
          email: hit?.email,
        };
      });
    } catch {
      selectedMembers.value = form.member_uuids.map((id) => ({ uuid: String(id), display_name: String(id) }));
    }
  }

  try {
    const statusResp = await service.getStaffWelcomeSyncStatus(editingStaffCodeUUID.value);
    welcomeSyncStatus.value = (statusResp as any)?.data || (statusResp as any) || null;
  } catch {
    welcomeSyncStatus.value = null;
  }

  await hydrateSelectedTagLabels();

  createOpen.value = true;
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

const deleteStaffCode = async (item: StaffLiveCodeRecord) => {
  const staffCodeUUID = String(item?.staff_code_uuid || "").trim();
  if (!staffCodeUUID) return;
  const ok = window.confirm(`确认删除员工活码「${item.activity_name || staffCodeUUID}」？\n将同步删除企微远端联系我配置。`);
  if (!ok) return;
  try {
    await service.deleteStaffCode(staffCodeUUID);
    staffCodes.value = staffCodes.value.filter((row) => row.staff_code_uuid !== staffCodeUUID);
    toast.add({ title: "删除成功", color: "success" });
  } catch (error: any) {
    toast.add({ title: "删除失败", description: error?.message || "unknown error", color: "error" });
  }
};

const ensureDraftStaffCode = async () => {
  const memberUUIDs = form.member_uuids.map((item) => item.trim()).filter(Boolean);
  const corpTagIDs = selectedTags.value.map((item) => item.id);
  if (!form.activity_name || memberUUIDs.length === 0) {
    throw new Error("请填写完整信息");
  }
  if (editingStaffCodeUUID.value) {
    await service.updateStaffCode(editingStaffCodeUUID.value, {
      activity_name: form.activity_name,
      member_uuids: memberUUIDs,
      corp_tag_ids: corpTagIDs,
      new_customer_remark_enabled: form.new_customer_remark_enabled,
    });
    return editingStaffCodeUUID.value;
  }
  const created = await service.createStaffCode({
    channel: "wechat",
    app_type: "wecom",
    channel_account_uuid: defaultChannelAccountUUID.value || undefined,
    activity_name: form.activity_name,
    member_uuids: memberUUIDs,
    corp_tag_ids: corpTagIDs,
    new_customer_remark_enabled: form.new_customer_remark_enabled,
  });
  const record = (created as any)?.data || (created as any);
  editingStaffCodeUUID.value = String(record?.staff_code_uuid || "").trim();
  formChannelAccountUUID.value = String(record?.channel_account_uuid || defaultChannelAccountUUID.value || "").trim();
  if (!editingStaffCodeUUID.value) {
    throw new Error("创建员工活码失败");
  }
  return editingStaffCodeUUID.value;
};

const buildWelcomeContentBlocks = () => {
  const blocks: Array<Record<string, any>> = [];
  for (const block of replyBlocks.value) {
    if (block.type === "text") {
      blocks.push({ type: "text", content: (block.content || "").trim() || "欢迎添加，我们将尽快联系你" });
      continue;
    }
    if (block.type === "image") {
      blocks.push({ type: "image", media_id: block.media_id || "", url: block.url || "" });
      continue;
    }
    if (block.type === "link") {
      blocks.push({ type: "link", title: block.title || "链接标题", url: block.url || "", desc: block.desc || "" });
      continue;
    }
    if (block.type === "mini_program") {
      blocks.push({ type: "mini_program", appid: block.appid || "", page: block.page || "", title: block.title || "" });
    }
  }
  if (blocks.length === 0) {
    blocks.push({ type: "text", content: (form.welcome_text || "").trim() || "欢迎添加，我们将尽快联系你" });
  }
  return blocks;
};

const submitStaffConfig = async () => {
  if (submitting.value) return;
  submitting.value = true;
  try {
    const staffCodeUUID = await ensureDraftStaffCode();
    await service.saveStaffWelcome(staffCodeUUID, {
      welcome_mode: form.welcome_mode === "send" ? "send" : "silent",
      content_blocks: buildWelcomeContentBlocks(),
    });
    await service.triggerStaffWelcomeSync(staffCodeUUID);
    const statusResp = await service.getStaffWelcomeSyncStatus(staffCodeUUID);
    welcomeSyncStatus.value = (statusResp as any)?.data || (statusResp as any) || null;
    const syncStatus = String(welcomeSyncStatus.value?.sync_status || "").toLowerCase();
    if (syncStatus === "success") {
      toast.add({ title: "提交成功", color: "success" });
      await loadData();
      closeCreatePanel();
    } else {
      const syncError = String(welcomeSyncStatus.value?.last_sync_error || "").trim();
      toast.add({
        title: "提交失败",
        description: syncError || `同步状态异常: ${syncStatus || "unknown"}`,
        color: "error",
      });
      await loadData();
    }
  } catch (error: any) {
    toast.add({ title: "提交失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    submitting.value = false;
  }
};

const resetCreateForm = () => {
  Object.assign(form, {
    activity_name: "",
    member_uuids: [],
    corp_tag_ids_text: "",
    new_customer_remark_enabled: false,
    customer_remark_position: "prefix",
    customer_remark_prefix: "亲爱的",
    welcome_mode: "send",
    welcome_text: "欢迎添加，我们将尽快联系你",
  });
  selectedMembers.value = [];
  selectedTags.value = [];
  memberKeyword.value = "";
  tagKeyword.value = "";
  replyBlocks.value = [createReplyBlock("text")];
  replyTypeMenuOpen.value = false;
  editingStaffCodeUUID.value = "";
  welcomeSyncStatus.value = null;
  formChannelAccountUUID.value = "";
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

const loadMembers = async () => {
  memberLoading.value = true;
  try {
    memberCandidates.value = await memberService.getMemberList({
      keyword: memberKeyword.value.trim(),
      org_sync_bound: true,
      status: "active",
      page: 1,
      page_size: 200,
    });
  } catch (error: any) {
    toast.add({ title: "加载成员失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    memberLoading.value = false;
  }
};

let memberSearchTimer: ReturnType<typeof setTimeout> | null = null;
const debouncedLoadMembers = () => {
  if (memberSearchTimer) clearTimeout(memberSearchTimer);
  memberSearchTimer = setTimeout(() => {
    loadMembers();
  }, 260);
};

const openMemberPicker = async () => {
  memberPickerOpen.value = true;
  await loadMembers();
};

const loadTagCandidates = async () => {
  const accountUUID = String(formChannelAccountUUID.value || defaultChannelAccountUUID.value || "").trim();
  if (!accountUUID) {
    tagCandidates.value = [];
    return;
  }
  tagLoading.value = true;
  try {
    const resp = await socialService.listFoundationTags({
      channel_account_uuid: accountUUID,
      limit: 500,
    });
    const items = ((resp as any)?.data?.items || []) as FoundationTagRecord[];
    const q = tagKeyword.value.trim().toLowerCase();
    tagCandidates.value = q
      ? items.filter((item) =>
          `${item.remote_group_name || ""} ${item.tag_name || ""} ${item.remote_tag_id || ""}`.toLowerCase().includes(q)
        )
      : items;
  } catch (error: any) {
    toast.add({ title: "加载标签失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    tagLoading.value = false;
  }
};

const openTagPicker = async () => {
  tagPickerOpen.value = true;
  await loadTagCandidates();
  await hydrateSelectedTagLabels();
};

const tagCheckedSet = computed(() => new Set(selectedTags.value.map((item) => item.id)));
const tagCheckedNormalizedSet = computed(() => new Set(selectedTags.value.map((item) => normalizeRemoteID(item.id))));

const setTagChecked = (item: FoundationTagRecord, checked: boolean) => {
  const id = String(item.remote_tag_id || "").trim();
  const normalizedID = normalizeRemoteID(id);
  if (!id) return;
  if (!checked) {
    selectedTags.value = selectedTags.value.filter((row) => normalizeRemoteID(row.id) !== normalizedID);
  } else if (!tagCheckedNormalizedSet.value.has(normalizedID)) {
    selectedTags.value.push({
      id,
      label: String(item.tag_name || id),
      group_name: item.remote_group_name || "",
    });
  }
  form.corp_tag_ids_text = selectedTags.value.map((row) => row.id).join(",");
};

const removeSelectedTag = (id: string) => {
  const key = String(id || "").trim();
  const normalizedKey = normalizeRemoteID(key);
  if (!key) return;
  selectedTags.value = selectedTags.value.filter((item) => normalizeRemoteID(item.id) !== normalizedKey);
  form.corp_tag_ids_text = selectedTags.value.map((row) => row.id).join(",");
};

const setMemberChecked = (item: Member, checked: boolean) => {
  const uuid = resolveMemberUUID(item as any);
  if (!uuid) return;
  if (!checked) {
    form.member_uuids = form.member_uuids.filter((id) => id !== uuid);
    selectedMembers.value = selectedMembers.value.filter((m) => m.uuid !== uuid);
    return;
  }
  if (!form.member_uuids.includes(uuid)) {
    form.member_uuids.push(uuid);
  }
  if (!selectedMembers.value.find((m) => m.uuid === uuid)) {
    selectedMembers.value.push({
      uuid,
      display_name: String((item as any).display_name || (item as any).name || uuid),
      username: (item as any).username,
      email: (item as any).email,
    });
  }
};

const removeSelectedMember = (uuid: string) => {
  const key = String(uuid || "").trim();
  if (!key) return;
  form.member_uuids = form.member_uuids.filter((id) => id !== key);
  selectedMembers.value = selectedMembers.value.filter((m) => m.uuid !== key);
};

const replyTypeOptions = [
  { label: "图片", value: "image" as const },
  { label: "链接", value: "link" as const },
  { label: "小程序", value: "mini_program" as const },
];

const replyTypeLabel = (type: string) => {
  const hit = replyTypeOptions.find((item) => item.value === type);
  return hit?.label || type;
};

const appendReplyBlock = (type: "text" | "image" | "link" | "mini_program") => {
  replyBlocks.value.push(createReplyBlock(type));
  replyTypeMenuOpen.value = false;
};

const removeReplyBlock = (idx: number) => {
  if (idx < 0 || idx >= replyBlocks.value.length) return;
  replyBlocks.value.splice(idx, 1);
  if (replyBlocks.value.length === 0) {
    replyBlocks.value.push(createReplyBlock("text"));
  }
};

const appendToTextBlock = (idx: number, token: string) => {
  const block = replyBlocks.value[idx];
  if (!block || block.type !== "text") return;
  block.content = `${block.content || ""}${token}`;
};

const toggleEmojiPicker = (idx: number) => {
  emojiPickerIndex.value = emojiPickerIndex.value === idx ? -1 : idx;
};

const pickEmoji = (idx: number, emoji: string) => {
  appendToTextBlock(idx, emoji);
  emojiPickerIndex.value = -1;
};

onMounted(loadData);

onMounted(async () => {
  try {
    const resp = await socialService.listChannelAccounts();
    const items = ((resp as any)?.data?.items || []) as Array<Record<string, any>>;
    const hit = items.find((item) => {
      const channel = String(item.channel_code || "").toLowerCase();
      const appType = String(item.app_type || "").toLowerCase();
      const status = String(item.status || "").toLowerCase();
      return channel === "wechat" && (appType === "wecom" || appType === "openwork") && status === "connected";
    });
    if (hit?.account_uuid) {
      defaultChannelAccountUUID.value = String(hit.account_uuid);
    }
  } catch {
    // ignore, UI will degrade gracefully
  }
});
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

.acq-chat-msg {
  max-width: 86%;
  border-radius: 16px;
  padding: 8px 12px;
  font-size: 12px;
  line-height: 18px;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.12);
}

.acq-chat-msg-left {
  border-top-left-radius: 6px;
  background: #ffffff;
  color: #334155;
}

.acq-chat-msg-right {
  margin-left: auto;
  border-top-right-radius: 6px;
  background: linear-gradient(180deg, #1aa7ef 0%, #129be7 100%);
  color: #ffffff;
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

.acq-reply-menu {
  background: #0b1730;
  border: 1px solid rgba(148, 163, 184, 0.35);
  border-radius: 10px;
}

.acq-reply-menu-item {
  color: #e2e8f0;
}

.acq-reply-menu-item:hover {
  background: rgba(59, 130, 246, 0.2);
}

.acq-tool-btn {
  height: 40px;
  width: 56px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
  background: #0b1730;
}

.acq-tool-btn:hover {
  color: #e2e8f0;
  background: rgba(59, 130, 246, 0.18);
}

.acq-text-editor-head {
  display: inline-flex;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.35);
  border-radius: 8px;
  background: #0b1730;
}

.acq-emoji-panel {
  width: 220px;
  display: grid;
  grid-template-columns: repeat(8, minmax(0, 1fr));
  gap: 4px;
  padding: 8px;
  background: #0b1730;
  border: 1px solid rgba(148, 163, 184, 0.35);
  border-radius: 10px;
}

.acq-emoji-item {
  height: 24px;
  width: 24px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.acq-emoji-item:hover {
  background: rgba(59, 130, 246, 0.2);
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
