package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/saeid-a/CoachAppBack/internal/config"
)

func TestRegisterDocsRoutesServesDocsPageAndSpec(t *testing.T) {
	app := fiber.New()
	cfg := &config.Config{AppEnv: "development", EnableDocs: true}

	if err := registerDocsRoutes(app, cfg); err != nil {
		t.Fatalf("registerDocsRoutes: %v", err)
	}

	pageReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	pageResp, err := app.Test(pageReq)
	if err != nil {
		t.Fatalf("app.Test docs page: %v", err)
	}
	defer pageResp.Body.Close()

	if pageResp.StatusCode != http.StatusOK {
		t.Fatalf("expected docs page status 200, got %d", pageResp.StatusCode)
	}
	if got := pageResp.Header.Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'none'") {
		t.Fatalf("expected restrictive CSP, got %q", got)
	}

	swaggerReq := httptest.NewRequest(http.MethodGet, "/docs/swagger", nil)
	swaggerResp, err := app.Test(swaggerReq)
	if err != nil {
		t.Fatalf("app.Test swagger page: %v", err)
	}
	defer swaggerResp.Body.Close()

	if swaggerResp.StatusCode != http.StatusOK {
		t.Fatalf("expected swagger page status 200, got %d", swaggerResp.StatusCode)
	}
	if got := swaggerResp.Header.Get("Content-Security-Policy"); strings.Contains(got, "cdn.jsdelivr.net") {
		t.Fatalf("expected swagger CSP to stay local-only, got %q", got)
	}

	redocReq := httptest.NewRequest(http.MethodGet, "/docs/redoc", nil)
	redocResp, err := app.Test(redocReq)
	if err != nil {
		t.Fatalf("app.Test redoc page: %v", err)
	}
	defer redocResp.Body.Close()

	if redocResp.StatusCode != http.StatusOK {
		t.Fatalf("expected redoc page status 200, got %d", redocResp.StatusCode)
	}

	scalarReq := httptest.NewRequest(http.MethodGet, "/docs/scalar", nil)
	scalarResp, err := app.Test(scalarReq)
	if err != nil {
		t.Fatalf("app.Test scalar page: %v", err)
	}
	defer scalarResp.Body.Close()

	if scalarResp.StatusCode != http.StatusOK {
		t.Fatalf("expected scalar page status 200, got %d", scalarResp.StatusCode)
	}

	assetReq := httptest.NewRequest(http.MethodGet, "/docs/assets/scalar/standalone.js", nil)
	assetResp, err := app.Test(assetReq)
	if err != nil {
		t.Fatalf("app.Test asset: %v", err)
	}
	defer assetResp.Body.Close()

	if assetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected asset status 200, got %d", assetResp.StatusCode)
	}
	if got := assetResp.Header.Get(fiber.HeaderContentType); !strings.Contains(got, "application/javascript") {
		t.Fatalf("expected js content type, got %q", got)
	}

	specReq := httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil)
	specResp, err := app.Test(specReq)
	if err != nil {
		t.Fatalf("app.Test docs spec: %v", err)
	}
	defer specResp.Body.Close()

	if specResp.StatusCode != http.StatusOK {
		t.Fatalf("expected docs spec status 200, got %d", specResp.StatusCode)
	}
	if got := specResp.Header.Get(fiber.HeaderContentType); !strings.Contains(got, "application/yaml") {
		t.Fatalf("expected yaml content type, got %q", got)
	}
}

func TestRegisterDocsRoutesSkipsWhenDisabled(t *testing.T) {
	app := fiber.New()
	cfg := &config.Config{AppEnv: "production", EnableDocs: true}

	if err := registerDocsRoutes(app, cfg); err != nil {
		t.Fatalf("registerDocsRoutes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 when docs are not in development, got %d", resp.StatusCode)
	}
}
