<template>
  <div class="space-y-4">
    <div class="flex flex-col md:flex-row md:items-center gap-2">
      <UInput
        v-model="search"
        :placeholder="t('iam.roles.permissionTree.search')"
        icon="i-heroicons-magnifying-glass"
        class="w-full md:flex-1"
      />
      <div class="flex gap-2">
        <UButton size="xs" variant="soft" @click="selectAll" :disabled="!permissions.length">
          {{ t("iam.roles.permissionTree.selectAll") }}
        </UButton>
        <UButton size="xs" variant="soft" color="neutral" @click="clearAll" :disabled="!selected.size">
          {{ t("iam.roles.permissionTree.clear") }}
        </UButton>
      </div>
    </div>

    <div v-if="loading">
      <USkeleton v-for="i in 3" :key="i" class="h-20" />
    </div>

    <div v-else-if="!filteredGroups.length" class="py-8">
      <UAlert
        icon="i-heroicons-information-circle"
        :title="t('iam.roles.permissionTree.emptyTitle')"
        :description="t('iam.roles.permissionTree.emptyDescription')"
        color="neutral"
        variant="soft"
      />
    </div>

    <div v-else class="space-y-3">
      <UCard v-for="group in filteredGroups" :key="group.key">
        <template #header>
          <div class="flex items-start justify-between gap-3">
            <div>
              <p class="text-sm font-semibold">
                {{ group.display }}
              </p>
              <p class="text-xs text-gray-500">
                {{ t("iam.roles.permissionTree.total", { count: group.permissions.length }) }}
              </p>
            </div>
            <UCheckbox
              :model-value="groupState(group).checked"
              :indeterminate="groupState(group).indeterminate"
              @update:modelValue="() => toggleGroup(group)"
            />
          </div>
        </template>

        <div class="space-y-2">
          <div
            v-for="perm in group.permissions"
            :key="perm.id"
            class="flex items-start justify-between rounded border border-gray-100 dark:border-gray-800 px-3 py-2"
          >
            <div class="text-sm space-y-1">
              <p class="font-mono text-xs">
                {{ perm.resource }}:{{ perm.action }}
              </p>
              <p class="text-xs text-gray-500" v-if="perm.description">
                {{ perm.description }}
              </p>
            </div>
            <UCheckbox
              :model-value="isChecked(perm.id)"
              @update:modelValue="() => togglePermission(perm.id)"
            />
          </div>
        </div>
      </UCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import type { PermissionRecord } from "~/composables/api/services/iamService";

const props = withDefaults(
  defineProps<{
    permissions?: PermissionRecord[];
    modelValue?: number[];
    loading?: boolean;
  }>(),
  {
    permissions: () => [],
    modelValue: () => [],
    loading: false,
  }
);

const emit = defineEmits<{
  (e: "update:modelValue", value: number[]): void;
}>();

const { t } = useI18n();
const search = ref("");

const permissions = computed(() => props.permissions ?? []);
const loading = computed(() => props.loading);
const selected = computed(() => new Set(props.modelValue ?? []));

const grouped = computed(() => {
  const map = new Map<
    string,
    {
      key: string;
      display: string;
      permissions: PermissionRecord[];
    }
  >();
  for (const perm of permissions.value) {
    const resource = perm.resource || "global";
    const [namespace, scope] = resource.split(":", 2);
    const key = scope ? `${namespace}:${scope}` : resource;
    if (!map.has(key)) {
      map.set(key, {
        key,
        display: key.replace(`${namespace}:`, `${namespace.toUpperCase()}:`),
        permissions: [],
      });
    }
    map.get(key)!.permissions.push(perm);
  }
  return Array.from(map.values()).sort((a, b) => a.key.localeCompare(b.key));
});

const filteredGroups = computed(() => {
  if (!search.value.trim()) {
    return grouped.value;
  }
  const term = search.value.trim().toLowerCase();
  return grouped.value
    .map((group) => ({
      ...group,
      permissions: group.permissions.filter(
        (perm) =>
          perm.resource.toLowerCase().includes(term) ||
          perm.action.toLowerCase().includes(term) ||
          (perm.description ?? "").toLowerCase().includes(term)
      ),
    }))
    .filter((group) => group.permissions.length > 0);
});

const updateValue = (set: Set<number>) => {
  emit("update:modelValue", Array.from(set.values()));
};

const togglePermission = (id: number) => {
  const next = new Set(selected.value);
  if (next.has(id)) {
    next.delete(id);
  } else {
    next.add(id);
  }
  updateValue(next);
};

const groupState = (group: { permissions: PermissionRecord[] }) => {
  const ids = group.permissions.map((perm) => perm.id);
  const hits = ids.filter((id) => selected.value.has(id)).length;
  return {
    checked: hits > 0 && hits === ids.length,
    indeterminate: hits > 0 && hits < ids.length,
  };
};

const toggleGroup = (group: { permissions: PermissionRecord[] }) => {
  const ids = group.permissions.map((perm) => perm.id);
  const next = new Set(selected.value);
  const current = groupState(group);
  if (!current.checked) {
    ids.forEach((id) => next.add(id));
  } else {
    ids.forEach((id) => next.delete(id));
  }
  updateValue(next);
};

const isChecked = (id: number) => selected.value.has(id);

const selectAll = () => {
  const next = new Set<number>();
  permissions.value.forEach((perm) => next.add(perm.id));
  updateValue(next);
};

const clearAll = () => {
  if (!selected.value.size) {
    return;
  }
  updateValue(new Set<number>());
};
</script>
