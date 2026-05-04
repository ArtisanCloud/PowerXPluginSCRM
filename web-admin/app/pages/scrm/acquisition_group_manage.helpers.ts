import type { GroupCustomerRelatedChatRecord } from "~/composables/api/services/acquisition";

export function shouldLoadCustomerDetailTab(tab: string, open: boolean, hasSelectedCustomer: boolean): boolean {
  if (!open || !hasSelectedCustomer) return false;
  return tab === "relation" || tab === "activity" || tab === "followups";
}

export function normalizeCustomerRelations(
  rows: GroupCustomerRelatedChatRecord[],
  currentChatID = ""
): GroupCustomerRelatedChatRecord[] {
  const current = String(currentChatID || "").trim();
  const uniq = new Map<string, GroupCustomerRelatedChatRecord>();
  for (const row of rows || []) {
    const chatID = String(row?.chat_id || "").trim();
    if (!chatID) continue;
    if (!uniq.has(chatID)) uniq.set(chatID, { ...row, chat_id: chatID });
  }
  const items = Array.from(uniq.values());
  items.sort((a, b) => {
    const aCurrent = String(a.chat_id || "") === current ? 0 : 1;
    const bCurrent = String(b.chat_id || "") === current ? 0 : 1;
    if (aCurrent !== bCurrent) return aCurrent - bCurrent;
    const aTime = Date.parse(String(a.updated_at || ""));
    const bTime = Date.parse(String(b.updated_at || ""));
    const aV = Number.isNaN(aTime) ? 0 : aTime;
    const bV = Number.isNaN(bTime) ? 0 : bTime;
    return bV - aV;
  });
  return items;
}
