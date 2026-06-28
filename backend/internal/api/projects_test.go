package api

import "testing"

func TestValidateRepoURL(t *testing.T) {
	cases := []struct {
		name     string
		url      string
		provider string
		wantErr  bool
	}{
		{"valid github", "https://github.com/owner/repo", "github", false},
		{"valid gitlab", "https://gitlab.com/owner/repo", "gitlab", false},
		{"http rejected", "http://github.com/owner/repo", "github", true},
		{"host mismatch", "https://gitlab.com/owner/repo", "github", true},
		{"missing repo segment", "https://github.com/owner", "github", true},
		{"not a url", "not-a-url", "github", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateRepoURL(c.url, c.provider)
			if (err != nil) != c.wantErr {
				t.Errorf("validateRepoURL(%q, %q) err=%v, wantErr=%v", c.url, c.provider, err, c.wantErr)
			}
		})
	}
}

func TestValidateCreateProject(t *testing.T) {
	valid := createProjectRequest{
		Name:        "My Service",
		GitProvider: "github",
		RepoURL:     "https://github.com/owner/repo",
		Branch:      "main",
		Runtime:     "go",
		Environment: "dev",
	}
	if msg := validateCreateProject(valid); msg != "" {
		t.Fatalf("valid request rejected: %s", msg)
	}

	bad := func(mutate func(r *createProjectRequest)) createProjectRequest {
		r := valid
		mutate(&r)
		return r
	}

	cases := []struct {
		name string
		req  createProjectRequest
	}{
		{"empty name", bad(func(r *createProjectRequest) { r.Name = "" })},
		{"bad provider", bad(func(r *createProjectRequest) { r.GitProvider = "bitbucket" })},
		{"bad runtime", bad(func(r *createProjectRequest) { r.Runtime = "rust" })},
		{"bad environment", bad(func(r *createProjectRequest) { r.Environment = "production" })},
		{"empty branch", bad(func(r *createProjectRequest) { r.Branch = "" })},
		{"bad repo url", bad(func(r *createProjectRequest) { r.RepoURL = "http://evil.com/a/b" })},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if validateCreateProject(c.req) == "" {
				t.Errorf("expected %q to be rejected", c.name)
			}
		})
	}
}
