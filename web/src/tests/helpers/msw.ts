import { setupServer } from "msw/node";
import { API, backend } from "../mocks/backend";

export { API };

/**
 * The one MSW server of a test run, serving the fake backend. setup.ts starts
 * it for every test file and resets the backend and any per test handlers
 * after each test. Anything unhandled fails the test.
 */
export const server = setupServer(...backend.handlers);

/**
 * The shared MSW server, for test files that add handlers of their own with
 * server.use. Those take precedence over the fake backend for the test.
 */
export function useApiServer() {
  return server;
}
