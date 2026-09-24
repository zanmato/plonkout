package template_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/zanmato/plonkout/server/internal/platform/apitest"
	"github.com/zanmato/plonkout/server/internal/template"
)

func TestTemplatesAndSettings(t *testing.T) {
	h := apitest.New(t)
	u, other := h.NewUser(), h.NewUser()

	body := map[string]any{
		"name": "Arm day", "notes": "light",
		"exercises": []map[string]any{{
			"name": "Wrist Curl", "muscleGroup": "Forearm", "type": "strength", "displayType": "reps",
			"singleArm": true, "intensity": "light",
			"sets": []map[string]any{{"type": "regular", "weight": nil, "distance": nil, "reps": nil, "time": "", "rpe": nil, "arm": "right", "notes": ""}},
		}},
	}
	var created template.Template
	h.Expect(h.Do(http.MethodPost, "/templates", body, u.Session), http.StatusCreated).Decode(t, &created)
	if len(created.Exercises) != 1 || created.Exercises[0].Sets[0].Arm != "right" {
		t.Fatalf("unexpected template %+v", created)
	}

	body["name"] = "Arm day B"
	var updated template.Template
	h.Expect(h.Do(http.MethodPut, "/templates/"+created.ID.String(), body, u.Session), http.StatusOK).Decode(t, &updated)
	if updated.Name != "Arm day B" {
		t.Fatal("the template was not renamed")
	}
	h.Expect(h.Do(http.MethodGet, "/templates/"+created.ID.String(), nil, other.Session), http.StatusNotFound)
	h.Expect(h.Do(http.MethodDelete, "/templates/"+created.ID.String(), nil, u.Session), http.StatusNoContent)

	h.Expect(h.Do(http.MethodPut, "/settings/weightUnit", map[string]any{"value": "lbs"}, u.Session), http.StatusNoContent)
	h.Expect(h.Do(http.MethodPut, "/settings/weightUnit", map[string]any{"value": "kg"}, u.Session), http.StatusNoContent)
	h.Expect(h.Do(http.MethodPut, "/settings/bad%20key!", map[string]any{"value": 1}, u.Session), http.StatusUnprocessableEntity)

	var settings map[string]json.RawMessage
	h.Expect(h.Do(http.MethodGet, "/settings", nil, u.Session), http.StatusOK).Decode(t, &settings)
	if string(settings["weightUnit"]) != `"kg"` {
		t.Fatalf("unexpected settings %s", settings["weightUnit"])
	}
	// A fresh map, since decoding into the old one would merge into it.
	var otherSettings map[string]json.RawMessage
	h.Expect(h.Do(http.MethodGet, "/settings", nil, other.Session), http.StatusOK).Decode(t, &otherSettings)
	if len(otherSettings) != 0 {
		t.Fatal("another user's settings are visible")
	}
}
