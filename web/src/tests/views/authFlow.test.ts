import { afterEach, describe, expect, it, vi } from "vitest";
import type { RouteLocationNormalizedLoaded, Router } from "vue-router";
import { ApiError } from "@/api";
import { describeError, returnTo } from "@/views/auth/authFlow";

function routeWith(returnToValue?: string): RouteLocationNormalizedLoaded {
  return { query: returnToValue === undefined ? {} : { returnTo: returnToValue } } as unknown as RouteLocationNormalizedLoaded;
}

describe("returnTo", () => {
  const replace = vi.fn();
  const router = { replace } as unknown as Router;

  afterEach(() => {
    replace.mockReset();
    vi.unstubAllGlobals();
  });

  it("goes back into the app", async () => {
    await returnTo(router, routeWith("/workout/12"));
    expect(replace).toHaveBeenCalledWith("/workout/12");
  });

  it("continues a waiting OAuth authorization with a full page load", async () => {
    const assign = vi.fn();
    vi.stubGlobal("location", { ...window.location, assign });
    await returnTo(router, routeWith("/oauth/authorize?client_id=x"));
    expect(assign).toHaveBeenCalledWith("/oauth/authorize?client_id=x");
    expect(replace).not.toHaveBeenCalled();
  });

  it.each(["https://evil.example/", "//evil.example/", "javascript:alert(1)", ""])(
    "ignores %s and goes to the log",
    async (target) => {
      await returnTo(router, routeWith(target));
      expect(replace).toHaveBeenCalledWith({ name: "log" });
    },
  );
});

describe("describeError", () => {
  const t = (key: string) => (key === "auth.errors.username_taken" ? "Taken!" : key);

  it("explains a closed passkey prompt", () => {
    expect(describeError(new DOMException("closed", "NotAllowedError"), t)).toBe("auth.errors.cancelled");
  });

  it("translates a known problem code", () => {
    expect(describeError(new ApiError(409, { code: "username_taken", detail: "x" }), t)).toBe("Taken!");
  });

  it("falls back to the server's detail for an unknown code", () => {
    expect(describeError(new ApiError(500, { code: "internal_error", detail: "boom" }), t)).toBe("boom");
  });
});
