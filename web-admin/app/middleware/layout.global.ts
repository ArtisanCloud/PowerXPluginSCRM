// app/plugins/layout.global.ts
import { isPluginAdminPath } from "~/utils/powerx-bridge";

export default defineNuxtPlugin(() => {
  addRouteMiddleware(
    "px-layout",
    (to) => {
      // _p 插件管理路径必须使用 iframe 内部滚动布局；本地代理环境不一定设置 insidePowerX。
      const useEmbedded = isPluginAdminPath(to.path);
      setPageLayout(useEmbedded ? "embedded" : "default");
    },
    { global: true }
  );
});
