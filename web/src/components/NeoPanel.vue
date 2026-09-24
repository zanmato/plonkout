<template>
  <div
    :class="[
      'bg-nb-bg border-3 border-black dark:bg-zinc-700 dark:text-white rounded-xl shadow-brutal',
      paddingClasses,
      {
        'w-full': fullWidth,
      },
    ]"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

export type PanelPadding = "none" | "sm" | "md" | "lg" | "xl";

const props = withDefaults(
  defineProps<{
    padding?: PanelPadding;
    fullWidth?: boolean;
  }>(),
  {
    padding: "md",
    fullWidth: false,
  },
);

const paddingClasses = computed(() => {
  const paddings: Record<PanelPadding, string> = {
    none: "",
    sm: "p-3",
    md: "p-4",
    lg: "p-6",
    xl: "p-8",
  };
  return paddings[props.padding] || paddings.md;
});
</script>
