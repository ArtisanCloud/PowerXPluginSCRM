// app/plugins/layout.global.ts
import { isPluginAdminPath } from "~/utils/powerx-bridge";

export default defineNuxtPlugin(() => {
  const config = useRuntimeConfig();
  const inside = !!config.public?.insidePowerX;

  addRouteMiddleware(
    "px-layout",
    (to) => {
      // Host 内，并且真实路径位于插件管理前缀下时才切换 embedded 布局
      const useEmbedded = inside && isPluginAdminPath(to.path);
      setPageLayout(useEmbedded ? "embedded" : "default");
    },
    { global: true }
  );
});
