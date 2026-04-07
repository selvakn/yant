//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const dockerImage = "yant:latest"

type testApp struct {
	baseURL string
	client  *http.Client
	cancel  context.CancelFunc
}

func startApp(t *testing.T) *testApp {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)

	req := testcontainers.ContainerRequest{
		Image:        dockerImage,
		ExposedPorts: []string{"8080/tcp"},
		Env: map[string]string{
			"SEMANTIC_SEARCH": "true",
		},
		WaitingFor: wait.ForHTTP("/login").
			WithPort("8080").
			WithStartupTimeout(90 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		cancel()
		t.Fatalf("start container: %v", err)
	}
	t.Cleanup(func() {
		container.Terminate(context.Background()) //nolint:errcheck
		cancel()
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("host: %v", err)
	}
	port, err := container.MappedPort(ctx, "8080")
	if err != nil {
		t.Fatalf("port: %v", err)
	}

	baseURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &testApp{baseURL: baseURL, client: client, cancel: cancel}
}

func (a *testApp) get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := a.client.Get(a.baseURL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func (a *testApp) postForm(t *testing.T, path string, values url.Values) *http.Response {
	t.Helper()
	resp, err := a.client.PostForm(a.baseURL+path, values)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

func (a *testApp) doRequest(t *testing.T, method, path string, body string, headers map[string]string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, a.baseURL+path, bodyReader)
	if err != nil {
		t.Fatalf("new request %s %s: %v", method, path, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

// TestHealthEndpoints verifies the app starts and responds to basic routes.
func TestHealthEndpoints(t *testing.T) {
	app := startApp(t)

	t.Run("login_page_accessible", func(t *testing.T) {
		resp := app.get(t, "/login")
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "Notes") {
			t.Error("expected login page to contain 'Notes'")
		}
	})

	t.Run("root_redirects_to_notes", func(t *testing.T) {
		resp := app.get(t, "/")
		if resp.StatusCode != 302 {
			t.Errorf("expected 302, got %d", resp.StatusCode)
		}
		loc := resp.Header.Get("Location")
		if loc != "/notes" {
			t.Errorf("expected redirect to /notes, got %s", loc)
		}
	})

	t.Run("protected_routes_redirect_to_login", func(t *testing.T) {
		resp := app.get(t, "/notes")
		if resp.StatusCode != 302 {
			t.Errorf("expected 302, got %d", resp.StatusCode)
		}
	})

	t.Run("404_for_unknown_routes", func(t *testing.T) {
		resp := app.get(t, "/nonexistent")
		if resp.StatusCode != 404 {
			t.Errorf("expected 404, got %d", resp.StatusCode)
		}
	})
}

func TestStaticAssets(t *testing.T) {
	app := startApp(t)

	t.Run("css_served", func(t *testing.T) {
		resp := app.get(t, "/static/css/app.css")
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		body := readBody(t, resp)
		if !strings.Contains(body, "layout") {
			t.Error("expected CSS to contain layout rules")
		}
	})
}
