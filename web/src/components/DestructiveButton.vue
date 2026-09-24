<template>
  <NeoButton
    :variant="isConfirming ? 'danger' : variant"
    :size="size"
    :disabled="disabled"
    :full-width="fullWidth"
    :class="$props.class"
    :title="iconOnly && isConfirming ? t('common.areYouSure') : undefined"
    @click="handleClick"
  >
    <template v-if="$slots.icon && !isConfirming" #icon>
      <slot name="icon" />
    </template>
    <template v-else-if="isConfirming" #icon>
      <span class="material-icons">warning</span>
    </template>
    <span v-if="!iconOnly">
      {{ isConfirming ? t("common.areYouSure") : confirmText }}
    </span>
  </NeoButton>
</template>

<script setup lang="ts">
import { ref, onUnmounted } from "vue";
import { useI18n } from "vue-i18n";
import NeoButton, { type ButtonSize, type ButtonVariant } from "./NeoButton.vue";

const props = withDefaults(
  defineProps<{
    confirmText: string;
    variant?: ButtonVariant;
    size?: ButtonSize;
    disabled?: boolean;
    fullWidth?: boolean;
    timeout?: number;
    class?: string;
    iconOnly?: boolean;
  }>(),
  {
    variant: "danger",
    size: "md",
    disabled: false,
    fullWidth: false,
    timeout: 3000, // 3 seconds
    class: "",
    iconOnly: false,
  },
);

const emit = defineEmits<{ (e: "confirm"): void }>();
const { t } = useI18n();

const isConfirming = ref(false);
let timeoutId: ReturnType<typeof setTimeout> | null = null;

function handleClick() {
  if (props.disabled) return;

  if (!isConfirming.value) {
    // First click - show confirmation
    isConfirming.value = true;
    startTimeout();
  } else {
    // Second click - confirm action
    clearTimeout();
    emit("confirm");
    isConfirming.value = false;
  }
}

function startTimeout() {
  clearTimeout();
  timeoutId = setTimeout(() => {
    if (isConfirming.value) {
      isConfirming.value = false;
    }
    timeoutId = null;
  }, props.timeout);
}

function clearTimeout() {
  if (timeoutId) {
    window.clearTimeout(timeoutId);
    timeoutId = null;
  }
}

onUnmounted(() => {
  clearTimeout();
});
</script>
