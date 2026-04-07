package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GitHubOAuth holds the configuration for GitHub OAuth.
type GitHubOAuth struct {
	ClientID     string
	ClientSecret string
	BaseURL      string // External base URL (e.g. "https://notes.example.com"), empty = auto-detect
}

// AuthorizeURL returns the GitHub authorization URL with a CSRF state token.
// The state is stored in the session for validation on callback.
func (g *GitHubOAuth) AuthorizeURL(r *http.Request) string {
	state := generateState()
	SessionManager.Put(r.Context(), "oauth_state", state)

	params := url.Values{
		"client_id":    {g.ClientID},
		"redirect_uri": {g.callbackURL(r)},
		"scope":        {"read:user"},
		"state":        {state},
	}
	return "https://github.com/login/oauth/authorize?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for a GitHub username.
// It validates the state parameter, exchanges the code for an access token,
// and fetches the authenticated user's login name.
func (g *GitHubOAuth) ExchangeCode(r *http.Request, code, state string) (string, error) {
	expected := SessionManager.GetString(r.Context(), "oauth_state")
	if expected == "" || state != expected {
		return "", fmt.Errorf("invalid oauth state")
	}
	SessionManager.Remove(r.Context(), "oauth_state")

	token, err := g.fetchAccessToken(code, g.callbackURL(r))
	if err != nil {
		return "", fmt.Errorf("token exchange: %w", err)
	}

	username, err := fetchGitHubUsername(token)
	if err != nil {
		return "", fmt.Errorf("fetch user: %w", err)
	}
	return username, nil
}

func (g *GitHubOAuth) fetchAccessToken(code, redirectURI string) (string, error) {
	data := url.Values{
		"client_id":     {g.ClientID},
		"client_secret": {g.ClientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("%s: %s", result.Error, result.ErrorDesc)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}
	return result.AccessToken, nil
}

func fetchGitHubUsername(token string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var user struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", err
	}
	if user.Login == "" {
		return "", fmt.Errorf("empty login from github")
	}
	return user.Login, nil
}

func generateState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (g *GitHubOAuth) callbackURL(r *http.Request) string {
	if g.BaseURL != "" {
		return strings.TrimRight(g.BaseURL, "/") + "/auth/github/callback"
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/auth/github/callback"
}
