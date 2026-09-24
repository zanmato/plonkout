import { http, HttpResponse } from "msw";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "@/api";
import { guessPasskeyName, login, signup } from "@/api/auth";
import { API, useApiServer } from "../helpers/msw";

const server = useApiServer();

/**
 * Stands in for the browser's passkey support. navigator.credentials answers
 * with a credential whose toJSON is what the server should receive.
 */
function fakePasskeys() {
  class FakeCredential {
    constructor(private readonly json: unknown) {}
    toJSON() {
      return this.json;
    }
  }
  const parseCreation = vi.fn((options: unknown) => ({ parsed: "creation", options }));
  const parseRequest = vi.fn((options: unknown) => ({ parsed: "request", options }));
  vi.stubGlobal(
    "PublicKeyCredential",
    Object.assign(FakeCredential, {
      parseCreationOptionsFromJSON: parseCreation,
      parseRequestOptionsFromJSON: parseRequest,
    }),
  );
  const create = vi.fn(async () => new FakeCredential({ id: "new-credential" }));
  const get = vi.fn(async () => new FakeCredential({ id: "existing-credential" }));
  vi.stubGlobal("navigator", { ...navigator, credentials: { create, get }, userAgent: "iPhone" });
  return { create, get, parseCreation, parseRequest };
}

describe("passkey sign up and sign in", () => {
  let passkeys: ReturnType<typeof fakePasskeys>;

  beforeEach(() => {
    passkeys = fakePasskeys();
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("creates a passkey from the server's options and finishes the signup", async () => {
    const pow = { challenge: "c", number: 7, salt: "s", signature: "sig" };
    let beginBody: unknown;
    let finishBody: unknown;
    server.use(
      http.post(`${API}/api/auth/signup/begin`, async ({ request }) => {
        beginBody = await request.json();
        return HttpResponse.json({ ceremony: "cer-1", options: { challenge: "abc" } });
      }),
      http.post(`${API}/api/auth/signup/finish`, async ({ request }) => {
        finishBody = await request.json();
        return HttpResponse.json({ user: { id: "u1", username: "andreas", created: "2026-09-24T00:00:00Z" }, recoveryCodes: ["a", "b"] });
      }),
    );

    const result = await signup("andreas", pow);

    expect(beginBody).toEqual({ username: "andreas", pow, website: "" });
    expect(passkeys.parseCreation).toHaveBeenCalledWith({ challenge: "abc" });
    expect(passkeys.create).toHaveBeenCalledWith({ publicKey: { parsed: "creation", options: { challenge: "abc" } } });
    expect(finishBody).toEqual({ ceremony: "cer-1", credential: { id: "new-credential" }, passkeyName: "iPhone" });
    expect(result.recoveryCodes).toEqual(["a", "b"]);
  });

  it("signs in with whichever passkey the person picks", async () => {
    let finishBody: unknown;
    server.use(
      http.post(`${API}/api/auth/login/begin`, () => HttpResponse.json({ ceremony: "cer-2", options: { challenge: "xyz" } })),
      http.post(`${API}/api/auth/login/finish`, async ({ request }) => {
        finishBody = await request.json();
        return HttpResponse.json({ user: { id: "u1", username: "andreas", created: "2026-09-24T00:00:00Z" } });
      }),
    );

    const result = await login();

    expect(passkeys.get).toHaveBeenCalledOnce();
    expect(finishBody).toEqual({ ceremony: "cer-2", credential: { id: "existing-credential" } });
    expect(result.user.username).toBe("andreas");
  });

  it("turns a problem document into an ApiError with its code", async () => {
    server.use(
      http.post(`${API}/api/auth/signup/begin`, () =>
        HttpResponse.json(
          { type: "about:blank", title: "Conflict", status: 409, code: "username_taken", detail: "that username is taken" },
          { status: 409, headers: { "Content-Type": "application/problem+json" } },
        ),
      ),
    );

    const error = await signup("taken", { challenge: "c", number: 1, salt: "s", signature: "sig" }).catch((e) => e);

    expect(error).toBeInstanceOf(ApiError);
    expect(error.status).toBe(409);
    expect(error.code).toBe("username_taken");
    expect(passkeys.create).not.toHaveBeenCalled();
  });
});

describe("guessPasskeyName", () => {
  it.each([
    ["Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X)", "iPhone"],
    ["Mozilla/5.0 (Linux; Android 15; Pixel 9)", "Android"],
    ["Mozilla/5.0 (Macintosh; Intel Mac OS X 15_0)", "Mac"],
    ["Mozilla/5.0 (Windows NT 10.0; Win64; x64)", "Windows"],
    ["curl/8", "Passkey"],
  ])("%s is %s", (userAgent, name) => {
    expect(guessPasskeyName(userAgent)).toBe(name);
  });
});
