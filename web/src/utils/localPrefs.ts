/**
 * Preferences that belong to one device rather than the account: the theme
 * and the language. They are needed before anyone is signed in, and a phone
 * and a laptop may well want different ones.
 */
export type Theme = "light" | "dark" | "system";

const KEY = "plonkout.prefs";

interface Prefs {
  theme: Theme;
  locale: string;
}

const defaults: Prefs = { theme: "system", locale: "en" };

function read(): Prefs {
  try {
    const stored = JSON.parse(localStorage.getItem(KEY) ?? "{}") as Partial<Prefs>;
    return { ...defaults, ...stored };
  } catch {
    return { ...defaults };
  }
}

export function getLocalPref<K extends keyof Prefs>(key: K): Prefs[K] {
  return read()[key];
}

export function setLocalPref<K extends keyof Prefs>(key: K, value: Prefs[K]): void {
  try {
    localStorage.setItem(KEY, JSON.stringify({ ...read(), [key]: value }));
  } catch {
    // Storage can be unavailable, e.g. in a private window. The choice then
    // lasts until reload, which is the best that can be done.
  }
}
