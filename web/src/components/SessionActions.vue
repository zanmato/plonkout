<template>
  <div>
    <div v-if="!skipping" class="flex gap-3">
      <NeoButton
        variant="primary"
        class="flex-1"
        :disabled="busy"
        data-testid="start-session"
        @click="$emit('start')"
      >
        <template #icon>
          <span class="material-icons">play_arrow</span>
        </template>
        {{ status === "in_progress" ? t("plan.continue") : t("plan.start") }}
      </NeoButton>
      <!-- Only a session not yet started can be skipped -->
      <NeoButton
        v-if="status === 'pending'"
        variant="secondary"
        :disabled="busy"
        data-testid="skip-session"
        @click="skipping = true"
      >
        <template #icon>
          <span class="material-icons">skip_next</span>
        </template>
        {{ t("plan.skip") }}
      </NeoButton>
    </div>

    <!-- Skip, with an optional reason for whoever adjusts the plan -->
    <div v-else class="space-y-3">
      <input
        v-model="reason"
        type="text"
        maxlength="500"
        class="w-full px-3 py-2 bg-white dark:bg-zinc-800 border-2 border-nb-border rounded-md text-black dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500"
        :placeholder="t('plan.skipReason')"
        data-testid="skip-reason"
        @keydown.enter="confirmSkip"
      />
      <div class="flex gap-3">
        <NeoButton
          variant="danger"
          class="flex-1"
          :disabled="busy"
          data-testid="confirm-skip"
          @click="confirmSkip"
        >
          {{ t("plan.confirmSkip") }}
        </NeoButton>
        <NeoButton variant="overlay" :disabled="busy" @click="cancel">
          {{ t("common.cancel") }}
        </NeoButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import NeoButton from "@/components/NeoButton.vue";
import type { PlannedSession } from "@/types/domain";

withDefaults(
  defineProps<{
    status: PlannedSession["status"];
    busy?: boolean;
  }>(),
  { busy: false },
);

const emit = defineEmits<{
  start: [];
  skip: [reason: string];
}>();

const { t } = useI18n();
const skipping = ref(false);
const reason = ref("");

function confirmSkip() {
  emit("skip", reason.value);
}

function cancel() {
  skipping.value = false;
  reason.value = "";
}
</script>
