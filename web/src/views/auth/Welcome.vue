<template>
  <div class="h-full overflow-auto bg-gray-100 dark:bg-zinc-800 safe-area-top">
    <div class="max-w-md mx-auto p-6 flex flex-col gap-6 min-h-full justify-center">
      <div class="text-center">
        <span class="material-icons text-6xl text-purple-500">fitness_center</span>
        <h1 class="text-3xl font-black text-black dark:text-white mt-2">Plonkout</h1>
        <p class="text-sm text-black dark:text-white opacity-70 mt-1">
          {{ t("auth.tagline") }}
        </p>
      </div>

      <NeoPanel class="flex flex-col gap-4">
        <p v-if="!supported" class="text-sm font-bold text-red-600">
          {{ t("auth.unsupported") }}
        </p>

        <NeoButton
          variant="primary"
          size="lg"
          full-width
          :disabled="busy || !supported"
          @click="signIn"
        >
          <template #icon>
            <span class="material-icons">fingerprint</span>
          </template>
          {{ busy ? t("auth.waitingForPasskey") : t("auth.signIn") }}
        </NeoButton>

        <p v-if="error" class="text-sm font-bold text-red-600" role="alert">{{ error }}</p>

        <NeoButton variant="secondary" full-width @click="go('signup')">
          {{ t("auth.createAccount") }}
        </NeoButton>
      </NeoPanel>

      <button
        type="button"
        class="text-sm underline text-black dark:text-white opacity-70 self-center"
        @click="go('recover')"
      >
        {{ t("auth.lostPasskey") }}
      </button>
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
import { login, passkeysSupported } from "@/api/auth";
import { useAuth } from "@/composables/useAuth";
import { describeError, returnTo } from "@/views/auth/authFlow";

const { t } = useI18n();
useHead({ title: () => t("auth.signIn") });
const route = useRoute();
const router = useRouter();
const { load } = useAuth();

const supported = passkeysSupported();
const busy = ref(false);
const error = ref("");

async function signIn(): Promise<void> {
  busy.value = true;
  error.value = "";
  try {
    await login();
    await load(true);
    await returnTo(router, route);
  } catch (e) {
    error.value = describeError(e, t);
  } finally {
    busy.value = false;
  }
}

function go(name: "signup" | "recover"): void {
  router.push({ name, query: route.query });
}
</script>
