export default defineNuxtPlugin(() => {
  const { initTheme, listenToParentMessages } = useTheme();

  // 初始化主题
  initTheme();

  // 监听外部消息
  listenToParentMessages();
});
