package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mintingSource issues a different token on each call, standing in for a source
// of short lived credentials such as GitHub App installation tokens.
type mintingSource struct {
	calls int
}

func (s *mintingSource) Token(context.Context) (*scm.Token, error) {
	s.calls++
	return &scm.Token{Token: fmt.Sprintf("token-%d", s.calls)}, nil
}

type errorSource struct{}

func (errorSource) Token(context.Context) (*scm.Token, error) {
	return nil, errors.New("cannot mint a token")
}

type nilSource struct{}

func (nilSource) Token(context.Context) (*scm.Token, error) {
	return nil, nil
}

// recorder captures the requests a transport actually sends.
func recorder(t *testing.T, requests *[]*http.Request) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		*requests = append(*requests, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func get(t *testing.T, client *http.Client, url string) {
	t.Helper()
	res, err := client.Get(url)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
}

func TestAuth_ResolvesTokenOnEveryRequest(t *testing.T) {
	var requests []*http.Request
	srv := recorder(t, &requests)

	client := &http.Client{
		Transport: &Auth{
			Source:     &mintingSource{},
			Credential: SchemeCredential("Bearer"),
		},
	}
	get(t, client, srv.URL)
	get(t, client, srv.URL)

	require.Len(t, requests, 2)
	assert.Equal(t, "Bearer token-1", requests[0].Header.Get("Authorization"))
	assert.Equal(t, "Bearer token-2", requests[1].Header.Get("Authorization"))
}

func TestAuth_Credentials(t *testing.T) {
	tests := []struct {
		name       string
		credential Credential
		check      func(t *testing.T, r *http.Request)
	}{
		{
			name:       "scheme",
			credential: SchemeCredential("token"),
			check: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "token token-1", r.Header.Get("Authorization"))
			},
		},
		{
			name:       "private token",
			credential: PrivateTokenCredential,
			check: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "token-1", r.Header.Get("Private-Token"))
			},
		},
		{
			name:       "basic auth with a username",
			credential: BasicAuthCredential("bot"),
			check: func(t *testing.T, r *http.Request) {
				username, password, ok := r.BasicAuth()
				require.True(t, ok)
				assert.Equal(t, "bot", username)
				assert.Equal(t, "token-1", password)
			},
		},
		{
			// Azure DevOps sends the token as the password with no username.
			name:       "basic auth without a username",
			credential: BasicAuthCredential(""),
			check: func(t *testing.T, r *http.Request) {
				username, password, ok := r.BasicAuth()
				require.True(t, ok)
				assert.Empty(t, username)
				assert.Equal(t, "token-1", password)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var requests []*http.Request
			srv := recorder(t, &requests)

			client := &http.Client{
				Transport: &Auth{Source: &mintingSource{}, Credential: tc.credential},
			}
			get(t, client, srv.URL)

			require.Len(t, requests, 1)
			tc.check(t, requests[0])
		})
	}
}

func TestAuth_SourceError(t *testing.T) {
	client := &http.Client{
		Transport: &Auth{Source: errorSource{}, Credential: SchemeCredential("Bearer")},
	}

	//nolint:bodyclose // the transport fails before a response exists
	res, err := client.Get("http://localhost")
	require.Nil(t, res)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot mint a token")
}

func TestAuth_NoTokenSendsRequestUnauthenticated(t *testing.T) {
	var requests []*http.Request
	srv := recorder(t, &requests)

	client := &http.Client{
		Transport: &Auth{Source: nilSource{}, Credential: SchemeCredential("Bearer")},
	}
	get(t, client, srv.URL)

	require.Len(t, requests, 1)
	assert.Empty(t, requests[0].Header.Get("Authorization"))
}

func TestAuth_DoesNotMutateRequest(t *testing.T) {
	var requests []*http.Request
	srv := recorder(t, &requests)

	client := &http.Client{
		Transport: &Auth{Source: &mintingSource{}, Credential: SchemeCredential("Bearer")},
	}
	req, err := http.NewRequest("GET", srv.URL, http.NoBody)
	require.NoError(t, err)

	res, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())

	assert.Empty(t, req.Header.Get("Authorization"))
	require.Len(t, requests, 1)
	assert.Equal(t, "Bearer token-1", requests[0].Header.Get("Authorization"))
}
