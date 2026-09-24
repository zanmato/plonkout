<template>
  <div class="h-full overflow-auto bg-gray-100 dark:bg-zinc-800 safe-area-top">
    <div class="max-w-md mx-auto p-6 flex flex-col gap-6">
      <div class="flex items-center gap-3">
        <NeoButton v-if="!codes" variant="overlay" size="sm" icon-only @click="back">
          <template #icon><span class="material-icons">arrow_back</span></template>
          {{ t("common.cancel") }}
        </NeoButton>
        <h1 class="text-2xl font-black text-black dark:text-white">
          {{ codes ? t("auth.recoveryCodes.title") : t("auth.createAccount") }}
        </h1>
      </div>

      <NeoPanel v-if="!codes" class="flex flex-col gap-4">
        <form class="flex flex-col gap-4" @submit.prevent="create">
          <div class="floating-label-container relative">
            <input
              id="signup-username"
              v-model.trim="username"
              type="text"
              class="floating-input"
              autocomplete="username webauthn"
              autocapitalize="none"
              spellcheck="false"
              minlength="3"
              maxlength="32"
              pattern="[A-Za-z0-9_.\-]+"
              required
              placeholder=" "
            />
            <label for="signup-username" class="floating-label">{{ t("auth.username") }}</label>
          </div>
          <p class="text-xs text-black dark:text-white opacity-70">{{ t("auth.usernameHint") }}</p>

          <!-- Hidden from people, filled in by scripts. -->
          <div class="absolute -left-[9999px]" aria-hidden="true">
            <label for="signup-website">Website</label>
            <input id="signup-website" v-model="website" type="text" tabindex="-1" autocomplete="off" />
          </div>

          <p class="text-sm text-black dark:text-white">{{ t("auth.passkeyExplain") }}</p>

          <NeoButton type="submit" variant="primary" size="lg" full-width :disabled="busy">
            <template #icon><span class="material-icons">fingerprint</span></template>
            {{ busy ? t("auth.waitingForPasskey") : t("auth.createPasskey") }}
          </NeoButton>
          <p v-if="error" class="text-sm font-bold text-red-600" role="alert">{{ error }}</p>
        </form>
      </NeoPanel>

      <NeoPanel v-else class="flex flex-col gap-4">
        <RecoveryCodes :codes="codes" :username="username" />
        <label class="flex items-center gap-3 text-sm font-bold text-black dark:text-white">
          <input v-model="saved" type="checkbox" class="w-5 h-5 accent-purple-500" />
          {{ t("auth.recoveryCodes.saved") }}
        </label>
        <NeoButton variant="primary" size="lg" full-width :disabled="!saved" @click="finish">
          {{ t("auth.recoveryCodes.continue") }}
        </NeoButton>
      </NeoPanel>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useHead } from "@unhead/vue";
import { useRoute, useRouter } from "vue-router";
import NeoButton from "@/components/NeoButton.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import RecoveryCodes from "@/components/RecoveryCodes.vue";
import { getSignupChallenge, signup, type PowSolution } from "@/api/auth";
import { useAuth } from "@/composables/useAuth";
import { solvePowInWorker } from "@/utils/pow";
import { describeError, returnTo } from "@/views/auth/authFlow";

const { t } = useI18n();
useHead({ title: () => t("auth.createAccount") });
const route = useRoute();
const router = useRouter();
const { load } = useAuth();

const username = ref("");
const website = ref("");
const busy = ref(false);
const error = ref("");
const codes = ref<string[] | null>(null);
const saved = ref(false);

// The proof of work starts as soon as the page opens, so it is usually done
// before the person has finished typing a username.
let solution: Promise<PowSolution> | null = null;

function startPow(): Promise<PowSolution> {
  solution = getSignupChallenge().then(solvePowInWorker);
  // A failure surfaces when the form is submitted, not as an unhandled rejection.
  solution.catch(() => {});
  return solution;
}

onMounted(startPow);

async function create(): Promise<void> {
  busy.value = true;
  error.value = "";
  try {
    const pow = await (solution ?? startPow());
    const result = await signup(username.value, pow, website.value);
    codes.value = result.recoveryCodes ?? [];
  } catch (e) {
    error.value = describeError(e, t);
    // A solution is spent once the server has seen it, so the next attempt needs a new one.
    startPow();
  } finally {
    busy.value = false;
  }
}

async function finish(): Promise<void> {
  await load(true);
  await returnTo(router, route);
}

function back(): void {
  router.push({ name: "welcome", query: route.query });
}
</script>
