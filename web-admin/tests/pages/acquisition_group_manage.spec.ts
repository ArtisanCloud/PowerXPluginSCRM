import { describe, expect, it } from "vitest";
import { normalizeCustomerRelations, shouldLoadCustomerDetailTab } from "../../app/pages/scrm/acquisition_group_manage.helpers";

describe("acquisition_group_manage interactions", () => {
  it("shouldLoadCustomerDetailTab only enables remote tabs when modal is ready", () => {
    expect(shouldLoadCustomerDetailTab("relation", true, true)).toBe(true);
    expect(shouldLoadCustomerDetailTab("activity", true, true)).toBe(true);
    expect(shouldLoadCustomerDetailTab("followups", true, true)).toBe(true);

    expect(shouldLoadCustomerDetailTab("basic", true, true)).toBe(false);
    expect(shouldLoadCustomerDetailTab("relation", false, true)).toBe(false);
    expect(shouldLoadCustomerDetailTab("relation", true, false)).toBe(false);
  });

  it("normalizeCustomerRelations deduplicates and keeps current chat first", () => {
    const rows = normalizeCustomerRelations(
      [
        { chat_id: "chat-b", updated_at: "2026-04-20T12:00:00Z" },
        { chat_id: "chat-a", updated_at: "2026-04-20T10:00:00Z" },
        { chat_id: "chat-b", updated_at: "2026-04-20T09:00:00Z" },
      ],
      "chat-a"
    );

    expect(rows).toHaveLength(2);
    expect(rows[0].chat_id).toBe("chat-a");
    expect(rows[1].chat_id).toBe("chat-b");
  });
});
