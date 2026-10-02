<template>
  <div
    class="fixed inset-0 z-50 flex items-end justify-center overscroll-none"
    style="background-color: rgba(0, 0, 0, 0.4)"
    @click="emit('close')"
  >
    <NeoPanel
      role="dialog"
      aria-modal="true"
      :aria-label="title"
      padding="none"
      class="w-full max-w-md max-h-[90vh] flex flex-col rounded-b-none safe-area-bottom"
      @click.stop
    >
      <div class="flex items-center justify-between gap-3 p-4 border-b-2 border-nb-border">
        <h3 class="text-xl font-bold text-black dark:text-white truncate">
          {{ title }}
        </h3>
        <button
          type="button"
          class="w-10 h-10 shrink-0 flex items-center justify-center rounded-full border-2 border-nb-border bg-white dark:bg-zinc-800 text-black dark:text-white"
          :aria-label="t('common.close')"
          data-testid="sheet-close"
          @click="emit('close')"
        >
          <span class="material-icons text-xl">close</span>
        </button>
      </div>
      <div class="flex-1 overflow-y-auto overscroll-contain p-4">
        <slot />
      </div>
      <div v-if="$slots.footer" class="p-4 border-t-2 border-nb-border">
        <slot name="footer" />
      </div>
    </NeoPanel>
  </div>
</template>

<script setup lang="ts">
import { onUnmounted } from "vue";
import { useI18n } from "vue-i18n";
import NeoPanel from "@/components/NeoPanel.vue";
import { lockScroll } from "@/utils/scrollLock";

defineProps<{ title: string }>();
const emit = defineEmits<{ (e: "close"): void }>();
const { t } = useI18n();

const release = lockScroll();
onUnmounted(release);
</script>
