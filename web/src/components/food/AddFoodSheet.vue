<template>
  <BottomSheet :title="t('food.search.title', { meal: mealName })" @close="emit('close')">
    <!-- Creating a food from its label -->
    <FoodForm v-if="creating" :initial-name="query" @saved="pickCreated" />

    <!-- The amount of the food picked -->
    <div v-else-if="selected" class="space-y-4" data-testid="amount">
      <button
        type="button"
        class="flex items-center gap-1 text-sm font-bold text-purple-700 dark:text-purple-300"
        @click="selected = null"
      >
        <span class="material-icons text-base!">arrow_back</span>
        {{ t("food.search.label") }}
      </button>
      <div>
        <div class="text-lg font-bold text-black dark:text-white">{{ selected.name }}</div>
        <div class="text-sm text-black dark:text-white opacity-70">
          <span v-if="selected.brand">{{ selected.brand }} · </span>
          {{ t("food.search.per100", { kcal: round(selected.per100g.kcal) }) }}
        </div>
      </div>

      <div class="flex items-stretch gap-2 flex-wrap">
        <div class="floating-label-container w-32">
          <input
            id="food-grams"
            v-model="gramsText"
            type="text"
            inputmode="decimal"
            class="floating-input"
            :placeholder="t('food.amount.label')"
            data-testid="grams"
            @input="portion = null"
          />
          <label for="food-grams" class="floating-label">{{ t("food.amount.label") }}</label>
        </div>
        <button
          v-for="p in selected.portions"
          :key="p.id"
          type="button"
          :class="[
            'min-h-11 px-3 border-2 border-nb-border rounded-lg text-sm font-bold',
            portion?.id === p.id ? 'bg-purple-500 text-white' : 'bg-white text-black dark:bg-zinc-800 dark:text-white',
          ]"
          @click="addPortion(p)"
        >
          {{ p.name }} {{ round(p.grams) }} g
        </button>
      </div>

      <div v-if="grams && !portion" class="flex items-stretch gap-2">
        <div class="floating-label-container flex-1">
          <input
            id="portion-name"
            v-model="portionName"
            type="text"
            maxlength="40"
            class="floating-input"
            :placeholder="t('food.amount.portionName', { grams: round(grams) })"
            data-testid="portion-name"
          />
          <label for="portion-name" class="floating-label">{{ t("food.amount.portionName", { grams: round(grams) }) }}</label>
        </div>
        <NeoButton
          variant="secondary"
          size="sm"
          :disabled="!portionName.trim() || saving"
          data-testid="save-portion"
          @click="keepPortion"
        >
          {{ t("food.amount.savePortion") }}
        </NeoButton>
      </div>

      <div v-if="grams" class="grid grid-cols-4 gap-2 text-center text-black dark:text-white">
        <div v-for="m in preview" :key="m.label">
          <div class="text-base font-bold">{{ m.value }}</div>
          <div class="text-xs opacity-70">{{ m.label }}</div>
        </div>
      </div>
    </div>

    <!-- Search -->
    <div v-else class="space-y-3">
      <div class="floating-label-container">
        <input
          id="food-search"
          ref="searchInput"
          v-model="query"
          type="search"
          class="floating-input"
          :placeholder="t('food.search.label')"
          autocomplete="off"
          autocorrect="off"
          spellcheck="false"
          data-testid="food-search"
        />
        <label for="food-search" class="floating-label">{{ t("food.search.label") }}</label>
      </div>

      <div v-if="!query.trim()" class="text-sm font-bold text-black dark:text-white">
        {{ myFoods.length ? t("food.search.yourFoods") : t("food.search.noFoods") }}
      </div>
      <p
        v-else-if="searched && matches.length === 0"
        class="text-sm text-black dark:text-white opacity-70"
        data-testid="no-matches"
      >
        {{ t("food.search.nothing") }}
      </p>

      <ul class="border-3 border-nb-border rounded-xl overflow-hidden bg-white dark:bg-zinc-800 empty:hidden">
        <li v-for="match in shown" :key="matchKey(match)" class="border-b border-gray-300 dark:border-zinc-600 last:border-b-0">
          <button
            type="button"
            class="w-full flex items-center gap-3 px-3 py-3 text-left"
            data-testid="food-match"
            @click="pick(match)"
          >
            <div class="flex-1 min-w-0">
              <div class="font-bold text-black dark:text-white">{{ match.name }}</div>
              <div class="text-xs text-black dark:text-white opacity-70 truncate">
                <span v-if="match.source === 'mine'">{{ match.brand || t("food.search.mine") }}</span>
                <span v-else>{{ match.group }}</span>
                <span v-if="match.uses"> · {{ t("food.search.uses", { count: match.uses }, match.uses) }}</span>
              </div>
            </div>
            <div class="text-right text-black dark:text-white shrink-0">
              <div class="text-sm font-bold">{{ Math.round(match.per100g.kcal) }}</div>
              <div class="text-[0.65rem] opacity-70">{{ t("food.search.kcalPer100") }}</div>
            </div>
          </button>
        </li>
      </ul>

      <NeoButton variant="secondary" full-width data-testid="create-food" @click="creating = true">
        <template #icon><span class="material-icons text-base!">add</span></template>
        {{ t("food.search.create") }}
      </NeoButton>

      <p v-if="source" class="text-xs text-center text-black dark:text-white opacity-70">
        <a :href="source.url" target="_blank" rel="noopener" class="underline">{{ source.name }}</a>{{ versionText }},
        <a :href="source.licenseUrl" target="_blank" rel="noopener" class="underline">{{ source.license }}</a>
      </p>
    </div>

    <template v-if="selected && !creating" #footer>
      <NeoButton variant="primary" full-width :disabled="!grams || saving" data-testid="add-food" @click="add">
        {{ t("food.amount.add", { grams: round(grams ?? 0), kcal: Math.round(preview[0]?.raw ?? 0) }) }}
      </NeoButton>
    </template>
  </BottomSheet>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, useTemplateRef, watch } from "vue";
import { useI18n } from "vue-i18n";
import BottomSheet from "@/components/BottomSheet.vue";
import NeoButton from "@/components/NeoButton.vue";
import FoodForm from "@/components/food/FoodForm.vue";
import { getFoods, logFood, savePortion, searchFoods } from "@/api/food";
import type { Day, DayKey, Food, FoodMatch, FoodSearch, Meal, Portion } from "@/types/domain";
import { parseDecimal, scale } from "@/utils/number";

const props = defineProps<{ day: DayKey; meal: Meal }>();
const emit = defineEmits<{
  (e: "close"): void;
  (e: "logged", day: Day): void;
  (e: "error"): void;
}>();
const { t } = useI18n();

const mealName = computed(() => t(`food.meals.${props.meal}`));
const query = ref("");
const matches = ref<FoodMatch[]>([]);
const source = ref<FoodSearch["source"] | null>(null);
const searched = ref(false);
const myFoods = ref<FoodMatch[]>([]);
const selected = ref<FoodMatch | null>(null);
const gramsText = ref("");
const portion = ref<Portion | null>(null);
const portions = ref(0);
const creating = ref(false);
const portionName = ref("");
const saving = ref(false);
const searchInput = useTemplateRef<HTMLInputElement>("searchInput");

const round = (n: number) => Math.round(n * 10) / 10;
const versionText = computed(() => (source.value?.version ? ` ${source.value.version}` : ""));
const shown = computed(() => (query.value.trim() ? matches.value : myFoods.value));
const grams = computed(() => {
  const value = parseDecimal(gramsText.value);
  return value && value > 0 && value <= 5000 ? value : undefined;
});
const preview = computed(() => {
  if (!selected.value || !grams.value) return [];
  const n = selected.value.per100g;
  const g = grams.value;
  return [
    { label: t("food.kcal"), raw: scale(n.kcal, g), value: Math.round(scale(n.kcal, g)) },
    { label: t("food.macros.carbs"), raw: scale(n.carbs, g), value: `${scale(n.carbs, g)} g` },
    { label: t("food.macros.protein"), raw: scale(n.protein, g), value: `${scale(n.protein, g)} g` },
    { label: t("food.macros.fat"), raw: scale(n.fat, g), value: `${scale(n.fat, g)} g` },
  ];
});

function matchKey(match: FoodMatch) {
  return match.foodId ?? `lmv-${match.lmvNumber}`;
}

function asMatch(food: Food): FoodMatch {
  return {
    source: "mine", foodId: food.id, lmvNumber: null, name: food.name, brand: food.brand, group: "",
    per100g: food.per100g, portions: food.portions, uses: 0,
  };
}

onMounted(async () => {
  searchInput.value?.focus();
  try {
    myFoods.value = (await getFoods()).map(asMatch);
    // The attribution shows before the first search too.
    source.value = (await searchFoods("", 1)).source;
  } catch (error) {
    console.error("Error loading foods:", error);
  }
});

// Search as the person types. Only the latest answer counts, an earlier one
// arriving late must not replace it.
let latest = 0;
let timer: ReturnType<typeof setTimeout> | undefined;
watch(query, (text) => {
  clearTimeout(timer);
  searched.value = false;
  if (!text.trim()) {
    matches.value = [];
    return;
  }
  timer = setTimeout(async () => {
    const mine = ++latest;
    try {
      const result = await searchFoods(text.trim());
      if (mine !== latest) return;
      matches.value = result.matches;
      source.value = result.source;
      searched.value = true;
    } catch (error) {
      console.error("Error searching foods:", error);
    }
  }, 250);
});

function pick(match: FoodMatch) {
  selected.value = match;
  portion.value = null;
  portions.value = 0;
  gramsText.value = match.portions.length === 1 ? String(match.portions[0]!.grams) : "100";
  if (match.portions.length === 1) {
    portion.value = match.portions[0]!;
    portions.value = 1;
  }
}

// Remembers the amount typed as a household measure of the food, e.g. a
// tablespoon of pesto is 15 g, picked with one tap from then on.
async function keepPortion() {
  const match = selected.value;
  if (!match || !grams.value || !portionName.value.trim()) return;
  saving.value = true;
  try {
    const saved = await savePortion(match, portionName.value.trim(), grams.value);
    match.portions = [...match.portions.filter((p) => p.id !== saved.id), saved].sort((a, b) => a.grams - b.grams);
    portion.value = saved;
    portions.value = 1;
    portionName.value = "";
  } catch (error) {
    console.error("Error saving a portion:", error);
    emit("error");
  } finally {
    saving.value = false;
  }
}

function pickCreated(food: Food) {
  creating.value = false;
  pick(asMatch(food));
}

// Tapping a portion again adds another one: two tablespoons is two taps.
function addPortion(p: Portion) {
  portions.value = portion.value?.id === p.id ? portions.value + 1 : 1;
  portion.value = p;
  gramsText.value = String(round(p.grams * portions.value));
}

async function add() {
  if (!selected.value || !grams.value) return;
  saving.value = true;
  try {
    const match = selected.value;
    const day = await logFood(props.day, [
      {
        meal: props.meal,
        ...(match.foodId ? { foodId: match.foodId } : { lmvNumber: match.lmvNumber! }),
        grams: grams.value,
        amount: portion.value ? `${portions.value} ${portion.value.name}` : undefined,
      },
    ]);
    emit("logged", day);
  } catch (error) {
    console.error("Error logging food:", error);
    emit("error");
  } finally {
    saving.value = false;
  }
}
</script>
