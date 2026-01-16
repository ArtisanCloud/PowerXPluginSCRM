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
2. Confirm the page loads and shows the governance placeholder.
3. Once APIs are wired, verify you can create an account and see it in the list.
