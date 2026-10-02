<template>
  <!--
    The overlay covers the visual viewport rather than the layout viewport. A
    phone's keyboard shrinks only the visual one, so a sheet at the bottom of
    the layout viewport would sit behind the keyboard, its end out of reach.
  -->
  <div
    class="fixed inset-x-0 z-50 flex items-end justify-center overscroll-none"
    :style="{ top: `${viewport.top}px`, height: `${viewport.height}px`, backgroundColor: 'rgba(0, 0, 0, 0.4)' }"
    @click="emit('close')"
  >
    <NeoPanel
      role="dialog"
      aria-modal="true"
      :aria-label="title"
      padding="none"
      :class="[
        'w-full max-w-md flex flex-col rounded-b-none safe-area-bottom',
        fill ? 'h-[92%]' : 'max-h-[92%]',
      ]"
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
      <slot name="header" />
      <div class="flex-1 min-h-0 overflow-y-auto overscroll-contain p-4">
        <slot />
      </div>
      <div v-if="$slots.footer" class="p-4 border-t-2 border-nb-border">
        <slot name="footer" />
      </div>
    </NeoPanel>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import NeoPanel from "@/components/NeoPanel.vue";
import { lockScroll } from "@/utils/scrollLock";

withDefaults(
  defineProps<{
    title: string;
    /** Keep the sheet at full height, so changing content scrolls inside it instead of resizing it. */
    fill?: boolean;
  }>(),
  { fill: false },
);
const emit = defineEmits<{ (e: "close"): void }>();
const { t } = useI18n();

const release = lockScroll();

const viewport = ref({ top: 0, height: window.innerHeight });
function follow() {
  const visual = window.visualViewport;
  viewport.value = visual
    ? { top: visual.offsetTop, height: visual.height }
    : { top: 0, height: window.innerHeight };
}

// Escape closes the sheet, like a native dialog.
function onKey(event: KeyboardEvent) {
  if (event.key === "Escape") emit("close");
}

onMounted(() => {
  follow();
  window.addEventListener("keydown", onKey);
  window.visualViewport?.addEventListener("resize", follow);
  window.visualViewport?.addEventListener("scroll", follow);
  window.addEventListener("resize", follow);
});
onUnmounted(() => {
  window.visualViewport?.removeEventListener("resize", follow);
  window.visualViewport?.removeEventListener("scroll", follow);
  window.removeEventListener("resize", follow);
  window.removeEventListener("keydown", onKey);
  release();
});
</script>
