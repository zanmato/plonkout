<template>
  <BottomSheet :title="entry.name" @close="emit('close')">
    <div class="space-y-4" data-testid="entry-sheet">
      <div>
        <div class="text-sm font-bold text-black dark:text-white mb-2">{{ t("food.entry.meal") }}</div>
        <div class="grid grid-cols-2 gap-2">
          <button
            v-for="m in meals"
            :key="m"
            type="button"
            :class="[
              'min-h-11 border-2 border-nb-border rounded-lg text-sm font-bold',
              meal === m ? 'bg-purple-500 text-white' : 'bg-white text-black dark:bg-zinc-800 dark:text-white',
            ]"
            @click="meal = m"
          >
            {{ t(`food.meals.${m}`) }}
          </button>
        </div>
      </div>
      <div class="floating-label-container">
        <input
          id="entry-grams"
          v-model="gramsText"
          type="text"
          inputmode="decimal"
          class="floating-input"
          :placeholder="t('food.entry.grams')"
          data-testid="entry-grams"
        />
        <label for="entry-grams" class="floating-label">{{ t("food.entry.grams") }}</label>
      </div>
      <div class="text-sm text-black dark:text-white opacity-70">
        {{ Math.round(kcal) }} {{ t("food.kcal") }}
      </div>
      <DestructiveButton :confirm-text="t('food.entry.delete')" variant="secondary" full-width data-testid="entry-delete" @confirm="remove" />
    </div>

    <template #footer>
      <NeoButton variant="primary" full-width :disabled="!grams || busy" data-testid="entry-save" @click="save">
        {{ t("food.entry.save") }}
      </NeoButton>
    </template>
  </BottomSheet>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import BottomSheet from "@/components/BottomSheet.vue";
import DestructiveButton from "@/components/DestructiveButton.vue";
import NeoButton from "@/components/NeoButton.vue";
import { deleteEntry, updateEntry } from "@/api/food";
import type { DayKey, FoodEntry, Meal } from "@/types/domain";
import { parseDecimal, scale } from "@/utils/number";

const props = defineProps<{ entry: FoodEntry; day: DayKey }>();
const emit = defineEmits<{ (e: "close"): void; (e: "changed"): void; (e: "error"): void }>();
const { t } = useI18n();

const meals: Meal[] = ["breakfast", "lunch", "dinner", "snack"];
const meal = ref<Meal>(props.entry.meal);
const gramsText = ref(String(props.entry.grams));
const busy = ref(false);

const grams = computed(() => {
  const value = parseDecimal(gramsText.value);
  return value && value > 0 && value <= 5000 ? value : undefined;
});
const kcal = computed(() => scale(props.entry.per100g.kcal, grams.value ?? 0));

async function run(action: () => Promise<unknown>) {
  busy.value = true;
  try {
    await action();
    emit("changed");
  } catch (error) {
    console.error("Error changing an entry:", error);
    emit("error");
  } finally {
    busy.value = false;
  }
}

function save() {
  if (!grams.value) return;
  // A changed weight leaves a household amount like "2 tbsp" untrue.
  const amount = grams.value === props.entry.grams ? props.entry.amount : "";
  return run(() => updateEntry(props.entry.id, { day: props.day, meal: meal.value, grams: grams.value!, amount }));
}

function remove() {
  return run(() => deleteEntry(props.entry.id));
}
</script>
