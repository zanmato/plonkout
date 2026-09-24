/// <reference lib="webworker" />
import type { PowChallenge } from "@/api";
import { solvePow } from "@/utils/pow";

// Solving runs off the main thread so the signup form stays responsive.
self.onmessage = async (event: MessageEvent<PowChallenge>) => {
  try {
    self.postMessage({ solution: await solvePow(event.data) });
  } catch (error) {
    self.postMessage({ error: error instanceof Error ? error.message : String(error) });
  }
};
