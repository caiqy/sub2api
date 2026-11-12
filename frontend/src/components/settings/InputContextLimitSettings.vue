<template>
  <section
    data-testid="input-context-limit-settings"
    class="card"
    :aria-labelledby="`${id}-title`"
  >
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2
        :id="`${id}-title`"
        class="text-lg font-semibold text-gray-900 dark:text-white"
      >
        {{ t("admin.settings.inputContextLimit.title") }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t("admin.settings.inputContextLimit.description") }}
      </p>
    </div>

    <div class="space-y-5 p-6">
      <!-- 6 项准入规则与生效说明 -->
      <div
        class="rounded-lg border border-blue-100 bg-blue-50/70 p-4 text-xs text-blue-900 dark:border-blue-900/40 dark:bg-blue-950/20 dark:text-blue-300"
      >
        <div class="flex items-center gap-2 font-medium">
          <Icon name="infoCircle" size="sm" class="text-blue-600 dark:text-blue-400" />
          <span>{{ t("admin.settings.inputContextLimit.noticeTitle") }}</span>
        </div>
        <ul class="mt-2 list-inside list-disc space-y-1 text-blue-800/90 dark:text-blue-300/90">
          <li v-for="(item, index) in noticeItems" :key="index">
            {{ item }}
          </li>
        </ul>
      </div>

      <!-- 加载中提示 -->
      <div
        v-if="loading"
        role="status"
        class="flex items-center gap-2 text-sm text-gray-500"
      >
        <div class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"></div>
        <span>{{ t("common.loading") }}</span>
      </div>

      <!-- 加载失败提示与重试 -->
      <div
        v-if="loadError"
        role="alert"
        class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-300"
      >
        <span>{{ loadError }}</span>
        <button
          type="button"
          class="btn btn-secondary btn-sm"
          :disabled="loading || saving"
          @click="load"
        >
          {{ t("admin.settings.inputContextLimit.retry") }}
        </button>
      </div>

      <!-- 表单主体（加载中或加载失败时整体禁用） -->
      <fieldset
        :disabled="loading || saving || !loaded || !!loadError"
        class="space-y-5"
      >
        <!-- 总开关 -->
        <div class="flex items-center justify-between gap-4">
          <div>
            <label
              :for="`${id}-enabled`"
              class="text-sm font-medium text-gray-700 dark:text-gray-300"
            >
              {{ t("admin.settings.inputContextLimit.enabled") }}
            </label>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t("admin.settings.inputContextLimit.enabledHint") }}
            </p>
          </div>
          <Toggle
            :id="`${id}-enabled`"
            v-model="form.enabled"
            data-testid="input-context-limit-toggle"
            :aria-label="t('admin.settings.inputContextLimit.enabled')"
          />
        </div>

        <!-- 孤儿配置警告与一键清理 -->
        <div
          v-if="orphanGroupIds.length > 0"
          role="alert"
          class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-300"
        >
          <div>
            <span class="font-medium">
              {{
                t("admin.settings.inputContextLimit.orphanWarning", {
                  count: orphanGroupIds.length,
                  ids: orphanGroupIds.join(", "),
                })
              }}
            </span>
          </div>
          <button
            type="button"
            class="btn btn-secondary btn-xs"
            :disabled="saving"
            @click="cleanOrphans"
          >
            {{ t("admin.settings.inputContextLimit.cleanOrphans") }}
          </button>
        </div>

        <!-- 分组配置区域（开关关闭时折叠或保留编辑态） -->
        <div class="space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <h3 class="text-sm font-medium text-gray-900 dark:text-white">
              {{ t("admin.settings.inputContextLimit.groupLimits") }}
            </h3>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ t("admin.settings.inputContextLimit.unlimitedHint") }}
            </span>
          </div>

          <!-- 搜索与筛选工具栏 -->
          <div class="flex flex-col gap-3 sm:flex-row">
            <div class="flex-1">
              <label :for="`${id}-search`" class="sr-only">
                {{ t("admin.settings.inputContextLimit.searchGroup") }}
              </label>
              <input
                :id="`${id}-search`"
                v-model="groupSearch"
                type="search"
                class="input w-full"
                :placeholder="t('admin.settings.inputContextLimit.searchGroup')"
                :aria-label="t('admin.settings.inputContextLimit.searchGroup')"
              />
            </div>
            <div class="w-full sm:w-48">
              <label :for="`${id}-platform-filter`" class="sr-only">
                {{ t("admin.settings.inputContextLimit.platformFilter") }}
              </label>
              <Select
                :id="`${id}-platform-filter`"
                v-model="platformFilter"
                :options="platformOptions"
                :aria-label="t('admin.settings.inputContextLimit.platformFilter')"
              />
            </div>
          </div>

          <!-- 分组表格 -->
          <div
            class="max-h-96 overflow-y-auto rounded-lg border border-gray-200 dark:border-dark-700"
          >
            <table
              class="w-full text-left text-sm"
              :aria-label="t('admin.settings.inputContextLimit.groupLimits')"
            >
              <thead
                class="sticky top-0 z-10 bg-gray-50 text-xs font-semibold uppercase text-gray-500 dark:bg-dark-800 dark:text-gray-400"
              >
                <tr>
                  <th scope="col" class="px-4 py-3">
                    {{ t("admin.settings.inputContextLimit.colGroup") }}
                  </th>
                  <th scope="col" class="w-48 px-4 py-3">
                    {{ t("admin.settings.inputContextLimit.colThreshold") }}
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <!-- 无任何分组 -->
                <tr v-if="allGroups.length === 0">
                  <td
                    colspan="2"
                    class="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400"
                  >
                    {{ t("admin.settings.inputContextLimit.noGroups") }}
                  </td>
                </tr>

                <!-- 筛选后无匹配 -->
                <tr v-else-if="filteredGroups.length === 0">
                  <td
                    colspan="2"
                    class="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400"
                  >
                    {{ t("admin.settings.inputContextLimit.noMatchingGroups") }}
                  </td>
                </tr>

                <!-- 正常分组列表 -->
                <tr
                  v-for="group in filteredGroups"
                  :key="group.id"
                  class="hover:bg-gray-50/50 dark:hover:bg-dark-800/50"
                >
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-2">
                      <GroupBadge :name="group.name" :platform="group.platform" />
                      <span class="text-xs text-gray-400 dark:text-gray-500 font-mono">
                        #{{ group.id }}
                      </span>
                    </div>
                  </td>
                  <td class="px-4 py-3">
                    <input
                      :id="`${id}-limit-${group.id}`"
                      v-model="draftLimits[group.id.toString()]"
                      type="text"
                      inputmode="numeric"
                      class="input w-36 font-mono text-sm"
                      placeholder="∞"
                      :data-testid="`input-context-limit-input-${group.id}`"
                      :aria-label="`${t('admin.settings.inputContextLimit.colThreshold')} - ${group.name}`"
                    />
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </fieldset>

      <!-- 成功与错误反馈 -->
      <div v-if="saveSuccess" role="status" class="text-sm font-medium text-green-600 dark:text-green-400">
        {{ t("admin.settings.inputContextLimit.saved") }}
      </div>
      <div v-if="saveError" role="alert" class="text-sm font-medium text-red-600 dark:text-red-400">
        {{ saveError }}
      </div>

      <!-- 保存操作栏 -->
      <div class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700">
        <button
          type="button"
          data-testid="input-context-limit-save"
          class="btn btn-primary btn-sm"
          :disabled="!canSave"
          @click="save"
        >
          <Icon name="check" size="sm" />
          <span>{{ t(saving ? "common.loading" : "admin.settings.inputContextLimit.save") }}</span>
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, getCurrentInstance } from "vue";
import { useI18n } from "vue-i18n";
import {
  inputContextLimitAPI,
  type InputContextLimitSettings,
} from "@/api/admin/settings";
import { getAllIncludingInactive } from "@/api/admin/groups";
import type { AdminGroup } from "@/types";
import Toggle from "@/components/common/Toggle.vue";
import Select from "@/components/common/Select.vue";
import GroupBadge from "@/components/common/GroupBadge.vue";
import Icon from "@/components/icons/Icon.vue";
import { listPlatformIds } from "@/constants/platformCatalog";
import { extractApiErrorMessage } from "@/utils/apiError";

const { t } = useI18n();
const id = `input-context-limit-settings-${getCurrentInstance()?.uid ?? "0"}`;

const form = reactive<InputContextLimitSettings>({
  enabled: false,
  group_limits: {},
});

// 使用字符串 Map 承载草稿状态，避免输入过程中产生非法的空/NaN/浮点
const draftLimits = reactive<Record<string, string>>({});

const allGroups = ref<AdminGroup[]>([]);
const loading = ref(true);
const saving = ref(false);
const loaded = ref(false);
const loadError = ref("");
const saveError = ref("");
const saveSuccess = ref(false);

const groupSearch = ref("");
const platformFilter = ref("");

const noticeItems = computed(() =>
  ["estimation", "inputOnly", "groups", "exemptions", "coverage", "propagation"].map(
    (key) => t(`admin.settings.inputContextLimit.noticeItems.${key}`),
  ),
);

const canSave = computed(
  () => loaded.value && !loading.value && !saving.value && !loadError.value,
);

const platformOptions = computed(() => [
  { label: t("admin.settings.inputContextLimit.platformFilter"), value: "" },
  ...listPlatformIds().map((p) => ({ label: p, value: p })),
]);

// 孤儿配置：在 group_limits 中存在限制但不存在于当前已知 allGroups 中的组 ID
const orphanGroupIds = computed(() => {
  const existingIds = new Set(allGroups.value.map((g) => g.id.toString()));
  return Object.keys(draftLimits).filter(
    (key) => !existingIds.has(key) && Number(draftLimits[key]) > 0,
  );
});

const filteredGroups = computed(() => {
  const q = groupSearch.value.trim().toLowerCase();
  const pf = platformFilter.value;
  return allGroups.value.filter((g) => {
    const matchesSearch =
      !q || g.name.toLowerCase().includes(q) || g.id.toString().includes(q);
    const matchesPlatform = !pf || g.platform === pf;
    return matchesSearch && matchesPlatform;
  });
});

function syncDraftFromForm() {
  // 清理并更新草稿，保留已有键
  for (const key of Object.keys(draftLimits)) {
    delete draftLimits[key];
  }
  for (const [key, val] of Object.entries(form.group_limits || {})) {
    if (typeof val === "number" && val > 0) {
      draftLimits[key] = String(val);
    }
  }
}

async function load() {
  loading.value = true;
  loaded.value = false;
  loadError.value = "";
  saveError.value = "";
  saveSuccess.value = false;
  try {
    const [settings, groups] = await Promise.all([
      inputContextLimitAPI.get(),
      getAllIncludingInactive(),
    ]);
    form.enabled = Boolean(settings?.enabled);
    form.group_limits = { ...(settings?.group_limits || {}) };
    allGroups.value = Array.isArray(groups) ? groups : [];
    syncDraftFromForm();
    loaded.value = true;
  } catch (e) {
    loadError.value = `${t("admin.settings.inputContextLimit.loadFailed")} ${extractApiErrorMessage(e, t("common.unknownError"))}`;
  } finally {
    loading.value = false;
  }
}

function cleanOrphans() {
  for (const orphanId of orphanGroupIds.value) {
    delete draftLimits[orphanId];
    delete form.group_limits[orphanId];
  }
  saveSuccess.value = false;
  saveError.value = "";
}

async function save() {
  if (!canSave.value) return;

  saveError.value = "";
  saveSuccess.value = false;

  // 严格校验每一个输入的数值：必须为 0..2147483647 的非负整数
  const cleanedLimits: Record<string, number> = {};

  for (const [gid, rawValue] of Object.entries(draftLimits)) {
    const trimmed = String(rawValue ?? "").trim();
    if (trimmed === "" || trimmed === "0") {
      // 留空或 0 表示不限制，不发往后端或设为 0
      continue;
    }

    // 检查是否全由纯数字构成且无小数点
    if (!/^\d+$/.test(trimmed)) {
      const group = allGroups.value.find((g) => g.id.toString() === gid);
      saveError.value = t("admin.settings.inputContextLimit.invalidValue", {
        name: group?.name ?? `#${gid}`,
      });
      return;
    }

    const num = Number(trimmed);
    if (!Number.isSafeInteger(num) || num < 0 || num > 2147483647) {
      const group = allGroups.value.find((g) => g.id.toString() === gid);
      saveError.value = t("admin.settings.inputContextLimit.invalidValue", {
        name: group?.name ?? `#${gid}`,
      });
      return;
    }

    cleanedLimits[gid] = num;
  }

  saving.value = true;
  try {
    const payload: InputContextLimitSettings = {
      enabled: form.enabled,
      group_limits: cleanedLimits,
    };
    const result = await inputContextLimitAPI.put(payload);
    form.enabled = Boolean(result.enabled);
    form.group_limits = { ...(result.group_limits || {}) };
    syncDraftFromForm();
    saveSuccess.value = true;
  } catch (e) {
    saveError.value = `${t("admin.settings.inputContextLimit.saveFailed")} ${extractApiErrorMessage(e, t("common.unknownError"))}`;
  } finally {
    saving.value = false;
  }
}

onMounted(load);
</script>
