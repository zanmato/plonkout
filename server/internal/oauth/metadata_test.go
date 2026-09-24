package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/zanmato/plonkout/server/internal/platform/config"
	"github.com/zanmato/plonkout/server/internal/platform/dbtest"
)

func TestPrivateAddressesAreRefused(t *testing.T) {
	for _, address := range []string{"127.0.0.1:443", "[::1]:443", "10.1.2.3:443", "192.168.1.1:443", "169.254.169.254:80", "0.0.0.0:443"} {
		if publicAddressesOnly("tcp", address, nil) == nil {
			t.Errorf("%s should be refused", address)
		}
	}
	if err := publicAddressesOnly("tcp", "160.79.104.10:443", nil); err != nil {
		t.Errorf("a public address should be allowed: %v", err)
	}

	// The real fetcher refuses a document served on loopback.
	ts := httptest.NewTLSServer(http.NotFoundHandler())
	defer ts.Close()
	if _, _, err := newMetadataFetcher().fetch(t.Context(), ts.URL+"/client.json"); err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("expected the loopback fetch to be refused, got %v", err)
	}
}

func TestClientMetadataDocuments(t *testing.T) {
	d := dbtest.New(t)
	var ts *httptest.Server
	doc := map[string]any{}
	ts = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "max-age=60")
		_ = json.NewEncoder(w).Encode(doc)
	}))
	defer ts.Close()
	clientID := ts.URL + "/oauth/claude-code-client-metadata"

	cfg := &config.Config{Server: config.Server{BaseURL: "http://localhost:5173"}}
	s := NewService(d.App, cfg)
	// The test server is on loopback with its own certificate, which the real
	// fetcher rightly refuses.
	s.metadata = &metadataFetcher{client: ts.Client()}

	query := url.Values{
		"client_id": {clientID}, "redirect_uri": {"http://localhost:61234/callback"}, "response_type": {"code"},
		"code_challenge": {"abc"}, "code_challenge_method": {"S256"},
	}

	doc["client_id"] = "https://someone.else/metadata"
	doc["redirect_uris"] = []string{"http://localhost/callback"}
	if _, err := s.Prepare(t.Context(), query); err == nil {
		t.Fatal("a document naming another client_id must be refused")
	}

	doc["client_id"] = clientID
	doc["client_name"] = "Claude Code"
	req, err := s.Prepare(t.Context(), query)
	if err != nil {
		t.Fatal(err)
	}
	if req.ClientName != "Claude Code" {
		t.Fatalf("unexpected client %+v", req)
	}

	// Cached for at least an hour, even though the document said a minute.
	row, err := s.q.GetClient(t.Context(), clientID)
	if err != nil {
		t.Fatal(err)
	}
	if row.CacheUntil == nil || time.Until(*row.CacheUntil) < 59*time.Minute {
		t.Fatalf("expected an hour of cache, got %v", row.CacheUntil)
	}
}
