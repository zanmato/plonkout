<template>
  <div class="food-goal h-full flex flex-col">
    <NeoHeader :title="t('food.goal.title')">
      <template #left>
        <NeoButton variant="primary" size="sm" data-testid="back-button" @click="router.push({ name: 'food' })">
          <template #icon><span class="material-icons">arrow_back</span></template>
        </NeoButton>
      </template>
    </NeoHeader>

    <div class="flex-1 overflow-y-auto hide-scrollbar p-4">
      <div v-if="loading" class="flex items-center justify-center h-32">
        <div class="text-gray-500">{{ t("common.loading") }}</div>
      </div>

      <form v-else class="space-y-4" data-testid="goal-form" @submit.prevent="save">
        <p class="text-sm text-black dark:text-white opacity-70">{{ t("food.goal.assistantHint") }}</p>

        <NeoPanel class="space-y-4">
          <div class="grid grid-cols-3 gap-2">
            <button
              v-for="option in directions"
              :key="option"
              type="button"
              :class="[
                'min-h-11 border-2 border-nb-border rounded-lg text-sm font-bold',
                direction === option ? 'bg-purple-500 text-white' : 'bg-white text-black dark:bg-zinc-800 dark:text-white',
              ]"
              :aria-pressed="direction === option"
              :data-testid="`direction-${option}`"
              @click="direction = option"
            >
              {{ t(`food.goal.direction.${option}`) }}
            </button>
          </div>

          <div class="floating-label-container">
            <input id="goal-kcal" v-model="fields.kcal" type="text" inputmode="numeric" class="floating-input" :placeholder="t('food.goal.kcal')" data-testid="goal-kcal" />
            <label for="goal-kcal" class="floating-label">{{ t("food.goal.kcal") }}</label>
          </div>
          <div class="grid grid-cols-3 gap-2">
            <div v-for="macro in macroFields" :key="macro" class="floating-label-container">
              <input
                :id="`goal-${macro}`"
                v-model="fields[macro]"
                type="text"
                inputmode="numeric"
                class="floating-input"
                :placeholder="t(`food.goal.${macro}`)"
                :data-testid="`goal-${macro}`"
              />
              <label :for="`goal-${macro}`" class="floating-label">{{ t(`food.goal.${macro}`) }}</label>
            </div>
          </div>
          <p v-if="macroKcal" class="text-xs text-black dark:text-white opacity-70" data-testid="macro-sum">
            {{ t("food.goal.macroSum", { kcal: macroKcal }) }}
          </p>

          <label class="flex items-center justify-between gap-3 min-h-11 text-sm font-bold text-black dark:text-white">
            <span>{{ t("food.goal.addActivities") }}</span>
            <input v-model="addActivities" type="checkbox" class="w-6 h-6 accent-purple-500" data-testid="goal-add-activities" />
          </label>
        </NeoPanel>

        <NeoPanel class="space-y-4">
          <div class="grid grid-cols-2 gap-2">
            <div class="floating-label-container">
              <input id="goal-target" v-model="fields.targetWeight" type="text" inputmode="decimal" class="floating-input" :placeholder="t('food.goal.targetWeight', { unit: weightUnit })" />
              <label for="goal-target" class="floating-label">{{ t("food.goal.targetWeight", { unit: weightUnit }) }}</label>
            </div>
            <div class="floating-label-container">
              <input id="goal-pace" v-model="fields.weeklyChange" type="text" inputmode="decimal" class="floating-input" :placeholder="t('food.goal.weeklyChange', { unit: weightUnit })" />
              <label for="goal-pace" class="floating-label">{{ t("food.goal.weeklyChange", { unit: weightUnit }) }}</label>
            </div>
          </div>
          <div class="floating-label-container">
            <textarea id="goal-notes" v-model="notes" rows="4" class="floating-textarea" :placeholder="t('food.goal.notes')"></textarea>
            <label for="goal-notes" class="floating-label">{{ t("food.goal.notes") }}</label>
          </div>
        </NeoPanel>

        <NeoButton type="submit" variant="primary" full-width :disabled="!draft || saving" data-testid="goal-save">
          {{ t("food.goal.save") }}
        </NeoButton>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import NeoHeader from "@/components/NeoHeader.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import NeoButton from "@/components/NeoButton.vue";
import { getNutritionGoal, saveNutritionGoal } from "@/api/food";
import { useToast } from "@/composables/useToast";
import { useUnits } from "@/composables/useUnits";
import type { NutritionGoalDraft } from "@/types/domain";
import { parseDecimal } from "@/utils/number";

type Direction = NutritionGoalDraft["direction"];

const { t } = useI18n();
const router = useRouter();
const { showError, showSuccess } = useToast();
const { weightUnit } = useUnits();

const directions: Direction[] = ["lose", "maintain", "gain"];
const macroFields = ["protein", "carbs", "fat"] as const;

const loading = ref(true);
const saving = ref(false);
const direction = ref<Direction>("maintain");
const addActivities = ref(true);
const notes = ref("");
const fields = reactive({ kcal: "", protein: "", carbs: "", fat: "", targetWeight: "", weeklyChange: "" });

const whole = (text: string) => {
  const value = parseDecimal(text);
  return value === undefined ? undefined : Math.round(value);
};

// 4 kcal per gram of protein and carbs, 9 of fat.
const macroKcal = computed(() => {
  const [protein, carbs, fat] = macroFields.map((m) => whole(fields[m]));
  if (protein === undefined || carbs === undefined || fat === undefined) return 0;
  return protein * 4 + carbs * 4 + fat * 9;
});

const draft = computed<NutritionGoalDraft | null>(() => {
  const kcal = whole(fields.kcal);
  const [protein, carbs, fat] = macroFields.map((m) => whole(fields[m]));
  if (kcal === undefined || kcal < 800 || kcal > 10000) return null;
  if (protein === undefined || carbs === undefined || fat === undefined) return null;
  if (protein < 0 || carbs < 0 || fat < 0) return null;
  const targetWeight = parseDecimal(fields.targetWeight);
  const weeklyChange = parseDecimal(fields.weeklyChange);
  if (fields.targetWeight.trim() && (targetWeight === undefined || targetWeight <= 0)) return null;
  if (fields.weeklyChange.trim() && (weeklyChange === undefined || Math.abs(weeklyChange) > 2)) return null;
  return {
    direction: direction.value,
    kcal,
    protein,
    carbs,
    fat,
    addActivities: addActivities.value,
    targetWeight: targetWeight ?? null,
    weeklyChange: weeklyChange ?? null,
    notes: notes.value.trim(),
  };
});

onMounted(async () => {
  try {
    const goal = await getNutritionGoal();
    if (goal) {
      direction.value = goal.direction;
      addActivities.value = goal.addActivities;
      notes.value = goal.notes ?? "";
      fields.kcal = String(goal.kcal);
      fields.protein = String(goal.protein);
      fields.carbs = String(goal.carbs);
      fields.fat = String(goal.fat);
      fields.targetWeight = goal.targetWeight != null ? String(goal.targetWeight) : "";
      fields.weeklyChange = goal.weeklyChange != null ? String(goal.weeklyChange) : "";
    }
  } catch (error) {
    console.error("Error loading the goal:", error);
    showError(t("food.loadError"));
  } finally {
    loading.value = false;
  }
});

async function save() {
  if (!draft.value) return;
  saving.value = true;
  try {
    await saveNutritionGoal(draft.value);
    showSuccess(t("food.goal.saved"));
    router.push({ name: "food" });
  } catch (error) {
    console.error("Error saving the goal:", error);
    showError(t("food.saveError"));
  } finally {
    saving.value = false;
  }
}
</script>
