<template>
  <div
    class="bg-nb-overlay border-2 border-nb-border rounded-lg p-4 shadow-brutal-sm dark:bg-zinc-800"
    :data-exercise-index="exerciseIndex"
    :data-set-index="setIndex"
  >
    <!-- What the plan asks of this set -->
    <div
      v-if="target"
      class="flex items-center justify-between gap-2 mb-2"
      data-testid="set-target"
    >
      <div
        class="text-xs font-semibold text-purple-700 dark:text-purple-300 min-w-0"
      >
        {{ targetLabel }}
      </div>
      <button
        v-if="canFillTarget"
        type="button"
        class="shrink-0 inline-flex items-center gap-1 bg-purple-300 text-black border-2 border-nb-border rounded-md px-2 py-0.5 text-xs font-bold shadow-brutal-sm transition-all duration-200 hover:shadow-none hover:translate-x-0.5 hover:translate-y-0.5 active:shadow-none active:translate-x-0.5 active:translate-y-0.5"
        :title="t('plan.asPlanned')"
        data-testid="as-planned"
        @click="fillTarget"
      >
        <span class="material-icons text-sm!">check</span>
        {{ t("plan.asPlanned") }}
      </button>
    </div>

    <div class="flex items-center justify-between mb-3">
      <!-- Set Number -->
      <button
        :class="[
          'w-9 h-9 min-w-9 min-h-9 flex-shrink-0 border-2 border-nb-border rounded-full flex items-center justify-center text-sm font-bold shadow-brutal-sm',
          set.type === 'warmup'
            ? 'bg-orange-500 text-white'
            : 'bg-purple-400 text-black',
        ]"
        @click="$emit('toggleSetType')"
      >
        {{ setNumber }}
      </button>

      <!-- Weight, Reps/Time, RPE, and Arm -->
      <div class="flex gap-4">
        <!-- Weight -->
        <div class="flex flex-col items-center text-center">
          <div class="relative flex items-center">
            <input
              ref="primaryInput"
              :value="set.weight"
              type="number"
              inputmode="decimal"
              step="0.5"
              tabindex="0"
              class="text-base font-bold text-black bg-transparent border-none text-center w-16 focus:outline-none dark:text-white"
              :placeholder="weightPlaceholder"
              @input="
                $emit(
                  'update:weight',
                  ($event.target as HTMLInputElement).value ? Number(($event.target as HTMLInputElement).value) : null,
                )
              "
            />
          </div>
          <div
            class="text-xs font-medium text-black opacity-70 dark:text-white"
          >
            {{ weightUnit }}
          </div>
        </div>

        <!-- Reps or Time -->
        <div class="flex flex-col items-center text-center">
          <div class="relative flex items-center">
            <input
              v-if="exercise.displayType === 'reps'"
              :value="set.reps"
              type="number"
              inputmode="decimal"
              tabindex="0"
              class="text-base font-bold text-black bg-transparent border-none text-center w-16 focus:outline-none dark:text-white"
              :placeholder="repsPlaceholder"
              @input="
                $emit(
                  'update:reps',
                  ($event.target as HTMLInputElement).value ? Number(($event.target as HTMLInputElement).value) : null,
                )
              "
            />
            <input
              v-else
              :value="set.time"
              type="text"
              tabindex="0"
              class="text-base font-bold text-black bg-transparent border-none text-center w-16 focus:outline-none dark:text-white"
              :placeholder="target?.time || '0:00'"
              @input="$emit('update:time', ($event.target as HTMLInputElement).value)"
            />
          </div>
          <div
            class="text-xs font-medium text-black lowercase dark:text-white relative"
          >
            <template v-if="exercise.displayType === 'reps'">
              <div class="flex items-center gap-1">
                <span class="opacity-70">{{ t("exercise.reps") }}</span>
                <button
                  v-if="set.weight && (previousReps || highestReps)"
                  class="text-purple-600 hover:text-purple-800 dark:text-purple-400 dark:hover:text-purple-300 ml-1 p-0.5 rounded hover:bg-purple-100 dark:hover:bg-purple-900/20"
                  :title="t('exercise.repHistory')"
                  @click="repHistoryPanel!.toggle($event)"
                >
                  <span class="m-icon text-sm">history</span>
                </button>
              </div>
              <span
                v-if="isRepRecord"
                class="absolute top-[1px] -right-6 text-yellow-500 text-sm m-icon animate-pulse"
                :title="t('exercise.newRepRecord')"
              >
                star
              </span>
            </template>
            <template v-else>
              {{ t("exercise.time") }}
            </template>
          </div>

          <!-- Rep History Popover -->
          <Popover
            ref="repHistoryPanel"
            :showCloseIcon="true"
            class="rep-history-popover"
          >
            <div class="p-3 min-w-40">
              <h4 class="font-semibold mb-2 text-sm">
                {{ t("exercise.repHistory") }}
              </h4>
              <div class="space-y-1 text-xs">
                <div
                  v-if="previousReps"
                  class="flex justify-between items-center"
                >
                  <span>{{ t("exercise.previousBestFull") }}:</span>
                  <span
                    class="font-medium text-purple-600 dark:text-purple-400"
                    >{{ previousReps }}</span>
                </div>
                <div
                  v-if="highestReps"
                  class="flex justify-between items-center"
                >
                  <span>{{ t("exercise.highestReps") }}:</span>
                  <span
                    class="font-medium text-purple-600 dark:text-purple-400"
                    >{{ highestReps }}</span>
                </div>
              </div>
            </div>
          </Popover>
        </div>

        <!-- RPE -->
        <div class="flex flex-col items-center text-center">
          <input
            :value="set.rpe"
            type="number"
            inputmode="numeric"
            min="1"
            max="10"
            tabindex="0"
            class="text-base font-bold text-black bg-transparent border-none text-center w-16 focus:outline-none dark:text-white"
            placeholder="0"
            @input="
              $emit(
                'update:rpe',
                ($event.target as HTMLInputElement).value ? Number(($event.target as HTMLInputElement).value) : null,
              )
            "
          />
          <div
            class="text-xs font-medium text-black opacity-70 dark:text-white"
          >
            RPE
          </div>
        </div>

        <!-- Arm (only for single arm exercises) -->
        <div
          v-if="exercise.singleArm"
          class="flex flex-col items-center text-center"
        >
          <select
            :value="set.arm"
            tabindex="0"
            class="text-base font-bold text-black bg-transparent text-align-last-center border-none text-center w-16 focus:outline-none appearance-none dark:text-white"
            @change="$emit('update:arm', ($event.target as HTMLSelectElement).value as Arm)"
            @blur="handleSelectBlur"
          >
            <option value="">-</option>
            <option value="left">
              {{ t("exercise.arms.left") }}
            </option>
            <option value="right">
              {{ t("exercise.arms.right") }}
            </option>
          </select>
          <div
            class="text-xs font-medium text-black opacity-70 lowercase dark:text-white"
          >
            {{ t("exercise.arm") }}
          </div>
        </div>
      </div>
    </div>

    <!-- Notes and % of max Row -->
    <div class="flex items-center">
      <!-- % of max -->
      <div class="flex flex-col items-center text-center mr-3">
        <div class="text-base font-bold text-black dark:text-white">
          {{ maxPercentage }}
        </div>
        <div class="text-xs font-medium text-black opacity-70 dark:text-white">
          {{ t("exercise.percentOfMax") }}
        </div>
      </div>

      <!-- Notes -->
      <div class="flex flex-col flex-1">
        <input
          :value="set.notes"
          type="text"
          tabindex="0"
          class="bg-white border-2 border-nb-border rounded-md px-2 py-1 text-sm font-medium text-black focus:outline-none dark:bg-black dark:text-white"
          :placeholder="t('exercise.notes')"
          @input="$emit('update:notes', ($event.target as HTMLInputElement).value)"
        />
      </div>

      <!-- Add Set Tab Field (minimal but tabbable) - only show on last set -->
      <div class="flex flex-col">
        <input
          ref="addSetInput"
          type="text"
          :tabindex="isLastSet ? 0 : -1"
          class="w-4 h-1 text-transparent bg-transparent border-none focus:bg-purple-100 focus:border-2 focus:border-purple-300 focus:rounded-md focus:px-2 focus:py-1 focus:text-sm focus:font-medium focus:text-purple-600 focus:outline-none focus:w-auto focus:h-auto focus:text-purple-600 dark:focus:bg-purple-900/20 dark:focus:border-purple-600 dark:focus:text-purple-400"
          :placeholder="t('workout.addSet')"
          @focus="handleAddSetFocus"
          @input="handleAddSetInput"
        />
      </div>

      <!-- Delete Set Button (only show if more than one set) -->
      <div v-if="exercise.sets.length > 1" class="flex flex-col">
        <DestructiveButton
          :confirm-text="t('workout.deleteSet')"
          size="sm"
          class="w-8 h-8 !px-0 !py-0 rounded-full"
          icon-only
          @confirm="$emit('delete-set')"
        >
          <template #icon>
            <span class="material-icons">delete</span>
          </template>
        </DestructiveButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import Popover from "@/volt/Popover.vue";
import DestructiveButton from "@/components/DestructiveButton.vue";
import { formatSetTarget } from "@/utils/plan";
import type { Arm, Target, WorkoutExercise, WorkoutSet } from "@/types/domain";

const { t } = useI18n();

const repHistoryPanel = ref<InstanceType<typeof Popover> | null>(null);
const addSetInput = ref<HTMLInputElement | null>(null);
const primaryInput = ref<HTMLInputElement | null>(null);

defineExpose({
  /** Focus the first input of this set */
  focus: () => primaryInput.value?.focus(),
});

const props = withDefaults(
  defineProps<{
    set: WorkoutSet;
    exercise: WorkoutExercise;
    exerciseIndex: number;
    setIndex: number;
    setNumber: string;
    weightUnit: string;
    isWeightRecord?: boolean;
    isRepRecord?: boolean;
    previousReps?: number | null;
    highestReps?: number | null;
    maxPercentage: string;
    /** The plan's target this set was prefilled for */
    target?: Target | null;
    /** The target's weight for this set's arm */
    targetWeight?: number | null;
  }>(),
  {
    isWeightRecord: false,
    isRepRecord: false,
    previousReps: null,
    highestReps: null,
    target: null,
    targetWeight: null,
  },
);

const targetLabel = computed(() =>
  props.target
    ? formatSetTarget(props.target, {
        unit: props.weightUnit,
        to: t("plan.to"),
        weight: props.targetWeight,
      })
    : "",
);

// The target shows through an empty input as a hint of what to lift
const weightPlaceholder = computed(() =>
  props.targetWeight !== null ? String(props.targetWeight) : "0",
);
const repsPlaceholder = computed(() =>
  props.target?.reps != null ? String(props.target.reps) : "0",
);

const isEmpty = (value: unknown) =>
  value === null || value === undefined || value === "";

// Offered until something is logged, so a set done as planned is one tap
const canFillTarget = computed(() => {
  if (!props.target || !isEmpty(props.set.weight)) return false;
  const hasTarget =
    props.targetWeight !== null ||
    (props.exercise.displayType === "reps"
      ? props.target.reps !== null
      : Boolean(props.target.time));
  const logged =
    props.exercise.displayType === "reps"
      ? !isEmpty(props.set.reps)
      : Boolean(props.set.time);
  return hasTarget && !logged;
});

// Check if this is the last set in the exercise (only last set shows add-set field)
const isLastSet = computed(() => {
  return props.setIndex === props.exercise.sets.length - 1;
});

const emit = defineEmits<{
  toggleSetType: [];
  "update:weight": [value: number | null];
  "update:reps": [value: number | null];
  "update:time": [value: string];
  "update:rpe": [value: number | null];
  "update:arm": [value: Arm];
  "update:notes": [value: string];
  "add-set": [];
  "delete-set": [];
}>();

/**
 * Log the set exactly as the target prescribes
 */
function fillTarget() {
  const target = props.target;
  if (!target) return;
  if (props.targetWeight !== null) emit("update:weight", props.targetWeight);
  if (props.exercise.displayType === "reps") {
    if (target.reps !== null) emit("update:reps", target.reps);
  } else if (target.time) {
    emit("update:time", target.time);
  }
}

function handleSelectBlur() {
  // Force viewport update for iOS when select loses focus
  if (window.visualViewport) {
    // Force reflow
    document.body.style.transform = "translateZ(0)";
    requestAnimationFrame(() => {
      document.body.style.transform = "";
    });
  }
}

function handleAddSetFocus() {
  // Trigger add set when field receives focus
  emit("add-set");
}

function handleAddSetInput() {
  // Trigger add set on any input and clear the field
  emit("add-set");
  if (addSetInput.value) {
    addSetInput.value.value = "";
  }
}
</script>
