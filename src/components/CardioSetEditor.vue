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

      <!-- Distance, Time, and RPE -->
      <div class="flex gap-4">
        <!-- Distance -->
        <div class="flex flex-col items-center text-center">
          <div class="relative flex items-center">
            <input
              :value="set.distance"
              @input="
                $emit(
                  'update:distance',
                  $event.target.value ? Number($event.target.value) : null
                )
              "
              type="number"
              inputmode="decimal"
              step="0.1"
              class="text-base font-bold text-black bg-transparent border-none text-center w-16 focus:outline-none dark:text-white"
              placeholder="0"
            />
          </div>
          <div
            class="text-xs font-medium text-black opacity-70 dark:text-white"
          >
            {{ distanceUnit }}
          </div>
        </div>

        <!-- Time -->
        <div class="flex flex-col items-center text-center">
          <div class="relative flex items-center">
            <input
              :value="set.time"
              @input="$emit('update:time', $event.target.value)"
              type="text"
              class="text-base font-bold text-black bg-transparent border-none text-center w-16 focus:outline-none dark:text-white"
              placeholder="0:00"
            />
          </div>
          <div
            class="text-xs font-medium text-black lowercase opacity-70 dark:text-white"
          >
            {{ t("exercise.time") }}
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
      </div>
    </div>

    <!-- Notes Row -->
    <div class="flex gap-3 items-center">
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
  distanceUnit: {
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
  "update:distance",
  "update:time",
  "update:rpe",
  "update:notes",
]);
</script>
