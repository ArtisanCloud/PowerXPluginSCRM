import { useRuntimeConfig } from "#imports";

export default defineNuxtPlugin(() => {
  const auth = useAuth();
  const runtimeConfig = useRuntimeConfig();
  const providerMode = String(runtimeConfig.public?.providerMode || "").trim().toLowerCase();
  const delegated = providerMode === "delegated";

  auth.setProviderModeFlags?.(delegated);

  if (process.client) {
    auth.initAuth();
  }
});
