<template>
  <UContainer class="py-10 space-y-6">
    <section class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex flex-col gap-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ $t("capabilities.list.title") }}
        </h1>
        <p class="text-gray-600 dark:text-gray-300">
          {{ $t("capabilities.list.description") }}
        </p>
      </div>
      <div class="flex flex-wrap gap-3">
        <UButton
          icon="i-heroicons-arrow-path"
          variant="ghost"
          color="neutral"
          :loading="catalogLoading"
          @click="loadCatalog"
        >
      {{ $t("capabilities.list.refresh") }}
    </UButton>
    <UButton icon="i-heroicons-plus" color="primary" @click="openForm">
      {{ $t("capabilities.list.createButton") }}
    </UButton>
  </div>
</section>

    <UAlert variant="soft" color="neutral" class="bg-white/5 dark:bg-white/5">
      <template #title>
        {{ $t("capabilities.catalogSync.title") }}
      </template>
      <template #description>
        <p class="text-sm text-gray-600 dark:text-gray-300">
          {{ $t("capabilities.catalogSync.desc") }}
        </p>
        <code class="mt-1 inline-flex select-all rounded bg-gray-900/80 px-2 py-1 font-mono text-xs text-white">
          {{ catalogCommand }}
        </code>
      </template>
    </UAlert>

    <UCard>
      <template #header>
        <div class="flex flex-col gap-1">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ $t("capabilities.list.tableTitle") }}
          </h2>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ $t("capabilities.list.tableHint") }}
          </p>
        </div>
      </template>
      <div v-if="catalogLoading" class="flex items-center justify-center py-16 text-gray-500 dark:text-gray-400">
        {{ $t("common.loading") }}
      </div>
      <div v-else>
        <div v-if="groupedCatalog.length" class="space-y-4">
          <section
            v-for="group in groupedCatalog"
            :key="group.module"
            class="rounded-2xl border border-gray-200 bg-white px-1 py-1 shadow-sm transition dark:border-white/5 dark:bg-[#0f192a]/80 dark:shadow-inner dark:shadow-black/40"
          >
            <button
              type="button"
              class="flex w-full items-center justify-between gap-4 rounded-2xl px-5 py-4 text-left transition hover:bg-gray-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-400 dark:hover:border-primary-500/40 dark:hover:bg-white/5"
              @click="toggleModule(group.module)"
            >
              <div>
                <p class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ group.displayName }}
                </p>
                <p class="text-xs text-blue-600 dark:text-[#93c5fd]">
                  {{ group.module }} · {{ group.items.length }} 个能力
                </p>
              </div>
              <div class="flex items-center gap-2">
                <UBadge
                  v-for="badge in group.kindBadges"
                  :key="badge.label"
                  :label="badge.label"
                  :color="badge.color"
                  variant="soft"
                  class="bg-gray-100 text-gray-700 backdrop-blur dark:bg-white/10 dark:text-white"
                />
                <UIcon
                  :name="isModuleExpanded(group.module) ? 'i-heroicons-chevron-down' : 'i-heroicons-chevron-right'"
                  class="h-5 w-5 text-gray-500 dark:text-white/70"
                />
              </div>
            </button>
            <div v-if="isModuleExpanded(group.module)" class="border-t border-gray-200 bg-gray-50 dark:border-white/5 dark:bg-black/10">
              <UTable
                :data="group.items"
                :columns="tableColumns"
                :ui="{ td: { base: 'align-top' } }"
              >
                <template #capability_id-cell="{ row }">
                  <div class="flex flex-col">
                    <span class="font-semibold text-gray-900 dark:text-white">
                      {{ row.original.capability_id }}
                    </span>
                    <small class="text-xs text-gray-500 dark:text-gray-400">
                      {{ $t("capabilities.list.versionLabel", { version: row.original.version }) }}
                    </small>
                  </div>
                </template>
                <template #kind-cell="{ row }">
                  <UBadge
                    :label="formatKindLabel(row.original.kind)"
                    :color="kindColor(row.original.kind)"
                    variant="soft"
                  />
                </template>
                <template #execution-cell="{ row }">
                  <UBadge
                    :label="row.original.execution.mode.toUpperCase()"
                    :color="row.original.execution.mode === 'async' ? 'primary' : 'gray'"
                    variant="soft"
                  />
                  <p v-if="row.original.execution.callback_url" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ row.original.execution.callback_url }}
                  </p>
                </template>
                <template #tags-cell="{ row }">
                  <div class="flex flex-wrap gap-1.5">
                    <UBadge
                      v-for="tag in row.original.tags"
                      :key="`${row.original.capability_id}-${tag}`"
                      :label="tag"
                      variant="subtle"
                    />
                    <span v-if="!row.original.tags.length" class="text-xs text-gray-400">
                      {{ $t("capabilities.list.noTags") }}
                    </span>
                  </div>
                </template>
                <template #exposure-cell="{ row }">
                  <div class="flex flex-col gap-1">
                    <UBadge
                      :label="exposureBadge(row.original.syncStatus).label"
                      :color="exposureBadge(row.original.syncStatus).color"
                      variant="soft"
                    />
                    <span v-if="row.original.updatedAt" class="text-xs text-gray-500 dark:text-gray-400">
                      {{ $t("capabilities.exposure.list.updatedAt", { time: row.original.updatedAt }) }}
                    </span>
                  </div>
                </template>
                <template #checksum-cell="{ row }">
                  <code class="text-xs text-gray-500 dark:text-gray-400">{{ row.original.checksum }}</code>
                </template>
                <template #actions-cell="{ row }">
                  <UButton
                    size="xs"
                    icon="i-heroicons-adjustments-horizontal"
                    @click="openExposureForm(row.original.capability_id)"
                  >
                    {{ $t("capabilities.exposure.list.actions.configure") }}
                  </UButton>
                  <UButton
                    v-if="isWorkflowKind(row.original.kind)"
                    size="xs"
                    variant="outline"
                    color="primary"
                    icon="i-heroicons-command-line"
                    @click="openMcpPanelForCapability(row.original.capability_id)"
                  >
                    {{ $t("capabilities.mcp.actions.debugCapability") }}
                  </UButton>
                  <UButton
                    v-if="!isWorkflowKind(row.original.kind)"
                    size="xs"
                    variant="ghost"
                    icon="i-heroicons-beaker-20-solid"
                    @click="openDebugPanelFromCatalog(row.original)"
                  >
                    {{ $t("capabilities.form.debug.openExisting") }}
                  </UButton>
                </template>
              </UTable>
            </div>
          </section>
        </div>
        <div v-else class="py-16 text-center">
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ $t("capabilities.list.empty") }}
          </p>
        </div>
      </div>
    </UCard>

    <UModal
      v-model:open="formOpen"
      prevent-close
      :ui="{ content: 'max-w-6xl w-[95vw] mx-auto' }"
    >
      <template #body>
        <div class="space-y-6">
          <div class="flex flex-col gap-1">
            <div class="flex items-center justify-between">
              <div>
                <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
                  {{ $t("capabilities.form.title") }}
                </h2>
                <p class="text-gray-600 dark:text-gray-300">
                  {{ $t("capabilities.form.description") }}
                </p>
              </div>
              <UButton icon="i-heroicons-x-mark" variant="ghost" color="neutral" @click="closeForm" />
            </div>
            <div class="flex flex-wrap items-center gap-3 text-sm text-gray-600 dark:text-gray-300">
              <div class="inline-flex items-center gap-1">
                <UIcon name="i-heroicons-identification" />
                <span class="font-medium">{{ $t("capabilities.form.capabilityId") }}:</span>
                <code class="rounded bg-gray-100 px-2 py-0.5 text-gray-900 dark:bg-gray-800 dark:text-gray-100">
                  {{ capabilityId || "—" }}
                </code>
              </div>
              <UBadge :label="validationBadge.label" :color="validationBadge.color" variant="soft" />
            </div>
          </div>

          <UCard :ui="{ body: 'space-y-8' }">
            <template #header>
              <div class="flex flex-col gap-2">
                <div class="flex flex-wrap gap-3">
                  <button
                    v-for="(item, idx) in stepItems"
                    :key="item.label"
                    type="button"
                    class="flex items-center gap-2 rounded-full border px-4 py-1 text-sm transition"
                    :class="idx === currentStep
                      ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-950 dark:text-primary-300'
                      : 'border-gray-200 text-gray-500 dark:border-gray-800 dark:text-gray-400'"
                    @click="currentStep = idx"
                  >
                    <span
                      class="flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold"
                      :class="idx === currentStep
                        ? 'bg-primary-600 text-white'
                        : 'bg-gray-200 text-gray-600 dark:bg-gray-800 dark:text-gray-300'"
                    >
                      {{ idx + 1 }}
                    </span>
                    {{ item.label }}
                  </button>
                </div>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ stepItems[currentStep]?.description }}
                </p>
              </div>
            </template>

            <div
              v-if="loadingTemplate"
              class="flex items-center justify-center py-16 text-gray-500 dark:text-gray-400"
            >
              {{ $t("common.loading") }}
            </div>

            <template v-else>
        <section v-if="currentStep === 0" class="space-y-6">
          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.namespace')" required>
              <UInput v-model="form.namespace" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.resource')" required>
              <UInput v-model="form.resource" />
            </UFormField>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.action')" required>
              <UInput v-model="form.action" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.sensitivity')" required>
              <USelectMenu
                v-model="form.sensitivity"
                :options="template?.sensitivity_options || ['low','medium','high']"
              />
            </UFormField>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.tenantScope')">
              <UInput v-model="form.tenant_scope" placeholder="global" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.tags')" :description="$t('capabilities.form.tagsHint')">
              <UInput v-model="tagsText" placeholder="workflow,agent" />
            </UFormField>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField class="md:col-span-2" :label="$t('capabilities.form.scenario')" :description="$t('capabilities.form.scenarioHint')" required>
              <UTextarea v-model="form.scenario" :rows="3" />
            </UFormField>
          </div>

          <div class="space-y-2">
            <p class="text-sm font-medium text-gray-700 dark:text-gray-200">
              {{ $t("capabilities.form.displayName") }}
              <span class="text-red-500">*</span>
            </p>
            <div class="grid gap-4 md:grid-cols-2">
              <UFormField :label="$t('capabilities.form.localeZh')" required>
                <UInput v-model="form.name.zh" />
              </UFormField>
              <UFormField :label="$t('capabilities.form.localeEn')" required>
                <UInput v-model="form.name.en" />
              </UFormField>
            </div>
          </div>

          <div class="space-y-2">
            <p class="text-sm font-medium text-gray-700 dark:text-gray-200">
              {{ $t("capabilities.form.summary") }}
              <span class="text-red-500">*</span>
            </p>
            <div class="grid gap-4 md:grid-cols-2">
              <UFormField :label="$t('capabilities.form.localeZh')" required>
                <UInput v-model="form.summary.zh" />
              </UFormField>
              <UFormField :label="$t('capabilities.form.localeEn')" required>
                <UInput v-model="form.summary.en" />
              </UFormField>
            </div>
          </div>

          <div class="space-y-2">
            <p class="text-sm font-medium text-gray-700 dark:text-gray-200">
              {{ $t("capabilities.form.descriptionField") }}
            </p>
            <div class="grid gap-4 md:grid-cols-2">
              <UFormField :label="$t('capabilities.form.localeZh')">
                <UTextarea v-model="form.description.zh" :rows="4" />
              </UFormField>
              <UFormField :label="$t('capabilities.form.localeEn')">
                <UTextarea v-model="form.description.en" :rows="4" />
              </UFormField>
            </div>
          </div>

        </section>

        <section v-else-if="currentStep === 1" class="space-y-6">
          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.inputSchema')" required :description="template?.field_hints?.['schemas.input']">
              <UInput v-model="form.schemas.input" placeholder="contracts/schema/input/xxx.json" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.outputSchema')" required :description="template?.field_hints?.['schemas.output']">
              <UInput v-model="form.schemas.output" placeholder="contracts/schema/output/xxx.json" />
            </UFormField>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.restPath')" :description="template?.protocol_samples?.rest_path">
              <div class="flex gap-3">
                <USelectMenu v-model="form.protocols.rest.method" :options="httpMethods" class="w-32" />
                <UInput v-model="form.protocols.rest.path" class="flex-1" />
              </div>
            </UFormField>
            <UFormField :label="$t('capabilities.form.grpcService')" :description="template?.protocol_samples?.grpc_service">
              <UInput v-model="form.protocols.grpc.service" placeholder="powerx.template.TemplateService/Create" />
            </UFormField>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.workflowTemplate')" :description="template?.protocol_samples?.workflow_template">
              <UInput v-model="form.protocols.workflow.template" placeholder="contracts/exposure/workflow/template-create.json" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.agentStream')">
              <UInput v-model="form.protocols.agent_stream.channel" placeholder="contracts/exposure/agent-streams/create.yaml" />
            </UFormField>
          </div>

          <div class="border-t border-gray-200 pt-4 dark:border-gray-800">
            <h3 class="text-base font-semibold">
              {{ $t("capabilities.form.asyncTitle") }}
            </h3>
            <div class="mt-3 grid gap-4 md:grid-cols-3">
              <UFormField :label="$t('capabilities.form.executionMode')">
                <USelectMenu v-model="form.async_mode" :options="template?.async_modes || ['sync','async']" />
              </UFormField>
              <UFormField :label="$t('capabilities.form.callbackUrl')" :disabled="form.async_mode !== 'async'">
                <UInput v-model="form.async_config.callback_url" :disabled="form.async_mode !== 'async'" />
              </UFormField>
              <UFormField :label="$t('capabilities.form.statusEndpoint')" :disabled="form.async_mode !== 'async'">
                <UInput v-model="form.async_config.status_endpoint" :disabled="form.async_mode !== 'async'" />
              </UFormField>
              <UFormField
                :label="$t('capabilities.form.sseChannel')"
                class="md:col-span-3"
                :disabled="form.async_mode !== 'async'"
              >
                <UInput v-model="form.async_config.sse_channel" :disabled="form.async_mode !== 'async'" />
              </UFormField>
            </div>
          </div>
        </section>

        <section v-else class="space-y-6">
          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.sampleRequest')" :description="$t('capabilities.form.jsonHint')">
              <UTextarea v-model="form.samples.requestText" :rows="8" class="font-mono text-xs" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.sampleResponse')" :description="$t('capabilities.form.jsonHint')">
              <UTextarea v-model="form.samples.responseText" :rows="8" class="font-mono text-xs" />
            </UFormField>
          </div>

          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <h3 class="text-base font-semibold">
                {{ $t("capabilities.form.errorCodes") }}
              </h3>
              <UButton icon="i-heroicons-plus" size="xs" variant="soft" @click="addErrorCode">
                {{ $t("capabilities.form.addRow") }}
              </UButton>
            </div>
            <div class="space-y-2">
              <div
                v-for="(error, idx) in form.samples.errors"
                :key="`error-${idx}`"
                class="grid gap-2 rounded border border-gray-200 p-3 dark:border-gray-800 md:grid-cols-3"
              >
                <UInput v-model="error.code" :placeholder="$t('capabilities.form.errorCode')" />
                <UInput v-model="error.message" :placeholder="$t('capabilities.form.errorMessage')" />
                <div class="flex gap-2">
                  <UInput v-model="error.solution" class="flex-1" :placeholder="$t('capabilities.form.errorSolution')" />
                  <UButton icon="i-heroicons-x-mark" size="xs" variant="ghost" color="neutral" @click="removeErrorCode(idx)" />
                </div>
              </div>
              <p v-if="!form.samples.errors.length" class="text-sm text-gray-500">
                {{ $t("capabilities.form.errorEmpty") }}
              </p>
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.demoUrl')">
              <UInput v-model="form.demo.url" placeholder="https://demo.powerx.cloud/template" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.demoHint')">
              <UInput v-model="form.demo.credential_hint" placeholder="使用测试租户 demo-tenant / demo123" />
            </UFormField>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.ownerName')" required>
              <UInput v-model="form.owner.name" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.ownerEmail')" required>
              <UInput v-model="form.owner.email" type="email" />
            </UFormField>
          </div>

          <UCard size="lg" :ui="{ body: 'space-y-2' }">
            <template #header>
              <div class="flex items-center gap-2">
                <UIcon name="i-heroicons-document-text" class="text-gray-400" />
                <h3 class="text-base font-semibold">
                  {{ $t("capabilities.form.previewTitle") }}
                </h3>
              </div>
            </template>
            <dl class="grid gap-2 md:grid-cols-2">
              <div>
                <dt class="text-sm text-gray-500">{{ $t("capabilities.form.previewName") }}</dt>
                <dd class="font-medium">{{ form.name.zh || "—" }}</dd>
              </div>
              <div>
                <dt class="text-sm text-gray-500">{{ $t("capabilities.form.previewOwner") }}</dt>
                <dd class="font-medium">{{ form.owner.email || "—" }}</dd>
              </div>
              <div class="md:col-span-2">
                <dt class="text-sm text-gray-500">{{ $t("capabilities.form.previewSummary") }}</dt>
                <dd class="font-medium">{{ form.summary.zh || $t("capabilities.form.previewPlaceholder") }}</dd>
              </div>
            </dl>
          </UCard>
        </section>

        <UAlert
          v-if="validationResult && !validationResult.valid"
          icon="i-heroicons-exclamation-triangle"
          color="rose"
          variant="soft"
        >
          <template #title>
            {{ $t("capabilities.form.validationFailed") }}
          </template>
          <template #description>
            <ul class="list-disc pl-5 text-sm">
              <li v-for="err in validationResult.errors" :key="`${err.field}-${err.message}`">
                <span class="font-semibold">{{ err.field }}:</span> {{ err.message }}
                <span v-if="err.suggestion" class="text-gray-500">({{ err.suggestion }})</span>
              </li>
            </ul>
          </template>
        </UAlert>

        <div class="flex flex-col gap-3 border-t border-gray-200 pt-4 dark:border-gray-800 md:flex-row md:items-center md:justify-between">
          <div class="text-sm text-gray-500">
            <span v-if="savedAt">
              {{ $t("capabilities.form.lastSaved", { time: savedAt }) }}
            </span>
          </div>
          <div class="flex flex-wrap gap-3">
            <UButton variant="ghost" color="neutral" :disabled="currentStep === 0" @click="currentStep = Math.max(0, currentStep - 1)">
              {{ $t("common.previous") }}
            </UButton>
            <UButton variant="ghost" color="neutral" :disabled="currentStep === stepItems.length - 1" @click="currentStep = Math.min(stepItems.length - 1, currentStep + 1)">
              {{ $t("common.next") }}
            </UButton>
            <UButton variant="outline" color="primary" :loading="validating" @click="handleValidate">
              {{ $t("capabilities.form.validateButton") }}
            </UButton>
            <UButton variant="soft" color="primary" :loading="savingDraft" @click="handleSaveDraft">
              {{ $t("capabilities.form.saveDraft") }}
            </UButton>
            <UButton variant="soft" color="neutral" @click="openDebugPanelForDraft">
              {{ $t("capabilities.form.debug.openDraft") }}
            </UButton>
            <UButton color="primary" :loading="submitting" @click="handleSubmit">
              {{ $t("common.submit") }}
            </UButton>
          </div>
        </div>
      </template>
    </UCard>
  </div>
</template>
    </UModal>

    <UModal
      v-model:open="debugModalOpen"
      prevent-close
      :title="$t('capabilities.form.debug.title')"
      :description="$t('capabilities.form.debug.description')"
      :ui="{ content: 'max-w-none w-full sm:w-[95vw] sm:max-w-6xl h-screen sm:h-[90vh] flex flex-col overflow-hidden' }"
    >
      <template #actions>
        <NuxtLink
          :href="pluginCapabilityDocUrl"
          target="_blank"
          rel="noreferrer"
          class="text-xs text-primary-600 dark:text-primary-200 hover:underline"
        >
          {{ $t("capabilities.form.debug.docLink") }}
        </NuxtLink>
      </template>
	      <template #body>
	        <div class="flex flex-1 flex-col gap-6 overflow-hidden">
	          <div class="grid flex-1 gap-6 overflow-hidden lg:grid-cols-[1.1fr_0.9fr]">
	            <UCard
	              class="h-full border border-gray-200 bg-white text-gray-900 shadow-sm dark:border-white/5 dark:bg-[#0f192a]/80 dark:text-white/80 dark:shadow-inner dark:shadow-black/30"
	              :ui="{ body: 'p-0 h-full' }"
	            >
              <div class="flex h-full flex-col">
                <div class="flex-1 overflow-y-auto px-5 py-5 text-sm">
                  <div class="grid gap-5 lg:grid-cols-12">
                    <UFormField class="lg:col-span-6" :label="$t('capabilities.form.debug.capability')">
                      <UInput v-model="debugForm.capabilityId" readonly />
                    </UFormField>
                    <UFormField class="lg:col-span-6" :label="$t('capabilities.form.debug.action')">
                      <UInput v-model="debugForm.action" />
                    </UFormField>
                    <UFormField class="lg:col-span-4" :label="$t('capabilities.form.debug.protocol')">
                      <USelectMenu
                        v-model="debugForm.preferredProtocol"
                        :options="debugProtocolOptions"
                        option-attribute="label"
                        value-attribute="value"
                        :portal="false"
                        class="w-full"
                      />
                    </UFormField>
                    <UFormField class="lg:col-span-4" :label="$t('capabilities.form.debug.targetMode')">
                      <USelectMenu
                        v-model="debugForm.targetMode"
                        :options="debugTargetOptions"
                        option-attribute="label"
                        value-attribute="value"
                        :portal="false"
                        class="w-full"
                      />
                    </UFormField>
                    <UFormField class="lg:col-span-4" :label="$t('capabilities.form.debug.apiBase')">
                      <UInput v-model="debugForm.apiBase" placeholder="http://127.0.0.1:8078" />
                    </UFormField>
                    <div class="lg:col-span-12 space-y-3">
	                      <label class="text-sm font-medium text-gray-900 dark:text-white">
	                        {{ $t("capabilities.form.debug.payload") }}
	                      </label>
                      <UTextarea
                        v-model="debugForm.payloadText"
                        :rows="14"
                        class="w-full min-h-[320px] font-mono text-xs"
                      />
	                      <div class="flex flex-wrap items-center justify-between text-xs text-gray-600 dark:text-gray-300">
                        <span>
                          {{ debugPayloadValid ? "JSON OK" : $t("capabilities.form.debug.payloadInvalid") }}
                        </span>
                        <div class="flex items-center gap-3">
                          <UButton variant="link" size="xs" color="neutral" @click="resetDebugPayload">
                            {{ $t("capabilities.form.debug.resetTemplate") }}
                          </UButton>
                          <UButton
                            variant="link"
                            size="xs"
                            color="neutral"
                            :disabled="!debugPayloadTemplate"
                            @click="applyDebugPayloadTemplate"
                          >
                            {{ $t("capabilities.form.debug.applyTemplate") }}
                          </UButton>
                        </div>
                      </div>
                    </div>
                    <UFormField class="lg:col-span-6" :label="$t('capabilities.form.debug.tenantUuid')">
                      <UInput v-model="debugForm.tenantUuid" placeholder="00000000-0000-0000-0000-000000000001" />
                    </UFormField>
                    <UFormField class="lg:col-span-6" :label="$t('capabilities.form.debug.mockModule')">
                      <UInput v-model="debugForm.mockModule" placeholder="media / workflow" />
                    </UFormField>
                    <UFormField class="lg:col-span-12" :label="$t('capabilities.form.debug.requestId')">
                      <div class="flex gap-2">
                        <UInput v-model="debugForm.requestId" />
                        <UButton
                          variant="ghost"
                          color="neutral"
                          size="xs"
                          @click="regenerateDebugRequestId"
                        >
                          <UIcon name="i-heroicons-arrow-path" />
                        </UButton>
                      </div>
                    </UFormField>
                    <div class="lg:col-span-12 space-y-2">
                      <div class="flex items-center justify-between">
	                        <p class="text-sm font-medium text-gray-900 dark:text-white">
	                          {{ $t("capabilities.form.debug.requestPreview") }}
	                        </p>
                        <UButton
                          color="gray"
                          variant="ghost"
                          size="xs"
                          :icon="debugShowRequestPreview ? 'i-heroicons-chevron-down' : 'i-heroicons-chevron-right'"
                          @click="debugShowRequestPreview = !debugShowRequestPreview"
                        />
                      </div>
	                      <pre
	                        v-if="debugShowRequestPreview"
	                        class="rounded bg-gray-100 p-4 text-xs text-gray-800 dark:bg-black/40 dark:text-white/80"
	                      >
{{ debugRequestPreviewText }}
	                      </pre>
	                      <p v-else class="text-xs text-gray-600 dark:text-gray-300">
	                        {{ $t("capabilities.form.debug.collapsedPreview") }}
	                      </p>
                    </div>
                  </div>
                </div>
	                <div class="border-t border-gray-200 px-5 py-4 dark:border-white/10">
	                  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
	                    <p class="text-xs text-gray-500 dark:text-white/60">
	                      {{ $t("capabilities.form.debug.payloadHint") }}
	                    </p>
                    <UButton
                      color="primary"
                      :loading="debugInvokeLoading"
                      :disabled="!debugPayloadValid"
                      @click="handleDebugInvoke"
                    >
                      {{ $t("capabilities.form.debug.invoke") }}
                    </UButton>
                  </div>
                </div>
              </div>
	            </UCard>

	            <UCard
	              class="h-full border border-gray-200 bg-white text-gray-900 shadow-sm dark:border-white/5 dark:bg-[#0d1828]/90 dark:text-white/80 dark:shadow-inner dark:shadow-black/30"
	              :ui="{ body: 'p-0 h-full' }"
	            >
              <div class="flex h-full flex-col">
                <div class="flex-1 space-y-4 overflow-y-auto px-5 py-5 text-sm">
                  <div class="flex flex-wrap gap-3">
                    <span>
                      {{ $t("capabilities.form.debug.status") }}：
                      <strong
                        :class="{
                          'text-green-400': !!debugResult,
                          'text-rose-400': !!debugErrorMessage,
                        }"
                      >
                        {{ debugResult?.status ||
                            (debugErrorMessage
                              ? $t("capabilities.form.debug.statusFailed")
                              : $t("capabilities.form.debug.statusIdle")) }}
                      </strong>
                    </span>
                    <span v-if="debugDuration">
                      {{ $t("capabilities.form.debug.duration") }}：{{ debugDuration.toFixed(1) }} ms
                    </span>
                    <span>
                      {{ $t("capabilities.form.debug.traceId") }}：
                      {{ debugLastTraceId || "—" }}
                    </span>
                  </div>

                  <div v-if="debugWarnings.length" class="space-y-1">
                    <p class="text-xs uppercase tracking-wide text-yellow-300">
                      {{ $t("capabilities.form.debug.warnings") }}
                    </p>
                    <ul class="list-disc pl-4 text-xs text-yellow-200">
                      <li v-for="warn in debugWarnings" :key="warn">{{ warn }}</li>
                    </ul>
                  </div>

                  <div v-if="debugErrorMessage" class="space-y-2">
                    <p class="text-xs uppercase tracking-wide text-rose-300">
                      {{ $t("capabilities.form.debug.errors") }}
                    </p>
	                    <p class="rounded bg-rose-50 p-3 text-sm text-rose-700 dark:bg-rose-500/10 dark:text-rose-100">
	                      {{ debugErrorMessage }}
	                    </p>
	                    <pre
	                      v-if="debugErrorDetails"
	                      class="rounded bg-gray-100 p-3 text-xs text-gray-800 dark:bg-black/30 dark:text-white/80"
	                    >
{{ JSON.stringify(debugErrorDetails, null, 2) }}
	                    </pre>
                  </div>

                  <div v-if="debugResult?.data" class="space-y-2">
	                    <p class="text-xs uppercase tracking-wide text-primary-700 dark:text-primary-200">
	                      {{ $t("capabilities.form.debug.responseData") }}
	                    </p>
	                    <pre class="rounded bg-gray-100 p-3 text-xs text-gray-800 dark:bg-black/30 dark:text-white/80">
{{ JSON.stringify(debugResult.data, null, 2) }}
	                    </pre>
                  </div>

                  <div v-if="debugResult?.raw" class="space-y-2">
	                    <p class="text-xs uppercase tracking-wide text-gray-700 dark:text-gray-200">
	                      {{ $t("capabilities.form.debug.rawResponse") }}
	                    </p>
	                    <pre class="rounded bg-gray-100 p-3 text-xs text-gray-800 dark:bg-black/30 dark:text-white/80">
{{ JSON.stringify(debugResult.raw, null, 2) }}
	                    </pre>
                  </div>

                  <div class="space-y-2">
                    <div class="flex items-center justify-between">
	                      <p class="text-sm font-medium text-gray-900 dark:text-white">
	                        {{ $t("capabilities.form.debug.history") }}
	                      </p>
                      <UButton
                        size="xs"
                        variant="ghost"
                        color="neutral"
                        @click="clearDebugHistory"
                      >
                        {{ $t("capabilities.form.debug.clearHistory") }}
                      </UButton>
                    </div>
                    <div v-if="debugHistoryEntries.length" class="space-y-3">
	                      <UCard
	                        v-for="entry in debugHistoryEntries"
	                        :key="entry.id"
	                        class="border border-gray-200 bg-gray-50 text-xs text-gray-800 dark:border-white/5 dark:bg-black/30 dark:text-white/80"
	                      >
                        <template #header>
                          <div class="flex flex-wrap items-center gap-2">
                            <span class="text-sm font-semibold">
                              {{ entry.capabilityId }} · {{ entry.action }}
                            </span>
                            <UBadge :color="entry.success ? 'green' : 'rose'" variant="soft">
                              {{ entry.success
                                  ? $t("capabilities.form.debug.historySuccess")
                                  : $t("capabilities.form.debug.historyFail") }}
                            </UBadge>
                          </div>
	                          <p class="text-[11px] text-gray-500 dark:text-white/60">
	                            {{ $t("capabilities.form.debug.duration") }}：{{ entry.duration.toFixed(1) }} ms ·
	                            {{ $t("capabilities.form.debug.traceId") }}：{{ entry.traceId || "—" }}
	                          </p>
                        </template>
                        <pre class="overflow-x-auto whitespace-pre-wrap text-[11px]">
{{ entry.rawText }}
                        </pre>
                      </UCard>
                    </div>
	                    <p v-else class="text-xs text-gray-500 dark:text-gray-400">
	                      {{ $t("capabilities.form.debug.noHistory") }}
	                    </p>
                  </div>
                </div>
              </div>
            </UCard>
          </div>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="mcpModalOpen"
      prevent-close
      :title="$t('capabilities.mcp.panelTitle')"
      :description="$t('capabilities.mcp.panelDescription')"
      :ui="{ content: 'max-w-6xl w-[95vw] mx-auto' }"
    >
      <template #actions>
        <NuxtLink
          :href="mcpGuideUrl"
          target="_blank"
          rel="noreferrer"
          class="text-xs text-primary-600 dark:text-primary-200 hover:underline"
        >
          {{ $t("capabilities.mcp.docLink") }}
        </NuxtLink>
      </template>
	      <template #body>
	        <div class="space-y-6">
	          <UCard class="border border-gray-200 bg-white text-gray-900 shadow-sm dark:border-white/5 dark:bg-[#0f192a]/80 dark:text-white/80 dark:shadow-inner dark:shadow-black/30">
	            <template #header>
	              <div>
	                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
	                  {{ $t("capabilities.mcp.session.title") }}
	                </h2>
	                <p class="text-sm text-gray-600 dark:text-white/70">
	                  {{ $t("capabilities.mcp.session.description") }}
	                </p>
	              </div>
	            </template>
            <div class="space-y-6 text-sm">
              <div class="grid gap-4 md:grid-cols-2">
                <UFormField :label="$t('capabilities.mcp.fields.runtimeAssignment')">
                  <div class="flex gap-2">
                    <UInput v-model="mcpSessionForm.runtimeAssignmentId" placeholder="uuid" />
                    <UButton
                      icon="i-heroicons-arrow-path"
                      variant="ghost"
                      color="neutral"
                      size="xs"
                      @click="regenerateMcpRuntimeAssignment"
                    />
                  </div>
                </UFormField>
                <UFormField :label="$t('capabilities.mcp.fields.jwtId')">
                  <UInput v-model="mcpSessionForm.jwtId" placeholder="dev-mcp-client" />
                </UFormField>
                <UFormField :label="$t('capabilities.mcp.fields.registerState')">
                  <UInput v-model="mcpSessionForm.registerState" placeholder="registering" />
                </UFormField>
                <UFormField :label="$t('capabilities.mcp.fields.capabilitiesHash')">
                  <UInput v-model="mcpSessionForm.capabilitiesHash" placeholder="sha256:..." />
                </UFormField>
              </div>
              <div class="flex flex-wrap gap-3">
                <UButton
                  color="primary"
                  :loading="mcpRegisterLoading"
                  @click="handleMcpRegister"
                >
                  {{ $t("capabilities.mcp.actions.register") }}
                </UButton>
              </div>
              <div class="grid gap-4 lg:grid-cols-3">
                <div class="space-y-2">
                  <UFormField :label="$t('capabilities.mcp.fields.ackState')">
                    <UInput v-model="mcpSessionForm.ackState" placeholder="ready" />
                  </UFormField>
                  <UButton
                    block
                    variant="soft"
                    color="primary"
                    :disabled="!mcpSessionForm.sessionId"
                    :loading="mcpAckLoading"
                    @click="handleMcpAck"
                  >
                    {{ $t("capabilities.mcp.actions.ack") }}
                  </UButton>
                </div>
                <div class="space-y-2">
                  <UFormField :label="$t('capabilities.mcp.fields.heartbeatMissed')">
                    <UInput
                      v-model.number="mcpSessionForm.heartbeatMissed"
                      type="number"
                      min="0"
                    />
                  </UFormField>
                  <UButton
                    block
                    variant="soft"
                    color="primary"
                    :disabled="!mcpSessionForm.sessionId"
                    :loading="mcpHeartbeatLoading"
                    @click="handleMcpHeartbeat"
                  >
                    {{ $t("capabilities.mcp.actions.heartbeat") }}
                  </UButton>
                </div>
                <div class="space-y-2">
                  <UFormField :label="$t('capabilities.mcp.fields.closeReason')">
                    <UInput v-model="mcpSessionForm.closeReason" placeholder="optional" />
                  </UFormField>
                  <UButton
                    block
                    variant="ghost"
                    color="neutral"
                    :disabled="!mcpSessionForm.sessionId"
                    :loading="mcpCloseLoading"
                    @click="handleMcpClose"
                  >
                    {{ $t("capabilities.mcp.actions.close") }}
                  </UButton>
                </div>
              </div>
	              <div
	                v-if="mcpSession"
	                class="rounded-2xl border border-gray-200 bg-gray-50 p-4 text-xs text-gray-800 dark:border-white/10 dark:bg-black/20 dark:text-white/80"
	              >
                <div class="flex flex-wrap items-center gap-2 text-sm">
                  <span class="font-semibold">ID：{{ mcpSession.id }}</span>
                  <UBadge :label="mcpSession.state" :color="mcpSession.state === 'ready' ? 'green' : 'gray'" />
                </div>
	                <dl class="mt-3 grid gap-2 md:grid-cols-2">
	                  <div>
	                    <dt class="text-gray-500 dark:text-white/60">{{ $t("capabilities.mcp.fields.tenantUuid") }}</dt>
	                    <dd>{{ mcpSession.tenant_uuid }}</dd>
	                  </div>
	                  <div>
	                    <dt class="text-gray-500 dark:text-white/60">{{ $t("capabilities.mcp.fields.lastPing") }}</dt>
	                    <dd>{{ mcpSession.last_ping_at || "—" }}</dd>
	                  </div>
	                  <div>
	                    <dt class="text-gray-500 dark:text-white/60">{{ $t("capabilities.mcp.fields.createdAt") }}</dt>
	                    <dd>{{ mcpSession.created_at }}</dd>
	                  </div>
	                  <div>
	                    <dt class="text-gray-500 dark:text-white/60">{{ $t("capabilities.mcp.fields.updatedAt") }}</dt>
	                    <dd>{{ mcpSession.updated_at }}</dd>
	                  </div>
	                </dl>
	              </div>
            </div>
          </UCard>

	          <UCard class="border border-gray-200 bg-white text-gray-900 shadow-sm dark:border-white/5 dark:bg-[#0f192a]/80 dark:text-white/80 dark:shadow-inner dark:shadow-black/30">
	            <template #header>
	              <div class="flex items-center justify-between">
	                <div>
	                  <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
	                    {{ $t("capabilities.mcp.invoke.title") }}
	                  </h2>
	                  <p class="text-sm text-gray-600 dark:text-white/70">
	                    {{ $t("capabilities.mcp.invoke.description") }}
	                  </p>
	                </div>
                <UButton
                  size="xs"
                  variant="ghost"
                  color="neutral"
                  @click="applyCurrentCapabilityToMcp"
                >
                  {{ $t("capabilities.mcp.actions.applyCapability") }}
                </UButton>
              </div>
            </template>
            <div class="space-y-4 text-sm">
              <div class="grid gap-4 md:grid-cols-2">
                <UFormField :label="$t('capabilities.mcp.fields.sessionId')">
                  <UInput v-model="mcpInvokeForm.sessionId" placeholder="session uuid" />
                </UFormField>
                <UFormField :label="$t('capabilities.mcp.fields.tenantUuid')">
                  <UInput v-model="mcpInvokeForm.tenantUuid" placeholder="00000000-..." />
                </UFormField>
                <UFormField :label="$t('capabilities.mcp.fields.toolScope')">
                  <UInput v-model="mcpInvokeForm.toolScope" placeholder="agent.template.compose" />
                </UFormField>
                <UFormField :label="$t('capabilities.mcp.fields.capabilityId')">
                  <UInput v-model="mcpInvokeForm.capabilityId" placeholder="com.powerx.plugins.base.template.compose" />
                </UFormField>
                <UFormField :label="$t('capabilities.mcp.fields.intent')">
                  <UInput v-model="mcpInvokeForm.intent" placeholder="template.compose" />
                </UFormField>
                <UFormField :label="$t('capabilities.mcp.fields.idempotencyKey')">
                  <UInput v-model="mcpInvokeForm.idempotencyKey" placeholder="optional" />
                </UFormField>
                <UFormField :label="$t('capabilities.mcp.fields.signature')">
                  <UInput v-model="mcpInvokeForm.signature" placeholder="stub-signature" />
                </UFormField>
              </div>
              <div class="grid gap-4 md:grid-cols-2">
                <UFormField :label="$t('capabilities.mcp.fields.messageId')">
                  <div class="flex gap-2">
                    <UInput v-model="mcpInvokeForm.messageId" />
                    <UButton
                      variant="ghost"
                      color="neutral"
                      size="xs"
                      @click="regenerateInvokeIdentifiers"
                    >
                      <UIcon name="i-heroicons-arrow-path" />
                    </UButton>
                  </div>
                </UFormField>
                <UFormField :label="$t('capabilities.mcp.fields.traceId')">
                  <UInput v-model="mcpInvokeForm.traceId" />
                </UFormField>
                <UFormField :label="$t('capabilities.mcp.fields.correlationId')">
                  <UInput v-model="mcpInvokeForm.correlationId" />
                </UFormField>
                <UFormField :label="$t('capabilities.mcp.fields.issuedAt')">
                  <div class="flex gap-2">
                    <UInput v-model="mcpInvokeForm.issuedAt" placeholder="2025-12-11T06:10:00Z" />
                    <UButton
                      variant="ghost"
                      color="neutral"
                      size="xs"
                      @click="setIssuedAtNow"
                    >
                      {{ $t("capabilities.mcp.actions.issuedNow") }}
                    </UButton>
                  </div>
                </UFormField>
              </div>
	              <div class="grid gap-4 md:grid-cols-2">
	                <div>
	                  <label class="text-xs font-medium text-gray-700 dark:text-white/80">
	                    {{ $t("capabilities.mcp.fields.payload") }}
	                  </label>
                  <UTextarea
                    v-model="mcpInvokeForm.payloadText"
                    :rows="10"
                    class="mt-1 font-mono text-xs"
                  />
	                </div>
	                <div>
	                  <label class="text-xs font-medium text-gray-700 dark:text-white/80">
	                    {{ $t("capabilities.mcp.fields.metadata") }}
	                  </label>
                  <UTextarea
                    v-model="mcpInvokeForm.metadataText"
                    :rows="10"
                    class="mt-1 font-mono text-xs"
                  />
                </div>
              </div>
              <UButton
                color="primary"
                :loading="mcpInvokeLoading"
                @click="handleMcpInvoke"
              >
                {{ $t("capabilities.mcp.actions.invoke") }}
              </UButton>
              <div class="space-y-2 text-xs">
                <div class="flex flex-wrap gap-3">
                  <span>
                    {{ $t("capabilities.mcp.invoke.status") }}：
                    <strong
                      :class="{
                        'text-green-400': !!mcpInvokeResult,
                        'text-rose-400': !!mcpInvokeErrorMessage,
                      }"
                    >
                      {{ mcpInvokeResult?.status || (mcpInvokeErrorMessage ? $t("capabilities.mcp.invoke.failed") : $t("capabilities.mcp.invoke.idle")) }}
                    </strong>
                  </span>
                  <span>
                    Trace：{{ mcpInvokeResult?.trace_id || "—" }}
                  </span>
                  <span>
                    Correlation：{{ mcpInvokeResult?.correlation_id || "—" }}
                  </span>
                </div>
	                <div v-if="mcpInvokeResult?.payload" class="space-y-1">
	                  <p class="text-xs uppercase tracking-wide text-primary-700 dark:text-primary-200">
	                    {{ $t("capabilities.mcp.fields.payload") }}
	                  </p>
	                  <pre class="rounded bg-gray-100 p-3 text-xs text-gray-800 dark:bg-black/30 dark:text-white/80">
{{ JSON.stringify(mcpInvokeResult.payload, null, 2) }}
	                  </pre>
	                </div>
	                <div v-if="mcpInvokeResult?.metadata" class="space-y-1">
	                  <p class="text-xs uppercase tracking-wide text-primary-700 dark:text-primary-200">
	                    {{ $t("capabilities.mcp.fields.metadata") }}
	                  </p>
	                  <pre class="rounded bg-gray-100 p-3 text-xs text-gray-800 dark:bg-black/30 dark:text-white/80">
{{ JSON.stringify(mcpInvokeResult.metadata, null, 2) }}
	                  </pre>
	                </div>
	                <div v-if="mcpInvokeErrorMessage" class="space-y-1">
	                  <p class="text-xs uppercase tracking-wide text-rose-300">
	                    {{ $t("capabilities.mcp.invoke.failed") }}
	                  </p>
	                  <p class="rounded bg-rose-50 p-3 text-xs text-rose-700 dark:bg-rose-500/10 dark:text-rose-100">
	                    {{ mcpInvokeErrorMessage }}
	                  </p>
	                  <pre
	                    v-if="mcpInvokeErrorDetails"
	                    class="rounded bg-gray-100 p-3 text-xs text-gray-800 dark:bg-black/30 dark:text-white/80"
	                  >
{{ JSON.stringify(mcpInvokeErrorDetails, null, 2) }}
	                  </pre>
	                </div>
              </div>
	              <div class="space-y-2">
	                <div class="flex items-center justify-between">
	                  <p class="text-sm font-medium text-gray-900 dark:text-white">
	                    {{ $t("capabilities.mcp.invoke.history") }}
	                  </p>
                  <UButton
                    size="xs"
                    variant="ghost"
                    color="neutral"
                    @click="clearMcpInvokeHistory"
                  >
                    {{ $t("capabilities.mcp.actions.clearHistory") }}
                  </UButton>
                </div>
	                <div v-if="mcpInvokeHistory.length" class="space-y-3">
	                  <UCard
	                    v-for="entry in mcpInvokeHistory"
	                    :key="entry.id"
	                    class="border border-gray-200 bg-gray-50 text-xs text-gray-800 dark:border-white/5 dark:bg-black/30 dark:text-white/80"
	                  >
                    <template #header>
                      <div class="flex flex-wrap items-center gap-2">
                        <span class="text-sm font-semibold">
                          {{ entry.status }}
                        </span>
                        <UBadge
                          :color="entry.success ? 'green' : 'rose'"
                          variant="soft"
                          :label="entry.success ? $t('capabilities.mcp.invoke.historySuccess') : $t('capabilities.mcp.invoke.historyFail')"
                        />
	                        <span class="text-gray-500 dark:text-white/60">{{ entry.timestamp }}</span>
	                      </div>
	                      <p class="text-gray-500 dark:text-white/60">
	                        Trace：{{ entry.traceId || "—" }} · Correlation：{{ entry.correlationId || "—" }}
	                      </p>
                    </template>
                    <pre class="overflow-x-auto whitespace-pre-wrap text-[11px]">
{{ entry.payload ? JSON.stringify(entry.payload, null, 2) : entry.error }}
                    </pre>
                  </UCard>
                </div>
	                <p v-else class="text-xs text-gray-500 dark:text-white/60">
	                  {{ $t("capabilities.mcp.invoke.noHistory") }}
	                </p>
	              </div>
            </div>
          </UCard>

	          <UCard class="border border-gray-200 bg-white text-gray-900 shadow-sm dark:border-white/5 dark:bg-[#0f192a]/80 dark:text-white/80 dark:shadow-inner dark:shadow-black/30">
	            <template #header>
	              <div class="flex items-center justify-between">
	                <div>
	                  <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
	                    {{ $t("capabilities.mcp.stream.title") }}
	                  </h2>
	                  <p class="text-sm text-gray-600 dark:text-white/70">
	                    {{ $t("capabilities.mcp.stream.description") }}
	                  </p>
	                </div>
                <div class="flex items-center gap-2">
                  <UBadge
                    :label="mcpStreamConnected ? $t('capabilities.mcp.stream.connected') : $t('capabilities.mcp.stream.disconnected')"
                    :color="mcpStreamConnected ? 'green' : 'gray'"
                  />
                  <UButton
                    size="xs"
                    variant="soft"
                    color="primary"
                    :disabled="!mcpSessionForm.sessionId"
                    @click="toggleMcpStream"
                  >
                    {{ mcpStreamConnected ? $t("capabilities.mcp.actions.disconnectStream") : $t("capabilities.mcp.actions.connectStream") }}
                  </UButton>
                  <UButton
                    size="xs"
                    variant="ghost"
                    color="neutral"
                    @click="clearMcpEvents"
                  >
                    {{ $t("capabilities.mcp.actions.clearEvents") }}
                  </UButton>
                </div>
              </div>
            </template>
            <div class="space-y-3 text-xs">
              <p v-if="mcpStreamError" class="text-rose-300">
                {{ mcpStreamError }}
              </p>
	              <div v-if="mcpEvents.length" class="space-y-3">
	                <UCard
	                  v-for="event in mcpEvents"
	                  :key="`${event.type}-${event.timestamp}`"
	                  class="border border-gray-200 bg-gray-50 text-gray-800 dark:border-white/5 dark:bg-black/30 dark:text-white/80"
	                >
                  <template #header>
                    <div class="flex items-center justify-between">
                      <span class="font-semibold">{{ event.type }}</span>
	                      <span class="text-gray-500 dark:text-white/60">{{ event.timestamp }}</span>
	                    </div>
	                  </template>
                  <pre class="overflow-x-auto whitespace-pre-wrap text-[11px]">
{{ JSON.stringify(event.payload, null, 2) }}
                  </pre>
                </UCard>
              </div>
	              <p v-else class="text-gray-500 dark:text-white/60">
	                {{ $t("capabilities.mcp.stream.empty") }}
	              </p>
            </div>
          </UCard>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="exposureFormOpen"
      prevent-close
      :ui="{ content: 'max-w-6xl w-[95vw] mx-auto' }"
    >
      <template #body>
        <div class="space-y-6">
          <div class="flex flex-col gap-2">
            <div class="flex items-start justify-between gap-4">
              <div>
                <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
                  {{ $t("capabilities.exposure.title") }}
                </h2>
                <p class="text-gray-600 dark:text-gray-300">
                  {{ $t("capabilities.exposure.description") }}
                </p>
              </div>
              <UButton icon="i-heroicons-x-mark" variant="ghost" color="neutral" @click="closeExposureForm" />
            </div>
            <div class="flex flex-wrap items-center gap-3 text-sm text-gray-600 dark:text-gray-300">
              <div class="inline-flex items-center gap-1">
                <UIcon name="i-heroicons-cube" />
                <span class="font-medium">{{ $t("capabilities.exposure.capability") }}:</span>
                <code class="rounded bg-gray-100 px-2 py-0.5 text-gray-900 dark:bg-gray-800 dark:text-gray-100">
                  {{ exposureForm.capability_id || "—" }}
                </code>
              </div>
              <div class="inline-flex items-center gap-2">
                <UBadge :label="exposureModalStatus.label" :color="exposureModalStatus.color" variant="soft" />
                <span class="text-xs text-gray-500">
                  {{ $t("capabilities.exposure.updatedAt", { time: exposurePackageInfo?.updated_at || "—" }) }}
                </span>
              </div>
            </div>
          </div>

          <UCard :ui="{ body: 'space-y-8' }">
            <section class="space-y-4">
              <div class="grid gap-4 md:grid-cols-2">
                <UFormField :label="$t('capabilities.exposure.fields.capabilityId')" required class="md:col-span-2">
                  <UInput v-model="exposureForm.capability_id" readonly />
                </UFormField>
                <UFormField :label="$t('capabilities.exposure.fields.docsVersion')">
                  <UInput v-model="exposureForm.docs_version" placeholder="1.0.0" />
                </UFormField>
                <UFormField :label="$t('capabilities.exposure.fields.sdkVersion')">
                  <UInput v-model="exposureForm.sdk_version" placeholder="1.0.0" />
                </UFormField>
              </div>
              <div class="flex flex-wrap gap-3">
                <UButton variant="ghost" color="neutral" :loading="exposureLoading" @click="handleExposureLoad">
                  {{ $t("capabilities.exposure.actions.load") }}
                </UButton>
                <UButton variant="ghost" color="neutral" @click="resetExposureChannels">
                  {{ $t("capabilities.exposure.actions.resetChannels") }}
                </UButton>
                <UButton color="primary" :disabled="!exposureForm.capability_id" :loading="exposureSaving" @click="handleExposureSave">
                  {{ $t("common.save") }}
                </UButton>
              </div>
            </section>

            <section class="space-y-4">
              <div class="flex items-center justify-between">
                <h2 class="text-lg font-semibold">
                  {{ $t("capabilities.exposure.sections.channels") }}
                </h2>
                <span class="text-sm text-gray-500">
                  {{ selectedExposureChannels.length }}/{{ exposureForm.channels.length }}
                  {{ $t("capabilities.exposure.sections.enabled") }}
                </span>
              </div>
              <div class="grid gap-4 lg:grid-cols-2">
                <div
                  v-for="channel in exposureForm.channels"
                  :key="channel.type"
                  class="border rounded-lg border-gray-200 dark:border-gray-800 p-4 space-y-3"
                >
                  <div class="flex items-center justify-between">
                    <div class="flex flex-col">
                      <span class="font-semibold">{{ channel.name }}</span>
                      <span class="text-xs text-gray-500">
                        <template v-if="channel.definitionKey && selectedCapability">
                          {{ $t("capabilities.exposure.sections.descriptorNote", {
                            path: selectedCapability.descriptor,
                            key: channel.definitionKey,
                          }) }}
                        </template>
                        <template v-else>
                          {{ $t("capabilities.exposure.sections.channelHint") }}
                        </template>
                      </span>
                    </div>
                    <USwitch v-model="channel.enabled" color="primary" />
                  </div>
                  <div class="space-y-3">
                    <div
                      v-if="channel.definition"
                      class="rounded-lg border border-gray-200 dark:border-gray-800 bg-gray-50/50 dark:bg-white/5 p-3 text-xs space-y-1"
                    >
                      <div
                        v-for="entry in channelDefinitionEntries(channel)"
                        :key="`${channel.type}-${entry.label}`"
                        class="flex items-start justify-between gap-3"
                      >
                        <span class="text-gray-500">{{ entry.label }}</span>
                        <span class="font-semibold text-gray-200 dark:text-white break-all">{{ entry.value }}</span>
                      </div>
                    </div>
                    <div v-else class="text-xs text-gray-500">
                      {{ $t("capabilities.exposure.sections.channelHint") }}
                    </div>
                    <UFormField :label="$t('capabilities.exposure.fields.scopes')">
                      <UInput
                        v-model="channel.scopesText"
                        :placeholder="$t('capabilities.exposure.fields.scopePlaceholder')"
                        @change="syncExposureScopeList(channel)"
                      />
                    </UFormField>
                  </div>
                </div>
              </div>
            </section>

            <section class="grid gap-4 md:grid-cols-2">
              <div class="space-y-3">
                <h2 class="text-lg font-semibold">
                  {{ $t("capabilities.exposure.sections.auth") }}
                </h2>
                <UFormField :label="$t('capabilities.exposure.fields.strategy')">
                  <USelectMenu v-model="exposureForm.auth.strategy" :options="exposureTemplate?.auth_strategies || []" />
                </UFormField>
                <UFormField :label="$t('capabilities.exposure.fields.audience')">
                  <UInput v-model="exposureForm.auth.audience" />
                </UFormField>
                <UFormField :label="$t('capabilities.exposure.fields.scopeList')">
                  <UInput
                    v-model="exposureAuthScopes"
                    :placeholder="$t('capabilities.exposure.fields.scopePlaceholder')"
                    @change="syncExposureAuthScopes"
                  />
                </UFormField>
              </div>
              <div class="space-y-3">
                <h2 class="text-lg font-semibold">
                  {{ $t("capabilities.exposure.sections.rateLimit") }}
                </h2>
                <UFormField :label="$t('capabilities.exposure.fields.rpm')">
                  <UInput v-model.number="exposureForm.rate_limit.requests_per_minute" type="number" min="1" />
                </UFormField>
                <UFormField :label="$t('capabilities.exposure.fields.burst')">
                  <UInput v-model.number="exposureForm.rate_limit.burst" type="number" min="1" />
                </UFormField>
                <UFormField :label="$t('capabilities.exposure.fields.concurrency')">
                  <UInput v-model.number="exposureForm.rate_limit.concurrency" type="number" min="1" />
                </UFormField>
              </div>
            </section>

            <section class="space-y-4">
              <div class="flex items-center justify-between">
                <h2 class="text-lg font-semibold">
                  {{ $t("capabilities.exposure.sections.quotas") }}
                </h2>
                <div class="text-sm text-gray-500">
                  {{ $t("capabilities.exposure.sections.tenantHint") }}
                </div>
              </div>
              <div class="overflow-x-auto border rounded-lg border-gray-200 dark:border-gray-800">
                <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-800 text-sm">
                  <thead class="bg-gray-50 dark:bg-gray-900">
                    <tr>
                      <th class="px-4 py-2 text-left font-semibold">{{ $t("capabilities.exposure.fields.tenant") }}</th>
                      <th class="px-4 py-2 text-left font-semibold">{{ $t("capabilities.exposure.fields.quota") }}</th>
                      <th class="px-4 py-2 text-left font-semibold">{{ $t("capabilities.exposure.fields.status") }}</th>
                      <th class="px-4 py-2 text-left font-semibold w-48">{{ $t("capabilities.exposure.fields.notes") }}</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
                    <tr v-if="exposureQuotasList.length === 0">
                      <td colspan="4" class="px-4 py-4 text-center text-gray-500">
                        {{ $t("capabilities.exposure.sections.noQuotas") }}
                      </td>
                    </tr>
                    <tr v-for="quota in exposureQuotasList" :key="quota.tenant_id">
                      <td class="px-4 py-2">
                        <div class="font-medium">{{ quota.tenant_id }}</div>
                        <div class="text-xs text-gray-500">{{ quota.tenant_name }}</div>
                      </td>
                      <td class="px-4 py-2">
                        {{ quota.used || 0 }} / {{ quota.quota }}
                      </td>
                      <td class="px-4 py-2">
                        <UBadge :label="quota.status || 'active'" color="primary" variant="soft" />
                      </td>
                      <td class="px-4 py-2 break-words">
                        <span class="text-xs text-gray-500">{{ quota.notes }}</span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <div class="border rounded-lg border-gray-200 dark:border-gray-800 p-4 space-y-3">
                <div class="grid gap-3 md:grid-cols-2">
                  <UFormField :label="$t('capabilities.exposure.fields.tenantId')" required>
                    <UInput v-model="newExposureQuota.tenant_id" />
                  </UFormField>
                  <UFormField :label="$t('capabilities.exposure.fields.tenantName')">
                    <UInput v-model="newExposureQuota.tenant_name" />
                  </UFormField>
                  <UFormField :label="$t('capabilities.exposure.fields.quota')" required>
                    <UInput v-model.number="newExposureQuota.quota" type="number" min="0" />
                  </UFormField>
                  <UFormField :label="$t('capabilities.exposure.fields.status')">
                    <USelectMenu v-model="newExposureQuota.status" :options="statusOptions" />
                  </UFormField>
                  <UFormField class="md:col-span-2" :label="$t('capabilities.exposure.fields.notes')">
                    <UInput v-model="newExposureQuota.notes" />
                  </UFormField>
                </div>
                <div class="flex justify-end">
                  <UButton
                    color="neutral"
                    variant="soft"
                    :disabled="!exposureForm.capability_id"
                    :loading="exposureQuotaSaving"
                    @click="handleExposureQuotaSave"
                  >
                    {{ $t("capabilities.exposure.actions.addTenant") }}
                  </UButton>
                </div>
              </div>
            </section>
          </UCard>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from "vue";
import { useDebounceFn } from "@vueuse/core";
import { useI18n, useRoute, useRouter, useToast } from "#imports";
import {
  useCapabilityRegistryApi,
  type CapabilityTemplate,
  type CapabilityRegisterPayload,
  type CapabilityValidationResult,
  useCapabilityCatalogApi,
  type CapabilityCatalogEntry,
} from "~/composables/api";
import {
  useCapabilityExposureApi,
  type ExposureTemplate,
  type ExposureChannel,
  type ExposurePackage,
  type TenantQuota,
} from "~/composables/api/useCapabilityExposure";
import {
  useMcpSessionApi,
  type McpSession,
  type McpInvokeResult,
} from "~/composables/api/useMcpSession";
import { resolveApiBase } from "~/composables/api/_base";
import { useNormalizedColumns } from "~/utils/table";

definePageMeta({
  alias: ["/capabilities/register", "/capabilities/exposure"],
});

const { t } = useI18n();
const toast = useToast();
const route = useRoute();
const router = useRouter();
const {
  fetchTemplate,
  validateDraft,
  submitDraft,
} = useCapabilityRegistryApi();
const { list: listCatalog } = useCapabilityCatalogApi();
const {
  getTemplate: getExposureTemplate,
  getPackage: fetchExposurePackage,
  upsertPackage: saveExposurePackage,
  listQuotas: fetchExposureQuotas,
  upsertQuota: saveExposureQuota,
} = useCapabilityExposureApi();
const runtimeConfig = useRuntimeConfig();
const pluginCapabilityDocUrl = "https://github.com/ArtisanCloud/PowerXPlugin/blob/main/docs/guides/develop/plugin-capability/README.md";
const mcpGuideUrl = "https://github.com/ArtisanCloud/PowerXPlugin/blob/main/docs/guides/publish/capabilities/mcp-guide.md";
const {
  invokeCapability: invokeDebugCapability,
  clearHistory: clearDebugHistory,
  loading: debugInvokeLoading,
  result: debugResult,
  warnings: debugWarnings,
  errorMessage: debugErrorMessage,
  lastTraceId: debugLastTraceId,
  durationMs: debugDuration,
  history: debugHistory,
  errorDetails: debugErrorDetails,
} = useCapabilityLab();

const runtimeDebugApiBase =
  (runtimeConfig.public?.powerx?.apiBase as string | undefined) ??
  (runtimeConfig.public?.apiBase as string | undefined) ??
  "";
const defaultDebugApiBase = normalizeLocalDebugApiBase(runtimeDebugApiBase);
const gatewayDebugApiFallback = runtimeDebugApiBase || "/api/v1";
const debugDefaults = {
  capabilityId: "",
  action: "",
  preferredProtocol: "",
  targetMode: "local" as "local" | "gateway",
  payloadText: "{\n  \n}",
  tenantUuid: "00000000-0000-0000-0000-000000000001",
  mockModule: "",
  requestId: generateDebugRequestId(),
  apiBase: defaultDebugApiBase,
};
const mcpInvokeSamples: Record<
  string,
  {
    toolScope: string;
    payload: JsonMap;
  }
> = {
  "com.powerx.plugins.base.template.compose": {
    toolScope: "agent.template.compose",
    payload: {
      draft: {
        name: "Demo Template",
        description: "由 MCP 指南自动创建",
        content: "## hello world",
      },
      review: {
        reviewer: "qa-bot",
        comment: "looks good",
      },
      publish_channel: "channel:demo",
      cleanup: {
        reason: "archived after publish",
      },
    },
  },
  "com.powerx.plugins.base.template.audit": {
    toolScope: "agent.template.audit",
    payload: {
      filters: {
        status: "",
        page: 1,
        page_size: 5,
      },
      update_payload: {
        description: "reviewed by qa-team",
        content: "## patched content",
        metadata: {
          owner: "qa-team",
        },
      },
    },
  },
  "com.powerx.plugins.base.template.quality_distribute": {
    toolScope: "agent.template.quality_distribute",
    payload: {
      scan_filter: {
        q: "demo",
        page: 1,
        page_size: 5,
      },
      validate_rules: ["name_not_empty", "content_min_length"],
      clone: {
        copies: 2,
        name_prefix: "qa-copy-",
        description_prefix: "[auto]",
      },
      update_payload: {
        description: "distributed by QA",
        content: "## fixed content",
      },
      publish_channel: "channel:qa-lab",
    },
  },
};
const debugForm = reactive({ ...debugDefaults });
const debugPayloadTouched = ref(false);
const debugAutoFillingPayload = ref(false);
const debugShowRequestPreview = ref(true);
const debugActionTouched = ref(false);
type DebugContext = "draft" | "catalog" | null;
type DebugProtocols = {
  rest?: { method?: string; path?: string };
  grpc?: { service?: string; method?: string };
  workflow?: { template?: string };
};
const debugModalOpen = ref(false);
const debugContext = ref<DebugContext>(null);
const debugSourceProtocols = ref<DebugProtocols | null>(null);

const formOpen = ref(false);
const loadingTemplate = ref(true);
const currentStep = ref(0);
const template = ref<CapabilityTemplate | null>(null);
const validating = ref(false);
const savingDraft = ref(false);
const submitting = ref(false);
const validationResult = ref<CapabilityValidationResult | null>(null);
const savedAt = ref<string | null>(null);
const autoValidate = ref(false);
const draftStorageKey = "powerxplugin::capability-register-draft";
const catalogLoading = ref(false);
const catalog = ref<CapabilityCatalogEntry[]>([]);
const catalogCommand = computed(() => t("capabilities.catalogSync.command"));
type CatalogRow = {
  capability_id: string;
  version: string;
  descriptor: string;
  tags: string[];
  checksum: string;
  execution: CapabilityCatalogEntry["execution"];
  module: string;
  kind: string;
  syncStatus: string;
  updatedAt?: string;
  protocols?: Record<string, any>;
};
const catalogRows = ref<CatalogRow[]>([]);
type ModuleGroup = {
  module: string;
  displayName: string;
  items: CatalogRow[];
  kindBadges: { label: string; color: string }[];
};
type ChannelFormEntry = ExposureChannel & {
  scopesText?: string;
  definition?: Record<string, any> | null;
  definitionKey?: string;
};
type ExposureMetaState = { status: string; updated_at?: string };
type StatusBadge = { label: string; color: string };
type JsonMap = Record<string, any>;
const exposureFormOpen = ref(false);
const exposureTemplate = ref<ExposureTemplate | null>(null);
const exposurePackageInfo = ref<ExposurePackage | null>(null);
const exposureLoading = ref(false);
const exposureSaving = ref(false);
const exposureQuotaSaving = ref(false);
const exposureForm = reactive({
  capability_id: "",
  docs_version: "1.0.0",
  sdk_version: "1.0.0",
  auth: {
    strategy: "powerx_session",
    audience: "",
    scopes: [] as string[],
  },
  rate_limit: {
    requests_per_minute: 600,
    burst: 120,
    concurrency: 10,
  },
  channels: [] as ChannelFormEntry[],
});
const exposureQuotas = ref<TenantQuota[]>([]);
const newExposureQuota = reactive({
  tenant_id: "",
  tenant_name: "",
  quota: 1000,
  status: "active",
  notes: "",
});
const exposureAuthScopes = ref("");
const exposureMeta = ref<Record<string, ExposureMetaState>>({});
const statusOptions = ["active", "suspended"];
const groupedCatalog = computed<ModuleGroup[]>(() => {
  const groups = new Map<string, CatalogRow[]>();
  catalogRows.value.forEach((row) => {
    const moduleKey = row.module || deriveModuleFromId(row.capability_id);
    if (!groups.has(moduleKey)) {
      groups.set(moduleKey, []);
    }
    groups.get(moduleKey)!.push(row);
  });
  return Array.from(groups.entries())
    .map(([module, items]) => {
      const stats = items.reduce<Record<string, number>>((acc, item) => {
        const key = normalizeKind(item.kind);
        acc[key] = (acc[key] || 0) + 1;
        return acc;
      }, {});
      return {
        module,
        displayName: formatModuleDisplay(module),
        items: items.sort((a, b) => a.capability_id.localeCompare(b.capability_id)),
        kindBadges: Object.entries(stats).map(([kind, count]) => ({
          label: `${formatKindLabel(kind)} · ${count}`,
          color: kindColor(kind),
        })),
      };
    })
    .sort((a, b) => a.module.localeCompare(b.module));
});
const selectedExposureChannels = computed(() =>
  exposureForm.channels.filter((channel) => channel.enabled),
);
const exposureQuotasList = computed(() => exposureQuotas.value ?? []);
const statusPresets = computed<Record<string, StatusBadge>>(() => ({
  unconfigured: {
    label: t("capabilities.exposure.list.status.unconfigured"),
    color: "gray",
  },
  synced: {
    label: t("capabilities.exposure.list.status.synced"),
    color: "primary",
  },
  pending: {
    label: t("capabilities.exposure.list.status.pending"),
    color: "amber",
  },
  failed: {
    label: t("capabilities.exposure.list.status.failed"),
    color: "rose",
  },
}));
const exposureModalStatus = computed(() =>
  exposureBadge(exposurePackageInfo.value?.sync_status),
);
const selectedCapability = computed(() =>
  catalog.value.find((entry) => entry.id === exposureForm.capability_id) ||
  null,
);
const expandedModules = ref<Record<string, boolean>>({});

const form = reactive(createDefaultForm());

const httpMethods = ["POST", "PUT", "PATCH", "GET", "DELETE"];

const capabilityId = computed(() =>
  buildCapabilityId(
    form.namespace || template.value?.namespace || "",
    form.resource,
    form.action,
  ),
);

const stepItems = computed(() => [
  {
    label: t("capabilities.form.steps.basic"),
    description: t("capabilities.form.steps.basicDesc"),
  },
  {
    label: t("capabilities.form.steps.protocol"),
    description: t("capabilities.form.steps.protocolDesc"),
  },
  {
    label: t("capabilities.form.steps.samples"),
    description: t("capabilities.form.steps.samplesDesc"),
  },
]);

const validationBadge = computed(() => {
  if (!validationResult.value) {
    return { label: t("capabilities.form.validationUnknown"), color: "neutral" };
  }
  if (validationResult.value.valid) {
    return { label: t("capabilities.form.validationPassed"), color: "green" };
  }
  return { label: t("capabilities.form.validationFailedShort"), color: "rose" };
});

const tagsText = computed({
  get() {
    return form.tags.join(", ");
  },
  set(value: string) {
    form.tags = value
      .split(",")
      .map((tag) => tag.trim())
      .filter(Boolean);
  },
});

const tableColumns = useNormalizedColumns([
  { key: "capability_id", label: t("capabilities.list.column.capability") },
  { key: "kind", label: t("capabilities.list.column.kind") },
  { key: "execution", label: t("capabilities.list.column.execution") },
  { key: "exposure", label: t("capabilities.exposure.list.columns.exposure") },
  { key: "tags", label: t("capabilities.list.column.tags") },
  { key: "checksum", label: t("capabilities.list.column.checksum") },
  { key: "actions", label: "" },
]);
const debugTargetOptions = computed(() => [
  { value: "local", label: t("capabilities.form.debug.target.local") },
  { value: "gateway", label: t("capabilities.form.debug.target.gateway") },
]);
const debugProtocolOptions = computed(() => {
  const protocols = debugSourceProtocols.value || {};
  const options: Array<{ value: string; label: string }> = [];
  if (protocols.rest) {
    options.push({ value: "rest", label: "REST" });
  }
  if (protocols.grpc) {
    options.push({ value: "grpc", label: "gRPC" });
  }
  if (protocols.workflow) {
    options.push({ value: "workflow", label: "Workflow" });
  }
  if (!options.length) {
    options.push({ value: "rest", label: "REST" });
  }
  return options;
});
const debugPayloadTemplate = computed(() => {
  const isLocal = debugForm.targetMode === "local";
  switch (debugForm.preferredProtocol) {
    case "rest":
      return buildDebugRestTemplate();
    case "grpc":
      return isLocal ? buildLocalGrpcTemplate() : buildDebugGrpcTemplate();
    case "workflow":
      return isLocal ? buildLocalWorkflowTemplate() : buildDebugWorkflowTemplate();
    default:
      return "";
  }
});
const debugPayloadValid = computed(() => {
  try {
    parseDebugPayload();
    return true;
  } catch {
    return false;
  }
});
const debugRequestPreview = computed(() => {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (debugForm.tenantUuid.trim()) {
    headers["X-Tenant-UUID"] = debugForm.tenantUuid.trim();
  }
  if (debugForm.mockModule.trim()) {
    headers["X-PX-Use-Mock"] = debugForm.mockModule.trim();
  }
  if (debugForm.requestId.trim()) {
    headers["X-Request-ID"] = debugForm.requestId.trim();
  }
  if (debugForm.targetMode === "local") {
    const payload = safeDebugPayload();
    const method = typeof payload?.method === "string" ? payload.method.toUpperCase() : "GET";
    const endpoint = typeof payload?.endpoint === "string" ? payload.endpoint : "<missing endpoint>";
    const url = buildPreviewLocalUrl(debugForm.apiBase || defaultDebugApiBase, endpoint, payload?.query);
    return {
      mode: "local",
      method,
      url,
      headers: {
        ...headers,
        ...(payload?.headers || {}),
      },
      body: payload?.body ?? {},
    };
  }
  const body: Record<string, any> = {
    capabilityId: debugForm.capabilityId || capabilityId.value || "<empty>",
    action: debugForm.action || form.action || "<empty>",
    payload: safeDebugPayload(),
  };
  if (debugForm.preferredProtocol) {
    body.preferredProtocol = debugForm.preferredProtocol;
  }
  return {
    mode: "gateway",
    url: combineURL(
      ensureGatewayApiBase(debugForm.apiBase || defaultDebugApiBase, gatewayDebugApiFallback),
      "/integration/capabilities/invoke",
    ),
    method: "POST",
    headers,
    body,
  };
});
const debugRequestPreviewText = computed(() =>
  JSON.stringify(debugRequestPreview.value, null, 2),
);
const debugHistoryEntries = computed(() => {
  const targetId = debugForm.capabilityId?.trim();
  const targetAction = debugForm.action?.trim();
  return debugHistory.value.filter((entry) => {
    if (targetId && entry.capabilityId !== targetId) {
      return false;
    }
    if (targetAction && entry.action !== targetAction) {
      return false;
    }
    return true;
  });
});

const {
  registerSession,
  ackSession,
  heartbeatSession,
  closeSession,
  invokeSession,
} = useMcpSessionApi();

type McpStreamEvent = {
  session_id: string;
  type: string;
  payload?: any;
  timestamp: string;
};

type McpInvokeHistoryEntry = {
  id: string;
  timestamp: string;
  status: string;
  traceId?: string;
  correlationId?: string;
  payload?: any;
  success: boolean;
  error?: string | null;
};

const mcpModalOpen = ref(false);
const mcpSession = ref<McpSession | null>(null);
const mcpSessionForm = reactive(createDefaultMcpSessionForm());
const mcpRegisterLoading = ref(false);
const mcpAckLoading = ref(false);
const mcpHeartbeatLoading = ref(false);
const mcpCloseLoading = ref(false);
const mcpInvokeForm = reactive(createDefaultMcpInvokeForm());
const mcpInvokeLoading = ref(false);
const mcpInvokeResult = ref<McpInvokeResult | null>(null);
const mcpInvokeErrorMessage = ref<string | null>(null);
const mcpInvokeErrorDetails = ref<any>(null);
const mcpInvokeHistory = ref<McpInvokeHistoryEntry[]>([]);
const mcpEvents = ref<McpStreamEvent[]>([]);
const mcpStreamConnected = ref(false);
const mcpStreamError = ref<string | null>(null);
const mcpEventSource = ref<EventSource | null>(null);

watch(debugModalOpen, (open) => {
  if (!open) {
    debugContext.value = null;
    debugSourceProtocols.value = null;
    debugPayloadTouched.value = false;
    debugActionTouched.value = false;
  }
});

watch(
  capabilityId,
  (value) => {
    if (debugContext.value === "draft" && debugModalOpen.value) {
      debugForm.capabilityId = value || "";
    }
  },
);

let syncingDebugAction = false;
watch(
  () => form.action,
  (value) => {
    if (debugContext.value !== "draft" || !debugModalOpen.value) {
      return;
    }
    if (debugActionTouched.value) {
      return;
    }
    syncingDebugAction = true;
    debugForm.action = value || "";
    syncingDebugAction = false;
  },
);

watch(
  () => debugForm.action,
  () => {
    if (!syncingDebugAction) {
      debugActionTouched.value = true;
    }
  },
);

watch(
  debugProtocolOptions,
  (options) => {
    if (!options.length) {
      debugForm.preferredProtocol = "";
      debugPayloadTouched.value = false;
      return;
    }
    if (!options.some((option) => option.value === debugForm.preferredProtocol)) {
      debugForm.preferredProtocol = options[0].value;
      debugPayloadTouched.value = false;
    }
  },
  { immediate: true },
);

watch(
  () => debugPayloadTemplate.value,
  (template) => {
    if (!debugModalOpen.value) return;
    if (!template) return;
    if (debugPayloadTouched.value) return;
    debugAutoFillingPayload.value = true;
    debugForm.payloadText = template;
    nextTick(() => {
      debugAutoFillingPayload.value = false;
    });
  },
  { immediate: true },
);

watch(
  () => debugForm.payloadText,
  () => {
    if (!debugAutoFillingPayload.value && debugModalOpen.value) {
      debugPayloadTouched.value = true;
    }
  },
);

watch(
  () => ({
    restMethod: form.protocols.rest.method,
    restPath: form.protocols.rest.path,
    grpcService: form.protocols.grpc.service,
    workflowTemplate: form.protocols.workflow.template,
  }),
  () => {
    if (debugContext.value !== "draft" || !debugModalOpen.value) {
      return;
    }
    debugSourceProtocols.value = buildDebugProtocolsFromForm();
  },
);

watch(
  () => mcpSessionForm.sessionId,
  (sessionId, previousSessionId) => {
    mcpInvokeForm.sessionId = sessionId || "";
    if (sessionId) {
      ensureMcpMetadataSession(sessionId);
      if (mcpStreamConnected.value && sessionId !== previousSessionId) {
        connectMcpStream(sessionId);
      }
    } else if (mcpStreamConnected.value) {
      disconnectMcpStream();
    }
  },
);

watch(
  () => mcpInvokeForm.capabilityId,
  (capabilityId) => {
    ensureMcpMetadataSession();
    maybeApplyMcpSample(capabilityId?.trim());
  },
);

watch(
  () => mcpInvokeForm.intent,
  () => {
    ensureMcpMetadataSession();
  },
);

watch(mcpModalOpen, (open) => {
  if (!open && mcpStreamConnected.value) {
    disconnectMcpStream();
  }
});

onUnmounted(() => {
  disconnectMcpStream();
});

onMounted(async () => {
  await Promise.all([loadTemplate(), loadCatalog(), hydrateExposureTemplate()]);
  hydrateDraft();
  watchFormForPersist();
  const capabilityFromQuery = route.query.capability as string | undefined;
  if (capabilityFromQuery) {
    await openExposureForm(capabilityFromQuery);
  }
});

watch(
  () => catalog.value,
  () => syncCatalogRows(),
  { deep: true },
);

watch(
  () => exposureMeta.value,
  () => syncCatalogRows(),
  { deep: true },
);

watch(
  () => route.query.capability,
  async (capability) => {
    if (typeof capability === "string" && capability) {
      if (exposureForm.capability_id !== capability || !exposureFormOpen.value) {
        await openExposureForm(capability);
      }
    }
  },
);

watch(formOpen, (isOpen) => {
  if (!isOpen) {
    currentStep.value = 0;
  }
});

watch(
  groupedCatalog,
  (groups) => {
    groups.forEach((group) => {
      if (!(group.module in expandedModules.value)) {
        expandedModules.value[group.module] = true;
      }
    });
  },
  { immediate: true },
);

function isModuleExpanded(module: string) {
  const state = expandedModules.value[module];
  return state === undefined ? true : state;
}

function toggleModule(module: string) {
  expandedModules.value[module] = !isModuleExpanded(module);
}

function deriveModuleFromId(id: string) {
  const parts = id.split(".");
  if (parts.length <= 1) {
    return id;
  }
  return parts.slice(0, -1).join(".");
}

function formatModuleDisplay(module: string) {
  const parts = module.split(".");
  if (parts.length <= 1) {
    return module;
  }
  return parts.slice(-1)[0];
}

function normalizeKind(kind?: string) {
  return kind?.trim() || "Capability";
}

function formatKindLabel(kind?: string) {
  const normalized = normalizeKind(kind).toLowerCase();
  if (normalized === "workflow" || normalized === "tool") {
    return t("capabilities.list.kind.workflow");
  }
  if (normalized === "capability") {
    return t("capabilities.list.kind.capability");
  }
  return t("capabilities.list.kind.default");
}

function kindColor(kind?: string) {
  const normalized = normalizeKind(kind).toLowerCase();
  if (normalized === "workflow" || normalized === "tool") {
    return "primary";
  }
  return "gray";
}

function isWorkflowKind(kind?: string) {
  const normalized = normalizeKind(kind).toLowerCase();
  return normalized === "workflow" || normalized === "tool";
}

function createDefaultMcpSessionForm() {
  return {
    runtimeAssignmentId: "",
    registerState: "registering",
    jwtId: "dev-mcp-client",
    capabilitiesHash: "",
    sessionId: "",
    ackState: "ready",
    heartbeatMissed: 0,
    closeReason: "",
  };
}

function createDefaultMcpInvokeForm() {
  return {
    sessionId: "",
    tenantUuid: debugDefaults.tenantUuid,
    toolScope: "",
    capabilityId: "",
    intent: "",
    messageId: generateUuid(),
    traceId: generateUuid(),
    correlationId: generateUuid(),
    issuedAt: new Date().toISOString(),
    idempotencyKey: "",
    payloadText: "{\n  \n}",
    metadataText: "{\n  \n}",
    signature: "stub-signature",
  };
}

function openMcpPanel() {
  if (!mcpSessionForm.runtimeAssignmentId) {
    regenerateMcpRuntimeAssignment();
  }
  mcpModalOpen.value = true;
}

function openMcpPanelForCapability(capabilityId?: string) {
  const trimmed = capabilityId?.trim();
  if (trimmed) {
    mcpInvokeForm.capabilityId = trimmed;
    mcpInvokeForm.intent = trimmed;
    applyMcpSample(trimmed, { force: true });
    if (!mcpInvokeForm.toolScope.trim()) {
      mcpInvokeForm.toolScope = deriveToolScopeFromCapability(trimmed);
    }
    ensureMcpMetadataSession();
  }
  openMcpPanel();
}

function regenerateMcpRuntimeAssignment() {
  mcpSessionForm.runtimeAssignmentId = generateUuid();
}

async function handleMcpRegister() {
  if (!mcpSessionForm.runtimeAssignmentId.trim()) {
    toast.add({
      title: t("capabilities.mcp.toast.runtimeRequired"),
      color: "rose",
    });
    return;
  }
  mcpRegisterLoading.value = true;
  try {
    const session = await registerSession({
      runtime_assignment_id: mcpSessionForm.runtimeAssignmentId.trim(),
      state: mcpSessionForm.registerState?.trim() || "registering",
      jwt_id: mcpSessionForm.jwtId?.trim() || undefined,
      capabilities_hash: mcpSessionForm.capabilitiesHash?.trim() || undefined,
    });
    applyMcpSession(session);
    toast.add({
      title: t("capabilities.mcp.toast.registered"),
      color: "green",
    });
  } catch (error) {
    toast.add({
      title: extractErrorMessage(error),
      color: "rose",
    });
  } finally {
    mcpRegisterLoading.value = false;
  }
}

async function handleMcpAck() {
  if (!mcpSessionForm.sessionId.trim()) {
    toast.add({
      title: t("capabilities.mcp.toast.sessionRequired"),
      color: "rose",
    });
    return;
  }
  mcpAckLoading.value = true;
  try {
    const session = await ackSession(mcpSessionForm.sessionId.trim(), {
      state: mcpSessionForm.ackState?.trim() || "ready",
      capabilities_hash: mcpSessionForm.capabilitiesHash?.trim() || undefined,
    });
    applyMcpSession(session);
    toast.add({
      title: t("capabilities.mcp.toast.acknowledged"),
      color: "green",
    });
  } catch (error) {
    toast.add({
      title: extractErrorMessage(error),
      color: "rose",
    });
  } finally {
    mcpAckLoading.value = false;
  }
}

async function handleMcpHeartbeat() {
  if (!mcpSessionForm.sessionId.trim()) {
    toast.add({
      title: t("capabilities.mcp.toast.sessionRequired"),
      color: "rose",
    });
    return;
  }
  mcpHeartbeatLoading.value = true;
  try {
    const session = await heartbeatSession(mcpSessionForm.sessionId.trim(), {
      missed_heartbeats: mcpSessionForm.heartbeatMissed || 0,
    });
    applyMcpSession(session);
    toast.add({
      title: t("capabilities.mcp.toast.heartbeat"),
      color: "green",
    });
  } catch (error) {
    toast.add({
      title: extractErrorMessage(error),
      color: "rose",
    });
  } finally {
    mcpHeartbeatLoading.value = false;
  }
}

async function handleMcpClose() {
  if (!mcpSessionForm.sessionId.trim()) {
    toast.add({
      title: t("capabilities.mcp.toast.sessionRequired"),
      color: "rose",
    });
    return;
  }
  mcpCloseLoading.value = true;
  try {
    const session = await closeSession(mcpSessionForm.sessionId.trim(), {
      reason: mcpSessionForm.closeReason?.trim() || undefined,
    });
    applyMcpSession(session);
    toast.add({
      title: t("capabilities.mcp.toast.closed"),
      color: "green",
    });
  } catch (error) {
    toast.add({
      title: extractErrorMessage(error),
      color: "rose",
    });
  } finally {
    mcpCloseLoading.value = false;
  }
}

async function handleMcpInvoke() {
  if (!mcpInvokeForm.sessionId.trim()) {
    toast.add({
      title: t("capabilities.mcp.toast.sessionRequired"),
      color: "rose",
    });
    return;
  }
  if (!mcpInvokeForm.toolScope.trim()) {
    toast.add({
      title: t("capabilities.mcp.toast.toolScopeRequired"),
      color: "rose",
    });
    return;
  }
  mcpInvokeLoading.value = true;
  try {
    const payloadRef = buildPayloadRef(mcpInvokeForm.payloadText);
    const metadata = buildInvokeMetadata();
    const issuedAt = normalizeIssuedAt(mcpInvokeForm.issuedAt);
    const result = await invokeSession(mcpInvokeForm.sessionId.trim(), {
      message_id: mcpInvokeForm.messageId.trim() || generateUuid(),
      trace_id: mcpInvokeForm.traceId.trim() || generateUuid(),
      correlation_id: mcpInvokeForm.correlationId.trim() || generateUuid(),
      tenant_uuid: mcpInvokeForm.tenantUuid.trim() || debugDefaults.tenantUuid,
      tool_scope: mcpInvokeForm.toolScope.trim(),
      issued_at: issuedAt,
      idempotency_key: mcpInvokeForm.idempotencyKey?.trim() || undefined,
      payload_ref: payloadRef,
      metadata,
      signature: mcpInvokeForm.signature?.trim() || "stub-signature",
    });
    mcpInvokeResult.value = result;
    mcpInvokeErrorMessage.value = null;
    mcpInvokeErrorDetails.value = null;
    pushMcpInvokeHistory({
      id: generateUuid(),
      timestamp: new Date().toISOString(),
      status: result.status,
      traceId: result.trace_id,
      correlationId: result.correlation_id,
      payload: result.payload ?? result.metadata ?? null,
      success: true,
      error: null,
    });
    toast.add({
      title: t("capabilities.mcp.toast.invoked"),
      color: "green",
    });
  } catch (error: any) {
    const message = error instanceof Error ? error.message : extractErrorMessage(error);
    const details =
      error?.response?._data?.error?.details ??
      error?.response?._data?.data ??
      error?.response?._data ??
      null;
    mcpInvokeResult.value = null;
    mcpInvokeErrorMessage.value = message;
    mcpInvokeErrorDetails.value = details;
    pushMcpInvokeHistory({
      id: generateUuid(),
      timestamp: new Date().toISOString(),
      status: "error",
      traceId: details?.trace_id || undefined,
      correlationId: details?.correlation_id || undefined,
      payload: details,
      success: false,
      error: message,
    });
    toast.add({
      title: message,
      color: "rose",
    });
  } finally {
    mcpInvokeLoading.value = false;
  }
}

function applyMcpSession(session: McpSession | null) {
  if (!session) {
    mcpSession.value = null;
    return;
  }
  const resolvedSessionId =
    session.id?.trim() ||
    mcpSessionForm.sessionId?.trim() ||
    mcpSession.value?.id ||
    "";
  const resolvedAssignmentId =
    session.runtime_assignment_id?.trim() ||
    mcpSessionForm.runtimeAssignmentId?.trim() ||
    mcpSession.value?.runtime_assignment_id ||
    "";
  const resolvedTenantUuid =
    session.tenant_uuid?.trim() || mcpSession.value?.tenant_uuid || "";

  mcpSession.value = {
    ...session,
    id: resolvedSessionId,
    runtime_assignment_id: resolvedAssignmentId,
    tenant_uuid: resolvedTenantUuid,
  };

  if (resolvedSessionId) {
    mcpSessionForm.sessionId = resolvedSessionId;
    mcpInvokeForm.sessionId = resolvedSessionId;
  }
  if (session.runtime_assignment_id?.trim()) {
    mcpSessionForm.runtimeAssignmentId = session.runtime_assignment_id.trim();
  }
  if (session.capabilities_hash) {
    mcpSessionForm.capabilitiesHash = session.capabilities_hash;
  }
  if (resolvedTenantUuid) {
    mcpInvokeForm.tenantUuid = resolvedTenantUuid;
  }
  ensureMcpMetadataSession(resolvedSessionId);
}

function pushMcpInvokeHistory(entry: McpInvokeHistoryEntry) {
  mcpInvokeHistory.value = [entry, ...mcpInvokeHistory.value].slice(0, 5);
}

function clearMcpInvokeHistory() {
  mcpInvokeHistory.value = [];
}

function buildPayloadRef(text: string) {
  const trimmed = text?.trim();
  if (!trimmed) {
    return "{}";
  }
  try {
    return JSON.stringify(JSON.parse(trimmed));
  } catch {
    throw new Error(t("capabilities.mcp.errors.invalidPayload"));
  }
}

function buildInvokeMetadata() {
  const trimmed = mcpInvokeForm.metadataText?.trim();
  let metadata: Record<string, any> = {};
  if (trimmed) {
    try {
      metadata = JSON.parse(trimmed);
    } catch {
      throw new Error(t("capabilities.mcp.errors.invalidMetadata"));
    }
  }
  if (mcpInvokeForm.sessionId) {
    metadata.session_id = metadata.session_id || mcpInvokeForm.sessionId;
  }
  if (mcpInvokeForm.intent) {
    metadata.intent = metadata.intent || mcpInvokeForm.intent;
  }
  if (mcpInvokeForm.capabilityId) {
    metadata.capability_id = metadata.capability_id || mcpInvokeForm.capabilityId;
  }
  return metadata;
}

function normalizeIssuedAt(value: string) {
  if (!value?.trim()) {
    return new Date().toISOString();
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    throw new Error(t("capabilities.mcp.errors.invalidIssuedAt"));
  }
  return parsed.toISOString();
}

function ensureMcpMetadataSession(sessionId?: string) {
  const trimmed = mcpInvokeForm.metadataText?.trim();
  if (!trimmed) {
    if (sessionId || mcpInvokeForm.intent || mcpInvokeForm.capabilityId) {
      const metadata: Record<string, any> = {};
      if (sessionId) metadata.session_id = sessionId;
      if (mcpInvokeForm.intent) metadata.intent = mcpInvokeForm.intent;
      if (mcpInvokeForm.capabilityId) metadata.capability_id = mcpInvokeForm.capabilityId;
      mcpInvokeForm.metadataText = JSON.stringify(metadata, null, 2);
    }
    return;
  }
  try {
    const parsed = JSON.parse(trimmed);
    let changed = false;
    if (sessionId && parsed.session_id !== sessionId) {
      parsed.session_id = sessionId;
      changed = true;
    }
    if (mcpInvokeForm.intent && !parsed.intent) {
      parsed.intent = mcpInvokeForm.intent;
      changed = true;
    }
    if (mcpInvokeForm.capabilityId && !parsed.capability_id) {
      parsed.capability_id = mcpInvokeForm.capabilityId;
      changed = true;
    }
    if (changed) {
      mcpInvokeForm.metadataText = JSON.stringify(parsed, null, 2);
    }
  } catch {
    // ignore invalid JSON until用户手动修复
  }
}

function regenerateInvokeIdentifiers() {
  mcpInvokeForm.messageId = generateUuid();
  mcpInvokeForm.traceId = generateUuid();
  mcpInvokeForm.correlationId = generateUuid();
}

function setIssuedAtNow() {
  mcpInvokeForm.issuedAt = new Date().toISOString();
}

function applyCurrentCapabilityToMcp() {
  const candidate = debugForm.capabilityId?.trim() || capabilityId.value || "";
  if (!candidate) {
    toast.add({
      title: t("capabilities.mcp.toast.capabilityMissing"),
      color: "rose",
    });
    return;
  }
  mcpInvokeForm.capabilityId = candidate;
  mcpInvokeForm.intent = candidate;
  applyMcpSample(candidate, { force: true });
  ensureMcpMetadataSession();
  toast.add({
    title: t("capabilities.mcp.toast.capabilitySynced"),
    color: "green",
  });
}

function maybeApplyMcpSample(capabilityId?: string | null) {
  if (!capabilityId) {
    return;
  }
  applyMcpSample(capabilityId, { force: false });
}

function applyMcpSample(capabilityId: string, options?: { force?: boolean }) {
  const sample = mcpInvokeSamples[capabilityId];
  if (!sample) {
    return;
  }
  const force = Boolean(options?.force);
  if (force || !mcpInvokeForm.toolScope.trim()) {
    mcpInvokeForm.toolScope = sample.toolScope;
  }
  if (force || needsPayloadSeed(mcpInvokeForm.payloadText)) {
    mcpInvokeForm.payloadText = JSON.stringify(sample.payload, null, 2);
  }
}

function needsPayloadSeed(current?: string | null) {
  if (!current) {
    return true;
  }
  const trimmed = current.trim();
  if (!trimmed || trimmed === "{}" || current === "{\n  \n}") {
    return true;
  }
  try {
    const parsed = JSON.parse(trimmed);
    if (Array.isArray(parsed)) {
      return parsed.length === 0;
    }
    if (isPlainObjectValue(parsed)) {
      return Object.keys(parsed).length === 0;
    }
  } catch {
    return false;
  }
  return false;
}

function isPlainObjectValue(value: unknown): value is JsonMap {
  return Object.prototype.toString.call(value) === "[object Object]";
}

function connectMcpStream(sessionId: string) {
  if (typeof window === "undefined" || !sessionId) {
    return;
  }
  disconnectMcpStream();
  const endpoint = buildApiUrl("mcp/sse");
  const url = new URL(endpoint);
  url.searchParams.set("session_id", sessionId);
  const source = new EventSource(url.toString());
  mcpEventSource.value = source;
  source.onopen = () => {
    mcpStreamConnected.value = true;
    mcpStreamError.value = null;
  };
  source.onmessage = (event) => {
    try {
      const parsed = JSON.parse(event.data);
      mcpEvents.value = [parsed, ...mcpEvents.value].slice(0, 20);
    } catch {
      // ignore parse error
    }
  };
  source.onerror = () => {
    mcpStreamConnected.value = false;
    mcpStreamError.value = t("capabilities.mcp.streamError");
  };
}

function disconnectMcpStream() {
  if (mcpEventSource.value) {
    mcpEventSource.value.close();
    mcpEventSource.value = null;
  }
  mcpStreamConnected.value = false;
}

function toggleMcpStream() {
  if (mcpStreamConnected.value) {
    disconnectMcpStream();
    return;
  }
  if (mcpSessionForm.sessionId) {
    connectMcpStream(mcpSessionForm.sessionId);
  }
}

function clearMcpEvents() {
  mcpEvents.value = [];
}

function buildApiUrl(path: string) {
  const base = resolveApiBase();
  if (/^https?:\/\//i.test(base)) {
    return `${base.replace(/\/+$/, "")}/${path.replace(/^\/+/, "")}`;
  }
  if (typeof window === "undefined") {
    return `${base.replace(/\/+$/, "")}/${path.replace(/^\/+/, "")}`;
  }
  const normalizedBase = base.startsWith("/") ? base : `/${base}`;
  return `${window.location.origin}${normalizedBase.replace(/\/+$/, "")}/${path.replace(/^\/+/, "")}`;
}

async function loadTemplate() {
  loadingTemplate.value = true;
  try {
    template.value = await fetchTemplate();
    if (!form.namespace) {
      form.namespace = template.value?.namespace || "";
    }
  } catch (error) {
    console.error("[capabilities] failed to load template", error);
    toast.add({
      title: t("capabilities.form.toast.templateFailed"),
      description: String(error),
      color: "rose",
    });
  } finally {
    loadingTemplate.value = false;
  }
}

async function loadCatalog() {
  catalogLoading.value = true;
  try {
    const entries = await listCatalog();
    catalog.value = entries;
    syncCatalogRows();
  } catch (error) {
    console.error("[capabilities] failed to load catalog", error);
    toast.add({
      title: t("capabilities.list.toast.loadFailed"),
      description: String(error),
      color: "rose",
    });
  } finally {
    catalogLoading.value = false;
  }
}

function syncCatalogRows() {
  catalogRows.value = (catalog.value || []).map((entry) => {
    const meta = exposureMeta.value[entry.id] || { status: "unconfigured" };
    return {
      capability_id: entry.id,
      version: entry.version,
      descriptor: entry.descriptor,
      tags: entry.tags || [],
      checksum: entry.checksum,
      execution: normalizeExecution(entry.execution),
      module: entry.module || deriveModuleFromId(entry.id),
      kind: normalizeKind(entry.kind),
      syncStatus: meta.status,
      updatedAt: meta.updated_at,
      protocols: entry.protocols || {},
    };
  });
}

function normalizeExecution(execution?: CapabilityCatalogEntry["execution"]) {
  if (!execution || !execution.mode) {
    return { mode: "sync" };
  }
  return execution;
}

function openForm() {
  formOpen.value = true;
}

function closeForm() {
  formOpen.value = false;
}

function createDefaultForm() {
  return {
    namespace: "",
    resource: "",
    action: "",
    name: { zh: "", en: "" },
    summary: { zh: "", en: "" },
    description: { zh: "", en: "" },
    scenario: "",
    sensitivity: "medium",
    tags: [] as string[],
    tenant_scope: "global",
    schemas: { input: "", output: "" },
    protocols: {
      rest: { method: "POST", path: "" },
      grpc: { service: "" },
      workflow: { template: "" },
      agent_stream: { channel: "" },
    },
    samples: {
      requestText: "{\n  \n}",
      responseText: "{\n  \n}",
      errors: [] as Array<{ code: string; message: string; solution?: string }>,
    },
    demo: { url: "", credential_hint: "" },
    owner: { name: "", email: "", slack: "" },
    async_mode: "sync",
    async_config: { callback_url: "", sse_channel: "", status_endpoint: "" },
    draft: true,
    metadata: { source: "web-admin" } as Record<string, string>,
  };
}

function resetFormState() {
  Object.assign(form, createDefaultForm());
  currentStep.value = 0;
  validationResult.value = null;
  autoValidate.value = false;
  savedAt.value = null;
}

function hydrateDraft() {
  if (typeof window === "undefined") return;
  const raw = window.localStorage.getItem(draftStorageKey);
  if (!raw) return;
  try {
    const saved = JSON.parse(raw);
    Object.assign(form, saved);
    savedAt.value = new Date().toLocaleString();
  } catch (error) {
    console.warn("[capabilities] failed to parse draft", error);
  }
}

function persistDraft() {
  if (typeof window === "undefined") return;
  const payload = JSON.stringify(form);
  window.localStorage.setItem(draftStorageKey, payload);
  savedAt.value = new Date().toLocaleString();
}

function clearDraft() {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(draftStorageKey);
  savedAt.value = null;
}

function watchFormForPersist() {
  const debounced = useDebounceFn(persistDraft, 500);
  watch(
    form,
    () => debounced(),
    { deep: true },
  );
  watch(
    () => [form.namespace, form.resource, form.action],
    () => {
      if (autoValidate.value) {
        debouncedValidate();
      }
    },
  );
}

const debouncedValidate = useDebounceFn(async () => {
  await runValidation();
}, 800);

async function handleValidate() {
  await runValidation();
  autoValidate.value = true;
}

async function runValidation() {
  const payload = buildPayload();
  if (!payload) return;
  validating.value = true;
  try {
    const result = await validateDraft(payload);
    validationResult.value = result;
  } catch (error: any) {
    validationResult.value =
      error?.response?._data?.error?.details ??
      error?.response?._data?.data ??
      null;
    toast.add({
      title: t("capabilities.form.toast.validateFailed"),
      description: extractErrorMessage(error),
      color: "rose",
    });
  } finally {
    validating.value = false;
  }
}

async function handleSaveDraft() {
  form.draft = true;
  const payload = buildPayload();
  if (!payload) return;
  savingDraft.value = true;
  try {
    const record = await submitDraft(payload);
    validationResult.value = {
      capability_id: record.capability_id,
      valid: true,
      errors: [],
    };
    persistDraft();
    toast.add({
      title: t("capabilities.form.toast.saved"),
      description: t("capabilities.form.toast.savedDesc"),
      color: "primary",
    });
  } catch (error) {
    handleSubmitError(error);
  } finally {
    savingDraft.value = false;
  }
}

async function handleSubmit() {
  form.draft = false;
  const payload = buildPayload();
  if (!payload) return;
  submitting.value = true;
  try {
    const validation = await validateDraft(payload);
    validationResult.value = validation;
    if (!validation.valid) {
      toast.add({
        title: t("capabilities.form.toast.validateBlock"),
        description: t("capabilities.form.toast.fixBeforeSubmit"),
        color: "rose",
      });
      submitting.value = false;
      autoValidate.value = true;
      return;
    }
    const record = await submitDraft(payload);
    validationResult.value = {
      capability_id: record.capability_id,
      valid: true,
      errors: [],
    };
    autoValidate.value = true;
    toast.add({
      title: t("capabilities.form.toast.submitted"),
      description: t("capabilities.form.toast.submittedDesc", {
        id: record.capability_id,
      }),
      color: "green",
    });
    clearDraft();
    await loadCatalog();
    resetFormState();
    closeForm();
  } catch (error) {
    handleSubmitError(error);
  } finally {
    submitting.value = false;
  }
}

function handleSubmitError(error: any) {
  const details =
    error?.response?._data?.error?.details ??
    error?.response?._data?.data ??
    null;
  if (details?.capability_id) {
    validationResult.value = details;
  }
  toast.add({
    title: t("capabilities.form.toast.submitFailed"),
    description: extractErrorMessage(error),
    color: "rose",
  });
}

function buildPayload(): CapabilityRegisterPayload | null {
  const localErrors: CapabilityValidationResult = {
    capability_id: capabilityId.value || "",
    valid: false,
    errors: [],
  };
  const parseJSON = (value: string, field: string) => {
    if (!value || !value.trim()) return null;
    try {
      return JSON.parse(value);
    } catch (error: any) {
      localErrors.errors.push({
        field,
        message: error?.message || "JSON 解析失败",
      });
      return null;
    }
  };

  const requestPayload = parseJSON(form.samples.requestText, "samples.request");
  const responsePayload = parseJSON(form.samples.responseText, "samples.response");
  if (localErrors.errors.length) {
    validationResult.value = localErrors;
    toast.add({
      title: t("capabilities.form.toast.invalidJson"),
      description: t("capabilities.form.toast.fixJson"),
      color: "rose",
    });
    return null;
  }

  return {
    namespace: form.namespace || template.value?.namespace || "",
    resource: form.resource,
    action: form.action,
    name: { ...form.name },
    summary: { ...form.summary },
    description: { ...form.description },
    scenario: form.scenario,
    sensitivity: form.sensitivity,
    tags: [...form.tags],
    tenant_scope: form.tenant_scope,
    schemas: { ...form.schemas },
    protocols: buildProtocols(),
    samples: {
      request: requestPayload,
      response: responsePayload,
      errors: form.samples.errors.filter(
        (err) => err.code || err.message || err.solution,
      ),
    },
    demo: { ...form.demo },
    owner: { ...form.owner },
    async_mode: form.async_mode,
    async_config: { ...form.async_config },
    draft: form.draft,
    metadata: { ...form.metadata },
  };
}

function buildProtocols() {
  const matrix: Record<string, unknown> = {};
  if (form.protocols.rest.path) {
    matrix.rest = {
      path: form.protocols.rest.path,
      method: form.protocols.rest.method,
    };
  }
  if (form.protocols.grpc.service) {
    matrix.grpc = { service: form.protocols.grpc.service };
  }
  if (form.protocols.workflow.template) {
    matrix.workflow_step = { template: form.protocols.workflow.template };
  }
  if (form.protocols.agent_stream.channel) {
    matrix.agent_stream = { channel: form.protocols.agent_stream.channel };
  }
  return matrix;
}

function buildDebugProtocolsFromForm(): DebugProtocols {
  const protocols: DebugProtocols = {};
  if (form.protocols.rest.path) {
    protocols.rest = {
      method: form.protocols.rest.method,
      path: form.protocols.rest.path,
    };
  }
  if (form.protocols.grpc.service) {
    const parsed = parseGrpcServiceField(form.protocols.grpc.service || "");
    protocols.grpc = {
      service: parsed.service || "",
      method: parsed.method || "",
    };
  }
  if (form.protocols.workflow.template) {
    protocols.workflow = {
      template: form.protocols.workflow.template,
    };
  }
  return protocols;
}

function normalizeDebugProtocols(source?: CapabilityCatalogEntry["protocols"] | null): DebugProtocols {
  const protocols: DebugProtocols = {};
  if (!source) {
    return protocols;
  }
  const rest = firstProtocolRecord(source.rest || source.http);
  if (rest) {
    protocols.rest = {
      method: rest.method || rest.httpMethod || rest.verb || "POST",
      path: rest.path || rest.endpoint || rest.url || "",
    };
  }
  const grpc = firstProtocolRecord(source.grpc);
  if (grpc) {
    protocols.grpc = {
      service: grpc.service || grpc.endpoint || "",
      method: grpc.method || grpc.rpc || "",
    };
  }
  const workflow = firstProtocolRecord(source.workflow_step || source.workflow);
  if (workflow) {
    protocols.workflow = {
      template: workflow.template || workflow.name || "",
    };
  }
  return protocols;
}

function firstProtocolRecord(value: any): Record<string, any> | null {
  if (Array.isArray(value)) {
    return ensureProtocolRecord(value[0]);
  }
  return ensureProtocolRecord(value);
}

function ensureProtocolRecord(value: any): Record<string, any> | null {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, any>;
  }
  return null;
}

function buildDebugRestTemplate() {
  const rest = debugSourceProtocols.value?.rest;
  const method = rest?.method || "POST";
  const path = rest?.path || "/api/v1/<resource>";
  return JSON.stringify(
    {
      method,
      endpoint: path,
      headers: {
        "Content-Type": "application/json",
      },
      query: {},
      body: {},
    },
    null,
    2,
  );
}

function parseGrpcServiceField(value: string) {
  const trimmed = (value || "").trim();
  if (!trimmed) {
    return { service: "", method: "" };
  }
  const [service, method] = trimmed.split("/", 2);
  return { service: service || "", method: method || "" };
}

function buildDebugGrpcTemplate() {
  const grpc = debugSourceProtocols.value?.grpc;
  const service = grpc?.service || "powerx.module.v1.Service";
  const method = grpc?.method || "Method";
  return JSON.stringify(
    {
      endpoint: service,
      rpc: method || service,
      metadata: {},
      body: {},
    },
    null,
    2,
  );
}

function buildDebugWorkflowTemplate() {
  return JSON.stringify(
    {
      workflow: {
        template: debugSourceProtocols.value?.workflow?.template || "contracts/exposure/workflow/example.json",
      },
      payload: {},
    },
    null,
    2,
  );
}

function buildLocalGrpcTemplate() {
  const grpc = debugSourceProtocols.value?.grpc;
  const service = grpc?.service || "powerx.module.v1.Service";
  const method = grpc?.method || "Method";
  return JSON.stringify(
    {
      method: "POST",
      endpoint: `/grpc/${service}/${method}`,
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        service,
        rpc: method,
        input: {},
        metadata: {},
      },
    },
    null,
    2,
  );
}

function buildLocalWorkflowTemplate() {
  const capability =
    debugForm.capabilityId?.trim() || capabilityId.value || "com.powerx.plugins.base.template.compose";
  const intent = capability;
  const sessionPlaceholder = "<session-id>";
  const toolScope = deriveToolScopeFromCapability(capability);
  const workflowTemplate = debugSourceProtocols.value?.workflow?.template || capability;
  const inlinePayload = {
    workflow_id: workflowTemplate,
    vars: {
      filters: {
        status: "",
        tags: [],
        page: 1,
        page_size: 20,
      },
      update_payload: {
        description: "来自调试面板的示例更新",
        content: "## Updated content\n\n- 保持 JSON 有效\n- 根据能力要求调整",
        metadata: {
          source: "debug-panel",
        },
      },
    },
  };
  return JSON.stringify(
    {
      method: "POST",
      endpoint: `/api/v1/admin/runtime/sessions/${sessionPlaceholder}/invoke`,
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        message_id: generateUuid(),
        trace_id: generateUuid(),
        correlation_id: generateUuid(),
        tenant_uuid: debugDefaults.tenantUuid,
        tool_scope: toolScope,
        issued_at: new Date().toISOString(),
        payload_ref: JSON.stringify(inlinePayload),
        metadata: {
          session_id: sessionPlaceholder,
          intent,
          capability_id: capability,
        },
        signature: "stub-signature",
      },
    },
    null,
    2,
  );
}

function parseDebugPayload() {
  if (!debugForm.payloadText.trim()) {
    return {};
  }
  return JSON.parse(debugForm.payloadText);
}

function safeDebugPayload() {
  try {
    return parseDebugPayload();
  } catch {
    return "<invalid json>";
  }
}

function applyDebugPayloadTemplate() {
  if (!debugPayloadTemplate.value) return;
  debugPayloadTouched.value = false;
  debugAutoFillingPayload.value = true;
  debugForm.payloadText = debugPayloadTemplate.value;
  nextTick(() => {
    debugAutoFillingPayload.value = false;
  });
}

function resetDebugPayload() {
  debugPayloadTouched.value = false;
  debugAutoFillingPayload.value = true;
  debugForm.payloadText = debugDefaults.payloadText;
  nextTick(() => {
    debugAutoFillingPayload.value = false;
  });
}

function generateUuid() {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.floor(Math.random() * 1e6)
    .toString(16)
    .padStart(5, "0")}`;
}

function generateDebugRequestId() {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `req-${Date.now()}`;
}

function buildPreviewLocalUrl(base: string, endpoint: string, query?: Record<string, any>) {
  const normalized = isAbsoluteURL(endpoint) ? endpoint : combineURL(base, endpoint);
  if (!query || typeof query !== "object") return normalized;
  const params = new URLSearchParams();
  Object.entries(query).forEach(([key, value]) => {
    if (value === undefined || value === null) return;
    if (Array.isArray(value)) {
      value.forEach((entry) => params.append(key, String(entry)));
    } else {
      params.append(key, String(value));
    }
  });
  const qs = params.toString();
  if (!qs) return normalized;
  return normalized.includes("?") ? `${normalized}&${qs}` : `${normalized}?${qs}`;
}

function addErrorCode() {
  form.samples.errors.push({ code: "", message: "", solution: "" });
}

function removeErrorCode(index: number) {
  form.samples.errors.splice(index, 1);
}

function buildCapabilityId(namespace: string, resource: string, action: string) {
  const clean = (value: string) =>
    value
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9.-]/g, "-")
      .replace(/^-+|[-.]+$/g, "");
  const ns = clean(namespace);
  const res = clean(resource);
  const act = clean(action);
  return [ns, res, act].filter(Boolean).join(".");
}

function extractErrorMessage(error: any) {
  return (
    error?.response?._data?.error?.message ||
    error?.message ||
    t("capabilities.form.toast.genericError")
  );
}

function normalizeLocalDebugApiBase(value?: string | null) {
  if (!value) return "";
  if (/^https?:\/\//i.test(value)) {
    return value.replace(/\/api\/v1\/?$/i, "");
  }
  return value;
}

function ensureGatewayApiBase(value: string | undefined, fallback: string) {
  const fallbackBase = fallback?.trim() || "/api/v1";
  const candidate = (value || "").trim();
  if (!candidate) {
    return fallbackBase;
  }
  const normalized = candidate.replace(/\/+$/, "");
  if (/\/api\/v1$/i.test(normalized)) {
    return normalized;
  }
  return `${normalized}/api/v1`;
}

function combineURL(base?: string, endpoint?: string) {
  const normalizedBase = (base || "").replace(/\/+$/, "");
  const normalizedEndpoint = ("/" + (endpoint || "").replace(/^\/+/, "")).replace(/\/{2,}/g, "/");
  if (!normalizedBase) {
    return normalizedEndpoint;
  }
  return `${normalizedBase}${normalizedEndpoint}`;
}

function isAbsoluteURL(value: string) {
  return /^https?:\/\//i.test(value || "");
}

function deriveActionFromCapability(capability: string) {
  const tail = (capability || "").split(".").pop() || "Invoke";
  return tail.replace(/[-_]+/g, " ").replace(/\b\w/g, (char) => char.toUpperCase()).replace(/\s+/g, "");
}

function deriveToolScopeFromCapability(capability: string) {
  const parts = (capability || "").split(".").filter(Boolean);
  if (parts.length >= 2) {
    const tail = parts.slice(-2).join(".");
    return `agent.${tail}`;
  }
  return "agent.workflow";
}

function openDebugPanelFromCatalog(row: CatalogRow) {
  debugContext.value = "catalog";
  debugSourceProtocols.value = normalizeDebugProtocols(row.protocols || {});
  debugForm.capabilityId = row.capability_id;
  debugForm.action = deriveActionFromCapability(row.capability_id);
  debugForm.apiBase = defaultDebugApiBase;
  debugForm.mockModule = "";
  debugForm.tenantUuid = debugDefaults.tenantUuid;
  debugForm.requestId = generateDebugRequestId();
  debugForm.targetMode = "local";
  debugPayloadTouched.value = false;
  debugActionTouched.value = false;
  resetDebugOutputs();
  debugModalOpen.value = true;
}

function openDebugPanelForDraft() {
  debugContext.value = "draft";
  debugSourceProtocols.value = buildDebugProtocolsFromForm();
  debugForm.capabilityId = capabilityId.value || "";
  debugForm.action = form.action || deriveActionFromCapability(capabilityId.value || "");
  debugForm.apiBase = defaultDebugApiBase;
  debugForm.mockModule = "";
  debugForm.tenantUuid = debugDefaults.tenantUuid;
  debugForm.requestId = generateDebugRequestId();
  debugForm.targetMode = "local";
  debugPayloadTouched.value = false;
  debugActionTouched.value = false;
  resetDebugOutputs();
  debugModalOpen.value = true;
}

async function handleDebugInvoke() {
  if (!debugForm.capabilityId && debugForm.targetMode !== "local") {
    toast.add({
      title: t("capabilities.form.debug.capabilityRequired"),
      color: "rose",
    });
    return;
  }
  if (!debugPayloadValid.value) {
    toast.add({
      title: t("capabilities.form.debug.payloadInvalid"),
      color: "rose",
    });
    return;
  }
  const headers: Record<string, string> = {};
  if (debugForm.tenantUuid.trim()) {
    headers["X-Tenant-UUID"] = debugForm.tenantUuid.trim();
  }
  if (debugForm.mockModule.trim()) {
    headers["X-PX-Use-Mock"] = debugForm.mockModule.trim();
  }
  try {
    const rawApiBase = (debugForm.apiBase?.trim() || defaultDebugApiBase || "").trim();
    const resolvedApiBase =
      debugForm.targetMode === "gateway"
        ? ensureGatewayApiBase(rawApiBase, gatewayDebugApiFallback)
        : rawApiBase;
    await invokeDebugCapability({
      capabilityId: debugForm.capabilityId.trim() || capabilityId.value || "local-debug",
      action: (debugForm.action || form.action || "Invoke").trim(),
      payload: parseDebugPayload(),
      payloadText: debugForm.payloadText,
      headers,
      requestId: debugForm.requestId?.trim() || undefined,
      apiBase: resolvedApiBase || undefined,
      preferredProtocol: debugForm.preferredProtocol || undefined,
      mode: debugForm.targetMode,
    });
    if (!debugForm.requestId) {
      debugForm.requestId = generateDebugRequestId();
    }
  } catch {
    // handled in composable
  }
}

function resetDebugOutputs() {
  debugResult.value = null;
  debugWarnings.value = [];
  debugErrorMessage.value = "";
  debugErrorDetails.value = null;
  debugLastTraceId.value = null;
}

function regenerateDebugRequestId() {
  debugForm.requestId = generateDebugRequestId();
}

async function hydrateExposureTemplate() {
  try {
    exposureTemplate.value = await getExposureTemplate();
    if (exposureTemplate.value?.default_rate) {
      exposureForm.rate_limit = { ...exposureTemplate.value.default_rate };
    }
    if (!exposureForm.channels.length) {
      exposureForm.channels = buildChannelEntriesForCapability(
        selectedCapability.value,
      );
    }
  } catch (error) {
    console.error("[capabilities] failed to load exposure template", error);
    toast.add({
      title: t("capabilities.exposure.toast.templateFail"),
      color: "rose",
    });
  }
}

function buildTemplateChannelEntries(types: string[]): ChannelFormEntry[] {
  return types.map((type) => ({
    type,
    name: formatChannelLabel(type),
    method: type === "rest" ? "POST" : "",
    path: "",
    target: "",
    description: "",
    enabled: false,
    scopes: [],
    scopesText: "",
    definition: null,
  }));
}

function buildChannelEntriesForCapability(capability?: CapabilityCatalogEntry | null) {
  const protocols = (capability?.protocols || {}) as Record<string, any>;
  const entries: ChannelFormEntry[] = [];
  const pushChannel = (
    type: string,
    definitionKey?: string,
    definition?: Record<string, any> | null,
    overrides?: Partial<ChannelFormEntry>,
  ) => {
    entries.push({
      type,
      name: overrides?.name || formatChannelLabel(type),
      method: overrides?.method || definition?.method || "",
      path: overrides?.path || definition?.path || "",
      target: overrides?.target || "",
      description: overrides?.description || definition?.description || "",
      enabled: false,
      scopes: [],
      scopesText: "",
      definition: definition || null,
      definitionKey,
    });
  };

  const ensureRecord = (value: any): Record<string, any> | null => {
    if (value && typeof value === "object" && !Array.isArray(value)) {
      return value as Record<string, any>;
    }
    return null;
  };

  const rest = ensureRecord(protocols.rest);
  if (rest) {
    pushChannel("rest", "rest", rest, {
      method: rest.method || "POST",
      path: rest.path || "",
    });
  }
  const grpc = ensureRecord(protocols.grpc);
  if (grpc) {
    pushChannel("grpc", "grpc", grpc, {
      target: [grpc.service, grpc.method].filter(Boolean).join("/"),
    });
  }
  const workflow = ensureRecord(protocols.workflow);
  if (workflow) {
    pushChannel("workflow", "workflow", workflow, {
      target: workflow.template || "",
    });
  }
  const webhook = ensureRecord(protocols.webhook);
  if (webhook) {
    pushChannel("webhook", "webhook", webhook);
  }
  const graphql = ensureRecord(protocols.graphql);
  if (graphql) {
    pushChannel("graphql", "graphql", graphql);
  }
  const agentStream = ensureRecord(protocols.agent_stream);
  if (agentStream) {
    pushChannel("agent_sse", "agent_stream", agentStream, {
      target: agentStream.channel || "",
    });
  }
  const agentTool = ensureRecord(protocols.agent_tool);
  if (agentTool) {
    pushChannel("agent", "agent_tool", agentTool, {
      target: agentTool.endpoint || "",
    });
  }

  if (entries.length === 0) {
    return buildTemplateChannelEntries(exposureTemplate.value?.channel_types || []);
  }
  return entries;
}

function resetExposureChannels() {
  exposureForm.channels = buildChannelEntriesForCapability(
    selectedCapability.value,
  );
}

function formatChannelLabel(type: string) {
  const map: Record<string, string> = {
    rest: "REST",
    grpc: "gRPC",
    graphql: "GraphQL",
    webhook: "Webhook",
    workflow: "Workflow",
    agent: "Agent",
    agent_sse: "Agent SSE",
    sdk: "SDK",
  };
  if (map[type]) {
    return map[type];
  }
  return type.replace(/_/g, " ").toUpperCase();
}

function channelDefinitionEntries(channel: ChannelFormEntry) {
  const definition = channel.definition || {};
  const entries: Array<{ label: string; value: string }> = [];
  const asText = (value: any) => (value === undefined || value === null ? "—" : String(value));

  switch (channel.definitionKey) {
    case "rest":
      entries.push(
        { label: t("capabilities.exposure.fields.method"), value: asText(definition.method) },
        { label: t("capabilities.exposure.fields.path"), value: asText(definition.path) },
      );
      if (definition.description) {
        entries.push({ label: t("capabilities.exposure.fields.description"), value: asText(definition.description) });
      }
      break;
    case "grpc":
      entries.push(
        { label: t("capabilities.exposure.fields.service"), value: asText(definition.service) },
        { label: t("capabilities.exposure.fields.method"), value: asText(definition.method) },
      );
      break;
    case "workflow":
      entries.push({ label: t("capabilities.exposure.fields.template"), value: asText(definition.template) });
      break;
    case "agent_stream":
      entries.push({ label: t("capabilities.exposure.fields.channelPath"), value: asText(definition.channel) });
      break;
    case "agent_tool":
      entries.push(
        { label: t("capabilities.exposure.fields.toolId"), value: asText(definition.id) },
        { label: t("capabilities.exposure.fields.scopeList"), value: asText(definition.scope) },
        { label: t("capabilities.exposure.fields.target"), value: asText(definition.endpoint) },
      );
      break;
    default:
      break;
  }

  if (!entries.length) {
    Object.entries(definition).forEach(([key, value]) => {
      entries.push({ label: key, value: asText(value) });
    });
  }
  return entries;
}

async function openExposureForm(capabilityId: string) {
  if (!capabilityId) {
    toast.add({
      title: t("capabilities.exposure.toast.capabilityRequired"),
      color: "rose",
    });
    return;
  }
  exposureForm.capability_id = capabilityId;
  exposureFormOpen.value = true;
  resetExposureChannels();
  await handleExposureLoad();
}

function closeExposureForm() {
  exposureFormOpen.value = false;
  const nextQuery = { ...route.query };
  if (Reflect.has(nextQuery, "capability")) {
    delete (nextQuery as Record<string, any>).capability;
    router.replace({ query: nextQuery });
  }
}

function setExposureMeta(capabilityId: string, status?: string, updatedAt?: string) {
  if (!capabilityId) return;
  exposureMeta.value = {
    ...exposureMeta.value,
    [capabilityId]: {
      status: status || "unconfigured",
      updated_at: updatedAt,
    },
  };
}

function exposureBadge(status?: string): StatusBadge {
  const presets = statusPresets.value;
  if (status && presets[status]) {
    return presets[status];
  }
  if (status && !presets[status]) {
    return { label: status, color: "gray" };
  }
  return presets.unconfigured;
}

async function handleExposureLoad() {
  if (!exposureForm.capability_id) {
    toast.add({
      title: t("capabilities.exposure.toast.capabilityRequired"),
      color: "rose",
    });
    return;
  }
  exposureLoading.value = true;
  try {
    const data = await fetchExposurePackage(exposureForm.capability_id);
    const pkg = data?.package || null;
    exposurePackageInfo.value = pkg;
    if (pkg) {
      exposureForm.docs_version = pkg.docs_version || "1.0.0";
      exposureForm.sdk_version = pkg.sdk_version || "1.0.0";
      exposureForm.auth = {
        strategy: pkg.auth?.strategy || "powerx_session",
        audience: pkg.auth?.audience || "",
        scopes: pkg.auth?.scopes || [],
      };
      exposureAuthScopes.value = (pkg.auth?.scopes || []).join(", ");
      exposureForm.rate_limit = { ...pkg.rate_limit };
      syncExposureChannelEntries(pkg.channels || []);
      exposureQuotas.value = pkg.tenants || [];
      setExposureMeta(pkg.capability_id, pkg.sync_status, pkg.updated_at);
    } else {
      exposurePackageInfo.value = null;
      resetExposureChannels();
      exposureQuotas.value = [];
      setExposureMeta(exposureForm.capability_id, "unconfigured", undefined);
    }
    const quotaResp = await fetchExposureQuotas(exposureForm.capability_id);
    exposureQuotas.value = quotaResp?.quotas || [];
    await router.replace({
      query: {
        ...route.query,
        capability: exposureForm.capability_id,
      },
    });
  } catch (error) {
    console.error("[capabilities] failed to load exposure", error);
    toast.add({
      title: t("capabilities.exposure.toast.loadFail"),
      color: "rose",
    });
  } finally {
    exposureLoading.value = false;
  }
}

function syncExposureChannelEntries(existing: ExposureChannel[]) {
  const combined = buildChannelEntriesForCapability(selectedCapability.value);
  const map = new Map(existing.map((channel) => [channel.type, channel] as const));
  for (const entry of combined) {
    const defined = map.get(entry.type);
    if (!defined) continue;
    entry.enabled = defined.enabled ?? true;
    entry.scopes = defined.scopes || [];
    entry.scopesText = (defined.scopes || []).join(", ");
  }
  exposureForm.channels = combined;
}

function syncExposureScopeList(channel: ChannelFormEntry) {
  channel.scopes = (channel.scopesText || "")
    .split(",")
    .map((scope) => scope.trim())
    .filter(Boolean);
}

function syncExposureAuthScopes() {
  exposureForm.auth.scopes = exposureAuthScopes.value
    .split(",")
    .map((scope) => scope.trim())
    .filter(Boolean);
}

async function handleExposureSave() {
  if (!exposureForm.capability_id) {
    toast.add({
      title: t("capabilities.exposure.toast.capabilityRequired"),
      color: "rose",
    });
    return;
  }
  exposureSaving.value = true;
  try {
    const payload = {
      capability_id: exposureForm.capability_id,
      docs_version: exposureForm.docs_version,
      sdk_version: exposureForm.sdk_version,
      auth: { ...exposureForm.auth },
      rate_limit: { ...exposureForm.rate_limit },
      channels: exposureForm.channels.map((channel) => ({
        type: channel.type,
        name: channel.name,
        enabled: channel.enabled,
        method: channel.method,
        path: channel.path,
        target: channel.target,
        description: channel.description,
        scopes: channel.scopes,
      })),
      tenants: exposureQuotas.value,
    };
    const record = await saveExposurePackage(payload);
    exposurePackageInfo.value = record;
    setExposureMeta(record.capability_id, record.sync_status, record.updated_at);
    await loadCatalog();
    toast.add({
      title: t("capabilities.exposure.toast.saveSuccess"),
      color: "primary",
    });
  } catch (error) {
    console.error("[capabilities] failed to save exposure", error);
    toast.add({
      title: t("capabilities.exposure.toast.saveFailed"),
      color: "rose",
    });
  } finally {
    exposureSaving.value = false;
  }
}

async function handleExposureQuotaSave() {
  if (!exposureForm.capability_id) {
    toast.add({
      title: t("capabilities.exposure.toast.capabilityRequired"),
      color: "rose",
    });
    return;
  }
  if (!newExposureQuota.tenant_id) {
    toast.add({
      title: t("capabilities.exposure.toast.tenantRequired"),
      color: "rose",
    });
    return;
  }
  exposureQuotaSaving.value = true;
  try {
    const payload: TenantQuota = {
      tenant_id: newExposureQuota.tenant_id,
      tenant_name: newExposureQuota.tenant_name,
      quota: newExposureQuota.quota,
      status: newExposureQuota.status,
      notes: newExposureQuota.notes,
    };
    const record = await saveExposureQuota(exposureForm.capability_id, payload);
    exposureQuotas.value = record.tenants || [];
    exposurePackageInfo.value = record;
    setExposureMeta(record.capability_id, record.sync_status, record.updated_at);
    Object.assign(newExposureQuota, {
      tenant_id: "",
      tenant_name: "",
      quota: 1000,
      status: "active",
      notes: "",
    });
    toast.add({
      title: t("capabilities.exposure.toast.quotaSaved"),
      color: "primary",
    });
  } catch (error) {
    console.error("[capabilities] failed to save quota", error);
    toast.add({
      title: t("capabilities.exposure.toast.quotaFailed"),
      color: "rose",
    });
  } finally {
    exposureQuotaSaving.value = false;
  }
}
</script>
