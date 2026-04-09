<!-- /components/settings/users/UsersTenantAdmin.vue -->
<script setup lang="ts">
import {
  ref,
  reactive,
  computed,
  h,
  resolveComponent,
  onMounted,
  watch,
} from "vue";
import { useI18n, useToast } from "#imports";
import SelectTree from "~/components/ui/SelectTree.vue";
import {
  useUserService,
  type MemberWithProfile,
} from "~/composables/api/services/userService";
import { useIAMService } from "~/composables/api/services/iamService";

// ==== 输入属性（Root 复用时传入 tenantUuid） ====
const props = defineProps<{ tenantUuid: string }>();
const { t, locale } = useI18n();
const toast = useToast();

const userService = useUserService();
const iamService = useIAMService();

// ===== 类型与数据 =====
type StatusType = "active" | "inactive";

interface RowUser {
  id: number; // Member ID
  userId?: number; // User ID
  name: string;
  username?: string;
  email?: string;
  phone?: string;
  department?: string;
  departmentId?: number | null;
  status: StatusType | string;
  avatar: string;
  meta?: Record<string, any> | null;
}

const resolveErrorMessage = (err: any, fallback = "操作失败") => {
  return (
    err?.data?.error?.message ||
    err?.response?._data?.error?.message ||
    err?.response?._data?.message ||
    err?.message ||
    fallback
  );
};

const notifyError = (title: string, err: any, fallback = "操作失败") => {
  toast.add({
    color: "error",
    title,
    description: resolveErrorMessage(err, fallback),
  });
};

const notifySuccess = (title: string, description?: string) => {
  toast.add({
    color: "success",
    title,
    description,
  });
};

// 表格数据和加载状态
const users = ref<RowUser[]>([]);
const loading = ref(false);
const roleOptions = ref<Array<{ label: string; value: number }>>([]);
const loadingRoles = ref(false);

// ====== 过滤/分页（与你现有一致） ======
const searchQuery = ref("");
const filters = reactive({
  department: null as string | null,
  status: null as string | null,
});

const pagination = reactive({ page: 1, pageSize: 10, total: 0, totalPages: 0 });

const departmentItems = ref<any[]>([]);

function buildDepartmentTree(items: any[]) {
  const nodeMap = new Map<string, any>();
  const roots: any[] = [];

  for (const item of items) {
    const id = String(item?.id || "");
    if (!id) continue;
    nodeMap.set(id, {
      label: item?.name || "未命名部门",
      value: id,
      icon: "i-heroicons-building-office-2",
      defaultExpanded: true,
      disabled: item?.status === 0,
      children: [] as any[],
      _parentId:
        item?.parent_id === null || item?.parent_id === undefined
          ? null
          : String(item.parent_id),
    });
  }

  for (const node of nodeMap.values()) {
    const parentId = node._parentId;
    if (!parentId || !nodeMap.has(parentId)) {
      roots.push(node);
      continue;
    }
    nodeMap.get(parentId).children.push(node);
  }

  const clean = (nodes: any[]): any[] =>
    nodes
      .map((n) => {
        const children = clean(n.children || []);
        return {
          label: n.label,
          value: n.value,
          icon: n.icon,
          defaultExpanded: n.defaultExpanded,
          disabled: n.disabled,
          children,
        };
      })
      .sort((a, b) => a.label.localeCompare(b.label, "zh-CN"));

  return clean(roots);
}

async function loadDepartmentOptions() {
  if (!props.tenantUuid) {
    departmentItems.value = [];
    return;
  }
  try {
    const response = await iamService.listDepartments(props.tenantUuid);
    const items = (response as any)?.data?.items ?? [];
    departmentItems.value = buildDepartmentTree(Array.isArray(items) ? items : []);
  } catch (error) {
    departmentItems.value = [];
    notifyError("加载部门失败", error);
  }
}

// 将部门数据转换为SelectTree需要的TreeNode格式
const departmentTreeItems = computed(() => {
  return departmentItems.value;
});
const hasDepartmentOptions = computed(() => departmentTreeItems.value.length > 0);
// ====== 导入导出 ======
type ExportFormat = "csv" | "json";

async function exportUsers(format: ExportFormat) {
  try {
    let content: string;
    let filename: string;
    let mimeType: string;

    if (format === "csv") {
      const { default: Papa } = await import("papaparse");
      content = Papa.unparse(
        users.value.map((u) => ({
          姓名: u.name,
          用户名: u.username,
          邮箱: u.email,
          部门: u.department || "",
          状态: u.status === "active" ? "激活" : "停用",
        }))
      );
      filename = `users_${new Date().toISOString().split("T")[0]}.csv`;
      mimeType = "text/csv;charset=utf-8;";
    } else {
      content = JSON.stringify(users.value, null, 2);
      filename = `users_${new Date().toISOString().split("T")[0]}.json`;
      mimeType = "application/json;charset=utf-8;";
    }

    const { saveAs } = await import("file-saver");
    const blob = new Blob([content], { type: mimeType });
    saveAs(blob, filename);
    notifySuccess("导出成功", `已导出 ${users.value.length} 条记录`);
  } catch (error) {
    console.error("导出失败:", error);
    notifyError("导出失败", error, "导出失败，请重试");
  }
}

function importUsers() {
  const input = document.createElement("input");
  input.type = "file";
  input.accept = ".csv,.json";
  input.onchange = async (e) => {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (!file) return;

    try {
      const text = await file.text();
      let importedData: any[];

      if (file.name.endsWith(".csv")) {
        const { default: Papa } = await import("papaparse");
        const result = Papa.parse(text, { header: true, skipEmptyLines: true });
        importedData = result.data;
      } else {
        importedData = JSON.parse(text);
      }

      // 这里可以添加数据验证和转换逻辑
      console.log("导入的数据:", importedData);
      notifySuccess("导入成功", `成功导入 ${importedData.length} 条记录`);
    } catch (error) {
      console.error("导入失败:", error);
      notifyError("导入失败", error, "导入失败，请检查文件格式");
    }
  };
  input.click();
}

const importExportItems = computed(() => [
  [
    {
      label: t("organization.user.export.csv"),
      icon: "i-heroicons-arrow-down-tray",
      click: () => exportUsers("csv"),
    },
    {
      label: t("organization.user.export.json"),
      icon: "i-heroicons-arrow-down-tray",
      click: () => exportUsers("json"),
    },
  ],
  [
    {
      label: t("organization.user.import.button"),
      icon: "i-heroicons-arrow-up-tray",
      click: () => importUsers(),
    },
  ],
]);

// ====== 新增/编辑 ======
const showForm = ref(false);
const isEditing = ref(false);
const editingId = ref<number | null>(null);

// 统一"扁平表单" -> 后端映射 User+Member（我们之前对齐的）
const userForm = reactive({
  name: "",
  username: "",
  email: "",
  phone: "",
  departmentId: null as number | null,
  roleId: null as number | null,
  avatarUrl: "",
  password: "",
  confirmPassword: "",
  status: "active" as "active" | "disabled" | "locked",
  meta: {} as Record<string, any>,
});

function resetForm() {
  userForm.name = "";
  userForm.username = "";
  userForm.email = "";
  userForm.phone = "";
  userForm.departmentId = null;
  userForm.roleId = null;
  userForm.avatarUrl = "";
  userForm.password = "";
  userForm.confirmPassword = "";
  userForm.status = "active";
  userForm.meta = {};
  isEditing.value = false;
  editingId.value = null;
}

function normalizeDepartmentId(value: unknown): number | null {
  if (value === null || value === undefined || value === "") return null;
  const n = Number(value);
  return Number.isFinite(n) && n > 0 ? n : null;
}

function openAddForm() {
  resetForm();
  showForm.value = true;
}

function openEditForm(row: RowUser) {
  resetForm();
  isEditing.value = true;
  editingId.value = row.id; // 这里使用的是Member的ID

  // 将行数据映射回表单
  userForm.name = row.name;
  userForm.username = row.username || "";
  userForm.email = row.email || "";
  userForm.phone = row.phone || "";
  userForm.departmentId = normalizeDepartmentId(row.departmentId);
  userForm.avatarUrl = row.avatar;
  userForm.status = row.status === "active" ? "active" : "disabled";
  userForm.meta = row.meta || {};
  showForm.value = true;
}

async function saveUser() {
  // 基础校验
  if (!userForm.name || !userForm.email) {
    notifyError("保存失败", null, t("organization.user.validation.requiredFields"));
    return;
  }
  if (!normalizeDepartmentId(userForm.departmentId)) {
    notifyError("保存失败", null, "请选择部门");
    return;
  }
  if (!userForm.roleId) {
    notifyError("保存失败", null, "请选择角色");
    return;
  }
  if (!isEditing.value && !userForm.username) {
    notifyError("保存失败", null, "用户名为必填项");
    return;
  }
  if (!isEditing.value && userForm.password !== userForm.confirmPassword) {
    notifyError("保存失败", null, t("organization.user.validation.passwordMismatch"));
    return;
  }

  try {
    if (isEditing.value && editingId.value) {
      // 更新用户
      const updatePayload: Record<string, any> = {
        display_name: userForm.name,
        email: userForm.email,
        phone: userForm.phone,
        departmentId: normalizeDepartmentId(userForm.departmentId),
        avatar_url: userForm.avatarUrl,
        status: userForm.status === "active" ? 1 : 0,
      };
      if (userForm.roleId) {
        updatePayload.roles = [userForm.roleId];
        updatePayload.replace_roles = true;
      }
      await userService.updateUser(editingId.value, updatePayload);
    } else {
      // 创建系统用户
      const createPayload = {
        tenant_uuid: props.tenantUuid,
        display_name: userForm.name,
        email: userForm.email,
        phone: userForm.phone,
        departmentId: normalizeDepartmentId(userForm.departmentId),
        avatar_url: userForm.avatarUrl,
        status: userForm.status === "active" ? 1 : 0,
        meta: userForm.meta ?? {},
        username: userForm.username || userForm.email.split("@")[0],
        initial_password: userForm.password,
        roles: userForm.roleId ? [userForm.roleId] : [],
      };
      await userService.createSystemUser(createPayload);
    }
    showForm.value = false;
    await loadUsers(); // 重新加载数据
    notifySuccess(isEditing.value ? "用户更新成功" : "用户创建成功");
  } catch (e: any) {
    notifyError("保存失败", e);
  }
}

async function loadRoleOptions() {
  if (!props.tenantUuid) {
    roleOptions.value = [];
    return;
  }
  loadingRoles.value = true;
  try {
    const resp = await iamService.listRoles({ tenantUuid: props.tenantUuid });
    const items = (resp as any)?.data?.items ?? [];
    roleOptions.value = Array.isArray(items)
      ? items.map((item: any) => ({
          label: String(item?.name || item?.code || item?.id || ""),
          value: Number(item?.id),
        })).filter((item: any) => Number.isFinite(item.value) && item.value > 0)
      : [];
    if (import.meta.dev) {
      console.info("[UsersTenantAdmin] role options loaded:", roleOptions.value.length);
    }
  } catch (err) {
    notifyError("加载角色失败", err);
    roleOptions.value = [];
  } finally {
    loadingRoles.value = false;
  }
}

async function deleteUser(id: number) {
  if (!confirm(t("organization.user.confirmDelete"))) return;
  try {
    // 注意：这里的id是Member的ID，但API可能需要User的ID
    // 根据后端实现调整
    await userService.deleteUser(id);
    await loadUsers(); // 重新加载数据
    notifySuccess("用户已停用");
  } catch (e: any) {
    notifyError("删除失败", e);
  }
}

async function toggleUserStatus(row: RowUser) {
  try {
    const newStatus = row.status === "active" ? 0 : 1;
    // 注意：这里的row.id是Member的ID，但API可能需要User的ID
    // 根据后端实现调整
    await userService.setUserStatus(row.id, { status: newStatus });
    await loadUsers(); // 重新加载数据
    notifySuccess("状态更新成功");
  } catch (e: any) {
    notifyError("状态更新失败", e);
  }
}

// ===== 过滤/分页逻辑 =====
const filteredUsers = computed(() => {
  // 由于使用API分页，直接返回当前用户数据
  return users.value;
});

const paginatedUsers = computed(() => {
  // API已经返回分页数据，直接使用
  return users.value;
});

const hasNextPage = computed(() => pagination.page < pagination.totalPages);
const hasPrevPage = computed(() => pagination.page > 1);

async function changePage(p: number) {
  if (p >= 1 && p <= pagination.totalPages) {
    pagination.page = p;
    await loadUsers();
  }
}

async function changePageSize(size: number) {
  pagination.pageSize = size;
  pagination.page = 1;
  await loadUsers();
}

function resetFilters() {
  filters.department = filters.status = null;
  searchQuery.value = "";
  pagination.page = 1;
  loadUsers();
}

// 监听搜索和过滤条件变化
watch(
  [
    searchQuery,
    () => filters.department,
    () => filters.status,
  ],
  () => {
    pagination.page = 1;
    loadUsers();
  }
);

watch(
  () => props.tenantUuid,
  () => {
    pagination.page = 1;
    loadUsers();
    loadDepartmentOptions();
    loadRoleOptions();
  }
);

// ===== 列定义：含"编辑/禁用/删除"操作 =====
const UButton = resolveComponent("UButton");
const UAvatar = resolveComponent("UAvatar");
const UBadge = resolveComponent("UBadge");

const columns = computed(() => {
  const _ = locale.value;
  return [
    {
      id: "avatar",
      accessorKey: "avatar",
      header: "",
      cell: ({ row }: any) => {
        const u = row.original as RowUser;
        return h(UAvatar, { src: u.avatar, alt: u.name, size: "sm" });
      },
    },
    {
      id: "name",
      accessorKey: "name",
      header: t("organization.user.table.name").toString(),
    },
    {
      id: "username",
      accessorKey: "username",
      header: t("organization.user.table.username").toString(),
    },
    {
      id: "email",
      accessorKey: "email",
      header: t("organization.user.table.email").toString(),
    },
    {
      id: "phone",
      accessorKey: "phone",
      header: t("organization.user.table.phone").toString(),
      cell: ({ row }: any) => {
        const u = row.original as RowUser;
        return maskPhone(u.phone || "");
      },
    },
    {
      id: "status",
      accessorKey: "status",
      header: t("organization.user.table.status").toString(),
      cell: ({ row }: any) => {
        const u = row.original as RowUser;
        return h(
          UBadge,
          {
            color: u.status === "active" ? "success" : "neutral",
            variant: "subtle",
            size: "sm",
          },
          () =>
            u.status === "active"
              ? t("organization.user.form.active")
              : t("organization.user.form.inactive")
        );
      },
    },
    {
      id: "actions",
      header: t("organization.user.table.actions").toString(),
      cell: ({ row }: any) => {
        const u = row.original as RowUser;
        return h("div", { class: "flex gap-2" }, [
          h(
            UButton,
            {
              size: "xs",
              variant: "ghost",
              icon: "i-heroicons-pencil-square",
              onClick: () => openEditForm(u),
            },
            () => t("organization.common.edit")
          ),
          h(
            UButton,
            {
              size: "xs",
              color: u.status === "active" ? "warning" : "success",
              variant: "ghost",
              icon:
                u.status === "active"
                  ? "i-heroicons-lock-closed"
                  : "i-heroicons-lock-open",
              onClick: () => toggleUserStatus(u),
            },
            () =>
              u.status === "active"
                ? t("organization.user.disable")
                : t("organization.user.enable")
          ),
          h(
            UButton,
            {
              size: "xs",
              color: "error",
              variant: "ghost",
              icon: "i-heroicons-trash",
              onClick: () => deleteUser(u.id),
            },
            () => t("organization.common.delete")
          ),
        ]);
      },
    },
  ];
});

// 手机号脱敏函数
function maskPhone(phone: string): string {
  if (!phone) return "";
  if (phone.length <= 7) return phone;
  return phone.slice(0, 3) + "****" + phone.slice(-4);
}

function fallbackAvatarDataUri(seed: string): string {
  const raw = String(seed || "U").trim();
  const initial = (raw[0] || "U").toUpperCase();
  const hue = Array.from(raw).reduce((acc, ch) => acc + ch.charCodeAt(0), 0) % 360;
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="96" height="96" viewBox="0 0 96 96"><rect width="96" height="96" rx="20" fill="hsl(${hue},70%,45%)"/><text x="50%" y="54%" text-anchor="middle" dominant-baseline="middle" fill="#fff" font-family="Arial, sans-serif" font-size="42" font-weight="700">${initial}</text></svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
}

// 转换API数据为组件需要的格式
function transformUserData(memberWithProfile: MemberWithProfile): RowUser {
  const { Member, User } = memberWithProfile;
  const rawDepartment = (Member.meta as any)?.department;
  const departmentId = normalizeDepartmentId(rawDepartment);
  const avatarSeed = User.email || Member.display_name || Member.username || "U";
  return {
    id: Member.id, // 使用Member的ID作为主要ID
    userId: User.id, // 保存User的ID以备后用
    name: Member.display_name || User.display_name,
    username: Member.username,
    email: User.email || "",
    phone: User.phone || "",
    department: Member.meta?.title || Member.meta?.department || "",
    departmentId,
    status: Member.status === 1 ? "active" : "inactive",
    avatar:
      Member.avatar_url ||
      User.avatar_url ||
      fallbackAvatarDataUri(avatarSeed),
    meta: { ...User.meta, ...Member.meta }, // 合并User和Member的meta
  };
}

// 加载用户数据
async function loadUsers() {
  if (!props.tenantUuid) {
    return;
  }
  try {
    loading.value = true;
    const params: any = {
      tenant_uuid: props.tenantUuid,
      page: pagination.page,
      page_size: pagination.pageSize,
      status: filters.status
        ? filters.status === "active"
          ? 1
          : 0
        : undefined, // 不传status则显示所有状态
    };

    // 添加搜索参数
    if (searchQuery.value.trim()) {
      params.q = searchQuery.value.trim(); // 后端使用q参数
    }

    const response = await userService.getUsers(params);

    if (response.data) {
      users.value = response.data.items.map(transformUserData);
      pagination.total = response.data.pagination.total;
      pagination.totalPages = response.data.pagination.pages;
    }
  } catch (error) {
    console.error("加载用户数据失败:", error);
    notifyError("加载用户数据失败", error);
  } finally {
    loading.value = false;
  }
}

// 初始化数据
onMounted(async () => {
  await loadDepartmentOptions();
  await loadUsers();
  await loadRoleOptions();
});
</script>

<template>
  <div>
    <!-- 顶部：导入导出 + 新增 -->
    <div class="flex justify-between items-center mb-6">
      <div>
        <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
          {{ $t("organization.user.title") }}
        </h2>
        <p class="text-sm text-gray-500 mt-1 dark:text-slate-200">
          {{ $t("organization.user.description") }}
        </p>
      </div>
      <div class="flex space-x-2">
        <UDropdownMenu :items="importExportItems">
          <UButton
            color="neutral"
            variant="outline"
            icon="i-heroicons-arrow-up-tray"
          >
            {{ $t("organization.user.importExport") }}
          </UButton>
        </UDropdownMenu>
        <UButton color="primary" icon="i-heroicons-plus" @click="openAddForm">
          {{ $t("organization.user.add") }}
        </UButton>
      </div>
    </div>

    <!-- 搜索与筛选（与你现有一致） -->
    <div class="mb-6 rounded-lg bg-white p-4 shadow-sm dark:bg-slate-950/70 dark:border dark:border-slate-800/60">
      <div class="flex flex-wrap gap-4 items-end">
        <div class="flex-grow min-w-[200px]">
          <UInput
            v-model="searchQuery"
            icon="i-heroicons-magnifying-glass"
        :placeholder="$t('organization.user.search')"
        />
      </div>
        <UFormField :label="$t('organization.user.form.department')">
          <SelectTree
            v-model="filters.department"
            :items="departmentTreeItems"
            :placeholder="$t('organization.user.form.selectDepartment')"
            searchable
            clearable
            class="w-full sm:min-w-[12rem]"
          />
        </UFormField>
        <UFormField :label="$t('organization.user.form.status')" class="mb-0">
          <USelect
            v-model="filters.status"
            :items="[
              { label: $t('organization.user.filter.allStatus'), value: null },
              { label: $t('organization.user.filter.active'), value: 'active' },
              {
                label: $t('organization.user.filter.inactive'),
                value: 'inactive',
              },
            ]"
            class="w-full sm:w-40"
            :placeholder="$t('organization.user.filter.allStatus')"
          />
        </UFormField>
        <UButton
          color="neutral"
          variant="ghost"
          icon="i-heroicons-arrow-path"
          @click="resetFilters"
        >
          {{ $t("organization.user.filter.reset") }}
        </UButton>
      </div>
    </div>

    <!-- 表格 + 分页 -->
    <div
      class="rounded-lg bg-white shadow-sm dark:bg-slate-950/70 dark:border dark:border-slate-800/60"
    >
      <UTable
        :data="paginatedUsers"
        :columns="columns"
        :loading="loading"
        :empty-state="{
          icon: 'i-heroicons-circle-stack-20-solid',
          label: '暂无用户数据',
          description: '当前没有找到任何用户信息',
        }"
      />
      <div
        v-if="pagination.totalPages > 1"
        class="px-6 py-4 border-t border-gray-200 dark:border-slate-800/60 flex justify-between items-center"
      >
        <div class="text-sm text-gray-600 dark:text-slate-200">
          第 {{ pagination.page }} / {{ pagination.totalPages }} 页， 共
          {{ pagination.total }} 条
        </div>
        <div class="flex gap-2">
          <UButton
            :disabled="!hasPrevPage || loading"
            variant="outline"
            size="sm"
            icon="i-heroicons-chevron-left"
            @click="changePage(pagination.page - 1)"
            >上一页</UButton
          >
          <UButton
            :disabled="!hasNextPage || loading"
            variant="outline"
            size="sm"
            icon="i-heroicons-chevron-right"
            @click="changePage(pagination.page + 1)"
            >下一页</UButton
          >
        </div>
      </div>
    </div>

    <!-- 表单弹窗（新增/编辑） -->
    <UModal
      v-model:open="showForm"
      :title="
        isEditing
          ? t('organization.user.form.editUser')
          : t('organization.user.form.addUser')
      "
      :description="
        isEditing
          ? t('organization.user.form.editUserDesc')
          : t('organization.user.form.addUserDesc')
      "
    >
      <template #content>
        <div class="py-8 px-8">
          <form
            @submit.prevent="saveUser"
            class="grid grid-cols-1 md:grid-cols-2 gap-4"
          >
            <UFormField :label="$t('organization.user.form.name')" required>
              <UInput v-model="userForm.name" />
            </UFormField>
            <UFormField
              :label="$t('organization.user.form.username')"
              :required="!isEditing"
            >
              <UInput
                v-model="userForm.username"
                :placeholder="isEditing ? '编辑时可选' : '必填，用于租户内登录'"
              />
            </UFormField>
            <UFormField
              :label="$t('organization.user.form.email')"
              required
              class="md:col-span-2"
            >
              <UInput v-model="userForm.email" type="email" />
            </UFormField>
            <UFormField
              :label="$t('organization.user.form.department')"
              required
              class="md:col-span-2"
            >
              <div class="space-y-2">
                <SelectTree
                  v-model="userForm.departmentId"
                  :items="departmentTreeItems"
                  :placeholder="$t('organization.user.form.selectDepartment')"
                  tree-class="w-[22rem] max-h-72 overflow-auto rounded-md border border-gray-200 px-1 py-1 dark:border-slate-700"
                  button-class="text-gray-900 dark:text-slate-100"
                  searchable
                  clearable
                />
                <div
                  v-if="!hasDepartmentOptions"
                  class="flex items-center justify-between rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-200"
                >
                  <span>当前租户暂无部门，请先新增部门后再选择。</span>
                  <UButton
                    size="xs"
                    color="warning"
                    variant="soft"
                    :to="{ path: '/admin/iam/members', query: { tab: 'departments' } }"
                  >
                    去新增部门
                  </UButton>
                </div>
              </div>
            </UFormField>
            <UFormField
              :label="$t('organization.user.form.role')"
              required
              class="md:col-span-2"
            >
              <div class="space-y-2">
                <div class="text-xs text-gray-500 dark:text-slate-300">
                  角色（必选），来源于当前租户角色配置。
                </div>
                <USelect
                  v-model="userForm.roleId"
                  :items="roleOptions"
                  option-attribute="label"
                  value-attribute="value"
                  :placeholder="$t('organization.user.form.selectRole')"
                  :loading="loadingRoles"
                  class="w-full min-h-10"
                />
                <div
                  v-if="!loadingRoles && roleOptions.length === 0"
                  class="flex items-center justify-between rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-200"
                >
                  <span>当前租户暂无角色，请先创建角色后再新增成员。</span>
                  <UButton
                    size="xs"
                    color="warning"
                    variant="soft"
                    :to="{ path: '/admin/iam/members', query: { tab: 'permissions' } }"
                  >
                    去权限页
                  </UButton>
                </div>
              </div>
            </UFormField>
            <UFormField :label="$t('organization.user.form.phone')">
              <UInput
                v-model="userForm.phone"
                type="tel"
                :placeholder="$t('organization.user.form.phonePlaceholder')"
              />
            </UFormField>
            <UFormField
              :label="$t('organization.user.form.password')"
              :required="!isEditing"
              ><UInput v-model="userForm.password" type="password"
            /></UFormField>
            <UFormField
              :label="$t('organization.user.form.confirmPassword')"
              :required="!isEditing"
              ><UInput v-model="userForm.confirmPassword" type="password"
            /></UFormField>
            <div class="md:col-span-2 flex justify-end gap-3 mt-2">
              <UButton
                color="neutral"
                variant="outline"
                @click="showForm = false"
                >{{ $t("organization.common.cancel") }}</UButton
              >
              <UButton type="submit" color="primary">{{ $t("organization.common.save") }}</UButton>
            </div>
          </form>
        </div>
      </template>
    </UModal>
  </div>
</template>
