import { getStoredTenantUUID } from "~/utils/tenant-context";
import { useIAMService, type MemberRecord } from "./iamService";

export interface Member extends MemberRecord {}

export interface MemberQuery {
  tenant_uuid?: string;
  keyword?: string;
  status?: string;
  page?: number;
  page_size?: number;
}

const normalize = (records: MemberRecord[] = []): Member[] => {
  return records.map((record) => ({
    ...record,
    display_name: record.display_name || record.email || record.username,
  }));
};

export function useMemberService() {
  const iamService = useIAMService();

  const resolveTenantUuid = (tenantUuid?: string): string | null => {
    const candidate =
      tenantUuid?.trim() ||
      getStoredTenantUUID() ||
      (process.client ? localStorage.getItem("px_current_tenant_uuid") : "") ||
      "";
    return candidate || null;
  };

  const fetchMembers = async (
    query: MemberQuery = {}
  ): Promise<Member[]> => {
    const tenantUuid = resolveTenantUuid(query.tenant_uuid);
    if (!tenantUuid) {
      return [];
    }

    const response = await iamService.listMembers({
      tenantUuid,
      query: query.keyword,
      status: query.status,
      page: query.page,
      pageSize: query.page_size,
    });
    const payload = (response as any)?.data ?? response ?? {};
    const items =
      payload?.data?.items ?? payload?.items ?? payload?.data ?? payload ?? [];
    return normalize(Array.isArray(items) ? items : []);
  };

  return {
    listAll: async (tenantUuid?: string) => fetchMembers({ tenant_uuid: tenantUuid }),
    getMemberList: fetchMembers,
    createMember: async (payload: Record<string, any>) => {
      const tenantUuid = resolveTenantUuid(payload.tenant_uuid);
      if (!tenantUuid) {
        throw new Error("请先选择租户");
      }
      const response = await iamService.createMember({
        ...payload,
        tenant_uuid: tenantUuid,
      });
      return (response as any)?.data ?? response;
    },
    updateMember: async (id: number, payload: Record<string, any>) => {
      const response = await iamService.updateMember(id, payload);
      return (response as any)?.data ?? response;
    },
    deleteMember: async (_id: number) => {
      console.warn("deleteMember not supported via IAM service.");
      return false;
    },
  };
}
