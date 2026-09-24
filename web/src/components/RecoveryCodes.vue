<template>
  <div class="flex flex-col gap-4">
    <p class="text-sm text-black dark:text-white">{{ t("auth.recoveryCodes.explain") }}</p>

    <ol
      class="grid grid-cols-2 gap-2 font-mono text-base font-bold bg-white dark:bg-zinc-900 text-black dark:text-white border-3 border-nb-border rounded-lg p-3"
      data-testid="recovery-codes"
    >
      <li v-for="code in codes" :key="code" class="select-all">{{ code }}</li>
    </ol>

    <div class="flex gap-2">
      <NeoButton variant="secondary" size="sm" class="flex-1" @click="copy">
        <template #icon><span class="material-icons text-base">content_copy</span></template>
        {{ copied ? t("auth.recoveryCodes.copied") : t("auth.recoveryCodes.copy") }}
      </NeoButton>
      <NeoButton variant="secondary" size="sm" class="flex-1" @click="download">
        <template #icon><span class="material-icons text-base">download</span></template>
        {{ t("auth.recoveryCodes.download") }}
      </NeoButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import NeoButton from "@/components/NeoButton.vue";

const props = defineProps<{ codes: string[]; username: string }>();

const { t } = useI18n();
const copied = ref(false);

function asText(): string {
  return `Plonkout recovery codes for ${props.username}\n\n${props.codes.join("\n")}\n`;
}

async function copy(): Promise<void> {
  await navigator.clipboard.writeText(asText());
  copied.value = true;
}

function download(): void {
  const url = URL.createObjectURL(new Blob([asText()], { type: "text/plain" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = `plonkout-recovery-codes-${props.username}.txt`;
  link.click();
  URL.revokeObjectURL(url);
}
</script>
