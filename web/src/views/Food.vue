<template>
  <div class="food h-full flex flex-col">
    <NeoHeader :title="t('food.title')">
      <template #right>
        <NeoButton variant="primary" size="sm" data-testid="goal-button" @click="router.push({ name: 'food-goal' })">
          <template #icon><span class="material-icons text-base!">flag</span></template>
          {{ t("food.goalButton") }}
        </NeoButton>
      </template>
    </NeoHeader>

    <div class="flex-1 overflow-y-auto hide-scrollbar p-4 space-y-4">
      <!-- Day picker -->
      <div class="flex items-center justify-between gap-3">
        <NeoButton
          variant="secondary"
          class="w-11 h-11 px-0! py-0!"
          icon-only
          :aria-label="t('food.previousDay')"
          data-testid="previous-day"
          @click="goTo(addDays(day, -1))"
        >
          <template #icon><span class="material-icons">chevron_left</span></template>
        </NeoButton>
        <button
          type="button"
          class="flex flex-col items-center text-black dark:text-white"
          :aria-label="t('food.goToToday')"
          data-testid="day-label"
          @click="goTo(today())"
        >
          <span class="text-lg font-bold">{{ dayName }}</span>
          <span class="text-xs opacity-70">{{ d(dateOf(day), "short") }}</span>
        </button>
        <NeoButton
          variant="secondary"
          class="w-11 h-11 px-0! py-0!"
          icon-only
          :aria-label="t('food.nextDay')"
          data-testid="next-day"
          @click="goTo(addDays(day, 1))"
        >
          <template #icon><span class="material-icons">chevron_right</span></template>
        </NeoButton>
      </div>

      <div v-if="!diary" class="flex items-center justify-center h-32">
        <div class="text-gray-500">{{ t("common.loading") }}</div>
      </div>

      <template v-else>
        <!-- No goal yet -->
        <NeoPanel v-if="!diary.goal" class="text-black dark:text-white" data-testid="no-goal">
          <div class="font-bold">{{ t("food.noGoal") }}</div>
          <p class="text-sm opacity-70 mt-1">{{ t("food.noGoalDescription") }}</p>
          <NeoButton variant="primary" size="sm" class="mt-3" @click="router.push({ name: 'food-goal' })">
            {{ t("food.setGoal") }}
          </NeoButton>
        </NeoPanel>

        <!-- Summary -->
        <NeoPanel class="space-y-5" data-testid="summary">
          <div class="grid grid-cols-3 items-center gap-2 text-black dark:text-white">
            <div class="text-center">
              <div class="text-2xl font-black" data-testid="eaten">{{ Math.round(diary.eaten.kcal) }}</div>
              <div class="text-xs uppercase tracking-wide opacity-70">{{ t("food.eaten") }}</div>
            </div>
            <div class="relative w-30 h-30 justify-self-center">
              <svg viewBox="0 0 120 120" class="w-full h-full -rotate-90" aria-hidden="true">
                <circle cx="60" cy="60" r="52" fill="none" stroke-width="12" class="stroke-nb-overlay dark:stroke-zinc-800" />
                <circle
                  v-if="ring > 0"
                  cx="60"
                  cy="60"
                  r="52"
                  fill="none"
                  stroke-width="12"
                  stroke-linecap="round"
                  :class="over ? 'stroke-red-500' : 'stroke-purple-500'"
                  :stroke-dasharray="`${ring} ${CIRCUMFERENCE}`"
                />
              </svg>
              <div class="absolute inset-0 flex flex-col items-center justify-center">
                <template v-if="diary.remaining !== null">
                  <div class="text-2xl font-black leading-none" data-testid="remaining">
                    {{ Math.abs(Math.round(diary.remaining)) }}
                  </div>
                  <div class="text-[0.65rem] uppercase tracking-wide opacity-70 mt-1">
                    {{ over ? t("food.over") : t("food.left") }}
                  </div>
                </template>
                <div v-else class="text-sm font-bold opacity-70">{{ t("food.kcal") }}</div>
              </div>
            </div>
            <div class="text-center">
              <div class="text-2xl font-black" data-testid="burned">{{ diary.burned }}</div>
              <div class="text-xs uppercase tracking-wide opacity-70">{{ t("food.burned") }}</div>
            </div>
          </div>

          <div class="grid grid-cols-3 gap-3 text-black dark:text-white">
            <div v-for="macro in macros" :key="macro.key" :data-testid="`macro-${macro.key}`">
              <div class="text-sm font-bold">{{ t(`food.macros.${macro.key}`) }}</div>
              <div class="h-2.5 my-1.5 bg-nb-overlay border-2 border-nb-border rounded-md overflow-hidden dark:bg-zinc-800">
                <div class="h-full" :class="macro.color" :style="{ width: `${macro.percent}%` }"></div>
              </div>
              <div class="text-xs opacity-80">
                {{ macro.eaten }}<template v-if="macro.goal !== undefined"> / {{ macro.goal }}</template> g
              </div>
            </div>
          </div>
        </NeoPanel>

        <!-- Meals -->
        <MealCard
          v-for="meal in diary.meals"
          :key="meal.meal"
          :meal="meal"
          :day="day"
          :last="recentOf(meal.meal)?.last"
          :busy="busy"
          @add="adding = meal.meal"
          @edit="editing = $event"
          @repeat="repeat(meal.meal)"
        />

        <!-- Exercise -->
        <section
          class="bg-white border-3 border-nb-border rounded-xl shadow-brutal overflow-hidden dark:bg-zinc-700"
          data-testid="activities"
        >
          <div class="flex items-center gap-3 py-3 pl-4 pr-3">
            <div class="flex-1">
              <h3 class="text-lg font-bold text-black dark:text-white">{{ t("food.activities.title") }}</h3>
              <div class="text-sm text-black dark:text-white opacity-70">
                {{ diary.burned ? t("food.activities.burned", { kcal: diary.burned }) : t("food.activities.none") }}
              </div>
            </div>
            <NeoButton
              variant="primary"
              size="sm"
              class="w-11 h-11 px-0! py-0! shrink-0"
              icon-only
              :aria-label="t('food.activities.add')"
              data-testid="add-activity"
              @click="activityOpen = true"
            >
              <template #icon><span class="material-icons">add</span></template>
            </NeoButton>
          </div>
          <ul v-if="diary.activities.length" class="border-t-2 border-nb-border">
            <li
              v-for="activity in diary.activities"
              :key="activity.id"
              class="flex items-center gap-3 px-4 py-3 border-b border-gray-300 last:border-b-0 dark:border-zinc-600"
            >
              <div class="flex-1 min-w-0 text-black dark:text-white">
                <div class="font-bold">{{ activity.label }}</div>
                <div v-if="activity.loggedBy === 'assistant'" class="text-xs opacity-70">
                  {{ t("food.activities.byAssistant") }}
                </div>
              </div>
              <div class="text-sm font-bold text-black dark:text-white">{{ activity.kcal }}</div>
              <DestructiveButton
                :confirm-text="t('food.activities.delete')"
                variant="secondary"
                size="sm"
                icon-only
                @confirm="run(() => deleteActivity(activity.id))"
              >
                <template #icon><span class="material-icons text-base!">delete</span></template>
              </DestructiveButton>
            </li>
          </ul>
        </section>

        <!-- Body weight -->
        <div
          class="flex items-center justify-between gap-3 px-4 py-3 bg-white border-3 border-nb-border rounded-xl dark:bg-zinc-700"
          data-testid="weight"
        >
          <div class="text-black dark:text-white">
            <div class="text-sm opacity-70">{{ t("food.weight.title") }}</div>
            <div v-if="diary.weight !== null" class="text-lg font-bold">{{ diary.weight }} {{ weightUnit }}</div>
          </div>
          <NeoButton variant="secondary" size="sm" data-testid="log-weight" @click="openWeight">
            {{ t("food.weight.log") }}
          </NeoButton>
        </div>
      </template>
    </div>

    <AddFoodSheet
      v-if="adding"
      :day="day"
      :meal="adding"
      :usual="recentOf(adding)?.foods"
      @close="adding = null"
      @logged="logged"
      @updated="diary = $event"
      @error="showError(t('food.saveError'))"
    />

    <EntrySheet
      v-if="editing"
      :entry="editing"
      :day="day"
      @close="editing = null"
      @changed="changed"
      @error="showError(t('food.saveError'))"
    />

    <BottomSheet v-if="activityOpen" :title="t('food.activities.add')" @close="activityOpen = false">
      <div class="space-y-3">
        <div class="floating-label-container">
          <input
            id="activity-label"
            v-model="activityLabel"
            type="text"
            class="floating-input"
            :placeholder="t('food.activities.label')"
            data-testid="activity-label"
          />
          <label for="activity-label" class="floating-label">{{ t("food.activities.label") }}</label>
        </div>
        <div class="floating-label-container">
          <input
            id="activity-kcal"
            v-model="activityKcal"
            type="text"
            inputmode="numeric"
            class="floating-input"
            :placeholder="t('food.activities.kcal')"
            data-testid="activity-kcal"
          />
          <label for="activity-kcal" class="floating-label">{{ t("food.activities.kcal") }}</label>
        </div>
        <p class="text-xs text-black dark:text-white opacity-70">{{ t("food.activities.hint") }}</p>
      </div>
      <template #footer>
        <NeoButton
          variant="primary"
          full-width
          :disabled="!activityValid || busy"
          data-testid="activity-save"
          @click="saveActivity"
        >
          {{ t("food.activities.add") }}
        </NeoButton>
      </template>
    </BottomSheet>

    <BottomSheet v-if="weightOpen" :title="t('food.weight.title')" @close="weightOpen = false">
      <div class="floating-label-container">
        <input
          id="weight"
          v-model="weightText"
          type="text"
          inputmode="decimal"
          class="floating-input"
          :placeholder="`${t('food.weight.label')} (${weightUnit})`"
          data-testid="weight-input"
        />
        <label for="weight" class="floating-label">{{ t("food.weight.label") }} ({{ weightUnit }})</label>
      </div>
      <template #footer>
        <div class="space-y-2">
          <NeoButton
            variant="primary"
            full-width
            :disabled="!weightValue || busy"
            data-testid="weight-save"
            @click="saveWeightOfDay"
          >
            {{ t("food.weight.save") }}
          </NeoButton>
          <DestructiveButton
            v-if="diary?.weight !== null"
            :confirm-text="t('food.weight.delete')"
            variant="secondary"
            full-width
            @confirm="removeWeight"
          />
        </div>
      </template>
    </BottomSheet>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import NeoHeader from "@/components/NeoHeader.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import NeoButton from "@/components/NeoButton.vue";
import BottomSheet from "@/components/BottomSheet.vue";
import DestructiveButton from "@/components/DestructiveButton.vue";
import MealCard from "@/components/food/MealCard.vue";
import AddFoodSheet from "@/components/food/AddFoodSheet.vue";
import EntrySheet from "@/components/food/EntrySheet.vue";
import {
  againEntry,
  deleteActivity,
  deleteWeight,
  getDay,
  getRecentMeals,
  logActivity,
  logFood,
  saveWeight,
} from "@/api/food";
import { useToast } from "@/composables/useToast";
import { useUnits } from "@/composables/useUnits";
import type { Day, DayKey, FoodEntry, Meal, RecentMeals } from "@/types/domain";
import { addDays, dateOf, isDayKey, today } from "@/utils/day";
import { parseDecimal } from "@/utils/number";

const CIRCUMFERENCE = 2 * Math.PI * 52;

const { t, d } = useI18n();
const router = useRouter();
const route = useRoute();
const { showError } = useToast();
const { weightUnit } = useUnits();

// The day lives in the address, so a reload or the back button keeps it.
const day = ref<DayKey>(isDayKey(route.query.day) ? route.query.day : today());
const diary = ref<Day | null>(null);
const recent = ref<RecentMeals | null>(null);
const adding = ref<Meal | null>(null);
const editing = ref<FoodEntry | null>(null);
const busy = ref(false);

const dayName = computed(() => {
  if (day.value === today()) return t("common.today");
  if (day.value === addDays(today(), -1)) return t("common.yesterday");
  return d(dateOf(day.value), { weekday: "long" });
});

const over = computed(() => (diary.value?.remaining ?? 0) < 0);
const ring = computed(() => {
  const budget = diary.value?.budget;
  if (!diary.value || !budget) return 0;
  return Math.min(1, diary.value.eaten.kcal / budget) * CIRCUMFERENCE;
});

const macros = computed(() => {
  const eaten = diary.value?.eaten;
  const goal = diary.value?.goal;
  return (
    [
      { key: "carbs", color: "bg-amber-500" },
      { key: "protein", color: "bg-blue-700" },
      { key: "fat", color: "bg-pink-500" },
    ] as const
  ).map((macro) => {
    const value = eaten?.[macro.key] ?? 0;
    const target = goal?.[macro.key];
    return {
      ...macro,
      eaten: Math.round(value),
      goal: target,
      percent: target ? Math.min(100, Math.round((value / target) * 100)) : 0,
    };
  });
});

async function load() {
  try {
    diary.value = await getDay(day.value);
  } catch (error) {
    console.error("Error loading the day:", error);
    showError(t("food.loadError"));
  }
}

// What was eaten before the day only changes with the day, not with logging.
async function loadRecent() {
  try {
    recent.value = await getRecentMeals(day.value);
  } catch (error) {
    // Quick add is a shortcut, the day works without it.
    console.error("Error loading recent meals:", error);
  }
}

function recentOf(meal: Meal) {
  return recent.value?.meals.find((m) => m.meal === meal);
}

// Logs the meal's last logging again, every food at its amount.
async function repeat(meal: Meal) {
  const last = recentOf(meal)?.last;
  if (!last) return;
  busy.value = true;
  try {
    diary.value = await logFood(day.value, last.entries.map((food) => againEntry(food, meal)));
  } catch (error) {
    console.error("Error repeating a meal:", error);
    showError(t("food.saveError"));
  } finally {
    busy.value = false;
  }
}

function goTo(next: DayKey) {
  if (next === day.value) return;
  day.value = next;
  router.replace({ query: next === today() ? {} : { day: next } });
}

watch(day, () => {
  diary.value = null;
  recent.value = null;
  load();
  loadRecent();
});
onMounted(() => {
  load();
  loadRecent();
});

function logged(updated: Day) {
  adding.value = null;
  diary.value = updated;
}

function changed() {
  editing.value = null;
  load();
}

async function run(action: () => Promise<unknown>) {
  busy.value = true;
  try {
    await action();
    await load();
    return true;
  } catch (error) {
    console.error("Error saving:", error);
    showError(t("food.saveError"));
    return false;
  } finally {
    busy.value = false;
  }
}

// Exercise
const activityOpen = ref(false);
const activityLabel = ref("");
const activityKcal = ref("");
const activityValid = computed(() => {
  const kcal = parseDecimal(activityKcal.value);
  return activityLabel.value.trim() !== "" && kcal !== undefined && kcal >= 1 && kcal <= 10000;
});

async function saveActivity() {
  const kcal = Math.round(parseDecimal(activityKcal.value)!);
  if (await run(() => logActivity(day.value, { label: activityLabel.value.trim(), kcal }))) {
    activityOpen.value = false;
    activityLabel.value = "";
    activityKcal.value = "";
  }
}

// Body weight
const weightOpen = ref(false);
const weightText = ref("");
const weightValue = computed(() => {
  const value = parseDecimal(weightText.value);
  return value && value > 0 && value < 1000 ? value : undefined;
});

function openWeight() {
  weightText.value = diary.value?.weight != null ? String(diary.value.weight) : "";
  weightOpen.value = true;
}

async function saveWeightOfDay() {
  if (await run(() => saveWeight(day.value, weightValue.value!))) weightOpen.value = false;
}

async function removeWeight() {
  if (await run(() => deleteWeight(day.value))) weightOpen.value = false;
}
</script>
