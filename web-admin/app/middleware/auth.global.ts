import { useAuth } from "~/composables/useAuth";

const PUBLIC_ROUTE_PREFIXES = ["/users"];

export default defineNuxtRouteMiddleware(async (to) => {
  if (!process.client) return;
  const runtimeConfig = useRuntimeConfig();
  const insidePowerX =
    runtimeConfig.public?.insidePowerX === true ||
    runtimeConfig.public?.insidePowerX === "true";

  if (to.meta?.public === true) {
    return;
  }

  if (PUBLIC_ROUTE_PREFIXES.some((prefix) => to.path.startsWith(prefix))) {
    return;
  }

  const auth = useAuth();
  await auth.ensureFreshToken();

  if (!auth.token.value) {
    if (insidePowerX) {
      if (!auth.delegatedError?.value) {
        auth.rememberAuthError?.("PowerX 会话已失效，请回到宿主重新登录");
      }
      return;
    }
    return navigateTo({
      path: "/users/login",
      query: { redirect: to.fullPath },
    });
  }
});
