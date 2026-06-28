package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	githubendpoint "golang.org/x/oauth2/github"
)

// GitHubOAuth wraps the GitHub OAuth2 authorization-code flow.
type GitHubOAuth struct {
	cfg *oauth2.Config
}

func NewGitHubOAuth(clientID, clientSecret, callbackURL string) *GitHubOAuth {
	return &GitHubOAuth{
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  callbackURL,
			Endpoint:     githubendpoint.Endpoint,
			Scopes:       []string{"read:user", "user:email"},
		},
	}
}

// AuthCodeURL builds the GitHub consent URL for the given anti-CSRF state.
func (g *GitHubOAuth) AuthCodeURL(state string) string {
	return g.cfg.AuthCodeURL(state)
}

// GitHubUser is the subset of the GitHub profile we persist.
type GitHubUser struct {
	ID        int64
	Login     string
	Email     string
	AvatarURL string
}

// Exchange swaps an auth code for a token and loads the user's profile.
func (g *GitHubOAuth) Exchange(ctx context.Context, code string) (*GitHubUser, error) {
	tok, err := g.cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("oauth exchange: %w", err)
	}

	client := g.cfg.Client(ctx, tok)
	client.Timeout = 10 * time.Second

	var profile struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := getJSON(ctx, client, "https://api.github.com/user", &profile); err != nil {
		return nil, err
	}

	user := &GitHubUser{ID: profile.ID, Login: profile.Login, Email: profile.Email, AvatarURL: profile.AvatarURL}
	if user.Email == "" {
		user.Email = primaryEmail(ctx, client)
	}
	return user, nil
}

func getJSON(ctx context.Context, c *http.Client, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github api %s: status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

// primaryEmail fetches the user's primary, verified email when the profile
// email is hidden. Best-effort: returns "" on any failure.
func primaryEmail(ctx context.Context, c *http.Client) string {
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := getJSON(ctx, c, "https://api.github.com/user/emails", &emails); err != nil {
		return ""
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	return ""
}
