export type OpenWorkAuthStatus = "idle" | "pending" | "authorized" | "failed" | "expired";

export function buildOpenWorkQrCodeUrl(authorizeUrl: string): string {
  const raw = authorizeUrl.trim();
  if (!raw) {
    return "";
  }
  return `https://quickchart.io/qr?size=320&margin=1&text=${encodeURIComponent(raw)}`;
}

export function normalizeOpenWorkAuthStatus(status?: string): OpenWorkAuthStatus {
  switch ((status || "").trim()) {
    case "pending":
    case "authorized":
    case "failed":
    case "expired":
      return status as OpenWorkAuthStatus;
    default:
      return "pending";
  }
}
