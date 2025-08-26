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
              <div class="flex items-center gap-1">
                <span class="opacity-70">{{ t("exercise.reps") }}</span>
                <button
                  v-if="set.weight && (previousReps || highestReps)"
                  @click="repHistoryPanel.toggle($event)"
                  class="text-purple-600 hover:text-purple-800 dark:text-purple-400 dark:hover:text-purple-300 ml-1 p-0.5 rounded hover:bg-purple-100 dark:hover:bg-purple-900/20"
                  :title="t('exercise.repHistory')"
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
                    >{{ previousReps }}</span
                  >
                </div>
                <div
                  v-if="highestReps"
                  class="flex justify-between items-center"
                >
                  <span>{{ t("exercise.highestReps") }}:</span>
                  <span
                    class="font-medium text-purple-600 dark:text-purple-400"
                    >{{ highestReps }}</span
                  >
                </div>
              </div>
            </div>
          </Popover>
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
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import Popover from "@/volt/Popover.vue";

const { t } = useI18n();

const repHistoryPanel = ref();

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
  highestReps: {
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
