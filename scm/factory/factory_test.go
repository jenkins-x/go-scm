package factory

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient("fake", "", "")
	if client == nil {
		t.Errorf("no client created")
	}
	if err != nil {
		t.Errorf("failed to create client %s", err)
	}
}

func TestGHEEndpoint(t *testing.T) {
	assert.Equal(t, "https://my.ghe.com/custom/api/v5", ensureGHEEndpoint("https://my.ghe.com/custom/api/v5"))
	assert.Equal(t, "https://my.ghe.com/custom/api/v3", ensureGHEEndpoint("https://my.ghe.com/custom"))
	assert.Equal(t, "https://my.ghe.com/api/v3", ensureGHEEndpoint("https://my.ghe.com"))
}

func TestNewClientWithOptionFunc(t *testing.T) {
	httpClient := &http.Client{}
	scmClient, err := NewClient("github", "", "", Client(httpClient))
	if err != nil {
		t.Errorf("failed to create client %s", err)
	}

	assert.Equal(t, scmClient.Client, httpClient)
}

func TestFromRepoURL(t *testing.T) {
	client, err := FromRepoURL("https://:abc123@gitlab.com/myorg/myrepo.git")
	if err != nil {
		t.Fatal(err)
	}
	if client.BaseURL.String() != "https://gitlab.com/" {
		t.Fatalf("BaseURL got %q, want %q", client.BaseURL, "https://gitlab.com/")
	}
	if client.Driver != scm.DriverGitlab {
		t.Fatalf("Driver got %q, want %q", client.Driver, client.Driver)
	}
	assert.Equal(t, "abc123", sentHeader(t, client, "Private-Token"))
}

// sentHeader issues a request through the client's transport and returns the
// value the transport attached, which is the credential the factory installed.
func sentHeader(t *testing.T, client *scm.Client, header string) string {
	t.Helper()
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(header)
	}))
	defer srv.Close()

	res, err := client.Client.Get(srv.URL)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	return got
}

func TestNewClientWithTokenSource(t *testing.T) {
	tests := []struct {
		driver string
		header string
		want   string
	}{
		{driver: "github", header: "Authorization", want: "Bearer token-1"},
		{driver: "gogs", header: "Authorization", want: "Bearer token-1"},
		{driver: "stash", header: "Authorization", want: "Bearer token-1"},
		{driver: "gitlab", header: "Private-Token", want: "token-1"},
		{driver: "azure", header: "Authorization", want: "Basic OnRva2VuLTE="},
	}
	for _, tc := range tests {
		t.Run(tc.driver, func(t *testing.T) {
			client, err := NewClientWithTokenSource(tc.driver, "https://example.com", &mintingSource{})
			require.NoError(t, err)
			assert.Equal(t, tc.want, sentHeader(t, client, tc.header))
		})
	}
}

// TestNewClientWithTokenSource_Gitea covers the Gitea SDK's own http.Client as
// well as scm.Client, since the driver serves calls from both.
func TestNewClientWithTokenSource_Gitea(t *testing.T) {
	var sdkAuth, clientAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/version" {
			sdkAuth = r.Header.Get("Authorization")
			fmt.Fprint(w, `{"version":"1.22.0"}`)
			return
		}
		clientAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	client, err := NewClientWithTokenSource("gitea", srv.URL, &mintingSource{})
	require.NoError(t, err)

	res, err := client.Client.Get(srv.URL)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())

	assert.Equal(t, "token token-1", sdkAuth)
	assert.Equal(t, "token token-2", clientAuth)
}

func TestNewClientWithTokenSource_BitbucketCloud(t *testing.T) {
	client, err := NewClientWithTokenSource("bitbucketcloud", "", &mintingSource{}, SetUsername("bot"))
	require.NoError(t, err)
	assert.Equal(t, "Basic Ym90OnRva2VuLTE=", sentHeader(t, client, "Authorization"))
}

func TestNewClientWithTokenSource_RefreshesToken(t *testing.T) {
	client, err := NewClientWithTokenSource("github", "", &mintingSource{})
	require.NoError(t, err)

	assert.Equal(t, "Bearer token-1", sentHeader(t, client, "Authorization"))
	assert.Equal(t, "Bearer token-2", sentHeader(t, client, "Authorization"))
}

// mintingSource issues a different token on each call, standing in for a source
// of short lived credentials such as GitHub App installation tokens.
type mintingSource struct {
	calls int
}

func (s *mintingSource) Token(context.Context) (*scm.Token, error) {
	s.calls++
	return &scm.Token{Token: fmt.Sprintf("token-%d", s.calls)}, nil
}
