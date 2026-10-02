<template>
  <section
    class="bg-white border-3 border-nb-border rounded-xl shadow-brutal overflow-hidden dark:bg-zinc-700"
    :data-testid="`meal-${meal.meal}`"
  >
    <div class="flex items-center gap-3 py-3 pl-4 pr-3">
      <div class="flex-1 min-w-0">
        <h3 class="text-lg font-bold text-black dark:text-white">{{ t(`food.meals.${meal.meal}`) }}</h3>
        <div class="text-sm text-black dark:text-white opacity-70" data-testid="meal-summary">
          <template v-if="meal.aim && meal.entries.length">
            {{ t("food.mealAim", { kcal: Math.round(meal.totals.kcal), min: meal.aim.min, max: meal.aim.max }) }}
          </template>
          <template v-else-if="meal.aim">
            {{ t("food.mealAimEmpty", { min: meal.aim.min, max: meal.aim.max }) }}
          </template>
          <template v-else>{{ t("food.mealKcal", { kcal: Math.round(meal.totals.kcal) }) }}</template>
        </div>
      </div>
      <NeoButton
        variant="primary"
        size="sm"
        class="w-11 h-11 px-0! py-0! shrink-0"
        icon-only
        :aria-label="t('food.addTo', { meal: t(`food.meals.${meal.meal}`) })"
        :data-testid="`add-${meal.meal}`"
        @click="emit('add')"
      >
        <template #icon><span class="material-icons">add</span></template>
      </NeoButton>
    </div>

    <div
      v-if="meal.aim && meal.entries.length"
      class="h-1.5 bg-nb-overlay border-y-2 border-nb-border dark:bg-zinc-800"
    >
      <div class="h-full bg-purple-500" :style="{ width: `${progress}%` }"></div>
    </div>

    <ul v-if="meal.entries.length">
      <li v-for="entry in meal.entries" :key="entry.id" class="border-b border-gray-300 last:border-b-0 dark:border-zinc-600">
        <button
          type="button"
          class="w-full flex items-center gap-3 px-4 py-3 text-left"
          data-testid="entry"
          @click="emit('edit', entry)"
        >
          <div class="flex-1 min-w-0">
            <div class="font-bold text-black dark:text-white">{{ entry.name }}</div>
            <div class="text-xs text-black dark:text-white opacity-70">
              {{ entry.amount ? `${entry.amount} (${entry.grams} g)` : `${entry.grams} g` }}
              <span v-if="entry.loggedBy === 'assistant'" class="inline-flex items-center gap-0.5 text-purple-700 dark:text-purple-300 font-bold">
                · <span class="material-icons text-xs!">auto_awesome</span>{{ t("food.byAssistant") }}
              </span>
            </div>
          </div>
          <div class="text-sm font-bold text-black dark:text-white shrink-0">{{ Math.round(entry.kcal) }}</div>
        </button>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import NeoButton from "@/components/NeoButton.vue";
import type { DayMeal, FoodEntry } from "@/types/domain";

const props = defineProps<{ meal: DayMeal }>();
const emit = defineEmits<{ (e: "add"): void; (e: "edit", entry: FoodEntry): void }>();
const { t } = useI18n();

// How far into its aim the meal is, full at the top of the range.
const progress = computed(() => {
  const max = props.meal.aim?.max ?? 0;
  return max > 0 ? Math.min(100, Math.round((props.meal.totals.kcal / max) * 100)) : 0;
});
</script>
