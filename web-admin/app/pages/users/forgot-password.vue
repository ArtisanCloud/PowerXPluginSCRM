<script setup lang="ts">
definePageMeta({
  layout: "default",
  fullBleed: true,
  public: true,
});

const { t } = useI18n();
const localePath = useLocalePath();

// 表单数据
const form = reactive({
  email: "",
});

// 表单验证状态
const loading = ref(false);
const error = ref("");
const success = ref(false);
const emailSent = ref(false);

// 表单验证
const validateForm = () => {
  if (!form.email.trim()) {
    error.value = t("auth.forgot.requiredEmail");
    return false;
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
    error.value = t("auth.invalidEmail");
    return false;
  }
  return true;
};

// 发送重置密码邮件
const handleSendResetEmail = async () => {
  if (!validateForm()) return;

  loading.value = true;
  error.value = "";

  try {
    // 这里添加实际的发送重置密码邮件逻辑
    await new Promise((resolve) => setTimeout(resolve, 2000)); // 模拟API调用

    emailSent.value = true;
    success.value = true;
  } catch (err) {
    error.value = t("auth.forgot.sendError");
  } finally {
    loading.value = false;
  }
};

// 重新发送邮件
const handleResendEmail = async () => {
  emailSent.value = false;
  success.value = false;
  await handleSendResetEmail();
};
</script>

<template>
  <div
    class="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 flex items-center justify-center p-4"
  >
    <div class="max-w-md w-full">
      <!-- 返回登录链接 -->
      <div class="mb-6 flex justify-between items-center">
        <NuxtLink
          :to="$localePath('/users/login')"
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
          {{ $t("auth.backToLogin") }}
        </NuxtLink>
      </div>

      <!-- 忘记密码卡片 -->
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
              {{ emailSent
                  ? $t("auth.forgot.sentTitleShort")
                  : $t("auth.forgot.title") }}
            </h2>
            <p class="text-gray-600 text-sm">
              {{ emailSent
                  ? $t("auth.forgot.sentSubtitle")
                  : $t("auth.forgot.subtitle") }}
            </p>
          </div>
        </template>

        <div class="px-6 pb-6">
          <!-- 成功状态 -->
          <div v-if="emailSent" class="text-center space-y-6">
            <!-- 成功图标 -->
            <div class="flex justify-center">
              <div
                class="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center"
              >
                <UIcon
                  name="i-heroicons-envelope"
                  class="w-8 h-8 text-green-600"
                />
              </div>
            </div>

            <!-- 成功提示 -->
            <UAlert
              color="success"
              variant="soft"
              :title="$t('auth.forgot.sentTitle', { email: form.email })"
              :description="$t('auth.forgot.sentDesc')"
              class="text-left"
            />

            <!-- 操作按钮 -->
            <div class="space-y-3">
              <UButton
                block
                size="lg"
                :loading="loading"
                @click="handleResendEmail"
                variant="outline"
              >
                {{ loading
                    ? $t("auth.forgot.sending")
                    : $t("auth.forgot.resend") }}
              </UButton>

              <NuxtLink :to="localePath('/users/login')">
                <UButton block size="lg" variant="soft">
                  {{ $t("auth.backToLogin") }}
                </UButton>
              </NuxtLink>
            </div>

            <!-- 提示信息 -->
            <div class="text-xs text-gray-500 space-y-2">
              <p>{{ $t("auth.forgot.tips.expire") }}</p>
              <p>{{ $t("auth.forgot.tips.spam") }}</p>
              <p>{{ $t("auth.forgot.tips.email") }}</p>
            </div>
          </div>

          <!-- 表单状态 -->
          <form v-else @submit.prevent="handleSendResetEmail" class="space-y-5">
            <!-- 错误提示 -->
            <UAlert
              v-if="error"
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

            <!-- 邮箱输入 -->
            <div class="mb-6">
              <label
                for="email"
                class="block text-sm font-medium text-gray-700 mb-3"
              >
                {{ $t("auth.forgot.emailLabel") }}
                <span class="text-red-500">*</span>
              </label>
              <UInput
                id="email"
                name="email"
                v-model="form.email"
                type="email"
                :placeholder="$t('auth.forgot.emailPlaceholder')"
                size="lg"
                :disabled="loading"
                class="w-full"
                icon="i-heroicons-envelope"
              />
              <p class="text-xs text-gray-500 mt-2">
                {{ $t("auth.forgot.emailHint") }}
              </p>
            </div>

            <!-- 发送按钮 -->
            <div class="pt-2">
              <UButton
                type="submit"
                block
                size="lg"
                :loading="loading"
                :disabled="!form.email.trim()"
                class="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700"
              >
                {{ loading
                    ? $t("auth.forgot.sending")
                    : $t("auth.forgot.send") }}
              </UButton>
            </div>
          </form>

          <!-- 其他选项 -->
          <div class="text-center mt-6 pt-4 border-t border-gray-200">
            <p class="text-gray-600 text-sm mb-3">还有其他问题？</p>
            <div class="flex flex-col sm:flex-row gap-2 justify-center">
              <NuxtLink
                :to="$localePath('/users/login')"
                class="text-blue-600 hover:text-blue-700 font-medium text-sm"
              >
                返回登录
              </NuxtLink>
              <span class="hidden sm:inline text-gray-400">|</span>
              <NuxtLink
                :to="$localePath('/users/register')"
                class="text-blue-600 hover:text-blue-700 font-medium text-sm"
              >
                创建新账户
              </NuxtLink>
            </div>
          </div>
        </div>
      </UCard>
    </div>
  </div>
</template>
