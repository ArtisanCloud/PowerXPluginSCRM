<template>
  <UContainer class="max-w-none py-8 space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">群活码</h1>
        <p class="text-sm text-gray-600 dark:text-gray-300">入群方式管理：支持本地保存、发布到渠道与同步状态追踪。</p>
      </div>
      <div class="flex items-center gap-2">
        <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="loadData">刷新</UButton>
        <UButton icon="i-heroicons-plus" color="primary" @click="openCreate">新建群活码</UButton>
      </div>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-2">
          <span class="font-medium">群活码列表</span>
          <UBadge color="neutral" variant="soft">{{ items.length }} 条</UBadge>
        </div>
      </template>

      <div class="overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead>
            <tr class="border-b border-gray-200 dark:border-gray-700 text-left">
              <th class="px-3 py-2">活动</th>
              <th class="px-3 py-2">渠道账号</th>
              <th class="px-3 py-2">状态</th>
              <th class="px-3 py-2">同步</th>
              <th class="px-3 py-2">config_id</th>
              <th class="px-3 py-2">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="6" class="px-3 py-8 text-center text-gray-500">加载中...</td>
            </tr>
            <tr v-else-if="items.length === 0">
              <td colspan="6" class="px-3 py-8 text-center text-gray-500">暂无群活码</td>
            </tr>
            <tr v-for="row in items" :key="row.group_code_uuid" class="border-b border-gray-100 dark:border-gray-800">
              <td class="px-3 py-2">{{ row.activity_name }}</td>
              <td class="px-3 py-2 font-mono text-xs">{{ row.channel_account_uuid }}</td>
              <td class="px-3 py-2"><UBadge variant="soft" :color="row.status === 'active' ? 'success' : 'neutral'">{{ row.status }}</UBadge></td>
              <td class="px-3 py-2">
                <UBadge
                  variant="soft"
                  :color="row.sync_status === 'success' ? 'success' : row.sync_status === 'failed' ? 'error' : 'warning'"
                >
                  {{ row.sync_status || 'pending' }}
                </UBadge>
              </td>
              <td class="px-3 py-2 font-mono text-xs">{{ row.config_id || '-' }}</td>
              <td class="px-3 py-2">
                <div class="flex flex-wrap gap-2">
                  <UButton size="xs" variant="soft" @click="openEdit(row)">编辑</UButton>
                  <UButton size="xs" color="primary" :loading="syncingId === row.group_code_uuid" @click="syncNow(row)">发布</UButton>
                  <UButton size="xs" color="error" variant="soft" @click="removeRow(row)">删除</UButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </UCard>

    <UModal v-model:open="createOpen" title="新建群活码" :ui="{ content: 'max-w-2xl' }">
      <template #body>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <UFormField label="活动名称"><UInput v-model="createForm.activity_name" /></UFormField>
          <UFormField label="渠道账号 UUID"><UInput v-model="createForm.channel_account_uuid" /></UFormField>
          <UFormField label="渠道"><UInput v-model="createForm.channel" /></UFormField>
          <UFormField label="应用类型"><UInput v-model="createForm.app_type" /></UFormField>
          <UFormField label="入群场景"><UInput v-model.number="createForm.join_scene" type="number" /></UFormField>
          <div class="flex items-center gap-3 pt-6">
            <USwitch v-model="createForm.skip_verify" />
            <span class="text-sm">免验证入群</span>
          </div>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" @click="createOpen = false">取消</UButton>
          <UButton color="primary" :loading="creating" @click="createRow">保存</UButton>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="editOpen" title="编辑群活码" :ui="{ content: 'max-w-2xl' }">
      <template #body>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <UFormField label="活动名称"><UInput v-model="editForm.activity_name" /></UFormField>
          <UFormField label="状态">
            <USelect v-model="editForm.status" :items="statusOptions" value-key="value" label-key="label" />
          </UFormField>
          <div class="flex items-center gap-3 pt-6">
            <USwitch v-model="editForm.skip_verify" />
            <span class="text-sm">免验证入群</span>
          </div>
          <div class="flex items-center gap-3 pt-6">
            <USwitch v-model="editForm.auto_create_room" />
            <span class="text-sm">满员自动建群</span>
          </div>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" @click="editOpen = false">取消</UButton>
          <UButton color="primary" :loading="editing" @click="saveEdit">保存</UButton>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { useAcquisitionService, type GroupLiveCodeRecord } from "~/composables/api/services/acquisition";

const toast = useToast();
const service = useAcquisitionService();
const loading = ref(false);
const creating = ref(false);
const editing = ref(false);
const syncingId = ref("");
const items = ref<GroupLiveCodeRecord[]>([]);

const createOpen = ref(false);
const editOpen = ref(false);
const current = ref<GroupLiveCodeRecord | null>(null);

const createForm = reactive({
  activity_name: "",
  channel_account_uuid: "",
  channel: "wechat",
  app_type: "wecom",
  join_scene: 1,
  skip_verify: false,
  auto_create_room: false,
});

const editForm = reactive({
  activity_name: "",
  status: "draft" as "draft" | "active" | "disabled",
  skip_verify: false,
  auto_create_room: false,
});

const statusOptions = [
  { label: "草稿", value: "draft" },
  { label: "启用", value: "active" },
  { label: "停用", value: "disabled" },
];

const loadData = async () => {
  loading.value = true;
  try {
    const resp = await service.listGroupCodes(200);
    items.value = (resp as any)?.data?.items || [];
  } catch (error: any) {
    toast.add({ title: "加载失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    loading.value = false;
  }
};

const openCreate = () => {
  createForm.activity_name = "";
  createOpen.value = true;
};

const createRow = async () => {
  creating.value = true;
  try {
    await service.createGroupCode({ ...createForm });
    toast.add({ title: "创建成功", color: "success" });
    createOpen.value = false;
    await loadData();
  } catch (error: any) {
    toast.add({ title: "创建失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    creating.value = false;
  }
};

const openEdit = (row: GroupLiveCodeRecord) => {
  current.value = row;
  editForm.activity_name = row.activity_name;
  editForm.status = (row.status || "draft") as any;
  editForm.skip_verify = Boolean(row.skip_verify);
  editForm.auto_create_room = Boolean(row.auto_create_room);
  editOpen.value = true;
};

const saveEdit = async () => {
  if (!current.value) return;
  editing.value = true;
  try {
    await service.updateGroupCode(current.value.group_code_uuid, {
      activity_name: editForm.activity_name,
      status: editForm.status,
      skip_verify: editForm.skip_verify,
      auto_create_room: editForm.auto_create_room,
    });
    toast.add({ title: "更新成功", color: "success" });
    editOpen.value = false;
    await loadData();
  } catch (error: any) {
    toast.add({ title: "更新失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    editing.value = false;
  }
};

const syncNow = async (row: GroupLiveCodeRecord) => {
  syncingId.value = row.group_code_uuid;
  try {
    await service.syncGroupCode(row.group_code_uuid);
    toast.add({ title: "发布成功", color: "success" });
    await loadData();
  } catch (error: any) {
    toast.add({ title: "发布失败", description: error?.message || "unknown error", color: "error" });
  } finally {
    syncingId.value = "";
  }
};

const removeRow = async (row: GroupLiveCodeRecord) => {
  try {
    await service.deleteGroupCode(row.group_code_uuid);
    toast.add({ title: "删除成功", color: "success" });
    await loadData();
  } catch (error: any) {
    toast.add({ title: "删除失败", description: error?.message || "unknown error", color: "error" });
  }
};

onMounted(loadData);
</script>
