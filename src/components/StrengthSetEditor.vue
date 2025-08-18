<template>
  <div
    class="bg-nb-overlay border-2 border-nb-border rounded-lg p-4 shadow-brutal-sm dark:bg-zinc-800"
  >
    <div class="flex items-center justify-between mb-3">
      <!-- Set Number -->
      <button
        @click="$emit('toggleSetType')"
        :class="[
          'w-9 h-9 min-w-9 min-h-9 flex-shrink-0 border-2 border-nb-border rounded-full flex items-center justify-center text-sm font-bold shadow-brutal-sm',
          set.type === 'warmup'
            ? 'bg-orange-500 text-white'
            : 'bg-purple-400 text-black',
        ]"
      >
        {{ setNumber }}
      </button>

      <!-- Weight, Reps/Time, RPE, and Arm -->
      <div class="flex gap-4">
        <!-- Weight -->
        <div class="flex flex-col items-center text-center">
          <div class="relative flex items-center">
            <input
              :value="set.weight"
              @input="
                $emit(
                  'update:weight',
                  $event.target.value ? Number($event.target.value) : null
                )
              "
              type="number"
              inputmode="decimal"
              step="0.5"
              class="text-base font-bold text-black bg-transparent border-none text-center w-16 focus:outline-none dark:text-white"
              placeholder="0"
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
              @input="
                $emit(
                  'update:reps',
                  $event.target.value ? Number($event.target.value) : null
                )
              "
              type="number"
              inputmode="decimal"
              class="text-base font-bold text-black bg-transparent border-none text-center w-16 focus:outline-none dark:text-white"
              placeholder="0"
            />
            <input
              v-else
              :value="set.time"
              @input="$emit('update:time', $event.target.value)"
              type="text"
              class="text-base font-bold text-black bg-transparent border-none text-center w-16 focus:outline-none dark:text-white"
              placeholder="0:00"
            />
          </div>
          <div
            class="text-xs font-medium text-black lowercase dark:text-white relative"
          >
            <template v-if="exercise.displayType === 'reps'">
              <span class="opacity-70">
                <template v-if="set.weight && previousReps">
                  {{ t("exercise.reps") }} ({{ t("exercise.previousBest") }}:
                  {{ previousReps }})
                </template>
                <template v-else> {{ t("exercise.reps") }} </template>
              </span>
              <span
                v-if="isRepRecord"
                class="absolute -top-[2px] -right-6 text-yellow-500 text-sm m-icon animate-pulse"
                :title="t('exercise.newRepRecord')"
              >
                star
              </span>
            </template>
            <template v-else>
              {{ t("exercise.time") }}
            </template>
          </div>
        </div>

        <!-- RPE -->
        <div class="flex flex-col items-center text-center">
          <select
            :value="set.rpe"
            @change="$emit('update:rpe', $event.target.value)"
            @blur="handleSelectBlur"
            class="text-base font-bold text-black bg-transparent border-none text-align-last-center w-16 focus:outline-none appearance-none dark:text-white"
          >
            <option value="">-</option>
            <option v-for="rpe in rpeOptions" :key="rpe" :value="rpe">
              {{ rpe.replace("RPE ", "") }}
            </option>
          </select>
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
            @change="$emit('update:arm', $event.target.value)"
            @blur="handleSelectBlur"
            class="text-base font-bold text-black bg-transparent text-align-last-center border-none text-center w-16 focus:outline-none appearance-none dark:text-white"
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
    <div class="flex gap-3 items-center">
      <!-- % of max -->
      <div class="flex flex-col items-center text-center">
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
          @input="$emit('update:notes', $event.target.value)"
          type="text"
          class="bg-white border-2 border-nb-border rounded-md px-2 py-1 text-sm font-medium text-black focus:outline-none dark:bg-black dark:text-white"
          :placeholder="t('exercise.notes')"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { useI18n } from "vue-i18n";

const { t } = useI18n();

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

defineProps({
  set: {
    type: Object,
    required: true,
  },
  exercise: {
    type: Object,
    required: true,
  },
  setNumber: {
    type: String,
    required: true,
  },
  weightUnit: {
    type: String,
    required: true,
  },
  isWeightRecord: {
    type: Boolean,
    default: false,
  },
  isRepRecord: {
    type: Boolean,
    default: false,
  },
  previousReps: {
    type: Number,
    default: null,
  },
  maxPercentage: {
    type: String,
    required: true,
  },
  rpeOptions: {
    type: Array,
    required: true,
  },
});

defineEmits([
  "toggleSetType",
  "update:weight",
  "update:reps",
  "update:time",
  "update:rpe",
  "update:arm",
  "update:notes",
]);
</script>
