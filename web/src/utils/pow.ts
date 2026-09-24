import type { PowChallenge } from "@/api";
import type { PowSolution } from "@/api/auth";

/** Hex SHA-256 of a string. */
export async function sha256Hex(input: string): Promise<string> {
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(input));
  return Array.from(new Uint8Array(digest), (b) => b.toString(16).padStart(2, "0")).join("");
}

/**
 * Finds the number the server hid in the challenge, by trying them in order.
 * It takes a phone about a second, which is the point: cheap for a person,
 * expensive for a script signing up thousands of accounts.
 */
export async function solvePow(challenge: PowChallenge): Promise<PowSolution> {
  for (let number = 0; number <= challenge.maxNumber; number++) {
    if ((await sha256Hex(challenge.salt + number)) === challenge.challenge) {
      return {
        challenge: challenge.challenge,
        number,
        salt: challenge.salt,
        signature: challenge.signature,
      };
    }
  }
  throw new Error("The signup challenge has no solution");
}

/** Solves the challenge in a worker, keeping the page responsive. */
export function solvePowInWorker(challenge: PowChallenge): Promise<PowSolution> {
  return new Promise((resolve, reject) => {
    const worker = new Worker(new URL("../workers/pow.ts", import.meta.url), { type: "module" });
    worker.onmessage = (event: MessageEvent<{ solution?: PowSolution; error?: string }>) => {
      worker.terminate();
      if (event.data.solution) resolve(event.data.solution);
      else reject(new Error(event.data.error ?? "The signup challenge could not be solved"));
    };
    worker.onerror = (event) => {
      worker.terminate();
      reject(new Error(event.message));
    };
    worker.postMessage(challenge);
  });
}
