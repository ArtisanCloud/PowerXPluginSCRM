export default defineNuxtPlugin(async () => {
  const overlay = useOverlay();
  const { insidePowerX, hostAvailable, local } = useGlobalLoadingAdapter();

  if (insidePowerX && hostAvailable) {
    return;
  }

  const modal = overlay.create(
    defineAsyncComponent(() => import("~/components/GlobalLoadingModal.vue")),
    { props: { message: "加载中…" } }
  );

  const opened = ref(false);

  watch([local.message, local.progress], ([m, p]) => {
    if (opened.value) modal.patch({ message: m, progress: p });
  });

  watch(
    local.visible,
    async (v) => {
      if (v && !opened.value) {
        const instance = modal.open({
          message: local.message.value,
          progress: local.progress.value,
        });
        opened.value = true;
        await instance.result;
        opened.value = false;
      } else if (!v && opened.value) {
        modal.close();
      }
    },
    { immediate: true }
  );
});
