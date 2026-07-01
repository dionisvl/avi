package app

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	httpswagger "github.com/swaggo/http-swagger"

	"github.com/dionisvl/avi/api-go/docs"
)

var swaggerSpec map[string]any

func init() {
	data, err := docs.FS.ReadFile("swagger.json")
	if err != nil {
		panic("failed to read embedded swagger.json: " + err.Error())
	}
	if err := json.Unmarshal(data, &swaggerSpec); err != nil {
		panic("failed to parse embedded swagger.json: " + err.Error())
	}
}

// mountSwagger wires up Swagger UI routes:
//   - /swagger/doc.json         — spec with host/schemes/basePath patched per request
//   - /swagger/swagger.json     — alias of doc.json
//   - /swagger, /swagger/index.html — custom UI with a host-selector dropdown
//   - /swagger/*                — static Swagger UI assets from swaggo/http-swagger
//
// In prod (APP_ENV=prod) all /swagger routes are protected by HTTP Basic Auth
// using HTTP_BASIC_ADMIN_USER / HTTP_BASIC_ADMIN_PASSWORD env vars.
func (a *App) mountSwagger(r chi.Router) {
	var wrap func(http.Handler) http.Handler
	if a.cfg.App.Env == "prod" {
		wrap = a.basicAuth("Swagger UI")
	} else {
		wrap = func(h http.Handler) http.Handler { return h }
	}

	r.With(wrap).Get("/swagger/doc.json", a.serveSwaggerSpec)
	r.With(wrap).Get("/swagger/swagger.json", a.serveSwaggerSpec)
	r.With(wrap).Get("/swagger", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/swagger/index.html", http.StatusFound)
	})
	r.With(wrap).Get("/swagger/index.html", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(buildSwaggerIndex(a.cfg.App.SwaggerHosts))
	})
	r.With(wrap).Get("/swagger/*", httpswagger.Handler())
}

func (a *App) basicAuth(realm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(a.cfg.App.AdminUser)) == 1
			passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(a.cfg.App.AdminPassword)) == 1
			if !ok || !userMatch || !passMatch || a.cfg.App.AdminPassword == "" {
				w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// serveSwaggerSpec returns the embedded swagger.json with host/schemes/basePath
// overridden per request. Host resolution priority:
//  1. ?host= query param
//  2. first entry of SWAGGER_HOSTS env
//  3. request Host header
//
// basePath is forced to "/" because our paths in swagger.json already contain
// the /api/v1 prefix (otherwise Swagger UI would produce /api/v1/api/v1/...).
func (a *App) serveSwaggerSpec(w http.ResponseWriter, req *http.Request) {
	spec := make(map[string]any, len(swaggerSpec))
	maps.Copy(spec, swaggerSpec)

	host := req.URL.Query().Get("host")
	if host == "" && len(a.cfg.App.SwaggerHosts) > 0 {
		host = a.cfg.App.SwaggerHosts[0]
	}
	if host == "" {
		host = req.Host
	}

	// Scheme follows how Swagger UI itself was loaded: if the spec request
	// came in over https (directly or via a TLS-terminating proxy), use https;
	// otherwise http. A ?scheme= override lets the UI force a specific one.
	scheme := "http"
	if req.TLS != nil || strings.EqualFold(req.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	if s := req.URL.Query().Get("scheme"); s == "http" || s == "https" {
		scheme = s
	}

	spec["host"] = host
	spec["schemes"] = []string{scheme}
	spec["basePath"] = "/"

	// Add a helpful description to the BearerAuth security definition so the
	// Authorize dialog tells users they can paste the raw JWT (our UI also
	// auto-prefixes "Bearer " via requestInterceptor — see buildSwaggerIndex).
	// Deep-copy securityDefinitions before mutating to avoid cross-request bleed.
	if defs, ok := swaggerSpec["securityDefinitions"].(map[string]any); ok {
		defsCopy := make(map[string]any, len(defs))
		maps.Copy(defsCopy, defs)
		if bearer, ok := defs["BearerAuth"].(map[string]any); ok {
			bearerCopy := make(map[string]any, len(bearer))
			maps.Copy(bearerCopy, bearer)
			bearerCopy["description"] = `JWT access token. Paste the raw token — "Bearer " is added automatically.`
			defsCopy["BearerAuth"] = bearerCopy
		}
		spec["securityDefinitions"] = defsCopy
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(spec)
}

// buildSwaggerIndex renders a Swagger UI index.html with a host-selector dropdown.
// Selecting a host reloads the page with ?host=<host>, and serveSwaggerSpec
// rewrites the "host" field of the spec accordingly, so "Try it out" requests
// target the chosen host.
func buildSwaggerIndex(hosts []string) []byte {
	if len(hosts) == 0 {
		hosts = []string{"localhost:8080"}
	}
	hostsJSON, _ := json.Marshal(hosts)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>avi API - Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; background: #fafafa; }
    .host-bar {
      display: flex; align-items: center; gap: 12px;
      padding: 10px 20px; background: #1b1b1b; color: #fff;
      font-family: sans-serif; font-size: 14px;
    }
    .host-bar label { font-weight: 600; }
    .host-bar select, .host-bar input {
      padding: 6px 10px; border-radius: 4px; border: 1px solid #444;
      background: #2a2a2a; color: #fff; min-width: 260px;
    }
    .host-bar button {
      padding: 6px 14px; border-radius: 4px; border: 0;
      background: #4990e2; color: #fff; cursor: pointer; font-weight: 600;
    }
  </style>
</head>
<body>
  <div class="host-bar">
    <label for="host-select">API host:</label>
    <select id="host-select"></select>
    <input id="host-custom" placeholder="or custom host, e.g. api.avi.app" />
    <button id="host-apply">Apply</button>
  </div>
  <div id="swagger-ui"></div>

  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
  <script>
    const HOSTS = %s;
    const params = new URLSearchParams(window.location.search);
    const currentHost = params.get("host") || HOSTS[0];

    const select = document.getElementById("host-select");
    HOSTS.forEach(h => {
      const opt = document.createElement("option");
      opt.value = h; opt.textContent = h;
      if (h === currentHost) opt.selected = true;
      select.appendChild(opt);
    });
    if (!HOSTS.includes(currentHost)) {
      const opt = document.createElement("option");
      opt.value = currentHost; opt.textContent = currentHost + " (custom)";
      opt.selected = true;
      select.appendChild(opt);
    }

    document.getElementById("host-apply").addEventListener("click", () => {
      const custom = document.getElementById("host-custom").value.trim();
      const host = custom || select.value;
      const url = new URL(window.location.href);
      url.searchParams.set("host", host);
      window.location.href = url.toString();
    });
    select.addEventListener("change", () => {
      const url = new URL(window.location.href);
      url.searchParams.set("host", select.value);
      window.location.href = url.toString();
    });

    window.ui = SwaggerUIBundle({
      url: "/swagger/doc.json?host=" + encodeURIComponent(currentHost),
      dom_id: "#swagger-ui",
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
      layout: "StandaloneLayout",
      // Auto-prefix the Authorization header with "Bearer " when the user
      // pasted just the raw JWT into the Authorize dialog. Our backend
      // rejects anything that isn't "Bearer <token>" (RFC 6750).
      requestInterceptor: (req) => {
        const auth = req.headers && req.headers["Authorization"];
        if (auth && !/^Bearer\s+/i.test(auth)) {
          req.headers["Authorization"] = "Bearer " + auth;
        }
        return req;
      },
    });
  </script>
</body>
</html>`, string(hostsJSON))

	return []byte(html)
}
