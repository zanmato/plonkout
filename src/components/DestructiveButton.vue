<template>
  <NeoButton
    @click="handleClick"
    :variant="isConfirming ? 'danger' : variant"
    :size="size"
    :disabled="disabled"
    :full-width="fullWidth"
    :class="$props.class"
    :title="iconOnly && isConfirming ? t('common.areYouSure') : undefined"
  >
    <template #icon v-if="$slots.icon && !isConfirming">
      <slot name="icon" />
    </template>
    <template #icon v-else-if="isConfirming">
      <span class="material-icons">warning</span>
    </template>
    <span v-if="!iconOnly">
      {{ isConfirming ? t("common.areYouSure") : confirmText }}
    </span>
  </NeoButton>
</template>

<script setup>
import { ref, onUnmounted } from "vue";
import { useI18n } from "vue-i18n";
import NeoButton from "./NeoButton.vue";

const props = defineProps({
  confirmText: {
    type: String,
    required: true,
  },
  variant: {
    type: String,
    default: "danger",
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
  timeout: {
    type: Number,
    default: 3000, // 3 seconds
  },
  class: {
    type: String,
    default: "",
  },
  iconOnly: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(["confirm"]);
const { t } = useI18n();

const isConfirming = ref(false);
let timeoutId = null;

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
