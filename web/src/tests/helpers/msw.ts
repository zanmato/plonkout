import { setupServer } from "msw/node";
import { afterAll, afterEach, beforeAll } from "vitest";
import { client } from "@/api/gen/client.gen";

/** Origin the API is served from in tests. Node's fetch needs absolute URLs. */
export const API = "http://localhost";

/**
 * An MSW server for a test file. Handlers are added per test with
 * server.use, and anything unhandled fails the test.
 */
export function useApiServer() {
  const server = setupServer();
  beforeAll(() => {
    client.setConfig({ baseUrl: API });
    server.listen({ onUnhandledRequest: "error" });
  });
  afterEach(() => server.resetHandlers());
  afterAll(() => server.close());
  return server;
}
