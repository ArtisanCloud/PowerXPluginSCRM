import { computed } from "vue";
import { useI18n } from "#imports";

export const SCRM_LEAD_LIFECYCLE_TEMPLATE_CODE = "scrm_private_domain_v1";

export const privateDomainStatusValues = [
  "captured",
  "enriched",
  "deduplicated",
  "routed",
  "engaging",
  "qualified_for_handoff",
  "handoff_pending",
  "handoff_accepted",
] as const;

export const terminalStatusValues = [
  "invalid",
  "archived",
  "disconnected",
  "handoff_failed",
] as const;

export type PrivateDomainLeadStatus = (typeof privateDomainStatusValues)[number];
export type TerminalLeadStatus = (typeof terminalStatusValues)[number];
export type LeadLifecycleNodeState = "done" | "current" | "pending" | "terminal";

export interface LeadLifecycleNode {
  key: PrivateDomainLeadStatus;
  templateCode: string;
  label: string;
  shortLabel: string;
  description: string;
  actionLabel: string;
  icon: string;
  color: "primary" | "warning" | "info" | "success" | "error" | "neutral";
  state: LeadLifecycleNodeState;
  count: number;
  toneClass: string;
  indexClass: string;
}

const nodeToneClasses = [
  {
    card: "bg-sky-50/80 hover:bg-sky-50 dark:bg-sky-950/30 dark:hover:bg-sky-950/50",
    index: "bg-sky-100 text-sky-700 dark:bg-sky-900 dark:text-sky-200",
  },
  {
    card: "bg-cyan-50/80 hover:bg-cyan-50 dark:bg-cyan-950/30 dark:hover:bg-cyan-950/50",
    index: "bg-cyan-100 text-cyan-700 dark:bg-cyan-900 dark:text-cyan-200",
  },
  {
    card: "bg-blue-50/80 hover:bg-blue-50 dark:bg-blue-950/30 dark:hover:bg-blue-950/50",
    index: "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-200",
  },
  {
    card: "bg-indigo-50/80 hover:bg-indigo-50 dark:bg-indigo-950/30 dark:hover:bg-indigo-950/50",
    index: "bg-indigo-100 text-indigo-700 dark:bg-indigo-900 dark:text-indigo-200",
  },
  {
    card: "bg-emerald-50/80 hover:bg-emerald-50 dark:bg-emerald-950/30 dark:hover:bg-emerald-950/50",
    index: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900 dark:text-emerald-200",
  },
  {
    card: "bg-green-50/80 hover:bg-green-50 dark:bg-green-950/30 dark:hover:bg-green-950/50",
    index: "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-200",
  },
  {
    card: "bg-amber-50/80 hover:bg-amber-50 dark:bg-amber-950/30 dark:hover:bg-amber-950/50",
    index: "bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-200",
  },
  {
    card: "bg-teal-50/80 hover:bg-teal-50 dark:bg-teal-950/30 dark:hover:bg-teal-950/50",
    index: "bg-teal-100 text-teal-700 dark:bg-teal-900 dark:text-teal-200",
  },
] as const;

const iconByStatus: Record<PrivateDomainLeadStatus, string> = {
  captured: "i-heroicons-inbox-arrow-down",
  enriched: "i-heroicons-sparkles",
  deduplicated: "i-heroicons-square-2-stack",
  routed: "i-heroicons-user-plus",
  engaging: "i-heroicons-chat-bubble-left-right",
  qualified_for_handoff: "i-heroicons-check-badge",
  handoff_pending: "i-heroicons-arrow-path-rounded-square",
  handoff_accepted: "i-heroicons-shield-check",
};

const colorByStatus: Record<string, LeadLifecycleNode["color"]> = {
  captured: "info",
  enriched: "primary",
  deduplicated: "primary",
  routed: "primary",
  engaging: "warning",
  qualified_for_handoff: "warning",
  handoff_pending: "info",
  handoff_accepted: "success",
  invalid: "neutral",
  archived: "neutral",
  disconnected: "error",
  handoff_failed: "error",
};

export function useLeadLifecycle() {
  const { t } = useI18n();

  const statusMeta = (status?: string) => {
    const key = String(status || "captured").trim();
    const known = [...privateDomainStatusValues, ...terminalStatusValues].includes(key as any)
      ? key
      : "captured";
    return {
      label: t(`leadCapture.status.${known}`),
      color: colorByStatus[known] || "neutral",
    };
  };

  const nodeState = (key: PrivateDomainLeadStatus, currentStatus?: string): LeadLifecycleNodeState => {
    const currentIndex = privateDomainStatusValues.indexOf(currentStatus as PrivateDomainLeadStatus);
    const index = privateDomainStatusValues.indexOf(key);
    if (key === "handoff_accepted" && currentStatus === key) return "terminal";
    if (currentIndex < 0) return index === 0 ? "current" : "pending";
    if (index < currentIndex) return "done";
    if (index === currentIndex) return "current";
    return "pending";
  };

  const buildLifecycleNodes = (
    currentStatus?: string,
    countResolver: (status: PrivateDomainLeadStatus) => number = () => 0
  ) => privateDomainStatusValues.map((key, index) => {
    const tone = nodeToneClasses[index % nodeToneClasses.length];
    return {
      key,
      templateCode: SCRM_LEAD_LIFECYCLE_TEMPLATE_CODE,
      label: t(`leadCapture.status.${key}`),
      shortLabel: t(`leadCapture.funnel.short.${key}`),
      description: t(`leadCapture.lifecycle.nodes.${key}.description`),
      actionLabel: t(`leadCapture.lifecycle.nodes.${key}.action`),
      icon: iconByStatus[key],
      color: colorByStatus[key],
      state: nodeState(key, currentStatus),
      count: countResolver(key),
      toneClass: tone.card,
      indexClass: tone.index,
    } satisfies LeadLifecycleNode;
  });

  const statusFilterOptions = computed(() => [
    { label: t("leadCapture.status.all"), value: "__all__" },
    ...privateDomainStatusValues.map((value) => ({
      label: t(`leadCapture.status.${value}`),
      value,
    })),
    ...terminalStatusValues.map((value) => ({
      label: t(`leadCapture.status.${value}`),
      value,
    })),
  ]);

  return {
    templateCode: SCRM_LEAD_LIFECYCLE_TEMPLATE_CODE,
    privateDomainStatusValues,
    terminalStatusValues,
    statusFilterOptions,
    statusMeta,
    buildLifecycleNodes,
  };
}
