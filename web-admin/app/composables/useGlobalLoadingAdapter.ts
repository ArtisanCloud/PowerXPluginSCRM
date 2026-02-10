type ShowOptions = {
  lock?: boolean;
  minMs?: number;
  message?: string;
  progress?: number;
};

type HostGlobalLoading = {
  show?: (opts?: ShowOptions) => void;
  hide?: () => void;
  lock?: () => void;
  unlock?: () => void;
  setMessage?: (message: string) => void;
  setProgress?: (progress?: number) => void;
};

const useLocalGL_AutoVisible = () =>
  useState<boolean>("scrm:gl:autoVisible", () => false);
const useLocalGL_ManualVisible = () =>
  useState<boolean>("scrm:gl:manualVisible", () => false);
const useLocalGL_LockCount = () =>
  useState<number>("scrm:gl:lockCount", () => 0);
const useLocalGL_Message = () =>
  useState<string>("scrm:gl:message", () => "加载中…");
const useLocalGL_Progress = () =>
  useState<number | undefined>("scrm:gl:progress", () => undefined);

let minHideTimer: ReturnType<typeof setTimeout> | null = null;
let minHideAt = 0;

const resolveHostGlobalLoading = (): HostGlobalLoading | null => {
  if (typeof window === "undefined") return null;
  const parent = window.parent as any;
  if (!parent || parent === window) return null;
  return (
    parent.__PX_GLOBAL_LOADING__ ||
    parent.__POWERX_GLOBAL_LOADING__ ||
    null
  );
};

export function useLocalGlobalLoading() {
  const autoVisible = useLocalGL_AutoVisible();
  const manualVisible = useLocalGL_ManualVisible();
  const lockCount = useLocalGL_LockCount();
  const message = useLocalGL_Message();
  const progress = useLocalGL_Progress();

  const visible = computed(
    () => manualVisible.value || autoVisible.value || lockCount.value > 0
  );

  function setMessage(msg: string) {
    message.value = msg;
  }

  function setProgress(value?: number) {
    progress.value = value;
  }

  function show(opts: ShowOptions = {}) {
    if (opts.message) message.value = opts.message;
    if (typeof opts.progress === "number") progress.value = opts.progress;
    manualVisible.value = true;
    if (opts.lock) lock();
    if (opts.minMs && opts.minMs > 0) {
      minHideAt = Date.now() + opts.minMs;
      if (minHideTimer) clearTimeout(minHideTimer);
      minHideTimer = null;
    }
  }

  function hide() {
    const remaining = minHideAt - Date.now();
    const doHide = () => {
      manualVisible.value = false;
      progress.value = undefined;
    };
    if (remaining > 0) {
      if (minHideTimer) clearTimeout(minHideTimer);
      minHideTimer = setTimeout(() => {
        minHideTimer = null;
        doHide();
      }, remaining);
    } else {
      doHide();
    }
  }

  function lock() {
    lockCount.value++;
  }

  function unlock() {
    lockCount.value = Math.max(0, lockCount.value - 1);
  }

  return {
    visible,
    message,
    progress,
    show,
    hide,
    lock,
    unlock,
    setMessage,
    setProgress,
  };
}

export function useGlobalLoadingAdapter() {
  const runtimeConfig = useRuntimeConfig();
  const insidePowerX = Boolean(runtimeConfig.public?.insidePowerX);
  const host = insidePowerX ? resolveHostGlobalLoading() : null;
  const local = useLocalGlobalLoading();

  const show = (opts: ShowOptions = {}) => {
    if (host?.show) {
      host.show(opts);
      if (opts.lock && host.lock) host.lock();
      return;
    }
    local.show(opts);
  };

  const hide = () => {
    if (host?.hide) {
      host.hide();
      return;
    }
    local.hide();
  };

  const lock = () => {
    if (host?.lock) {
      host.lock();
      return;
    }
    local.lock();
  };

  const unlock = () => {
    if (host?.unlock) {
      host.unlock();
      return;
    }
    local.unlock();
  };

  const setMessage = (message: string) => {
    if (host?.setMessage) {
      host.setMessage(message);
      return;
    }
    local.setMessage(message);
  };

  const setProgress = (progress?: number) => {
    if (host?.setProgress) {
      host.setProgress(progress);
      return;
    }
    local.setProgress(progress);
  };

  return {
    insidePowerX,
    hostAvailable: Boolean(host),
    show,
    hide,
    lock,
    unlock,
    setMessage,
    setProgress,
    local,
  };
}
