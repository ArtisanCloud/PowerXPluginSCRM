<script setup lang="ts">
import { computed } from "vue";
import { useAuthService } from "~/composables/api/services/authService";

definePageMeta({
  layout: "default",
  fullBleed: true,
  public: true,
});

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const auth = useAuth();
const { setAuth, consumeAuthError } = auth;
const isDelegatedMode = computed(() => auth.delegatedIAM?.value ?? false);
const runtimeConfig = useRuntimeConfig();
const insidePowerX = computed(
  () =>
    runtimeConfig.public?.insidePowerX === true ||
    runtimeConfig.public?.insidePowerX === "true"
);

const sanitizeRedirectTo = (raw: unknown) => {
  if (typeof raw !== "string" || !raw.trim()) return "/";
  const redirect = raw.trim();

  // Avoid open redirects / malformed input
  if (/^https?:\/\//i.test(redirect)) return "/";

  const pathOnly = redirect.startsWith("/") ? redirect : "/";

  // Never redirect back to login (prevents loops)
  if (/\/users\/login(?:$|[?#/])/i.test(pathOnly)) return "/";

  // In standalone preview, strip the PowerX plugin admin base prefix.
  if (!insidePowerX.value) {
    const match = pathOnly.match(/^\/_p\/[^/]+\/admin(\/.*)?$/);
    if (match) {
      const stripped = match[1] || "/";
      if (/\/users\/login(?:$|[?#/])/i.test(stripped)) return "/";
      return stripped;
    }
  }

  return pathOnly;
};

// 导入认证服务
const { login } = useAuthService();

// 表单数据
const form = reactive({
  identifier: "",
  password: "",
  remember: false,
});

// 表单验证状态
const loading = ref(false);
const error = ref("");

// 登录处理
const handleLogin = async () => {
  if (isDelegatedMode.value) {
    error.value = t("auth.delegated.loginDisabled");
    return;
  }
  if (!form.identifier || !form.password) {
    error.value = t("auth.required");
    return;
  }

  loading.value = true;
  error.value = "";

  try {
    // 调用登录API
    const response = await login({
      tenant: "",
      identifier: form.identifier,
      password: form.password,
    });

    // console.log("登录结果:", response);

    if (response.success) {
      // 保存认证信息
      setAuth(response.data);

      // 获取重定向URL
      const redirectTo = sanitizeRedirectTo(route.query.redirect);

      // 登录成功后跳转
      await navigateTo(redirectTo);
    } else {
      error.value = response.message || t("auth.loginFailed");
    }
  } catch (err: any) {
    console.error("登录错误:", err);
    error.value = err.response?.data?.message || t("auth.loginFailed");
  } finally {
    loading.value = false;
  }
};

// 忘记密码
const handleForgotPassword = () => {
  // 这里添加忘记密码逻辑
  console.log("忘记密码");
};

onMounted(() => {
  const stored = consumeAuthError?.();
  if (stored) {
    error.value = stored;
  }
  if (typeof route.query.error === "string" && route.query.error) {
    error.value = route.query.error;
    router.replace({ path: route.path, query: { ...route.query, error: undefined } });
  }
});
</script>

<template>
  <div
    class="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 flex items-center justify-center p-4"
  >
    <div class="max-w-md w-full">
      <!-- 返回首页链接 -->
      <div class="mb-6 flex justify-between items-center">
        <NuxtLink
          :to="$localePath('/')"
          class="inline-flex items-center text-gray-600 hover:text-blue-600 transition-colors text-sm"
        >
          <svg
            class="w-4 h-4 mr-2"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M15 19l-7-7 7-7"
            ></path>
          </svg>
          {{ $t("auth.backToHome") }}
        </NuxtLink>
      </div>

      <!-- 登录卡片 -->
      <UCard class="shadow-xl border-0">
        <template #header>
          <div class="text-center py-6">
            <!-- Logo -->
            <h1
              class="text-3xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent mb-3"
            >
              PowerX
            </h1>
            <h2 class="text-xl font-semibold text-gray-900 mb-2">
              {{ $t("auth.welcomeBack") }}
            </h2>
            <p class="text-gray-600 text-sm">{{ $t("auth.loginSubtitle") }}</p>
          </div>
        </template>

        <div class="px-6 pb-6">
          <form @submit.prevent="handleLogin" class="space-y-5">
            <!-- 错误提示 -->
            <div v-if="error" role="alert">
              <UAlert
                color="error"
                variant="soft"
                :title="error"
                :close-button="{
                  icon: 'i-heroicons-x-mark-20-solid',
                  color: 'gray',
                  variant: 'link',
                  padded: false,
                }"
                @close="error = ''"
                class="mb-1"
              />
            </div>

            <UAlert
              v-if="isDelegatedMode"
              color="warning"
              variant="soft"
              :title="$t('auth.delegated.bannerTitle')"
              :description="$t('auth.delegated.loginDisabled')"
              class="mb-1"
            />

            <!-- 邮箱输入 -->
            <div class="mb-6">
              <label
                for="identifier"
                class="block text-sm font-medium text-gray-700 mb-3"
              >
                {{ $t("auth.identifier") }} <span class="text-red-500">*</span>
              </label>
              <UInput
                id="identifier"
                name="identifier"
                v-model="form.identifier"
                type="text"
                :placeholder="$t('auth.identifier')"
                size="lg"
                :disabled="loading || isDelegatedMode"
                class="w-full"
              />
            </div>

            <!-- 密码输入 -->
            <div class="mb-6">
              <label
                for="password"
                class="block text-sm font-medium text-gray-700 mb-3"
              >
                {{ $t("auth.password") }} <span class="text-red-500">*</span>
              </label>
              <UInput
                id="password"
                name="password"
                v-model="form.password"
                type="password"
                :placeholder="$t('auth.password')"
                size="lg"
                :disabled="loading || isDelegatedMode"
                class="w-full"
              />
            </div>

            <!-- 记住我和忘记密码 -->
            <div class="flex items-center justify-between mb-6">
              <UCheckbox
                v-model="form.remember"
                :disabled="loading || isDelegatedMode"
                :label="$t('auth.remember')"
              />
              <NuxtLink
                :to="$localePath('/users/forgot-password')"
                class="text-sm text-blue-600 hover:text-blue-700"
              >
                {{ $t("auth.login.forgotPassword") }}
              </NuxtLink>
            </div>

            <!-- 登录按钮 -->
            <div class="pt-2">
              <UButton
                type="submit"
                block
                size="lg"
                :loading="loading"
                :disabled="isDelegatedMode"
                class="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700"
              >
                {{ loading ? $t("auth.signingIn") : $t("auth.loginButton") }}
              </UButton>
            </div>
          </form>

          <!-- 注册链接 -->
          <div class="text-center mt-6 pt-4 border-t border-gray-200">
            <p class="text-gray-600 text-sm">
              {{ $t("auth.login.noAccount") }}
              <NuxtLink
                :to="$localePath('/users/register')"
                class="text-blue-600 hover:text-blue-700 font-medium"
              >
                {{ $t("auth.signUpNow") }}
              </NuxtLink>
            </p>
          </div>
        </div>
      </UCard>
    </div>
  </div>
</template>
