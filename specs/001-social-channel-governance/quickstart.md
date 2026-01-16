# Quickstart: Social Channel Governance

## Prerequisites

- Backend running on the plugin server.
- Web-admin running with access to the plugin API base.

## Run Backend

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/backend
go run ./cmd/plugin
```

## Run Web Admin

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugin.scrm/web-admin
npm run dev
```

## Verify

1. Open `http://localhost:3100/scrm/social_channel_governance/account-permission`.
2. Confirm the Channel Accounts list loads under the admin UI.
3. Create a channel account and verify it appears in the list with status.
4. Update channel account members and confirm the account updates in the list.
5. Update channel account capabilities and confirm the update response persists.
