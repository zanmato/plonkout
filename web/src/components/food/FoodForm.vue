<template>
  <form class="space-y-3" data-testid="food-form" @submit.prevent="save">
    <div class="floating-label-container">
      <input id="food-name" v-model="name" type="text" class="floating-input" :placeholder="t('food.form.name')" required />
      <label for="food-name" class="floating-label">{{ t("food.form.name") }}</label>
    </div>
    <div class="grid grid-cols-2 gap-3">
      <div class="floating-label-container">
        <input id="food-brand" v-model="brand" type="text" class="floating-input" :placeholder="t('food.form.brand')" />
        <label for="food-brand" class="floating-label">{{ t("food.form.brand") }}</label>
      </div>
      <div class="floating-label-container">
        <input
          id="food-barcode"
          v-model="barcode"
          type="text"
          inputmode="numeric"
          pattern="[0-9]{8,14}"
          class="floating-input"
          :placeholder="t('food.form.barcode')"
        />
        <label for="food-barcode" class="floating-label">{{ t("food.form.barcode") }}</label>
      </div>
    </div>

    <div class="text-sm font-bold text-black dark:text-white pt-1">{{ t("food.form.per100") }}</div>
    <div class="grid grid-cols-2 gap-3">
      <div v-for="field in fields" :key="field.key" class="floating-label-container">
        <input
          :id="`food-${field.key}`"
          v-model="values[field.key]"
          type="text"
          inputmode="decimal"
          class="floating-input"
          :placeholder="t(field.label)"
          :data-testid="`food-${field.key}`"
          required
        />
        <label :for="`food-${field.key}`" class="floating-label">{{ t(field.label) }}</label>
      </div>
    </div>

    <p v-if="error" class="text-sm font-bold text-red-600 dark:text-red-400" role="alert">{{ error }}</p>

    <NeoButton type="submit" variant="primary" full-width :disabled="!valid || saving" data-testid="food-save">
      {{ t("food.form.save") }}
    </NeoButton>
  </form>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import NeoButton from "@/components/NeoButton.vue";
import { ApiError } from "@/api";
import { createFood } from "@/api/food";
import type { Food } from "@/types/domain";
import { parseDecimal } from "@/utils/number";

const props = defineProps<{ initialName?: string }>();
const emit = defineEmits<{ (e: "saved", food: Food): void }>();
const { t } = useI18n();

type Field = "kcal" | "protein" | "carbs" | "fat";
const fields: { key: Field; label: string }[] = [
  { key: "kcal", label: "food.form.kcal" },
  { key: "protein", label: "food.form.protein" },
  { key: "carbs", label: "food.form.carbs" },
  { key: "fat", label: "food.form.fat" },
];

const name = ref(props.initialName ?? "");
const brand = ref("");
const barcode = ref("");
const values = reactive<Record<Field, string>>({ kcal: "", protein: "", carbs: "", fat: "" });
const saving = ref(false);
const error = ref("");

const parsed = computed(() => {
  const out = {} as Record<Field, number | undefined>;
  for (const field of fields) out[field.key] = parseDecimal(values[field.key]);
  return out;
});
const valid = computed(
  () => name.value.trim() !== "" && fields.every((field) => (parsed.value[field.key] ?? -1) >= 0),
);

async function save() {
  if (!valid.value) return;
  saving.value = true;
  error.value = "";
  try {
    const food = await createFood({
      name: name.value.trim(),
      brand: brand.value.trim() || undefined,
      gtin: barcode.value.trim() || undefined,
      per100g: {
        kcal: parsed.value.kcal!,
        protein: parsed.value.protein!,
        carbs: parsed.value.carbs!,
        fat: parsed.value.fat!,
      },
    });
    emit("saved", food);
  } catch (e) {
    error.value = e instanceof ApiError && e.code === "food_exists" ? t("food.form.exists") : t("food.saveError");
  } finally {
    saving.value = false;
  }
}
</script>
