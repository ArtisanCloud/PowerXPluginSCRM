import type { OpportunityPipelineGroup } from "~/composables/api/services/opportunity";

export const createPipelineModeOptions = [
  { label: "行业模板", value: "template" },
  { label: "复制现有", value: "copy" },
];

export const fixedStageOptions = [
  { label: "打开", value: "open" },
  { label: "已确认", value: "qualified" },
  { label: "方案", value: "proposal" },
  { label: "谈判", value: "negotiation" },
  { label: "赢单", value: "won" },
  { label: "输单", value: "lost" },
];

export const stageTypeOptions = [
  { label: "推进阶段", value: "active" },
  { label: "赢单终态", value: "won" },
  { label: "输单终态", value: "lost" },
];

export function dedupePipelineGroups(items: OpportunityPipelineGroup[]) {
  const seen = new Set<string>();
  return items.filter((item) => {
    const groupKey = String(item.group_key || "").trim().toLowerCase();
    const key = groupKey ? `key:${groupKey}` : `uuid:${String(item.group_uuid || "").trim().toLowerCase()}`;
    if (!key || key === "uuid:") return true;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}
