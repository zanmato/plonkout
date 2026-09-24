/**
 * The consent step of an MCP client signing in, and the list of apps that
 * have been let in.
 */
import * as sdk from "./gen";
import type { ConnectedApp, ConsentPrompt } from "./gen/types.gen";
import { unwrap } from "./index";

/** The /authorize parameters, carried from the server to the consent page. */
export type AuthorizationParams = {
  client_id: string;
  redirect_uri: string;
  response_type: string;
  code_challenge: string;
  code_challenge_method: string;
  scope?: string;
  state?: string;
  resource?: string;
};

const keys = [
  "client_id",
  "redirect_uri",
  "response_type",
  "code_challenge",
  "code_challenge_method",
  "scope",
  "state",
  "resource",
] as const;

/** Picks the authorization parameters out of a route query. */
export function authorizationParams(query: Record<string, unknown>): AuthorizationParams {
  const out: Record<string, string> = {};
  for (const key of keys) {
    const value = query[key];
    if (typeof value === "string" && value !== "") out[key] = value;
  }
  return out as AuthorizationParams;
}

export function getConsentPrompt(params: AuthorizationParams): Promise<ConsentPrompt> {
  return unwrap(sdk.getConsentPrompt({ query: params }));
}

/** Records the decision and answers with where to send the browser. */
export async function decideConsent(params: AuthorizationParams, approve: boolean): Promise<string> {
  const { redirectTo } = await unwrap(sdk.decideConsent({ body: { params, approve } }));
  return redirectTo;
}

export function listConnectedApps(): Promise<ConnectedApp[]> {
  return unwrap(sdk.listConnectedApps());
}

export async function disconnectApp(id: string): Promise<void> {
  await unwrap(sdk.disconnectApp({ path: { id } }));
}
