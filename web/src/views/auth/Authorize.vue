<template>
  <div class="h-full overflow-auto bg-gray-100 dark:bg-zinc-800 safe-area-top">
    <div class="max-w-md mx-auto p-6 flex flex-col gap-6 min-h-full justify-center">
      <div class="text-center">
        <span class="material-icons text-6xl text-purple-500">link</span>
        <h1 class="text-2xl font-black text-black dark:text-white mt-2">{{ t("oauth.title") }}</h1>
      </div>

      <NeoPanel v-if="loading" class="text-center text-black dark:text-white">
        {{ t("common.loading") }}
      </NeoPanel>

      <NeoPanel v-else-if="error" class="flex flex-col gap-3" data-testid="consent-error">
        <p class="font-bold text-red-600">{{ t("oauth.invalid") }}</p>
        <p class="text-sm text-black dark:text-white opacity-70">{{ error }}</p>
      </NeoPanel>

      <NeoPanel v-else-if="prompt" class="flex flex-col gap-4" data-testid="consent-prompt">
        <p class="text-base text-black dark:text-white">
          <i18n-t keypath="oauth.wants" tag="span">
            <template #client>
              <strong>{{ prompt.clientName || prompt.clientId }}</strong>
            </template>
          </i18n-t>
        </p>
        <ul class="text-sm text-black dark:text-white flex flex-col gap-2">
          <li class="flex gap-2"><span class="material-icons text-base text-purple-500">visibility</span>{{ t("oauth.read") }}</li>
          <li class="flex gap-2"><span class="material-icons text-base text-purple-500">edit_calendar</span>{{ t("oauth.write") }}</li>
        </ul>
        <p class="text-xs text-black dark:text-white opacity-70">
          <i18n-t keypath="oauth.sendsTo" tag="span">
            <template #host>
              <strong class="font-mono">{{ prompt.redirectHost }}</strong>
            </template>
          </i18n-t>
          <span v-if="prompt.loopback"> {{ t("oauth.loopback") }}</span>
        </p>
        <p class="text-xs text-black dark:text-white opacity-70">{{ t("oauth.revokeLater") }}</p>

        <NeoButton variant="primary" size="lg" full-width :disabled="busy" @click="decide(true)">
          {{ t("oauth.allow") }}
        </NeoButton>
        <NeoButton variant="secondary" full-width :disabled="busy" @click="decide(false)">
          {{ t("oauth.deny") }}
        </NeoButton>
      </NeoPanel>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useHead } from "@unhead/vue";
import { useRoute } from "vue-router";
import NeoButton from "@/components/NeoButton.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import type { ConsentPrompt } from "@/api";
import { authorizationParams, decideConsent, getConsentPrompt } from "@/api/oauth";
import { describeError } from "@/views/auth/authFlow";

const { t } = useI18n();
useHead({ title: () => t("oauth.title") });
const route = useRoute();

const params = authorizationParams(route.query);
const prompt = ref<ConsentPrompt | null>(null);
const loading = ref(true);
const busy = ref(false);
const error = ref("");

onMounted(async () => {
  try {
    prompt.value = await getConsentPrompt(params);
  } catch (e) {
    error.value = describeError(e, t);
  } finally {
    loading.value = false;
  }
});

async function decide(approve: boolean): Promise<void> {
  busy.value = true;
  try {
    // A full navigation back to the app that asked, with a code or a refusal.
    window.location.assign(await decideConsent(params, approve));
  } catch (e) {
    error.value = describeError(e, t);
    busy.value = false;
  }
}
</script>
