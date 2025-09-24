<template>
  <div
    class="bg-nb-overlay border-2 border-nb-border rounded-lg p-4 shadow-brutal-sm dark:bg-zinc-800"
    :data-exercise-index="exerciseIndex"
    :data-set-index="setIndex"
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
              tabindex="0"
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
              tabindex="0"
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
          <input
            :value="set.rpe"
            @input="
              $emit(
                'update:rpe',
                $event.target.value ? Number($event.target.value) : null
              )
            "
            type="number"
            inputmode="numeric"
            min="1"
            max="10"
            tabindex="0"
            class="text-base font-bold text-black bg-transparent border-none text-center w-16 focus:outline-none dark:text-white"
            placeholder="0"
          />
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
          tabindex="0"
          class="bg-white border-2 border-nb-border rounded-md px-2 py-1 text-sm font-medium text-black focus:outline-none dark:bg-black dark:text-white"
          :placeholder="t('exercise.notes')"
        />
      </div>

      <!-- Delete Set Button (only show if more than one set) -->
      <div v-if="exercise.sets.length > 1" class="flex flex-col">
        <DestructiveButton
          @confirm="$emit('delete-set')"
          :confirm-text="t('workout.deleteSet')"
          size="sm"
          class="w-8 h-8 !px-0 !py-0 rounded-full"
          icon-only
        >
          <template #icon>
            <span class="material-icons">delete</span>
          </template>
        </DestructiveButton>
      </div>

      <!-- Add Set Tab Field (minimal but tabbable) - only show on last set -->
      <div v-if="isLastSet" class="flex flex-col">
        <input
          @focus="handleAddSetFocus"
          @input="handleAddSetInput"
          @blur="clearAddSetField"
          ref="addSetInput"
          type="text"
          tabindex="0"
          class="w-4 h-1 text-transparent bg-transparent border-none focus:bg-purple-100 focus:border-2 focus:border-purple-300 focus:rounded-md focus:px-2 focus:py-1 focus:text-sm focus:font-medium focus:text-purple-600 focus:outline-none focus:w-auto focus:h-auto focus:text-purple-600 dark:focus:bg-purple-900/20 dark:focus:border-purple-600 dark:focus:text-purple-400"
          :placeholder="t('workout.addSet')"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import DestructiveButton from "@/components/DestructiveButton.vue";

const { t } = useI18n();

const addSetInput = ref();

const props = defineProps({
  set: {
    type: Object,
    required: true,
  },
  exercise: {
    type: Object,
    required: true,
  },
  exerciseIndex: {
    type: Number,
    required: true,
  },
  setIndex: {
    type: Number,
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
});

// Check if this is the last set in the exercise (only last set shows add-set field)
const isLastSet = computed(() => {
  return props.setIndex === props.exercise.sets.length - 1;
});

const emit = defineEmits([
  "toggleSetType",
  "update:distance",
  "update:time",
  "update:rpe",
  "update:notes",
  "add-set",
  "delete-set",
]);

function handleAddSetFocus() {
  // Trigger add set when field receives focus
  emit('add-set');
}

function handleAddSetInput() {
  // Trigger add set on any input and clear the field
  emit('add-set');
  if (addSetInput.value) {
    addSetInput.value.value = '';
  }
}

function clearAddSetField() {
  // Clear the field when it loses focus
  if (addSetInput.value) {
    addSetInput.value.value = '';
  }
}
</script>
