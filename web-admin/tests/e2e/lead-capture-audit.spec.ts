import { expect, test, type Page, type Route } from "@playwright/test";

const tenantUUID = "00000000-0000-4000-8000-000000000001";
const otherTenantUUID = "00000000-0000-4000-8000-000000000002";
const memberUUID = "10000000-0000-4000-8000-000000000001";
const leadUUID = "20000000-0000-4000-8000-000000000001";
const attachmentUUID = "30000000-0000-4000-8000-000000000001";
const activityUUID = "40000000-0000-4000-8000-000000000001";
const fileName = "lead-audit-e2e.txt";
const activitySubject = "lead-audit-e2e-subject";
const activityContent = "lead-audit-e2e-content";

const tokenFor = (tid: string) => {
  const encode = (value: object) => Buffer.from(JSON.stringify(value)).toString("base64url");
  return `${encode({ alg: "none", typ: "JWT" })}.${encode({ tid, mid: memberUUID })}.signature`;
};

type TimelineEvent = {
  event_uuid: string;
  event_type: string;
  actor_type: "member" | "system";
  actor_member_uuid?: string;
  stage_key: string;
  action_key: string;
  occurred_at: string;
  data: Record<string, unknown>;
};

const createState = () => ({
  lead: {
    lead_uuid: leadUUID,
    tenant_uuid: tenantUUID,
    display_name: "lead-audit-e2e",
    phone: "13800000000",
    email: "lead-audit-e2e@example.test",
    status: "captured",
    source_channel: "local",
    source_app_type: "manual",
    lead_origin_type: "local",
    created_at: "2026-08-30T12:00:00Z",
    updated_at: "2026-08-30T12:00:00Z",
  },
  attachments: [] as Array<Record<string, unknown>>,
  events: [] as TimelineEvent[],
});

const success = (route: Route, data: unknown, status = 200) =>
  route.fulfill({ status, contentType: "application/json", body: JSON.stringify({ success: status < 400, data }) });

const installAuth = async (page: Page, tenant: string) => {
  const token = tokenFor(tenant);
  await page.addInitScript(({ authToken, expiresAt }) => {
    localStorage.setItem("access_token", authToken);
    localStorage.setItem("token_type", "Bearer");
    localStorage.setItem("expires_at", String(expiresAt));
    localStorage.setItem("expires_in", "3600");
    localStorage.setItem("scope", "scrm");
    document.cookie = `token=${encodeURIComponent(authToken)}; path=/; SameSite=Lax`;
  }, { authToken: token, expiresAt: Date.now() + 3_600_000 });
};

const installStatefulAPI = async (page: Page, state: ReturnType<typeof createState>, visibleTenant = tenantUUID) => {
  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^.*\/api\/v1/, "");
    const method = request.method();
    const now = new Date().toISOString();

    if (path === "/admin/leads" && method === "GET") {
      return success(route, { items: visibleTenant === tenantUUID ? [state.lead] : [] });
    }
    if (path === `/admin/leads/${leadUUID}` && method === "GET") {
      return visibleTenant === tenantUUID
        ? success(route, state.lead)
        : success(route, { code: "LEAD_NOT_FOUND" }, 404);
    }
    if (path === `/admin/leads/${leadUUID}` && method === "PUT") {
      Object.assign(state.lead, request.postDataJSON());
      state.lead.updated_at = now;
      return success(route, state.lead);
    }
    if (path === `/admin/leads/${leadUUID}/node-attachments` && method === "POST") {
      const attachment = {
        attachment_uuid: attachmentUUID,
        tenant_uuid: tenantUUID,
        lead_uuid: leadUUID,
        stage_key: state.lead.status,
        action_key: state.lead.status === "captured" ? "capture" : state.lead.status,
        file_name: fileName,
        content_type: "text/plain",
        file_size: 16,
        storage_provider: "database",
        created_at: now,
      };
      state.attachments = [attachment];
      state.events.unshift({
        event_uuid: "50000000-0000-4000-8000-000000000001",
        event_type: "attachment_uploaded",
        actor_type: "member",
        actor_member_uuid: memberUUID,
        stage_key: "captured",
        action_key: "capture",
        occurred_at: now,
        data: { ...attachment, activity_type: "attachment_uploaded" },
      });
      return success(route, attachment, 201);
    }
    if (path === `/admin/leads/${leadUUID}/node-attachments` && method === "GET") {
      const stage = url.searchParams.get("stage_key");
      return success(route, { items: state.attachments.filter((item) => item.stage_key === stage) });
    }
    if (path === `/admin/leads/${leadUUID}/activities` && method === "POST") {
      const payload = request.postDataJSON();
      const activity = {
        activity_uuid: activityUUID,
        tenant_uuid: tenantUUID,
        lead_uuid: leadUUID,
        activity_type: "manual_activity",
        payload: { ...payload, actor_type: "member", operator_member_uuid: memberUUID },
        created_at: now,
        updated_at: now,
      };
      state.events.unshift({
        event_uuid: activityUUID,
        event_type: "manual_activity",
        actor_type: "member",
        actor_member_uuid: memberUUID,
        stage_key: "captured",
        action_key: "activity",
        occurred_at: now,
        data: { ...activity.payload, activity_uuid: activityUUID, activity_type: "manual_activity" },
      });
      return success(route, activity, 201);
    }
    if (path === `/admin/leads/${leadUUID}/attachments/${attachmentUUID}` && method === "DELETE") {
      state.attachments = [];
      state.events.unshift({
        event_uuid: "50000000-0000-4000-8000-000000000002",
        event_type: "attachment_deleted",
        actor_type: "member",
        actor_member_uuid: memberUUID,
        stage_key: "captured",
        action_key: "capture",
        occurred_at: now,
        data: { attachment_uuid: attachmentUUID, file_name: fileName, activity_type: "attachment_deleted" },
      });
      return success(route, { attachment_uuid: attachmentUUID });
    }
    if (path === `/admin/leads/${leadUUID}/status` && method === "POST") {
      const fromStatus = state.lead.status;
      const { status } = request.postDataJSON();
      state.lead.status = status;
      state.lead.updated_at = now;
      state.events.unshift({
        event_uuid: "60000000-0000-4000-8000-000000000001",
        event_type: "status_changed",
        actor_type: "member",
        actor_member_uuid: memberUUID,
        stage_key: status,
        action_key: "status",
        occurred_at: now,
        data: { from_status: fromStatus, to_status: status },
      });
      return success(route, state.lead);
    }
    if (path === `/admin/leads/${leadUUID}/timeline` && method === "GET") {
      if (visibleTenant !== tenantUUID) return success(route, { code: "LEAD_NOT_FOUND" }, 404);
      const stage = url.searchParams.get("stage_key");
      const eventType = url.searchParams.get("event_type");
      const items = state.events.filter((event) => {
        const stageMatches = !stage || event.stage_key === stage ||
          (event.event_type === "status_changed" && [event.data.from_status, event.data.to_status].includes(stage));
        return stageMatches && (!eventType || event.event_type === eventType);
      });
      return success(route, { items, total: items.length, page: 1, page_size: 20 });
    }
    if (path.includes(`/admin/leads/${leadUUID}/activities/`) && path.endsWith("/attachments") && method === "GET") {
      return success(route, { items: [] });
    }
    if (path.includes(`/admin/leads/${leadUUID}/attachments/`) && path.endsWith("/download")) {
      return visibleTenant === tenantUUID
        ? route.fulfill({ status: 200, contentType: "text/plain", body: "lead-audit-e2e" })
        : success(route, { code: "ATTACHMENT_NOT_FOUND" }, 404);
    }
    return success(route, { items: [] });
  });
};

const openWorkbench = async (page: Page) => {
  await page.locator(`[data-testid="lead-lifecycle-open"][data-lead-uuid="${leadUUID}"]`).click();
  await expect(page.getByTestId("lead-lifecycle-workbench")).toBeVisible();
};

test("lead_capture_audit_persists_across_refresh", async ({ page }) => {
  const state = createState();
  await installAuth(page, tenantUUID);
  await installStatefulAPI(page, state);
  await page.goto("/scrm/lead_capture");
  await openWorkbench(page);

  await page.getByTestId("lead-node-attachment-input").setInputFiles({
    name: fileName,
    mimeType: "text/plain",
    buffer: Buffer.from("lead-audit-e2e"),
  });
  await expect(page.getByTestId("lead-node-attachment-row")).toContainText(fileName);
  await expect(page.getByTestId("lead-timeline-attachment")).toBeVisible();

  await page.getByTestId("lead-activity-open").click();
  await page.locator('input[data-testid="lead-activity-subject"], [data-testid="lead-activity-subject"] input').fill(activitySubject);
  await page.locator('textarea[data-testid="lead-activity-content"], [data-testid="lead-activity-content"] textarea').fill(activityContent);
  await page.getByTestId("lead-activity-submit").click();
  await expect(page.getByTestId("lead-activity-row")).toContainText(activitySubject);
  await expect(page.getByTestId("lead-timeline-activity")).toBeVisible();

  await page.reload();
  await openWorkbench(page);
  await expect(page.getByTestId("lead-node-attachment-row")).toContainText(fileName);
  await expect(page.getByTestId("lead-activity-row")).toContainText(activitySubject);

  await page.getByTestId("lead-node-attachment-delete").click();
  await page.getByTestId("lead-node-attachment-delete-confirm").click();
  await expect(page.getByTestId("lead-node-attachment-row")).toHaveCount(0);
  await expect(page.getByTestId("lead-timeline-attachmentDeleted")).toBeVisible();

  await page.getByTestId("lead-lifecycle-enrich-submit").click();
  await expect(page.getByTestId("lead-timeline-status")).toBeVisible();
});

test("lead_capture_cross_tenant_is_hidden", async ({ page }) => {
  const state = createState();
  await installAuth(page, otherTenantUUID);
  await installStatefulAPI(page, state, otherTenantUUID);
  await page.goto("/scrm/lead_capture");
  await expect(page.locator(`[data-testid="lead-lifecycle-open"][data-lead-uuid="${leadUUID}"]`)).toHaveCount(0);

  const response = await page.evaluate(async ({ targetLeadUUID }) => {
    const authToken = localStorage.getItem("access_token") || "";
    const result = await fetch(`/api/v1/admin/leads/${targetLeadUUID}/timeline`, {
      headers: { Authorization: `Bearer ${authToken}` },
    });
    return result.status;
  }, { targetLeadUUID: leadUUID });
  expect(response).toBe(404);
});
