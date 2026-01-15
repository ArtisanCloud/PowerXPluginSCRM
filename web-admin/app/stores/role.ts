import { defineStore } from "pinia";
import { ref, computed } from "vue";

import {
  useRoleService,
  type Role,
  type RoleListParams,
  type RoleListResponse,
  type RoleCreateParams,
  type RoleUpdateParams,
  type CreateRoleWithPermsResponse,
} from "~/composables/api/services/roleService";

export const useRoleStore = defineStore("role", () => {
  const roleService = useRoleService();
  const isApiSuccess = (response: any): boolean => {
    if (typeof response?.success === "boolean") {
      return response.success;
    }
    if (typeof response?.code === "number") {
      return response.code >= 200 && response.code < 400;
    }
    return false;
  };
  const unwrapResponse = <T>(response: any): T | undefined =>
    (response?.data ?? response) as T | undefined;

  // 状态
  const roles = ref<Role[]>([]);
  const currentRole = ref<Role | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);
  const initialized = ref(false);
  const lastFetchTime = ref<number | null>(null);

  // 分页信息
  const pagination = ref({
    total: 0,
    page: 1,
    page_size: 20,
    pages: 1,
  });

  // 计算属性
  const systemRoles = computed(() =>
    roles.value.filter((role) => role.scope === "system")
  );

  const tenantRoles = computed(() =>
    roles.value.filter((role) => role.scope === "tenant")
  );

  const builtinRoles = computed(() =>
    roles.value.filter((role) => role.builtin)
  );

  const customRoles = computed(() =>
    roles.value.filter((role) => !role.builtin)
  );

  const isInitialized = computed(() => initialized.value);

  const needsRefresh = computed(() => {
    if (!initialized.value) return true;
    // 可以添加基于时间的缓存策略，比如5分钟后需要刷新
    // const fiveMinutes = 5 * 60 * 1000;
    // return lastFetchTime.value && (Date.now() - lastFetchTime.value) > fiveMinutes;
    return false;
  });

  const normalizeRole = (role: any): Role => {
    const scope = role.scope || role.scope_type || "tenant";
    return {
      ...role,
      tenant_uuid: role.tenant_uuid || role.tenantUuid,
      scope,
      scope_type: scope,
      createdAt: role.createdAt || role.created_at,
      updatedAt: role.updatedAt || role.updated_at,
      permission_ids:
        role.permission_ids || role.permissionIds || role.permissions || [],
      member_ids: role.member_ids || role.memberIds || [],
      member_count: role.member_count ?? role.memberCount ?? 0,
      builtin: role.builtin ?? false,
    };
  };

  // 操作方法
  const fetchRoles = async (params: RoleListParams, force = false) => {
    if (!params?.tenant_uuid) {
      throw new Error("tenant_uuid is required to fetch roles");
    }
    // 如果已经初始化且不强制刷新，则跳过
    if (initialized.value && !force && !needsRefresh.value) {
      return;
    }

    loading.value = true;
    error.value = null;

    try {
      const response = await roleService.getRoles(params);
      if (!isApiSuccess(response)) {
        throw new Error(response?.message || "获取角色列表失败");
      }
      const payload = unwrapResponse<RoleListResponse>(response);
      const items = payload?.items ?? [];
      roles.value = items.map(normalizeRole);
      const paginationData =
        payload?.pagination || {
          total: items.length,
          page: 1,
          page_size: items.length || 1,
          pages: 1,
        };
      pagination.value = paginationData;
      initialized.value = true;
      lastFetchTime.value = Date.now();
    } catch (err) {
      error.value = err instanceof Error ? err.message : "获取角色列表失败";
      console.error("获取角色列表失败:", err);
    } finally {
      loading.value = false;
    }
  };

  const fetchRole = async (id: number) => {
    loading.value = true;
    error.value = null;

    try {
      const response = await roleService.getRole(id);
      if (!isApiSuccess(response)) {
        throw new Error(response?.message || "获取角色信息失败");
      }
      const payload = unwrapResponse<Role>(response);
      if (!payload) {
        throw new Error("角色数据为空");
      }
      currentRole.value = normalizeRole(payload);
      return currentRole.value;
    } catch (err) {
      error.value = err instanceof Error ? err.message : "获取角色信息失败";
      console.error("获取角色信息失败:", err);
      throw err;
    } finally {
      loading.value = false;
    }
  };

  // ✅ 修正：只保留一个 createRole 定义，并带返回类型
  const createRole = async (
    data: RoleCreateParams
  ): Promise<CreateRoleWithPermsResponse> => {
    loading.value = true;
    error.value = null;

    try {
      const response = await roleService.createRole(data);
      if (!isApiSuccess(response)) {
        throw new Error(response?.message || "创建角色失败");
      }
      const payload = unwrapResponse<Role>(response);
      if (!payload) {
        throw new Error("创建角色失败：返回数据为空");
      }
      const roleData = normalizeRole(payload);
      roles.value.push(roleData);
      return {
        role: roleData,
        perm: (payload as any)?.perm,
      };
    } catch (err) {
      error.value = err instanceof Error ? err.message : "创建角色失败";
      console.error("创建角色失败:", err);
      throw err;
    } finally {
      loading.value = false;
    }
  };

  const updateRole = async (id: number, data: RoleUpdateParams) => {
    loading.value = true;
    error.value = null;

    try {
      const response = await roleService.updateRole(id, data);
      if (!isApiSuccess(response)) {
        throw new Error(response?.message || "更新角色失败");
      }
      const payload = unwrapResponse<Role>(response);
      const nextRole = payload
        ? normalizeRole(payload)
        : normalizeRole({
            ...(roles.value.find((role) => role.id === id) || {}),
            ...data,
          });

      const index = roles.value.findIndex((role) => role.id === id);
      if (index !== -1) {
        roles.value[index] = nextRole;
      }

      if (currentRole.value?.id === id) {
        currentRole.value = nextRole;
      }

      return true;
    } catch (err) {
      error.value = err instanceof Error ? err.message : "更新角色失败";
      throw err;
    } finally {
      loading.value = false;
    }
  };

  const deleteRole = async (id: number) => {
    loading.value = true;
    error.value = null;

    try {
      const response = await roleService.deleteRole(id);
      if (!isApiSuccess(response)) {
        throw new Error(response?.message || "删除角色失败");
      }

      const index = roles.value.findIndex((role) => role.id === id);
      if (index !== -1) {
        roles.value.splice(index, 1);
      }

      if (currentRole.value?.id === id) {
        currentRole.value = null;
      }

      pagination.value.total = Math.max(0, pagination.value.total - 1);

      return true;
    } catch (err) {
      error.value = err instanceof Error ? err.message : "删除角色失败";
      console.error("删除角色失败:", err);
      throw err;
    } finally {
      loading.value = false;
    }
  };

  // 确保已初始化（如果没有则自动加载）
  const ensureInitialized = async (params: RoleListParams) => {
    if (!initialized.value) {
      await fetchRoles(params);
    }
  };

  // 强制刷新数据
  const forceRefresh = async (params: RoleListParams) => {
    await fetchRoles(params, true);
  };

  // 重置状态
  const resetState = () => {
    roles.value = [];
    currentRole.value = null;
    loading.value = false;
    error.value = null;
    initialized.value = false;
    lastFetchTime.value = null;
    pagination.value = {
      total: 0,
      page: 1,
      page_size: 20,
      pages: 1,
    };
  };

  // 清除错误
  const clearError = () => {
    error.value = null;
  };

  return {
    // 状态
    roles,
    currentRole,
    loading,
    error,
    pagination,
    initialized,
    lastFetchTime,

    // 计算属性
    systemRoles,
    tenantRoles,
    builtinRoles,
    customRoles,
    isInitialized,
    needsRefresh,

    // 方法
    fetchRoles,
    fetchRole,
    createRole,
    updateRole,
    deleteRole,
    ensureInitialized,
    forceRefresh,
    resetState,
    clearError,
  };
});
