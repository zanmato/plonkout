<template>
  <button
    :class="[
      'border-3 border-nb-border rounded-lg px-4 py-2 text-sm font-bold flex items-center justify-center shadow-brutal-sm hover-capable transition-all duration-200',
      variantClasses,
      sizeClasses,
      {
        'disabled:opacity-50': disabled,
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

<script setup lang="ts">
import { computed } from "vue";

export type ButtonVariant =
  | "primary"
  | "secondary"
  | "success"
  | "danger"
  | "overlay";
export type ButtonSize = "sm" | "md" | "lg";

const props = withDefaults(
  defineProps<{
    variant?: ButtonVariant;
    size?: ButtonSize;
    disabled?: boolean;
    fullWidth?: boolean;
    iconOnly?: boolean;
  }>(),
  {
    variant: "primary",
    size: "md",
    disabled: false,
    fullWidth: false,
    iconOnly: false,
  },
);

const slots = defineSlots<{
  default?: () => unknown;
  icon?: () => unknown;
}>();

const hasIcon = computed(() => !!slots.icon);

const variantClasses = computed(() => {
  const variants: Record<ButtonVariant, string> = {
    primary: "bg-purple-300 text-black",
    secondary: "bg-nb-overlay text-black",
    success: "bg-green-500 text-white",
    danger: "bg-red-500 text-white",
    overlay: "bg-nb-overlay text-black",
  };
  return variants[props.variant] || variants.primary;
});

const sizeClasses = computed(() => {
  const sizes: Record<ButtonSize, string> = {
    sm: "px-3 py-1.5 text-xs",
    md: "px-4 py-2 text-sm",
    lg: "px-6 py-3 text-base",
  };
  return sizes[props.size] || sizes.md;
});
</script>
