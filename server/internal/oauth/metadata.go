package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Client ID Metadata Documents: a client names itself with an https URL and
// publishes its name and redirect URIs there, so it never has to register.
// Claude Code signs in this way.
const (
	metadataMaxBytes = 32 << 10
	metadataTimeout  = 5 * time.Second
	metadataMinCache = time.Hour
	metadataMaxCache = 24 * time.Hour
)

// clientMetadata is the part of a metadata document the server uses.
type clientMetadata struct {
	ClientID                string   `json:"client_id"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

type metadataFetcher struct {
	client *http.Client
}

// newMetadataFetcher fetches over a connection that refuses private,
// loopback and link local addresses after DNS resolution, so a client_id
// cannot turn this server into a probe of its own network.
func newMetadataFetcher() *metadataFetcher {
	dialer := &net.Dialer{Timeout: metadataTimeout, Control: publicAddressesOnly}
	transport := &http.Transport{
		DialContext:            dialer.DialContext,
		TLSHandshakeTimeout:    metadataTimeout,
		ResponseHeaderTimeout:  metadataTimeout,
		MaxResponseHeaderBytes: 16 << 10,
	}
	return &metadataFetcher{client: &http.Client{
		Transport: transport,
		Timeout:   metadataTimeout,
		// A redirect could point anywhere, so none are followed.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// publicAddressesOnly runs after DNS resolution, on the address actually dialed.
func publicAddressesOnly(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsInterfaceLocalMulticast() {
		return fmt.Errorf("refusing to fetch client metadata from %s", host)
	}
	return nil
}

// isMetadataURL says whether a client_id is an https URL with a path, which
// is what marks it as a metadata document rather than a registered id.
func isMetadataURL(id string) bool {
	parsed, err := url.Parse(id)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.Path != "" &&
		parsed.Path != "/" && parsed.Fragment == "" && parsed.User == nil
}

// fetch downloads and checks a metadata document, and says how long to cache it.
func (f *metadataFetcher) fetch(ctx context.Context, id string) (*clientMetadata, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, id, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch client metadata: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("client metadata answered %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, metadataMaxBytes+1))
	if err != nil {
		return nil, 0, err
	}
	if len(body) > metadataMaxBytes {
		return nil, 0, errors.New("client metadata is too large")
	}
	var doc clientMetadata
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, 0, fmt.Errorf("client metadata is not JSON: %w", err)
	}
	if doc.ClientID != id {
		return nil, 0, errors.New("client metadata names a different client_id")
	}
	if len(doc.RedirectURIs) == 0 {
		return nil, 0, errors.New("client metadata has no redirect_uris")
	}
	for _, redirect := range doc.RedirectURIs {
		if err := validateRedirectURI(redirect); err != nil {
			return nil, 0, err
		}
	}
	if doc.TokenEndpointAuthMethod != "" && doc.TokenEndpointAuthMethod != "none" {
		return nil, 0, errors.New("only public clients are supported")
	}
	return &doc, cacheDuration(resp.Header.Get("Cache-Control")), nil
}

// cacheDuration honours max-age within an hour and a day.
func cacheDuration(header string) time.Duration {
	age := metadataMinCache
	for _, directive := range strings.Split(header, ",") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(directive), "max-age="); ok {
			if seconds, err := strconv.Atoi(value); err == nil {
				age = time.Duration(seconds) * time.Second
			}
		}
	}
	return min(max(age, metadataMinCache), metadataMaxCache)
}
