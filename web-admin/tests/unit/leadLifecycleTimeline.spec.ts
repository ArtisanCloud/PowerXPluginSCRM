import { describe, expect, it } from "vitest";
import {
  mergeLeadLifecycleTimelineItems,
  resolveLeadAuditOperatorName,
} from "../../app/utils/leadLifecycleTimeline";

describe("lead lifecycle timeline", () => {
  it("merges manual activities with audit traces in descending time order", () => {
    const activities = [
      { id: "activity:1", kind: "activity", time: "2026-08-29T10:01:00Z" },
    ];
    const traces = [
      { id: "status:1", kind: "status", time: "2026-08-29T10:00:00Z" },
      { id: "assignment:1", kind: "assignment", time: "2026-08-29T10:02:00Z" },
    ];

    expect(mergeLeadLifecycleTimelineItems(activities, traces).map((item) => item.id)).toEqual([
      "assignment:1",
      "activity:1",
      "status:1",
    ]);
  });

  it("keeps items with missing time at the end", () => {
    const result = mergeLeadLifecycleTimelineItems(
      [{ id: "missing", time: "" }],
      [{ id: "dated", time: "2026-08-29T10:00:00Z" }],
    );

    expect(result.map((item) => item.id)).toEqual(["dated", "missing"]);
  });

  it("places attachment audit events in the same descending timeline", () => {
    const result = mergeLeadLifecycleTimelineItems(
      [{ id: "activity:manual", kind: "activity", time: "2026-08-29T10:00:00Z" }],
      [{ id: "activity:attachment", kind: "attachment", time: "2026-08-29T10:03:00Z" }],
      [{ id: "activity:attachment-deleted", kind: "attachmentDeleted", time: "2026-08-29T10:04:00Z" }],
      [{ id: "status:1", kind: "status", time: "2026-08-29T10:01:00Z" }],
    );

    expect(result.map((item) => item.id)).toEqual([
      "activity:attachment-deleted",
      "activity:attachment",
      "status:1",
      "activity:manual",
    ]);
  });

  it("resolves the audit operator from the member UUID without exposing the UUID", () => {
    const memberUUID = "30000000-0000-4000-8000-000000000001";
    const names = new Map([[memberUUID, "王采集"]]);

    expect(resolveLeadAuditOperatorName(
      { operator_member_uuid: memberUUID },
      names,
      "",
      "",
      "系统记录",
    )).toBe("王采集");
  });

  it("uses the authenticated current member name when the directory is not loaded", () => {
    const memberUUID = "30000000-0000-4000-8000-000000000001";

    expect(resolveLeadAuditOperatorName(
      { operator_member_uuid: memberUUID },
      new Map(),
      memberUUID,
      "当前操作人",
      "系统记录",
    )).toBe("当前操作人");
  });

  it("keeps system records explicit when no trusted operator can be resolved", () => {
    expect(resolveLeadAuditOperatorName(
      {},
      new Map(),
      "",
      "",
      "系统记录",
    )).toBe("系统记录");
  });
});
