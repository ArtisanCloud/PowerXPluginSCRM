import { useRuntimeConfig } from "#imports";

export default defineNuxtPlugin(() => {
  const auth = useAuth();
  const runtimeConfig = useRuntimeConfig();
  const iamMode = String(runtimeConfig.public?.iamMode || "").trim().toLowerCase();
  const delegated =
    iamMode === "delegated" ||
    (iamMode === "" &&
      (runtimeConfig.public?.insidePowerX === true ||
        runtimeConfig.public?.insidePowerX === "true"));

  auth.setIAMModeFlags?.(delegated);

  if (process.client) {
    auth.initAuth();
  }
});
