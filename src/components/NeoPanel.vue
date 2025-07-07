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

<script setup>
import { computed } from "vue";

const props = defineProps({
  padding: {
    type: String,
    default: "md",
    validator: (value) => ["none", "sm", "md", "lg", "xl"].includes(value),
  },
  fullWidth: {
    type: Boolean,
    default: false,
  },
});

const paddingClasses = computed(() => {
  const paddings = {
    none: "",
    sm: "p-3",
    md: "p-4",
    lg: "p-6",
    xl: "p-8",
  };
  return paddings[props.padding] || paddings.md;
});
</script>
