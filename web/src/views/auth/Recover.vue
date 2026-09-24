<template>
  <div class="h-full overflow-auto bg-gray-100 dark:bg-zinc-800 safe-area-top">
    <div class="max-w-md mx-auto p-6 flex flex-col gap-6">
      <div class="flex items-center gap-3">
        <NeoButton variant="overlay" size="sm" icon-only @click="back">
          <template #icon><span class="material-icons">arrow_back</span></template>
          {{ t("common.cancel") }}
        </NeoButton>
        <h1 class="text-2xl font-black text-black dark:text-white">{{ t("auth.recover.title") }}</h1>
      </div>

      <NeoPanel>
        <form class="flex flex-col gap-4" @submit.prevent="submit">
          <p class="text-sm text-black dark:text-white">{{ t("auth.recover.explain") }}</p>

          <div class="floating-label-container relative">
            <input
              id="recover-username"
              v-model.trim="username"
              type="text"
              class="floating-input"
              autocomplete="username"
              autocapitalize="none"
              spellcheck="false"
              required
              placeholder=" "
            />
            <label for="recover-username" class="floating-label">{{ t("auth.username") }}</label>
          </div>

          <div class="floating-label-container relative">
            <input
              id="recover-code"
              v-model.trim="code"
              type="text"
              class="floating-input font-mono"
              autocomplete="one-time-code"
              autocapitalize="none"
              spellcheck="false"
              required
              placeholder=" "
            />
            <label for="recover-code" class="floating-label">{{ t("auth.recover.code") }}</label>
          </div>

          <NeoButton type="submit" variant="primary" size="lg" full-width :disabled="busy">
            <template #icon><span class="material-icons">fingerprint</span></template>
            {{ busy ? t("auth.waitingForPasskey") : t("auth.recover.submit") }}
          </NeoButton>
          <p v-if="error" class="text-sm font-bold text-red-600" role="alert">{{ error }}</p>
        </form>
      </NeoPanel>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useHead } from "@unhead/vue";
import { useRoute, useRouter } from "vue-router";
import NeoButton from "@/components/NeoButton.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import { recover } from "@/api/auth";
import { useAuth } from "@/composables/useAuth";
import { describeError, returnTo } from "@/views/auth/authFlow";

const { t } = useI18n();
useHead({ title: () => t("auth.recover.title") });
const route = useRoute();
const router = useRouter();
const { load } = useAuth();

const username = ref("");
const code = ref("");
const busy = ref(false);
const error = ref("");

async function submit(): Promise<void> {
  busy.value = true;
  error.value = "";
  try {
    await recover(username.value, code.value);
    await load(true);
    await returnTo(router, route);
  } catch (e) {
    error.value = describeError(e, t);
  } finally {
    busy.value = false;
  }
}

function back(): void {
  router.push({ name: "welcome", query: route.query });
}
</script>
