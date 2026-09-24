import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { http, HttpResponse } from "msw";
import Authorize from "@/views/auth/Authorize.vue";
import { API, useApiServer } from "../helpers/msw";

const query = {
  client_id: "dcr_abc",
  redirect_uri: "https://claude.ai/api/mcp/auth_callback",
  response_type: "code",
  code_challenge: "challenge",
  code_challenge_method: "S256",
  state: "xyz",
  scope: "training offline_access",
  // Not an authorization parameter, and must not be sent on.
  utm: "ignored",
};

vi.mock("vue-router", () => ({
  useRoute: () => ({ params: {}, query, path: "/authorize", name: "authorize" }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
}));

const server = useApiServer();

describe("Authorize", () => {
  const assign = vi.fn();

  beforeEach(() => {
    vi.stubGlobal("location", { ...window.location, assign });
  });
  afterEach(() => {
    assign.mockReset();
    vi.unstubAllGlobals();
  });

  function promptHandler(seen: { query?: URLSearchParams }) {
    return http.get(`${API}/api/oauth/consent`, ({ request }) => {
      seen.query = new URL(request.url).searchParams;
      return HttpResponse.json({
        clientName: "Claude",
        clientId: "dcr_abc",
        redirectHost: "claude.ai",
        loopback: false,
        scopes: ["training", "offline_access"],
      });
    });
  }

  it("shows who is asking and where the answer goes", async () => {
    const seen: { query?: URLSearchParams } = {};
    server.use(promptHandler(seen));

    const wrapper = mount(Authorize);
    await flushPromises();

    expect(wrapper.find("[data-testid=consent-prompt]").text()).toContain("Claude");
    expect(wrapper.text()).toContain("claude.ai");
    expect(seen.query?.get("client_id")).toBe("dcr_abc");
    expect(seen.query?.has("utm")).toBe(false);
  });

  it("sends the decision and follows the redirect back to the app", async () => {
    let body: unknown;
    server.use(
      promptHandler({}),
      http.post(`${API}/api/oauth/consent`, async ({ request }) => {
        body = await request.json();
        return HttpResponse.json({ redirectTo: "https://claude.ai/api/mcp/auth_callback?code=c&state=xyz" });
      }),
    );

    const wrapper = mount(Authorize);
    await flushPromises();
    const allow = wrapper.findAll("button").find((b) => b.text() === "Allow");
    await allow!.trigger("click");
    await flushPromises();

    const { utm: _utm, ...params } = query;
    expect(body).toEqual({ params, approve: true });
    expect(assign).toHaveBeenCalledWith("https://claude.ai/api/mcp/auth_callback?code=c&state=xyz");
  });

  it("says so when the request is not valid", async () => {
    server.use(
      http.get(`${API}/api/oauth/consent`, () =>
        HttpResponse.json(
          { title: "Unprocessable Entity", status: 422, code: "invalid_request", detail: "redirect_uri is not registered" },
          { status: 422, headers: { "Content-Type": "application/problem+json" } },
        ),
      ),
    );

    const wrapper = mount(Authorize);
    await flushPromises();

    expect(wrapper.find("[data-testid=consent-error]").exists()).toBe(true);
    expect(wrapper.find("[data-testid=consent-prompt]").exists()).toBe(false);
  });
});
