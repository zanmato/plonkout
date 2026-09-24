package mcpserver_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zanmato/plonkout/server/internal/platform/apitest"
)

// bearer adds the access token to every request, like a signed in client.
type bearer struct {
	token string
	next  http.RoundTripper
}

func (b bearer) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+b.token)
	return b.next.RoundTrip(req)
}

func connect(t *testing.T, ts *httptest.Server, token string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint:   ts.URL + "/mcp",
		HTTPClient: &http.Client{Transport: bearer{token: token, next: http.DefaultTransport}},
		MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func call(t *testing.T, s *mcp.ClientSession, name string, args map[string]any) (map[string]any, bool) {
	t.Helper()
	result, err := s.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		var list []any
		if json.Unmarshal([]byte(text), &list) == nil {
			return map[string]any{"items": list}, result.IsError
		}
		t.Fatalf("%s answered %q", name, text)
	}
	return out, result.IsError
}

func TestUnauthenticatedClientsAreToldWhereToSignIn(t *testing.T) {
	h := apitest.New(t)
	ts := httptest.NewServer(h.Handler)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	challenge := resp.Header.Get("WWW-Authenticate")
	if resp.StatusCode != http.StatusUnauthorized || !strings.Contains(challenge, `resource_metadata="`+apitest.Origin+"/.well-known/oauth-protected-resource/mcp") {
		t.Fatalf("expected a 401 pointing at the resource metadata, got %d %q", resp.StatusCode, challenge)
	}
}

func TestToolsAndPrompts(t *testing.T) {
	h := apitest.New(t)
	ts := httptest.NewServer(h.Handler)
	defer ts.Close()
	s := connect(t, ts, h.NewAccessToken(h.NewUser()))

	tools, err := s.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
		schema, _ := json.Marshal(tool.InputSchema)
		if !strings.Contains(string(schema), `"type":"object"`) {
			t.Errorf("%s has no object input schema: %s", tool.Name, schema)
		}
	}
	want := []string{
		"add_sessions", "create_plan", "get_context", "get_exercise_stats", "get_plan", "get_workout_history",
		"list_exercises", "list_plans", "reorder_sessions", "set_session_status", "update_plan", "update_session",
	}
	if !slices.Equal(names, want) {
		t.Fatalf("tools %v, want %v", names, want)
	}

	prompts, err := s.ListPrompts(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(prompts.Prompts) != 2 {
		t.Fatalf("expected 2 prompts, got %d", len(prompts.Prompts))
	}
	design, err := s.GetPrompt(t.Context(), &mcp.GetPromptParams{Name: "design_plan", Arguments: map[string]string{"goal": "180 kg bench"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(design.Messages[0].Content.(*mcp.TextContent).Text, "180 kg bench") {
		t.Fatal("the prompt should carry the goal")
	}
}

func TestPlanningThroughTools(t *testing.T) {
	h := apitest.New(t)
	user := h.NewUser()
	for _, name := range []string{"Bench Press", "Side Pressure"} {
		h.Expect(h.Do(http.MethodPost, "/exercises", map[string]any{
			"name": name, "muscleGroup": "Chest", "type": "strength", "displayType": "reps", "singleArm": name == "Side Pressure",
		}, user.Session), http.StatusCreated)
	}
	ts := httptest.NewServer(h.Handler)
	defer ts.Close()
	s := connect(t, ts, h.NewAccessToken(user))

	ctx, isErr := call(t, s, "get_context", nil)
	if isErr || ctx["weightUnit"] != "kg" || ctx["dominantArm"] != "right" {
		t.Fatalf("unexpected context %v", ctx)
	}

	session := func(label string, weight float64) map[string]any {
		return map[string]any{
			"label": label, "day": "A", "intensity": "heavy",
			"exercises": []any{
				map[string]any{"exercise": "bench press", "targets": []any{
					map[string]any{"setType": "Top set", "sets": 1, "reps": 5, "weight": weight, "rpeMin": 8, "rpeMax": 9},
				}},
				map[string]any{"exercise": "Side Pressure", "offArmPercent": 82.5, "notes": "Elbow pinned", "targets": []any{
					map[string]any{"setType": "Top set", "sets": 1, "reps": 5, "weight": 21.25},
				}},
			},
		}
	}

	// A typo is refused with the name the model probably meant.
	_, isErr = call(t, s, "create_plan", map[string]any{"name": "Peak", "sessions": []any{
		map[string]any{"label": "W1 A", "exercises": []any{map[string]any{"exercise": "Bench Pres", "targets": []any{map[string]any{"setType": "Top set", "sets": 1}}}}},
	}})
	if !isErr {
		t.Fatal("an unknown exercise should be refused")
	}

	created, isErr := call(t, s, "create_plan", map[string]any{
		"name": "Peak", "goal": "Bench 180",
		"sessions": []any{session("W1 A", 145), session("W2 A", 150)},
	})
	if isErr {
		t.Fatalf("create_plan failed: %v", created)
	}
	planID := created["id"].(string)
	sessions := created["sessions"].([]any)
	first := sessions[0].(map[string]any)["id"].(string)

	updated, isErr := call(t, s, "update_session", map[string]any{"id": first, "notes": "Pause the first rep"})
	if isErr || updated["notes"] != "Pause the first rep" {
		t.Fatalf("update_session failed: %v", updated)
	}

	skipped, isErr := call(t, s, "set_session_status", map[string]any{"id": first, "status": "skipped", "reason": "sore elbow"})
	if isErr || skipped["session"].(map[string]any)["status"] != "skipped" {
		t.Fatalf("set_session_status failed: %v", skipped)
	}

	plan, isErr := call(t, s, "get_plan", map[string]any{"id": planID, "actuals": true})
	if isErr || plan["name"] != "Peak" {
		t.Fatalf("get_plan failed: %v", plan)
	}

	listed, isErr := call(t, s, "list_plans", map[string]any{"status": "active"})
	if isErr || len(listed["items"].([]any)) != 1 {
		t.Fatalf("list_plans failed: %v", listed)
	}

	_, isErr = call(t, s, "get_plan", map[string]any{"bogus": 1})
	if !isErr {
		t.Fatal("a missing id should be refused")
	}
}

func TestToolsActAsTheirUser(t *testing.T) {
	h := apitest.New(t)
	alice, bob := h.NewUser(), h.NewUser()
	h.Expect(h.Do(http.MethodPost, "/exercises", map[string]any{
		"name": "Bench Press", "muscleGroup": "Chest", "type": "strength", "displayType": "reps", "singleArm": false,
	}, alice.Session), http.StatusCreated)
	var plan struct{ ID string }
	h.Expect(h.Do(http.MethodPost, "/plans", map[string]any{"name": "Alice's", "sessions": []any{}}, alice.Session), http.StatusCreated).Decode(t, &plan)

	ts := httptest.NewServer(h.Handler)
	defer ts.Close()
	s := connect(t, ts, h.NewAccessToken(bob))

	if _, isErr := call(t, s, "get_plan", map[string]any{"id": plan.ID}); !isErr {
		t.Fatal("bob's assistant must not see alice's plan")
	}
	exercises, _ := call(t, s, "list_exercises", nil)
	if len(exercises["items"].([]any)) != 0 {
		t.Fatal("bob's assistant must not see alice's exercises")
	}
}
