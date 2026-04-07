# Quickstart: Social Channel Governance

## Prerequisites

- Backend running on the plugin server.
- Web-admin running with access to the plugin API base.

## Run Backend

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/backend
go run ./cmd/plugin
```

## Run Web Admin

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.scrm/web-admin
npm run dev
```

## Verify

1. Open `http://localhost:3100/scrm/social_channel_governance/account-permission`.
2. Confirm the Channel Accounts list loads under the admin UI.
3. Create a channel account and verify it appears in the list with status.
4. Update channel account members and confirm the account updates in the list.
5. Update channel account capabilities and confirm the update response persists.

## Callback Reliability Verification (SaaS)

1. Trigger duplicated OpenWork callback delivery (`create_auth`) with the same `auth_code`.
2. Confirm only one callback task performs real auth exchange; duplicate callbacks are idempotent hits.
3. Validate metrics at `/api/v1/admin/runtime/metrics`:
   - `plugin_openwork_callback_total`
   - `plugin_openwork_callback_idempotent_hits_total`
   - `plugin_openwork_auth_complete_total`
4. If `40078 invalid auth_code` appears, follow runbook:
   - `specs/001-social-channel-governance/openwork-invalid-auth-code-runbook.md`
