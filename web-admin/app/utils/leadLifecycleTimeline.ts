export type LeadLifecycleTimedItem = {
  time?: string;
};

export type LeadLifecycleAttachmentAudit = {
  attachment_uuid: string;
  file_name: string;
  content_type?: string;
  file_size?: number;
  storage_provider?: string;
};

export type LeadLifecycleTimelineItem = LeadLifecycleTimedItem & {
  id: string;
  kind: "activity" | "status" | "assignment" | "source" | "attachment" | "attachmentDeleted";
  title: string;
  subtitle: string;
  description: string;
  time: string;
  stageLabel: string;
  operator: string;
  actionLabel: string;
  badgeLabel: string;
  fields: { label: string; value: string }[];
  bodyLabel?: string;
  body?: string;
  footerLabel?: string;
  footer?: string;
  attachment?: LeadLifecycleAttachmentAudit;
};

export const mergeLeadLifecycleTimelineItems = <T extends LeadLifecycleTimedItem>(
  ...groups: T[][]
): T[] => groups
  .flat()
  .sort((left, right) => new Date(right.time || 0).getTime() - new Date(left.time || 0).getTime());

export const resolveLeadAuditOperatorName = (
  payload: Record<string, unknown> | undefined,
  memberNames: ReadonlyMap<string, string>,
  currentMemberUUID: string | null | undefined,
  currentMemberName: string | null | undefined,
  systemLabel: string,
): string => {
  const explicitName = String(payload?.operator_display_name || "").trim();
  if (explicitName) return explicitName;

  const memberUUID = String(payload?.operator_member_uuid || "").trim().toLowerCase();
  if (!memberUUID) return systemLabel;

  const directoryName = String(memberNames.get(memberUUID) || "").trim();
  if (directoryName) return directoryName;

  if (memberUUID === String(currentMemberUUID || "").trim().toLowerCase()) {
    return String(currentMemberName || "").trim() || systemLabel;
  }

  return systemLabel;
};
