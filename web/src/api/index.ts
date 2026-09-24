/**
 * The only place the app talks to the network. Views and composables call the
 * functions exported here, never fetch or the generated SDK directly, so auth
 * handling and error shapes live in one spot.
 */
import { client } from "./gen/client.gen";

client.setConfig({
  // The API is served from the app's own origin, the session cookie rides along.
  credentials: "same-origin",
});

export { getHealth } from "./gen";
export type * from "./gen/types.gen";
