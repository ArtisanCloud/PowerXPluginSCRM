import { defineStore } from "pinia";
import type {
  Department,
  MemberRecord,
  TenantSummary,
} from "~/composables/api/services/iamService";

export const useIAMStore = defineStore("iam.organization", () => {
  const tenants = ref<TenantSummary[]>([]);
  const departments = ref<Department[]>([]);
  const members = ref<MemberRecord[]>([]);
  const activeTenantUuid = ref("");

  const setTenants = (items: TenantSummary[]) => {
    tenants.value = items ?? [];
    if (!activeTenantUuid.value && tenants.value.length > 0) {
      activeTenantUuid.value = tenants.value[0].key;
    } else if (
      activeTenantUuid.value &&
      !tenants.value.some((tenant) => tenant.key === activeTenantUuid.value)
    ) {
      activeTenantUuid.value = tenants.value[0]?.key ?? "";
    }
  };

  const setDepartments = (items: Department[]) => {
    departments.value = items ?? [];
  };

  const setMembers = (items: MemberRecord[]) => {
    members.value = items ?? [];
  };

  const setActiveTenant = (tenantUuid: string) => {
    activeTenantUuid.value = tenantUuid;
  };

  const activeTenant = computed(() =>
    tenants.value.find((tenant) => tenant.key === activeTenantUuid.value)
  );

  return {
    tenants,
    departments,
    members,
    activeTenantUuid,
    activeTenant,
    setTenants,
    setDepartments,
    setMembers,
    setActiveTenant,
  };
});
