import { useAuth } from "~/composables/useAuth";

export default defineNuxtRouteMiddleware(() => {
  const config = useRuntimeConfig();
  const auth = useAuth();
  const insidePowerX =
    config.public?.insidePowerX === true ||
    config.public?.insidePowerX === "true";
  const delegated = auth.delegatedIAM?.value ?? insidePowerX;
  if (!delegated) return;

  const blocked = new Set([
    "/scrm/org",
    "/scrm/org/",
    "/scrm/org_sync/org",
  ]);

  const path = useRoute().path || "";
  if (blocked.has(path)) {
    return navigateTo("/scrm/org_sync");
  }
});
