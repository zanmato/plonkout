/**
 * Training plans. Plans are written by an assistant over MCP, the app reads
 * them, starts their sessions and skips or restores them.
 */
import * as sdk from "./gen";
import type {
  Comparison,
  Id,
  Plan,
  PlannedSession,
  QueueEntry,
} from "@/types/domain";
import { ApiError, unwrap } from "./index";

/** The sessions still to do in active plans, in order. */
export function getQueue(): Promise<QueueEntry[]> {
  return unwrap(sdk.getQueue());
}

/** Plans, newest first, optionally of one status. */
export function getPlans(status?: Plan["status"]): Promise<Plan[]> {
  return unwrap(sdk.listPlans(status ? { query: { status } } : {}));
}

/**
 * One plan. With actuals, every session that has a workout carries its
 * planned versus done comparison.
 */
export async function getPlan(id: Id, actuals = false): Promise<Plan | undefined> {
  try {
    return await unwrap(sdk.getPlan({ path: { id }, query: actuals ? { actuals } : {} }));
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return undefined;
    throw error;
  }
}

export async function getPlannedSession(id: Id): Promise<PlannedSession | undefined> {
  try {
    return await unwrap(sdk.getPlannedSession({ path: { id } }));
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return undefined;
    throw error;
  }
}

/**
 * Starts a session and answers with the id of its workout. A session that was
 * started before answers with the workout it already has.
 */
export async function startSession(id: Id): Promise<Id> {
  return (await unwrap(sdk.startSession({ path: { id } }))).workoutId;
}

/** Skips a pending session. The reason is kept for the assistant to read. */
export async function skipSession(id: Id, reason = ""): Promise<PlannedSession | undefined> {
  const body = reason.trim() ? { status: "skipped" as const, reason: reason.trim() } : { status: "skipped" as const };
  return (await unwrap(sdk.setSessionStatus({ path: { id }, body }))).session;
}

/** Puts a skipped session back in the queue. */
export async function restoreSession(id: Id): Promise<PlannedSession | undefined> {
  return (await unwrap(sdk.setSessionStatus({ path: { id }, body: { status: "pending" } }))).session;
}

export function getSessionComparison(id: Id): Promise<Comparison> {
  return unwrap(sdk.getSessionComparison({ path: { id } }));
}
