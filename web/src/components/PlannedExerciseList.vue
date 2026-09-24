<template>
  <ul class="space-y-3">
    <li
      v-for="exercise in exercises"
      :key="exercise.id"
      data-testid="planned-exercise"
    >
      <div class="font-bold text-black dark:text-white">
        {{ exercise.exercise }}
      </div>
      <ul class="mt-1 space-y-0.5 text-sm text-black dark:text-white">
        <li v-for="target in exercise.targets" :key="target.id">
          <span class="font-semibold mr-1">{{ target.setType }}</span>
          <span class="opacity-80">{{ prescription(target) }}</span>
          <span v-if="target.notes" class="ml-1 opacity-70">({{ target.notes }})</span>
        </li>
      </ul>
      <div
        v-if="exercise.singleArm && exercise.offArmPercent !== null"
        class="mt-1 inline-block bg-nb-overlay text-black border-2 border-nb-border rounded-md px-2 py-0.5 text-xs font-semibold dark:bg-zinc-800 dark:text-white"
      >
        {{ t("plan.offArm", { percent: exercise.offArmPercent }) }}
      </div>
      <p
        v-if="exercise.notes"
        class="mt-1 flex gap-1 text-sm italic text-black opacity-80 whitespace-pre-line dark:text-white"
      >
        <span class="material-icons text-base! not-italic">lightbulb</span>
        <span>{{ exercise.notes }}</span>
      </p>
    </li>
  </ul>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import { formatTarget } from "@/utils/plan";
import type { PlannedExercise, Target } from "@/types/domain";

const props = defineProps<{
  exercises: PlannedExercise[];
  weightUnit: string;
}>();

const { t } = useI18n();

function prescription(target: Target): string {
  return formatTarget(target, { unit: props.weightUnit, to: t("plan.to") });
}
</script>
