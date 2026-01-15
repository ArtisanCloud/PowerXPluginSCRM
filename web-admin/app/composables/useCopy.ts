import { ref } from "vue";
import { useToast } from "#imports";

type CopyOptions = {
  showToast?: boolean;
  successText?: string;
  failText?: string;
};

export function useCopy(defaultOptions: CopyOptions = { showToast: true }) {
  const copying = ref(false);
  const lastText = ref<string | null>(null);
  const toast = useToast();

  const notify = (ok: boolean, text?: string, opts?: CopyOptions) => {
    const merged = { ...defaultOptions, ...opts };
    if (!merged.showToast || process.server) {
      return;
    }
    toast.add({
      title: ok
        ? merged.successText ?? "已复制"
        : merged.failText ?? "复制失败",
      description: text,
      icon: ok ? "i-heroicons-clipboard" : "i-heroicons-exclamation-triangle",
      color: ok ? undefined : "red",
    });
  };

  const copy = async (text: string, opts?: CopyOptions) => {
    copying.value = true;
    lastText.value = text;
    try {
      if (
        process.client &&
        window.isSecureContext &&
        navigator.clipboard?.writeText
      ) {
        await navigator.clipboard.writeText(text);
        notify(true, text, opts);
        return true;
      }
      if (process.client) {
        const textarea = document.createElement("textarea");
        textarea.value = text;
        textarea.setAttribute("readonly", "");
        textarea.style.position = "fixed";
        textarea.style.top = "-9999px";
        textarea.style.left = "-9999px";
        document.body.appendChild(textarea);
        textarea.select();
        const ok = document.execCommand?.("copy");
        document.body.removeChild(textarea);
        if (ok) {
          notify(true, text, opts);
          return true;
        }
      }
      throw new Error("Clipboard not supported");
    } catch (error) {
      console.error("[useCopy] copy failed:", error);
      notify(false, text, opts);
      return false;
    } finally {
      copying.value = false;
    }
  };

  return { copy, copying, lastText };
}

export function cloneWithFilteredChildren<T extends object>(
  obj: T,
  predicate: (child: any) => boolean
): T {
  const descriptors = Object.getOwnPropertyDescriptors(obj);
  delete descriptors.children;
  const clone = Object.create(Object.getPrototypeOf(obj), descriptors) as T;
  const rawChildren = (obj as any).children;
  Object.defineProperty(clone, "children", {
    enumerable: true,
    configurable: true,
    get() {
      return Array.isArray(rawChildren)
        ? rawChildren.filter(predicate)
        : rawChildren;
    },
  });
  return clone;
}
