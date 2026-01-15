import { useRuntimeConfig } from "#imports";

export default defineNuxtPlugin(() => {
  const auth = useAuth();
  const runtimeConfig = useRuntimeConfig();
  const insidePowerX =
    runtimeConfig.public?.insidePowerX === true ||
    runtimeConfig.public?.insidePowerX === "true";

  auth.setIAMModeFlags?.(insidePowerX);

  if (process.client) {
    auth.initAuth();
  }
});
