# Security & Quality Hardening — Design Spec

**Date:** 2026-09-03
**Status:** Approved
**Scope:** Critical + High + Medium findings from code review

## Context

Full code review identified 54 issues across backend (5C/12H/15M), frontend (3C/5H/15M), and infra (3C/7H/8M). This spec covers all Critical, High, and Medium fixes. Low-severity items are out of scope.

## Architecture

Six isolated layers, each independently mergeable. Only dependency: Layer 2 benefits from Layer 1 being done first (gitignored db files won't conflict). Layers 2-6 can execute in parallel.

```
Layer 1 (git hygiene) ──┬──> Layer 2 (backend auth/security)
                        ├──> Layer 3 (backend perf/correctness)
                        ├──> Layer 4 (frontend fixes)
                        ├──> Layer 5 (infra/CI)
                        └──> Layer 6 (docs sync) [after 2,5]
```

## Layer 1 — Git Hygiene

### Files
- `.gitignore`
- `.gitattributes` (new)
- `backend/.dockerignore` (new)

### Changes
1. `git rm --cached backend/dashboard.db-shm backend/dashboard.db-wal` — remove SQLite sidecars from tracking (committed in 7de1f4c, may contain WebAuthn credentials/session data)
2. `git rm -r --cached .idea/` — remove 7 JetBrains config files from tracking
3. `.gitignore`: replace granular `.idea/**/...` rules (lines 29-140) with single `.idea/` entry; add `*.db-shm` and `*.db-wal` (or `*.db*` glob)
4. Create `.gitattributes`: `* text=auto eol=lf`, `*.db binary`, `*.db-shm binary`, `*.db-wal binary`
5. Create `backend/.dockerignore`: exclude `.env`, `*.db`, `*.db-shm`, `*.db-wal`, `.air.toml`, `server` (compiled binary), `.git`

### Out of scope
- Tailscale API key rotation — user action (key is expired per user)
- Git history purge of db files — user runs `git filter-repo` manually if needed

## Layer 2 — Backend Auth & Security

### Files
- `internal/middleware/auth.go`
- `internal/middleware/cors.go`
- `internal/handler/router.go`
- `internal/handler/auth.go`
- `internal/handler/respond.go`
- `internal/handler/tailscale.go`
- `internal/handler/containers.go`
- `internal/handler/services.go`
- `internal/handler/simplex.go`
- `internal/auth/session.go`
- `internal/auth/store.go`
- `cmd/server/main.go`
- `internal/config/config.go`

### Changes

#### 2.1 AuthRequired middleware (C1)
Add `AuthRequired` middleware in `middleware/auth.go`:
```go
func AuthRequired(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !IsAuthenticated(r) {
            respondError(w, http.StatusUnauthorized, "authentication required")
            return
        }
        next.ServeHTTP(w, r)
    })
}
```
In `router.go`, wrap all `/api/*` data routes (except `/api/auth/*` and `/health`) with `r.With(mw.AuthRequired).Group(...)`.

#### 2.2 CORS hardening (H1, H2)
- `cors.go`: replace `*` with allowed origin derived from `cfg.WebAuthnOrigin` (already in config) — no new env var needed
- Add `POST, DELETE` to `Access-Control-Allow-Methods`
- Add `Authorization` to `Access-Control-Allow-Headers`

#### 2.3 WebAuthn temp cookie HMAC (C2)
In `handler/auth.go`:
- `setTempSession`: HMAC-sign the base64 payload using `SessionManager`'s secret, format: `base64(payload).base64(hmac)`
- `getTempSession`: verify HMAC before deserializing; return error if mismatch

#### 2.4 Registration race fix (C3)
In `auth/store.go`: combine `HasUser()` check + `CreateUser` into single method `CreateUserIfAbsent()` that holds `Lock` for the entire check-and-insert:
```go
func (s *Store) CreateUserIfAbsent() (*User, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    var count int
    if err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
        return nil, err
    }
    if count > 0 {
        return s.getUserLocked()
    }
    // insert new user
}
```
`RegisterBegin` calls `CreateUserIfAbsent()` instead of `HasUser()` + `CreateUser()`.

#### 2.5 SESSION_SECRET fail-fast (C4)
In `auth/session.go`:
- Check `rand.Read` error
- If `SESSION_SECRET` empty AND `WEBAUTHN_RP_ID` != `localhost` (production), `log.Fatal("SESSION_SECRET must be set in production")`
- Dev mode: generate random secret with error check

#### 2.6 Store error handling (H3, H4)
- `store.go:34-43`: return errors from `json.Unmarshal`/`json.Marshal` in `AddCredential`
- `store.go:82`: `HasUser` returns `(bool, error)`, propagate scan error
- `store.go:23-32`: log unmarshal error in `WebAuthnCredentials`

#### 2.7 SQLite pool (H5)
In `auth/store.go` after `sql.Open`:
```go
db.SetMaxOpenConns(1)
db.Exec("PRAGMA busy_timeout=5000")
db.Exec("PRAGMA journal_mode=WAL")
```

#### 2.8 Request body limit (H12)
In `handler/auth.go`: wrap all `r.Body` reads with `http.MaxBytesReader(w, r.Body, 1<<20)` (1MB)

#### 2.9 Error message sanitization (H10)
In `handler/respond.go`: `respondError` logs full error server-side, returns generic message to client:
```go
func respondError(w http.ResponseWriter, code int, internalErr error) {
    log.Printf("request error: %v", internalErr)
    respondJSON(w, code, model.Error{Error: http.StatusText(code)})
}
```
Update all callers to pass `error` instead of `err.Error()`.

#### 2.10 HMAC token length-prefixing (M14)
In `session.go:84-88`: write `len(username)` as 4-byte prefix before username bytes, then `exp` as fixed 8 bytes.

#### 2.11 Server-side session revocation (C5)
- New SQLite table `sessions(id TEXT PRIMARY KEY, username TEXT, exp INTEGER)`
- Session token format: `sessionID.hmac(username,exp)` — sessionID is random 32-byte hex
- `Login`: insert session row
- `Logout`: delete session row by ID
- `Validate`: check session exists in DB + not expired + HMAC valid
- Add `sessions` table to `initSchema()`

#### 2.12 Graceful shutdown + timeouts (H8, H9)
In `cmd/server/main.go`:
```go
srv := &http.Server{
    Addr:              cfg.ListenAddr,
    Handler:           router,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       120 * time.Second,
}
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
go func() { srv.ListenAndServe() }()
<-ctx.Done()
shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
srv.Shutdown(shutdownCtx)
```
Also check `store.Close()` error.

## Layer 3 — Backend Performance & Correctness

### Files
- `internal/client/healthcheck.go`
- `internal/client/docker.go`
- `internal/handler/respond.go`
- `internal/config/config.go`
- `internal/auth/store.go` (SignCount column)
- `go.mod`

### Changes

#### 3.1 Concurrent health checks (H6)
In `healthcheck.go`: use `errgroup` or `sync.WaitGroup` to fan out `Check()` calls concurrently. Each check gets per-call context timeout (10s).

#### 3.2 Concurrent Docker log fetch (H7)
In `docker.go:117-125`: fan out `parseServerAddresses` concurrently per container with `errgroup`, per-call timeout.

#### 3.3 DialContext context propagation (M4)
`docker.go:31-33`: replace with `(&net.Dialer{Timeout: 5*time.Second}).DialContext(ctx, "unix", socketPath)`

#### 3.4 Docker response status check (M3)
`docker.go:58-61, 146-149`: check `resp.StatusCode` before decoding; return error if not 200.

#### 3.5 scanner.Err check (M10)
`docker.go:169-197`: check `scanner.Err()` after loop, return error if non-nil.

#### 3.6 stripANSI improvement (M9)
`docker.go:199-220`: replace manual parsing with regex:
```go
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b[()][AB0]`)
```

#### 3.7 Per-target health path (M5)
`healthcheck.go`: accept targets as `map[string]string` where value is full URL (including path), not just base URL. Update `config.go` to parse `name=url` pairs.

#### 3.8 respondJSON buffer (M2)
`respond.go:10-14`: encode to `bytes.Buffer`, then `w.WriteHeader(status)` + `w.Write(buf.Bytes())`.

#### 3.9 Dead code removal (M6)
Remove `Validate` function from `healthcheck.go:61-65`.

#### 3.10 getEnv with LookupEnv (L3)
`config.go:36-41`: use `os.LookupEnv` to distinguish unset from empty.

#### 3.11 go mod tidy (M1)
Run `go mod tidy` to fix `// indirect` markers on direct deps.

#### 3.12 Drop SignCount column (M7)
- `store.go`: remove `sign_count` from `User` struct, SQL schema, scan/insert
- `auth.go:139`: don't set `user.SignCount` after login
- Migration: `ALTER TABLE users DROP COLUMN sign_count` (or recreate table if SQLite doesn't support DROP COLUMN in older versions — use `CREATE TABLE users_new ...; INSERT; DROP; RENAME`)

## Layer 4 — Frontend Fixes

### Files
- `src/routes/+page.svelte`
- `src/routes/+layout.svelte`
- `src/lib/components/LoginCard.svelte`
- `src/lib/components/ThemeSwitcher.svelte`
- `src/lib/api.ts`
- `src/lib/stores.ts`
- `src/lib/types.ts`
- `src/lib/themes/global.css`
- `src/lib/themes/minimal.css`
- `src/lib/themes/glass.css`
- `src/app.html`
- `package.json`

### Changes

#### 4.1 loadDevices race fix (C1)
In `+page.svelte`:
- Add module-level `let loadSeq = 0` counter
- In `loadDevices`: `const seq = ++loadSeq; ... if (seq !== loadSeq) return;` before each `.set()`
- `onMount`: only call `loadDevices()` if `authStatus` is already known (add `authKnown` store flag set by `loadAuth`)
- Use `AbortController` per in-flight request, abort on new call + on destroy

#### 4.2 bufToB64 chunked (C2)
`LoginCard.svelte:132-137`:
```ts
function bufToB64(buf: ArrayBuffer): string {
    const bytes = new Uint8Array(buf);
    let binary = '';
    for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}
```

#### 4.3 addKey checkStatus (C3)
`LoginCard.svelte:61-75`: add `await checkStatus();` after `authAddKeyFinish()`.

#### 4.4 FOUC fix (H1)
`app.html`: add inline script in `<head>` before `%sveltekit.head%`:
```html
<script>
    const t = localStorage.getItem('theme');
    document.documentElement.setAttribute('data-theme',
        t === 'terminal' || t === 'minimal' || t === 'glass' ? t : 'terminal');
</script>
```

#### 4.5 Font loading (H2)
Move `@import url(...)` from theme CSS files to `app.html` `<link>` tags. Only load active theme's font. Use `<link rel="preload" as="style" onload="this.rel='stylesheet'">` pattern, or self-host fonts in `static/fonts/`.

#### 4.6 fetchJSON content-type check (H4)
`api.ts:5-11`: check `content-type` header before `res.json()`:
```ts
const ct = res.headers.get('content-type') ?? '';
if (!ct.includes('application/json')) throw new Error(`Expected JSON, got ${ct}`);
```

#### 4.7 Stale data on error (H5)
`+page.svelte`: change error blocks to `{#if $devicesError && $devices.length === 0}` — show stale data + error banner if data exists.

#### 4.8 logout error + disabled (M7, L8)
`LoginCard.svelte:52-59`: catch block sets `error = 'Logout failed'`. Add `disabled={loading}` to logout button.

#### 4.9 A11y (M8, M9, M10, L10)
- Error blocks: add `role="alert"`
- Logo: replace `<span role="button">` with native `<button>` (remove default styling)
- ThemeSwitcher: add `aria-pressed={$theme === t.id}`
- Auth buttons: add `aria-busy={loading}`

#### 4.10 prefers-reduced-motion (M4)
`global.css`:
```css
@media (prefers-reduced-motion: reduce) {
    *, *::before, *::after {
        animation-duration: 0.01ms !important;
        animation-iteration-count: 1 !important;
        transition-duration: 0.01ms !important;
    }
}
```

#### 4.11 Theme fixes (M1, M2, M3)
- `minimal.css:11`: `--color-text-secondary: #6a6a62`
- `glass.css:25`: `--font-mono: 'JetBrains Mono', monospace` (keep Outfit as `--font-body`)
- `glass.css`: add `--color-skeleton: rgba(255, 255, 255, 0.08)`, use in skeleton styles

#### 4.12 WebAuthn support check (M11)
`LoginCard.svelte:18`: guard all WebAuthn operations:
```ts
if (typeof PublicKeyCredential === 'undefined') {
    error = 'WebAuthn not supported in this browser';
    return;
}
```

#### 4.13 Typed credentials (M14, M15)
`types.ts`: add interfaces:
```ts
interface RegistrationCredentialJSON { id: string; rawId: string; type: string; response: { attestationObject: string; clientDataJSON: string }; }
interface AuthenticationCredentialJSON { id: string; rawId: string; type: string; response: { authenticatorData: string; clientDataJSON: string; signature: string; userHandle: string }; }
```
Update `api.ts` and `LoginCard.svelte` to use these instead of `unknown`.

#### 4.14 Error clear on re-show (L13)
`LoginCard.svelte`: `$effect` on `showAuth` — when it becomes true, clear `error`.

#### 4.15 Skeleton keys (L3)
`+page.svelte`: `{#each Array(4) as _, i (i)}`

#### 4.16 transition: all → specific (L5)
`ThemeSwitcher.svelte:47`, `LoginCard.svelte:216`: `transition: background 0.2s ease, color 0.2s ease, border-color 0.2s ease`

#### 4.17 Dead deps (L1)
`package.json`: remove `@sveltejs/adapter-auto`, `vite-plugin-resolve`. Run `yarn install`.

#### 4.18 Remove meta text-scale (L2)
`app.html:6`: remove `<meta name="text-scale" content="scale" />`

## Layer 5 — Infra & CI

### Files
- `docker-compose.yml`
- `backend/Dockerfile`
- `frontend/Dockerfile`
- `frontend/Caddyfile`
- `.github/workflows/deploy.yml`

### Changes

#### 5.1 Compose healthchecks (H1)
```yaml
backend:
  healthcheck:
    test: ["CMD", "wget", "-qO-", "http://localhost:8081/health"]
    interval: 10s
    timeout: 3s
    retries: 5
frontend:
  depends_on:
    backend:
      condition: service_healthy
```

#### 5.2 Non-root Dockerfiles (H2)
`backend/Dockerfile` final stage:
```dockerfile
RUN adduser -D -u 10001 app && chown -R app /app
USER app
```
`frontend/Dockerfile`: use `USER caddy` (ensure Caddy config/log paths writable).

#### 5.3 CGO_ENABLED=0 (M2)
`backend/Dockerfile:6`: `RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server ./cmd/server`

#### 5.4 Caddy security headers (H3, M1)
`frontend/Caddyfile`:
```caddyfile
header {
    Content-Security-Policy "default-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; script-src 'self'; frame-ancestors 'none';"
    X-Frame-Options "DENY"
    X-Content-Type-Options "nosniff"
    Referrer-Policy "strict-origin-when-cross-origin"
}
header /api/auth/* Cache-Control "no-store"
encode gzip zstd
```

#### 5.5 CI test gate (H5)
`deploy.yml`: add `test` job before `build-and-push`:
```yaml
test:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with: { go-version: '1.26' }
    - run: go vet ./...
    - run: go test ./...
    - uses: actions/setup-node@v4
      with: { node-version: '24' }
    - run: cd frontend && yarn install --immutable && yarn svelte-check --tsconfig ./tsconfig.json
```
`build-and-push`: add `needs: [test]`.

#### 5.6 CI environment protection (H6)
`deploy.yml`: `deploy` job add `environment: production`.

#### 5.7 BuildKit cache (M3)
`deploy.yml` build step: add `cache-from: type=gha`, `cache-to: type=gha,mode=max`.

#### 5.8 docker compose v2 (M7)
`deploy.yml:104-105`: replace `docker-compose` with `docker compose`.

#### 5.9 Deploy permissions (L6)
`deploy.yml`: add `permissions: { contents: read, deployments: write }` to `deploy` job.

#### 5.10 Image version pinning (M6)
- `frontend/Dockerfile:9`: `caddy:2.8-alpine` (or latest stable pinned)
- `backend/Dockerfile`: `golang:1.26-alpine`, `alpine:3.20` (or current stable)

## Layer 6 — Docs Sync

### Files
- `README.md`
- `.env.example`
- `backend/.env.example`

### Changes
- Consolidate `.env.example` (root) with `backend/.env.example` — root should include all vars
- Add comment: "Override WEBAUTHN_RP_ID/WEBAUTHN_ORIGIN for production" (CORS origin derived from WEBAUTHN_ORIGIN automatically)
- README: document `SESSION_SECRET` requirement for production
- README: note Docker socket security limitation (C1 infra — known trade-off, document mitigation: socket-proxy)

## Out of Scope

- Stores → `$state` runes refactor (M13 — large, optional)
- Structured logging / slog migration (L9)
- `svelte.config.js` runes function simplification (L16)
- Docker socket proxy (C1 infra — documented as known limitation)
- Tailscale API key rotation (user action)
- Git history purge of db files (user runs `git filter-repo` if needed)
- `+error.svelte` custom error page (L9 frontend)
- Dynamic document title (L14 frontend)
- `cacheMigrationMode` (L11 frontend — minor)

## Verification

Per layer:
- **Layers 2,3**: `cd backend && go vet ./... && go build -o /dev/null ./cmd/server`
- **Layer 4**: `cd frontend && yarn svelte-check --tsconfig ./tsconfig.json && yarn build`
- **Layer 5**: `docker compose config` (validate compose), `docker build` both Dockerfiles
- **Layer 1**: `git status` confirms db/idea untracked, `.gitignore` matches
- **All**: `git diff --stat` for scope check

## Execution Plan

Dispatch 6 subagents in parallel (after Layer 1 completes for Layer 2's benefit, but all can run concurrently since they touch different files):
- Agent A: Layer 1 (git hygiene) — quick
- Agent B: Layer 2 (backend auth/security)
- Agent C: Layer 3 (backend perf/correctness)
- Agent D: Layer 4 (frontend)
- Agent E: Layer 5 (infra/CI)
- Agent F: Layer 6 (docs) — runs after B, E complete (needs final env var names)

File ownership is non-overlapping between layers, so parallel execution is safe.