import { computed, ref } from "vue";
import { ApiError, type Me } from "@/api";
import { getMe } from "@/api/auth";
import { clearSettingsCache } from "@/api/data";

// Module level, so every component shares one signed in user.
const me = ref<Me | null>(null);
const loaded = ref(false);

/** The signed in user, loaded once and shared app wide. */
export function useAuth() {
  /** Loads the user unless already known. Resolves to null when signed out. */
  async function load(force = false): Promise<Me | null> {
    if (loaded.value && !force) return me.value;
    // A different user may be signing in, whose settings are not these.
    if (force) clearSettingsCache();
    try {
      me.value = await getMe();
    } catch (error) {
      if (!(error instanceof ApiError) || error.status !== 401) throw error;
      me.value = null;
    }
    loaded.value = true;
    return me.value;
  }

  function clear(): void {
    clearSettingsCache();
    me.value = null;
    loaded.value = true;
  }

  return {
    me,
    signedIn: computed(() => me.value !== null),
    load,
    clear,
  };
}
