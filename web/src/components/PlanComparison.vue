<template>
  <NeoPanel v-if="comparison" data-testid="plan-comparison">
    <h3 class="text-lg font-bold text-black dark:text-white mb-3">
      {{ t("plan.comparison.title") }}
    </h3>

    <div class="space-y-4">
      <div
        v-for="exercise in comparison.exercises"
        :key="exercise.plannedExerciseId"
      >
        <div class="flex items-baseline justify-between gap-2">
          <div class="font-bold text-black dark:text-white min-w-0 truncate">
            {{ exercise.exercise }}
          </div>
          <div
            v-if="exercise.bestEstimated1RM !== null"
            class="shrink-0 text-xs text-black dark:text-white"
          >
            <span class="opacity-70">{{ t("plan.comparison.best1RM") }}</span>
            <span class="ml-1 font-bold text-purple-600 dark:text-purple-400">
              {{ exercise.bestEstimated1RM }} {{ weightUnit }}
            </span>
          </div>
        </div>

        <ul class="mt-1 space-y-1 text-sm text-black dark:text-white">
          <li
            v-for="target in exercise.targets"
            :key="target.target.id"
            class="flex items-center gap-2"
            data-testid="comparison-target"
          >
            <span
              :class="[
                'material-icons text-base!',
                target.met
                  ? 'text-green-600 dark:text-green-400'
                  : 'text-red-600 dark:text-red-400',
              ]"
              :title="
                target.met
                  ? t('plan.comparison.met')
                  : t('plan.comparison.missed')
              "
            >
              {{ target.met ? "check_circle" : "cancel" }}
            </span>
            <span class="flex-1 min-w-0 truncate font-semibold">
              {{ target.target.setType }}
            </span>
            <span class="tabular-nums">
              {{ target.done }}/{{ target.target.sets }}
            </span>
            <span class="w-14 text-right text-xs opacity-70 tabular-nums">
              {{ target.topRpe !== null ? `RPE ${target.topRpe}` : "" }}
            </span>
          </li>
        </ul>

        <div
          v-if="exercise.extraSets.length"
          class="mt-1 text-xs text-black dark:text-white opacity-70"
        >
          {{
            t(
              "plan.comparison.extraSets",
              { count: exercise.extraSets.length },
              exercise.extraSets.length,
            )
          }}
        </div>
      </div>
    </div>

    <div
      v-if="comparison.unplanned.length"
      class="mt-4 text-sm text-black dark:text-white"
    >
      <span class="opacity-70">{{ t("plan.comparison.unplanned") }}:</span>
      <span class="ml-1 font-semibold">
        {{ comparison.unplanned.join(", ") }}
      </span>
    </div>
  </NeoPanel>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { getSessionComparison } from "@/api/plans";
import NeoPanel from "@/components/NeoPanel.vue";
import type { Comparison, Id } from "@/types/domain";

const props = defineProps<{
  sessionId: Id;
  weightUnit: string;
  /** Changes whenever the workout was saved, to compare again */
  revision?: number;
}>();

const { t } = useI18n();
const comparison = ref<Comparison | null>(null);

async function load() {
  try {
    comparison.value = await getSessionComparison(props.sessionId);
  } catch (error) {
    console.error(t("plan.comparison.loadError"), error);
  }
}

watch(() => [props.sessionId, props.revision], load, { immediate: true });
</script>
