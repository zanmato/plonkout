import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { createClient } from "@hey-api/openapi-ts";

/**
 * Generates the typed API client from the server's committed OpenAPI document.
 * The output is committed too, and `make check` fails when either is stale.
 */
const here = dirname(fileURLToPath(import.meta.url));

await createClient({
  input: resolve(here, "../../server/api/openapi.json"),
  output: { path: resolve(here, "../src/api/gen") },
  plugins: [
    // baseUrl false: the client talks to its own origin, so paths stay relative.
    { name: "@hey-api/client-fetch", baseUrl: false },
    { name: "@hey-api/typescript" },
    { name: "@hey-api/sdk" },
  ],
});
