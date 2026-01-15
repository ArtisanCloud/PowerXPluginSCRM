import { useIAMService, type MemberRecord } from "./iamService";
import type { ApiResponse } from "../types/types";

export interface User {
  id: number;
  uuid: string;
  createdAt: string;
  updatedAt: string;
  email?: string;
  phone?: string;
  display_name: string;
  avatar_url?: string;
  status: number;
  meta?: Record<string, any>;
}

export interface Member {
  id: number;
  uuid: string;
  tenant_uuid: string;
  user_id: number;
  username: string;
  display_name: string;
  avatar_url?: string;
  status: number;
  meta?: Record<string, any>;
  createdAt: string;
  updatedAt: string;
}

export interface MemberWithProfile {
  Member: Member;
  User: User;
  DeptIDs: number[] | null;
}

export interface UserListParams {
  tenant_uuid: string;
  q?: string;
  page?: number;
  page_size?: number;
  status?: number | string;
}

export interface RoleMembersPayload {
  member_ids: number[];
}

type MemberListResponse = {
  items: MemberWithProfile[];
  pagination: {
    total: number;
    page: number;
    page_size: number;
    pages: number;
  };
};

const toMemberWithProfile = (record: MemberRecord): MemberWithProfile => {
  const status =
    record.status === "disabled" || record.status === "locked" ? 0 : 1;
  return {
    Member: {
      id: record.member_id,
      uuid: `${record.member_id}`,
      tenant_uuid: record.tenant_uuid,
      user_id: record.user_id,
      username: record.username,
      display_name: record.display_name || record.email || record.username,
      avatar_url: (record as any).avatar_url,
      status,
      meta: {
        department: record.department_id,
        phone: record.phone,
      },
      createdAt: record.created_at,
      updatedAt: (record as any).updated_at || record.created_at,
    },
    User: {
      id: record.user_id,
      uuid: `${record.user_id}`,
      createdAt: record.created_at,
      updatedAt: (record as any).updated_at || record.created_at,
      email: record.email,
      phone: record.phone,
      display_name: record.display_name || record.email || record.username,
      avatar_url: (record as any).avatar_url,
      status,
      meta: {},
    },
    DeptIDs: record.department_id ? [record.department_id] : [],
  };
};

export const useUserService = () => {
  const iamService = useIAMService();

  const getUsers = async (
    params: UserListParams
  ): Promise<ApiResponse<MemberListResponse>> => {
    if (!params?.tenant_uuid) {
      throw new Error("需要提供租户 UUID");
    }
    const response = await iamService.listMembers({
      tenantUuid: params.tenant_uuid,
      query: params.q,
      status: typeof params.status === "number" ? `${params.status}` : params.status,
      page: params.page,
      pageSize: params.page_size,
    });
    const payload = (response as any)?.data ?? response ?? {};
    const list =
      payload?.data?.items ??
      payload?.items ??
      payload?.data ??
      payload ??
      [];
    const items = Array.isArray(list)
      ? (list as MemberRecord[]).map(toMemberWithProfile)
      : [];
    const total =
      payload?.data?.pagination?.total ??
      payload?.pagination?.total ??
      payload?.data?.total ??
      payload?.total ??
      items.length;
    const page = params.page ?? payload?.data?.pagination?.page ?? 1;
    const pageSize =
      params.page_size ??
      payload?.data?.pagination?.page_size ??
      payload?.pagination?.page_size ??
      (items.length || 1);
    const pages =
      payload?.data?.pagination?.pages ??
      payload?.pagination?.pages ??
      Math.max(1, Math.ceil(total / Math.max(1, pageSize)));

    return {
      code: (response as any)?.code ?? 200,
      message: (response as any)?.message ?? "",
      timestamp: Date.now(),
      data: {
        items,
        pagination: {
          total,
          page,
          page_size: pageSize,
          pages,
        },
      },
    };
  };

  const createSystemUser = async (data: Record<string, any>) => {
    const payload = {
      tenant_uuid: data.tenant_uuid,
      email: data.email,
      display_name: data.display_name ?? data.name,
      username: data.username,
      phone: data.phone,
      department_id: data.departmentId ?? data.department_id,
      status: data.status,
    };
    const response = await iamService.createMember(payload);
    return (response as any)?.data ?? response;
  };

  const updateUser = async (memberId: number, data: Record<string, any>) => {
    const payload = {
      email: data.email,
      display_name: data.display_name ?? data.name,
      username: data.username,
      phone: data.phone,
      department_id: data.departmentId ?? data.department_id,
      status: data.status,
    };
    const response = await iamService.updateMember(memberId, payload);
    return (response as any)?.data ?? response;
  };

  const setUserStatus = async (memberId: number, data: { status: number }) => {
    const status = data.status === 1 ? "active" : "disabled";
    return updateUser(memberId, { status });
  };

  const deleteUser = async (memberId: number) => {
    await setUserStatus(memberId, { status: 0 });
    return { ok: true };
  };

  return {
    getUsers,
    createSystemUser,
    updateUser,
    deleteUser,
    setUserStatus,
    addUserToTenant: async () => ({ member_id: 0 }),
    forceLogout: async () => ({ ok: true }),
  };
};
