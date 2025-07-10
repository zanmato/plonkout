<template>
  <button
    :class="[
      'border-3 border-nb-border rounded-lg px-4 py-2 text-sm font-bold flex items-center justify-center shadow-brutal-sm hover:shadow-none hover:translate-x-1 hover:translate-y-1 transition-all duration-200',
      variantClasses,
      sizeClasses,
      {
        'disabled:opacity-50 disabled:hover:shadow-brutal-sm disabled:hover:translate-x-0 disabled:hover:translate-y-0':
          disabled,
        'w-full': fullWidth,
        'gap-2': $slots.default && hasIcon,
      },
    ]"
    :disabled="disabled"
    v-bind="$attrs"
  >
    <slot name="icon" />
    <slot v-if="!iconOnly" />
  </button>
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  variant: {
    type: String,
    default: "primary",
    validator: (value) =>
      ["primary", "secondary", "success", "danger", "overlay"].includes(value),
  },
  size: {
    type: String,
    default: "md",
    validator: (value) => ["sm", "md", "lg"].includes(value),
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  fullWidth: {
    type: Boolean,
    default: false,
  },
  iconOnly: {
    type: Boolean,
    default: false,
  },
});

const slots = defineSlots();

const hasIcon = computed(() => !!slots.icon);

const variantClasses = computed(() => {
  const variants = {
    primary: "bg-purple-300 text-black",
    secondary: "bg-nb-overlay text-black",
    success: "bg-green-500 text-white",
    danger: "bg-red-500 text-white",
    overlay: "bg-nb-overlay text-black",
  };
  return variants[props.variant] || variants.primary;
});

const sizeClasses = computed(() => {
  const sizes = {
    sm: "px-3 py-1.5 text-xs",
    md: "px-4 py-2 text-sm",
    lg: "px-6 py-3 text-base",
  };
  return sizes[props.size] || sizes.md;
});
</script>
