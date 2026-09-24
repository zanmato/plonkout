/**
 * The only place the app talks to the network. Views and composables call the
 * functions exported from src/api, never fetch or the generated SDK directly,
 * so auth handling and error shapes live in one spot.
 */
import { client } from "./gen/client.gen";
import type { Problem } from "./gen/types.gen";

client.setConfig({
  // The API is served from the app's own origin, the session cookie rides along.
  credentials: "same-origin",
});

/** A failed API call, carrying the server's problem document. */
export class ApiError extends Error {
  readonly status: number;
  /** The stable machine readable reason, e.g. "username_taken". */
  readonly code: string;

  constructor(status: number, problem: Partial<Problem> | undefined) {
    super(problem?.detail || problem?.title || `Request failed with ${status}`);
    this.name = "ApiError";
    this.status = status;
    this.code = problem?.code ?? "request_failed";
  }
}

let unauthorizedHandler: (() => void) | null = null;

/** Called whenever the server says the session is gone. */
export function onUnauthorized(handler: () => void): void {
  unauthorizedHandler = handler;
}

client.interceptors.response.use((response) => {
  if (response.status === 401) unauthorizedHandler?.();
  return response;
});

interface Result<T> {
  data?: T;
  error?: unknown;
  response?: Response;
}

/** Resolves to the data of a generated SDK call, or throws an ApiError. */
export async function unwrap<T>(call: Promise<Result<T>>): Promise<T> {
  const { data, error, response } = await call;
  if (error !== undefined || !response?.ok) {
    throw new ApiError(response?.status ?? 0, error as Partial<Problem> | undefined);
  }
  return data as T;
}

export { getHealth } from "./gen";
export type * from "./gen/types.gen";
