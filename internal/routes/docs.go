package routes

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/saeid-a/CoachAppBack/internal/config"
)

//go:embed docs_static/*/*
var docsStaticFS embed.FS

const docsIndexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ .Title }}</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f6f7f4;
      --panel: #ffffff;
      --text: #132019;
      --muted: #536258;
      --accent: #1f6f4a;
      --border: #d8ddd6;
      --code-bg: #0f172a;
      --code-text: #e2e8f0;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: Georgia, "Times New Roman", serif;
      color: var(--text);
      background:
        radial-gradient(circle at top left, rgba(31, 111, 74, 0.12), transparent 30%),
        linear-gradient(180deg, #fcfcfa 0%, var(--bg) 100%);
    }
    main {
      max-width: 1120px;
      margin: 0 auto;
      padding: 48px 20px 64px;
    }
    .hero, .panel {
      background: rgba(255, 255, 255, 0.92);
      border: 1px solid var(--border);
      border-radius: 18px;
      box-shadow: 0 20px 60px rgba(19, 32, 25, 0.08);
      backdrop-filter: blur(8px);
    }
    .hero {
      padding: 28px;
      margin-bottom: 20px;
    }
    .hero h1 {
      margin: 0 0 12px;
      font-size: clamp(2rem, 5vw, 3.5rem);
      line-height: 0.96;
    }
    .hero p {
      margin: 0;
      max-width: 48rem;
      color: var(--muted);
      font-size: 1rem;
      line-height: 1.6;
    }
    .actions, .cards {
      display: flex;
      flex-wrap: wrap;
      gap: 12px;
      margin-top: 20px;
    }
    .button, .card a {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      padding: 11px 16px;
      border-radius: 999px;
      border: 1px solid var(--accent);
      color: #fff;
      background: var(--accent);
      text-decoration: none;
      font-weight: 600;
    }
    .button.secondary {
      background: transparent;
      color: var(--accent);
    }
    .cards {
      align-items: stretch;
    }
    .card {
      flex: 1 1 240px;
      min-width: 240px;
      padding: 22px;
      border: 1px solid var(--border);
      border-radius: 18px;
      background: rgba(255, 255, 255, 0.92);
      box-shadow: 0 20px 60px rgba(19, 32, 25, 0.06);
    }
    .card h2 {
      margin: 0 0 10px;
      font-size: 1.35rem;
    }
    .card p {
      margin: 0 0 16px;
      line-height: 1.6;
      color: var(--muted);
    }
    .meta {
      display: grid;
      gap: 12px;
      margin: 20px 0;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    }
    .panel {
      padding: 24px;
    }
    .meta strong, .panel h2 {
      display: block;
      margin-bottom: 6px;
      font-size: 0.92rem;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      color: var(--muted);
    }
    .meta span {
      font-size: 1rem;
    }
    pre {
      margin: 0;
      padding: 20px;
      overflow: auto;
      border-radius: 14px;
      background: var(--code-bg);
      color: var(--code-text);
      font-size: 0.92rem;
      line-height: 1.5;
    }
  </style>
</head>
<body>
  <main>
    <section class="hero">
      <h1>{{ .Title }}</h1>
      <p>The OpenAPI spec is served from the same origin at <code>/docs/openapi.yaml</code>. UI routes are read-only viewers layered on top of that spec. All viewer assets are vendored into this service and served locally, and the entire docs surface is intended for development-only exposure.</p>
      <div class="actions">
        <a class="button" href="/docs/openapi.yaml">Open Raw Spec</a>
        <a class="button secondary" href="/docs/openapi.yaml" download="openapi.yaml">Download YAML</a>
      </div>
    </section>
    <section class="cards">
      <article class="card">
        <h2>Swagger UI</h2>
        <p>Familiar reference view with operation explorer, but without server-side changes to your API.</p>
        <a href="/docs/swagger">Open Swagger UI</a>
      </article>
      <article class="card">
        <h2>ReDoc</h2>
        <p>Dense documentation layout optimized for structured schema browsing and long-form API reading.</p>
        <a href="/docs/redoc">Open ReDoc</a>
      </article>
      <article class="card">
        <h2>Scalar</h2>
        <p>Modern API reference view with a cleaner information hierarchy and a more application-style shell.</p>
        <a href="/docs/scalar">Open Scalar</a>
      </article>
    </section>
    <section class="meta">
      <div class="panel">
        <strong>Spec Path</strong>
        <span>/docs/openapi.yaml</span>
      </div>
      <div class="panel">
        <strong>Last Loaded</strong>
        <span>{{ .LoadedAt }}</span>
      </div>
      <div class="panel">
        <strong>UI Mode</strong>
        <span>Read-only same-origin spec</span>
      </div>
    </section>
    <section class="panel">
      <h2>OpenAPI YAML</h2>
      <pre>{{ .Spec }}</pre>
    </section>
  </main>
</body>
</html>
`

const swaggerUIHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>CoachAppBack Swagger UI</title>
  <link rel="stylesheet" href="/docs/assets/swagger-ui/swagger-ui.css">
  <style>
    html, body { margin: 0; height: 100%; background: #f7f8f4; }
    #swagger-ui { min-height: 100%; }
    .swagger-ui .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="/docs/assets/swagger-ui/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: '/docs/openapi.yaml',
      dom_id: '#swagger-ui',
      deepLinking: true,
      displayRequestDuration: true,
      docExpansion: 'list',
      defaultModelsExpandDepth: 1,
      defaultModelExpandDepth: 1,
      filter: true,
      supportedSubmitMethods: []
    });
  </script>
</body>
</html>
`

const redocHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>CoachAppBack ReDoc</title>
  <style>
    html, body { margin: 0; height: 100%; background: #ffffff; }
  </style>
</head>
<body>
  <redoc spec-url="/docs/openapi.yaml" hide-download-button></redoc>
  <script src="/docs/assets/redoc/redoc.standalone.js"></script>
</body>
</html>
`

const scalarHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>CoachAppBack Scalar</title>
  <link rel="stylesheet" href="/docs/assets/scalar/style.css">
  <style>
    html, body, #app { margin: 0; min-height: 100%; background: #ffffff; }
  </style>
</head>
<body>
  <div id="app"></div>
  <script src="/docs/assets/scalar/standalone.js"></script>
  <script>
    Scalar.createApiReference('#app', {
      url: '/docs/openapi.yaml',
      theme: 'kepler',
      layout: 'modern',
      hideDownloadButton: false
    });
  </script>
</body>
</html>
`

type docsPageData struct {
	Title    string
	LoadedAt string
	Spec     string
}

func registerDocsRoutes(app fiber.Router, cfg *config.Config) error {
	if !cfg.DocsEnabled() {
		return nil
	}

	spec, err := loadOpenAPISpec()
	if err != nil {
		return fmt.Errorf("load openapi spec: %w", err)
	}

	indexTemplate, err := template.New("docs-index").Parse(docsIndexHTML)
	if err != nil {
		return fmt.Errorf("parse docs template: %w", err)
	}

	pageData := docsPageData{
		Title:    "CoachAppBack API Docs",
		LoadedAt: time.Now().UTC().Format(time.RFC3339),
		Spec:     string(spec),
	}

	indexHandler := func(c *fiber.Ctx) error {
		applyDocsBaseHeaders(c, fiber.MIMETextHTMLCharsetUTF8)
		c.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src 'self' data:; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")

		var body bytes.Buffer
		if err := indexTemplate.Execute(&body, pageData); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to render api docs")
		}

		return c.Status(fiber.StatusOK).Send(body.Bytes())
	}

	app.Get("/docs", indexHandler)
	app.Get("/docs/", indexHandler)
	app.Get("/docs/assets/:viewer/:file", func(c *fiber.Ctx) error {
		viewer := c.Params("viewer")
		file := c.Params("file")

		asset, contentType, ok := loadDocsAsset(viewer, file)
		if !ok {
			return fiber.ErrNotFound
		}

		applyDocsBaseHeaders(c, contentType)
		c.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		return c.Status(fiber.StatusOK).Send(asset)
	})
	app.Get("/docs/swagger", func(c *fiber.Ctx) error {
		applyDocsBaseHeaders(c, fiber.MIMETextHTMLCharsetUTF8)
		c.Set("Content-Security-Policy", "default-src 'none'; connect-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; font-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
		return c.Status(fiber.StatusOK).SendString(swaggerUIHTML)
	})
	app.Get("/docs/redoc", func(c *fiber.Ctx) error {
		applyDocsBaseHeaders(c, fiber.MIMETextHTMLCharsetUTF8)
		c.Set("Content-Security-Policy", "default-src 'none'; connect-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; font-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
		return c.Status(fiber.StatusOK).SendString(redocHTML)
	})
	app.Get("/docs/scalar", func(c *fiber.Ctx) error {
		applyDocsBaseHeaders(c, fiber.MIMETextHTMLCharsetUTF8)
		c.Set("Content-Security-Policy", "default-src 'none'; connect-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; font-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
		return c.Status(fiber.StatusOK).SendString(scalarHTML)
	})
	app.Get("/docs/openapi.yaml", func(c *fiber.Ctx) error {
		applyDocsBaseHeaders(c, "application/yaml; charset=utf-8")
		c.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		c.Set(fiber.HeaderContentDisposition, `inline; filename="openapi.yaml"`)
		return c.Status(fiber.StatusOK).Send(spec)
	})

	return nil
}

func loadOpenAPISpec() ([]byte, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("resolve source path")
	}

	specPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "docs", "openapi.yaml")
	spec, err := os.ReadFile(specPath)
	if err != nil {
		return nil, err
	}

	return spec, nil
}

func applyDocsBaseHeaders(c *fiber.Ctx, contentType string) {
	c.Set(fiber.HeaderContentType, contentType)
	c.Set(fiber.HeaderCacheControl, "no-store, max-age=0")
	c.Set(fiber.HeaderPragma, "no-cache")
	c.Set(fiber.HeaderExpires, "0")
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	c.Set(fiber.HeaderXFrameOptions, "DENY")
	c.Set("Referrer-Policy", "no-referrer")
	c.Set("Permissions-Policy", "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()")
	c.Set("Cross-Origin-Resource-Policy", "same-origin")
	c.Set("Cross-Origin-Opener-Policy", "same-origin")
	c.Set("Cross-Origin-Embedder-Policy", "require-corp")
	c.Set("X-Robots-Tag", "noindex, nofollow")
}

func loadDocsAsset(viewer, file string) ([]byte, string, bool) {
	var path string
	var contentType string

	switch viewer + "/" + file {
	case "swagger-ui/swagger-ui.css":
		path = "docs_static/swagger-ui/swagger-ui.css"
		contentType = "text/css; charset=utf-8"
	case "swagger-ui/swagger-ui-bundle.js":
		path = "docs_static/swagger-ui/swagger-ui-bundle.js"
		contentType = "application/javascript; charset=utf-8"
	case "redoc/redoc.standalone.js":
		path = "docs_static/redoc/redoc.standalone.js"
		contentType = "application/javascript; charset=utf-8"
	case "scalar/standalone.js":
		path = "docs_static/scalar/standalone.js"
		contentType = "application/javascript; charset=utf-8"
	case "scalar/style.css":
		path = "docs_static/scalar/style.css"
		contentType = "text/css; charset=utf-8"
	default:
		return nil, "", false
	}

	asset, err := docsStaticFS.ReadFile(path)
	if err != nil {
		return nil, "", false
	}

	return asset, contentType, true
}
