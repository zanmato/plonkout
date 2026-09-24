<template>
  <NeoPanel class="flex flex-col gap-6">
    <div>
      <h3 class="text-xl font-semibold text-black dark:text-white mb-1">
        {{ t("settings.account.title") }}
      </h3>
      <p v-if="me" class="text-sm text-black dark:text-white opacity-70">
        {{ t("settings.account.signedInAs", { username: me.username }) }}
      </p>
    </div>

    <!-- Passkeys -->
    <div>
      <h4 class="text-base font-bold text-black dark:text-white mb-2">
        {{ t("settings.account.passkeys") }}
      </h4>
      <ul class="flex flex-col gap-2 mb-3">
        <li
          v-for="passkey in passkeys"
          :key="passkey.id"
          class="border-3 border-nb-border rounded-lg p-3 bg-white dark:bg-zinc-900 flex items-center gap-3"
        >
          <span class="material-icons text-purple-500">key</span>
          <div class="flex-1 min-w-0">
            <input
              v-if="renaming === passkey.id"
              v-model.trim="newName"
              class="w-full border-2 border-nb-border rounded px-2 py-1 text-sm bg-white dark:bg-zinc-800 text-black dark:text-white"
              :aria-label="t('settings.account.renamePrompt')"
              maxlength="64"
              @keydown.enter="saveName(passkey)"
              @keydown.esc="renaming = null"
            />
            <div v-else class="font-bold text-black dark:text-white truncate">
              {{ passkey.name }}
              <span v-if="passkey.synced" class="ml-1 text-xs font-semibold opacity-60">
                {{ t("settings.account.synced") }}
              </span>
            </div>
            <div class="text-xs text-black dark:text-white opacity-60">
              {{
                passkey.lastUsed
                  ? t("settings.account.lastUsed", { date: d(new Date(passkey.lastUsed), "short") })
                  : t("settings.account.neverUsed")
              }}
            </div>
          </div>
          <NeoButton
            v-if="renaming === passkey.id"
            variant="primary"
            size="sm"
            @click="saveName(passkey)"
          >
            {{ t("settings.account.done") }}
          </NeoButton>
          <template v-else>
            <NeoButton variant="overlay" size="sm" icon-only @click="startRename(passkey)">
              <template #icon><span class="material-icons text-base">edit</span></template>
              {{ t("settings.account.rename") }}
            </NeoButton>
            <DestructiveButton
              v-if="passkeys.length > 1"
              :confirm-text="t('settings.account.remove')"
              size="sm"
              icon-only
              @confirm="remove(passkey)"
            >
              <template #icon><span class="material-icons text-base">delete</span></template>
            </DestructiveButton>
          </template>
        </li>
      </ul>
      <NeoButton variant="secondary" full-width :disabled="busy" @click="add">
        <template #icon><span class="material-icons">add</span></template>
        {{ t("settings.account.addPasskey") }}
      </NeoButton>
    </div>

    <!-- Connected apps -->
    <div>
      <h4 class="text-base font-bold text-black dark:text-white mb-2">
        {{ t("settings.account.connectedApps") }}
      </h4>
      <p v-if="apps.length === 0" class="text-sm text-black dark:text-white opacity-70">
        {{ t("settings.account.connectedAppsEmpty") }}
      </p>
      <ul v-else class="flex flex-col gap-2" data-testid="connected-apps">
        <li
          v-for="app in apps"
          :key="app.id"
          class="border-3 border-nb-border rounded-lg p-3 bg-white dark:bg-zinc-900 flex items-center gap-3"
        >
          <span class="material-icons text-purple-500">link</span>
          <div class="flex-1 min-w-0">
            <div class="font-bold text-black dark:text-white truncate">{{ app.name }}</div>
            <div class="text-xs text-black dark:text-white opacity-60">
              {{
                app.lastUsed
                  ? t("settings.account.connectedLastUsed", { date: d(new Date(app.lastUsed), "short") })
                  : t("settings.account.neverUsed")
              }}
            </div>
          </div>
          <DestructiveButton
            :confirm-text="t('settings.account.disconnect')"
            size="sm"
            @confirm="disconnect(app.id)"
          />
        </li>
      </ul>
    </div>

    <!-- Recovery codes -->
    <div>
      <h4 class="text-base font-bold text-black dark:text-white mb-1">
        {{ t("settings.account.recoveryCodes") }}
      </h4>
      <p v-if="me" class="text-sm text-black dark:text-white opacity-70 mb-3">
        {{ t("settings.account.recoveryCodesLeft", { count: me.recoveryCodesLeft }) }}
      </p>
      <RecoveryCodes v-if="freshCodes && me" :codes="freshCodes" :username="me.username" class="mb-3" />
      <DestructiveButton
        v-else
        :confirm-text="t('settings.account.regenerate')"
        variant="secondary"
        full-width
        @confirm="regenerate"
      >
        <template #icon><span class="material-icons">autorenew</span></template>
      </DestructiveButton>
    </div>

    <div class="flex flex-col gap-3">
      <NeoButton variant="secondary" full-width @click="signOut">
        <template #icon><span class="material-icons">logout</span></template>
        {{ t("settings.account.signOut") }}
      </NeoButton>
    </div>

    <!-- Delete account -->
    <div class="border-t-2 border-nb-border pt-4">
      <h4 class="text-base font-bold text-red-600 mb-1">{{ t("settings.account.deleteAccount") }}</h4>
      <p class="text-sm text-black dark:text-white opacity-70 mb-3">
        {{ t("settings.account.deleteAccountExplain") }}
      </p>
      <input
        v-model.trim="confirmUsername"
        class="w-full border-3 border-nb-border rounded-lg px-3 py-2 mb-3 text-sm bg-white dark:bg-zinc-900 text-black dark:text-white"
        :placeholder="t('settings.account.deleteAccountConfirm')"
        autocapitalize="none"
        spellcheck="false"
      />
      <DestructiveButton
        :confirm-text="t('settings.account.deleteAccount')"
        full-width
        :disabled="!me || confirmUsername !== me.username"
        @confirm="deleteEverything"
      >
        <template #icon><span class="material-icons">delete_forever</span></template>
      </DestructiveButton>
    </div>
  </NeoPanel>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import NeoButton from "@/components/NeoButton.vue";
import NeoPanel from "@/components/NeoPanel.vue";
import DestructiveButton from "@/components/DestructiveButton.vue";
import RecoveryCodes from "@/components/RecoveryCodes.vue";
import type { ConnectedApp, Passkey } from "@/api";
import { disconnectApp, listConnectedApps } from "@/api/oauth";
import * as auth from "@/api/auth";
import { useAuth } from "@/composables/useAuth";
import { useToast } from "@/composables/useToast";
import { describeError } from "@/views/auth/authFlow";

const { t, d } = useI18n();
const router = useRouter();
const { me, load, clear } = useAuth();
const { showError } = useToast();

const passkeys = ref<Passkey[]>([]);
const apps = ref<ConnectedApp[]>([]);
const busy = ref(false);
const renaming = ref<string | null>(null);
const newName = ref("");
const freshCodes = ref<string[] | null>(null);
const confirmUsername = ref("");

async function refresh(): Promise<void> {
  const [list, connected] = await Promise.all([auth.listPasskeys(), listConnectedApps(), load(true)]);
  passkeys.value = list;
  apps.value = connected;
}

async function disconnect(id: string): Promise<void> {
  try {
    await disconnectApp(id);
    await refresh();
  } catch (e) {
    showError(describeError(e, t));
  }
}

onMounted(() => refresh().catch((e) => showError(describeError(e, t))));

async function add(): Promise<void> {
  busy.value = true;
  try {
    await auth.addPasskey();
    await refresh();
  } catch (e) {
    showError(describeError(e, t));
  } finally {
    busy.value = false;
  }
}

function startRename(passkey: Passkey): void {
  renaming.value = passkey.id;
  newName.value = passkey.name;
}

async function saveName(passkey: Passkey): Promise<void> {
  if (!newName.value) return;
  try {
    await auth.renamePasskey(passkey.id, newName.value);
    renaming.value = null;
    await refresh();
  } catch (e) {
    showError(describeError(e, t));
  }
}

async function remove(passkey: Passkey): Promise<void> {
  try {
    await auth.deletePasskey(passkey.id);
    await refresh();
  } catch (e) {
    showError(describeError(e, t));
  }
}

async function regenerate(): Promise<void> {
  try {
    freshCodes.value = await auth.regenerateRecoveryCodes();
    await load(true);
  } catch (e) {
    showError(describeError(e, t));
  }
}

async function signOut(): Promise<void> {
  try {
    await auth.logout();
  } finally {
    clear();
    await router.replace({ name: "welcome" });
  }
}

async function deleteEverything(): Promise<void> {
  try {
    await auth.deleteAccount();
    clear();
    await router.replace({ name: "welcome" });
  } catch (e) {
    showError(describeError(e, t));
  }
}
</script>
