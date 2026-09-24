import type { RouteLocationNormalizedLoaded, Router } from "vue-router";
import { ApiError } from "@/api";

type Translate = (key: string) => string;

/**
 * Where to go once signed in. An OAuth authorization waiting for the user is a
 * full page load on the server, anything else stays in the app. Only same
 * origin paths are followed, so a crafted link cannot bounce elsewhere.
 */
export async function returnTo(router: Router, route: RouteLocationNormalizedLoaded): Promise<void> {
  const target = typeof route.query.returnTo === "string" ? route.query.returnTo : "";
  if (target.startsWith("/oauth/authorize?")) {
    window.location.assign(target);
    return;
  }
  if (target.startsWith("/") && !target.startsWith("//")) {
    await router.replace(target);
    return;
  }
  await router.replace({ name: "log" });
}

/** A message for a failed sign in, signup or recovery. */
export function describeError(error: unknown, t: Translate): string {
  // The person closed the passkey prompt or it timed out.
  if (error instanceof DOMException && (error.name === "NotAllowedError" || error.name === "AbortError")) {
    return t("auth.errors.cancelled");
  }
  if (error instanceof DOMException && error.name === "InvalidStateError") {
    return t("auth.errors.passkeyExists");
  }
  if (error instanceof ApiError) {
    const key = `auth.errors.${error.code}`;
    const message = t(key);
    return message === key ? error.message : message;
  }
  return t("auth.errors.generic");
}
