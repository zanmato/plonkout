/**
 * Signing in and account management. Every passkey ceremony is two calls: the
 * server hands out options, the browser asks the authenticator, and the result
 * goes back to the matching finish operation.
 */
import * as sdk from "./gen";
import type { CeremonyStart, Me, Passkey, PowChallenge, SignedIn } from "./gen/types.gen";
import { unwrap } from "./index";

async function createPasskey(start: CeremonyStart): Promise<unknown> {
  const options = PublicKeyCredential.parseCreationOptionsFromJSON(
    start.options as PublicKeyCredentialCreationOptionsJSON,
  );
  const credential = await navigator.credentials.create({ publicKey: options });
  if (!(credential instanceof PublicKeyCredential)) {
    throw new Error("No passkey was created");
  }
  return credential.toJSON();
}

async function usePasskey(start: CeremonyStart): Promise<unknown> {
  const options = PublicKeyCredential.parseRequestOptionsFromJSON(
    start.options as PublicKeyCredentialRequestOptionsJSON,
  );
  const credential = await navigator.credentials.get({ publicKey: options });
  if (!(credential instanceof PublicKeyCredential)) {
    throw new Error("No passkey was chosen");
  }
  return credential.toJSON();
}

/** Whether this browser can use passkeys at all. */
export function passkeysSupported(): boolean {
  return (
    typeof window !== "undefined" &&
    typeof window.PublicKeyCredential !== "undefined" &&
    typeof PublicKeyCredential.parseCreationOptionsFromJSON === "function"
  );
}

/** A name that helps tell passkeys apart later, guessed from the device. */
export function guessPasskeyName(userAgent = navigator.userAgent): string {
  if (/iPhone/.test(userAgent)) return "iPhone";
  if (/iPad/.test(userAgent)) return "iPad";
  if (/Android/.test(userAgent)) return "Android";
  if (/Macintosh|Mac OS X/.test(userAgent)) return "Mac";
  if (/Windows/.test(userAgent)) return "Windows";
  if (/Linux/.test(userAgent)) return "Linux";
  return "Passkey";
}

export function getSignupChallenge(): Promise<PowChallenge> {
  return unwrap(sdk.getSignupChallenge());
}

export interface PowSolution {
  challenge: string;
  number: number;
  salt: string;
  signature: string;
}

/** Creates the account and its first passkey. Answers with the recovery codes. */
export async function signup(username: string, pow: PowSolution, website = ""): Promise<SignedIn> {
  const start = await unwrap(sdk.beginSignup({ body: { username, pow, website } }));
  const credential = await createPasskey(start);
  return unwrap(
    sdk.finishSignup({
      body: { ceremony: start.ceremony, credential, passkeyName: guessPasskeyName() },
    }),
  );
}

export async function login(): Promise<SignedIn> {
  const start = await unwrap(sdk.beginLogin());
  const credential = await usePasskey(start);
  return unwrap(sdk.finishLogin({ body: { ceremony: start.ceremony, credential } }));
}

/** Registers a new passkey with a recovery code and signs in. */
export async function recover(username: string, recoveryCode: string): Promise<SignedIn> {
  const start = await unwrap(sdk.beginRecovery({ body: { username, recoveryCode } }));
  const credential = await createPasskey(start);
  return unwrap(
    sdk.finishRecovery({
      body: { ceremony: start.ceremony, credential, passkeyName: guessPasskeyName() },
    }),
  );
}

export async function logout(): Promise<void> {
  await unwrap(sdk.logout());
}

export function getMe(): Promise<Me> {
  return unwrap(sdk.getMe());
}

export function listPasskeys(): Promise<Passkey[]> {
  return unwrap(sdk.listPasskeys());
}

export async function addPasskey(): Promise<Passkey> {
  const start = await unwrap(sdk.beginAddPasskey());
  const credential = await createPasskey(start);
  return unwrap(
    sdk.finishAddPasskey({
      body: { ceremony: start.ceremony, credential, passkeyName: guessPasskeyName() },
    }),
  );
}

export async function renamePasskey(id: string, name: string): Promise<void> {
  await unwrap(sdk.renamePasskey({ path: { id }, body: { name } }));
}

export async function deletePasskey(id: string): Promise<void> {
  await unwrap(sdk.deletePasskey({ path: { id } }));
}

export function regenerateRecoveryCodes(): Promise<string[]> {
  return unwrap(sdk.regenerateRecoveryCodes());
}

export async function deleteAccount(): Promise<void> {
  await unwrap(sdk.deleteAccount());
}
