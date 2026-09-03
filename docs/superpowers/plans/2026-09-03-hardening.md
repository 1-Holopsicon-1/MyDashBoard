# Security & Quality Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all Critical, High, and Medium issues from the code review across backend, frontend, and infra.

**Architecture:** Six parallel layers with non-overlapping file ownership. Layer 1 (git hygiene) first, then Layers 2-6 in parallel. Each layer is independently mergeable.

**Tech Stack:** Go 1.26 + chi, SvelteKit 5 (runes), SQLite (modernc.org/sqlite), Docker, Caddy, GitHub Actions

## Global Constraints

- Go backend: external deps limited to chi and go-webauthn (per CLAUDE.md)
- Frontend: Svelte 5 runes (`$props`, `$derived`, `$effect`), not legacy `export let`
- All components styled via CSS custom properties, not hardcoded values
- Never run git commands — user manages git manually (per CLAUDE.md)
- Yarn 4 with node-modules linker
- SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- No comments in code unless explicitly requested

---

## Task 1: Git Hygiene (Layer 1)

**Files:**
- Modify: `.gitignore`
- Create: `.gitattributes`
- Create: `backend/.dockerignore`

**Interfaces:**
- Consumes: nothing
- Produces: clean git tracking state for Layers 2-6

- [ ] **Step 1: Remove SQLite sidecars and .idea/ from git tracking**

Run:
```bash
cd /home/holopsicon/Projects/MyDashBoard
git rm --cached backend/dashboard.db-shm backend/dashboard.db-wal
git rm -r --cached .idea/
```
Expected: files staged for deletion from index, still on disk.

- [ ] **Step 2: Update .gitignore**

In `.gitignore`, replace all granular `.idea/**/...` rules (the block spanning ~lines 29-140) with a single line:
```
.idea/
```
Add these lines (in the appropriate section near existing `*.db` rule):
```
*.db-shm
*.db-wal
```

- [ ] **Step 3: Create .gitattributes**

Create `/home/holopsicon/Projects/MyDashBoard/.gitattributes`:
```
* text=auto eol=lf
*.db binary
*.db-shm binary
*.db-wal binary
```

- [ ] **Step 4: Create backend/.dockerignore**

Create `/home/holopsicon/Projects/MyDashBoard/backend/.dockerignore`:
```
*.db
*.db-shm
*.db-wal
.env
.air.toml
server
tmp/
```

- [ ] **Step 5: Verify**

Run: `git status`
Expected: `backend/dashboard.db-shm`, `backend/dashboard.db-wal`, `.idea/` show as deleted from tracking. New files `.gitattributes`, `backend/.dockerignore` show as untracked/new.

---

## Task 2: AuthRequired Middleware + CORS Hardening (Layer 2, part 1)

**Files:**
- Modify: `backend/internal/middleware/auth.go`
- Modify: `backend/internal/middleware/cors.go`
- Modify: `backend/internal/handler/router.go`
- Modify: `backend/internal/config/config.go`

**Interfaces:**
- Consumes: `IsAuthenticated` from `middleware/auth.go`, `cfg.WebAuthnOrigin` from `config.go`
- Produces: `AuthRequired` middleware function, hardened CORS middleware

- [ ] **Step 1: Read current files**

Read these files to understand current structure:
- `backend/internal/middleware/auth.go`
- `backend/internal/middleware/cors.go`
- `backend/internal/handler/router.go`
- `backend/internal/config/config.go`
- `backend/internal/handler/respond.go`

- [ ] **Step 2: Add AuthRequired middleware**

In `backend/internal/middleware/auth.go`, add:
```go
func AuthRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAuthenticated(r) {
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 3: Harden CORS middleware**

In `backend/internal/middleware/cors.go`, change the CORS middleware to accept an allowed origin parameter instead of `*`. The function signature becomes:
```go
func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowedOrigin == "" || origin == allowedOrigin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```
Note: if `allowedOrigin == ""`, allow any (dev mode fallback). In production, `cfg.WebAuthnOrigin` is set.

- [ ] **Step 4: Update router to use AuthRequired**

In `backend/internal/handler/router.go`, restructure routes so all `/api/*` data endpoints (tailscale, services, containers, simplex) are wrapped with `AuthRequired`. Auth endpoints and `/health` remain public. Example pattern:
```go
r.Route("/api", func(r chi.Router) {
	r.Use(mw.AuthRequired)
	r.Get("/tailscale/devices", h.TailscaleDevices)
	r.Get("/services", h.Services)
	r.Get("/containers", h.Containers)
	r.Get("/simplex/links", h.SimplexLinks)
})
r.Route("/api/auth", func(r chi.Router) {
	r.Post("/register/begin", h.RegisterBegin)
	r.Post("/register/finish", h.RegisterFinish)
	r.Post("/login/begin", h.LoginBegin)
	r.Post("/login/finish", h.LoginFinish)
	r.Post("/logout", h.Logout)
	r.Get("/status", h.AuthStatus)
	r.With(mw.AuthRequired).Post("/add-key/begin", h.AddKeyBegin)
	r.With(mw.AuthRequired).Post("/add-key/finish", h.AddKeyFinish)
})
```
Update the CORS middleware call in `main.go` or router setup to pass `cfg.WebAuthnOrigin`.

- [ ] **Step 5: Verify build**

Run: `cd backend && go vet ./... && go build -o /dev/null ./cmd/server`
Expected: no errors.

---

## Task 3: WebAuthn Temp Cookie HMAC + Registration Race + Session Secret (Layer 2, part 2)

**Files:**
- Modify: `backend/internal/handler/auth.go`
- Modify: `backend/internal/auth/session.go`
- Modify: `backend/internal/auth/store.go`

**Interfaces:**
- Consumes: `SessionManager` secret for HMAC, `Store` DB access
- Produces: signed temp cookie, atomic `CreateUserIfAbsent`, fail-fast session secret, server-side session revocation

- [ ] **Step 1: Read current files**

Read:
- `backend/internal/handler/auth.go`
- `backend/internal/auth/session.go`
- `backend/internal/auth/store.go`

- [ ] **Step 2: Add CreateUserIfAbsent to store.go**

In `backend/internal/auth/store.go`, add a method that atomically checks for existing user and creates one if absent, all under a single `Lock`:
```go
func (s *Store) CreateUserIfAbsent() (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return nil, fmt.Errorf("check existing users: %w", err)
	}
	if count > 0 {
		return s.getUserLocked()
	}
	id := make([]byte, 64)
	if _, err := rand.Read(id); err != nil {
		return nil, fmt.Errorf("generate user id: %w", err)
	}
	username := "admin"
	_, err := s.db.Exec("INSERT INTO users (id, username) VALUES (?, ?)", id, username)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &User{ID: id, Username: username}, nil
}
func (s *Store) getUserLocked() (*User, error) {
	u := &User{}
	err := s.db.QueryRow("SELECT id, username FROM users LIMIT 1").Scan(&u.ID, &u.Username)
	if err != nil {
		return nil, err
	}
	return u, nil
}
```
Add `getUserLocked()` if not present (private, called under lock). Update `RegisterBegin` in `auth.go` to call `CreateUserIfAbsent()` instead of `HasUser()` + `CreateUser()`.

- [ ] **Step 3: Fix HasUser to return error**

Change `HasUser()` signature to `HasUser() (bool, error)`. Check `row.Scan` error. Update all callers (`auth.go` `RegisterBegin`).

- [ ] **Step 4: Fix AddCredential error handling**

In `store.go` `AddCredential`, return errors from `json.Unmarshal` and `json.Marshal` instead of ignoring them:
```go
func (s *Store) AddCredential(userID []byte, cred *webauthn.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := &User{}
	if err := s.db.QueryRow("SELECT id, username, credentials FROM users WHERE id = ?", userID).Scan(&u.ID, &u.Username, &u.Credential); err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	var creds []webauthn.Credential
	if len(u.Credential) > 0 {
		if err := json.Unmarshal(u.Credential, &creds); err != nil {
			return fmt.Errorf("unmarshal credentials: %w", err)
		}
	}
	creds = append(creds, *cred)
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	_, err = s.db.Exec("UPDATE users SET credentials = ? WHERE id = ?", data, userID)
	return err
}
```

- [ ] **Step 5: Fix WebAuthnCredentials to log parse errors**

In `store.go` `WebAuthnCredentials`, if `json.Unmarshal` fails, log the error:
```go
if err := json.Unmarshal(u.Credential, &creds); err != nil {
	log.Printf("warning: failed to unmarshal credentials for user %s: %v", u.Username, err)
}
```

- [ ] **Step 6: HMAC-sign WebAuthn temp cookie**

In `handler/auth.go`, modify `setTempSession` and `getTempSession`. The `SessionManager` has a `secret` field — add a method `Sign(data []byte) string` and `Verify(data []byte, sig string) bool` to `SessionManager` (or reuse the HMAC logic). The temp cookie format becomes: `base64(json).base64(hmac)`. In `getTempSession`, verify HMAC before deserializing; if mismatch, return error.

- [ ] **Step 7: SESSION_SECRET fail-fast + rand.Read error check**

In `session.go` `NewSessionManager`:
```go
func NewSessionManager(secret string) (*SessionManager, error) {
	if secret == "" {
		// dev mode: generate random
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("generate session secret: %w", err)
		}
		secret = base64.StdEncoding.EncodeToString(b)
		log.Println("warning: SESSION_SECRET not set, generated random secret (sessions will not survive restart)")
	}
	return &SessionManager{secret: []byte(secret)}, nil
}
```
In `main.go`, if `cfg.WebAuthnRPID != "localhost" && cfg.SessionSecret == ""`, `log.Fatal("SESSION_SECRET must be set in production")`.

- [ ] **Step 8: Add SQLite pool settings**

In `store.go` after `sql.Open`:
```go
db.SetMaxOpenConns(1)
if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
	return nil, fmt.Errorf("set busy_timeout: %w", err)
}
if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
	return nil, fmt.Errorf("set journal_mode: %w", err)
}
```

- [ ] **Step 9: Add MaxBytesReader to auth POST handlers**

In `handler/auth.go`, at the start of each POST handler (`RegisterBegin`, `RegisterFinish`, `LoginBegin`, `LoginFinish`, `Logout`, `AddKeyBegin`, `AddKeyFinish`), wrap the body:
```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
```

- [ ] **Step 10: HMAC token length-prefixing**

In `session.go`, modify the HMAC signing to length-prefix fields:
```go
func (sm *SessionManager) sign(username string, exp int64) string {
	mac := hmac.New(sha256.New, sm.secret)
	mac.Write([]byte(fmt.Sprintf("%d:%s%d", len(username), username, exp)))
	// format: base64(username).exp.base64(hmac)
}
```
Use a separator format like `len:username` + `exp` to avoid ambiguity.

- [ ] **Step 11: Server-side session revocation table**

In `store.go` `initSchema`, add:
```sql
CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL,
	exp INTEGER NOT NULL
);
```
Add methods:
```go
func (s *Store) CreateSession(id, username string, exp int64) error
func (s *Store) DeleteSession(id string) error
func (s *Store) SessionExists(id string) (bool, error)
```
In `session.go`, change token format to include a session ID: `sessionID.base64(username).exp.hmac`. `Validate` checks: HMAC valid, not expired, session exists in DB. `Logout` calls `DeleteSession`.

- [ ] **Step 12: Verify build**

Run: `cd backend && go vet ./... && go build -o /dev/null ./cmd/server`
Expected: no errors.

---

## Task 4: Error Sanitization + Graceful Shutdown + Server Timeouts (Layer 2, part 3)

**Files:**
- Modify: `backend/internal/handler/respond.go`
- Modify: `backend/internal/handler/tailscale.go`
- Modify: `backend/internal/handler/containers.go`
- Modify: `backend/internal/handler/services.go`
- Modify: `backend/internal/handler/simplex.go`
- Modify: `backend/internal/handler/auth.go`
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes: `model.Error` struct
- Produces: sanitized error responses, graceful shutdown, server timeouts

- [ ] **Step 1: Read current respond.go and handler files**

Read `backend/internal/handler/respond.go` and all handler files to see current `respondError` usage.

- [ ] **Step 2: Rewrite respondError**

In `respond.go`:
```go
func respondError(w http.ResponseWriter, code int, err error) {
	log.Printf("request error [%d]: %v", code, err)
	respondJSON(w, code, model.Error{Error: http.StatusText(code)})
}
```
If `respondJSON` now returns error (from Task 6), handle it. Otherwise keep existing `respondJSON` signature.

- [ ] **Step 3: Update all respondError callers**

In each handler file, change `respondError(w, code, err.Error())` to `respondError(w, code, err)`. Files to update:
- `tailscale.go`
- `containers.go`
- `services.go`
- `simplex.go`
- `auth.go`

- [ ] **Step 4: Graceful shutdown + server timeouts in main.go**

In `cmd/server/main.go`, replace `http.ListenAndServe` with:
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
go func() {
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}()
<-ctx.Done()
log.Println("shutting down...")
shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
if err := srv.Shutdown(shutdownCtx); err != nil {
	log.Printf("shutdown error: %v", err)
}
if err := store.Close(); err != nil {
	log.Printf("store close error: %v", err)
}
```
Add imports: `context`, `os/signal`, `syscall`.

- [ ] **Step 5: Verify build**

Run: `cd backend && go vet ./... && go build -o /dev/null ./cmd/server`
Expected: no errors.

---

## Task 5: Backend Performance & Correctness (Layer 3)

**Files:**
- Modify: `backend/internal/client/healthcheck.go`
- Modify: `backend/internal/client/docker.go`
- Modify: `backend/internal/handler/respond.go`
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/auth/store.go`
- Modify: `backend/internal/handler/auth.go`
- Modify: `backend/go.mod`

**Interfaces:**
- Consumes: existing client/service structures
- Produces: concurrent health checks, concurrent Docker log fetch, fixed Docker client, improved ANSI stripping

- [ ] **Step 1: Read all target files**

Read:
- `backend/internal/client/healthcheck.go`
- `backend/internal/client/docker.go`
- `backend/internal/handler/respond.go`
- `backend/internal/config/config.go`
- `backend/internal/auth/store.go`
- `backend/internal/handler/auth.go`

- [ ] **Step 2: Concurrent health checks**

In `healthcheck.go`, change `CheckAll` (or equivalent method that loops over targets) to use `sync.WaitGroup` + concurrent goroutines:
```go
func (c *HealthChecker) CheckAll(ctx context.Context, targets map[string]string) []model.ServiceStatus {
	var mu sync.Mutex
	var wg sync.WaitGroup
	results := make([]model.ServiceStatus, 0, len(targets))
	for name, url := range targets {
		wg.Add(1)
		go func(name, url string) {
			defer wg.Done()
			res := c.Check(ctx, name, url)
			mu.Lock()
			results = append(results, res)
			mu.Unlock()
		}(name, url)
	}
	wg.Wait()
	return results
}
```

- [ ] **Step 3: Concurrent Docker log fetch**

In `docker.go`, change the loop that calls `parseServerAddresses` per container to use `errgroup` or `sync.WaitGroup`:
```go
var mu sync.Mutex
var wg sync.WaitGroup
for _, cont := range containers {
	wg.Add(1)
	go func(contID, contName string) {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		links := c.parseServerAddresses(ctx, contID, contName)
		mu.Lock()
		allLinks = append(allLinks, links...)
		mu.Unlock()
	}(cont.ID, cont.Name)
}
wg.Wait()
```

- [ ] **Step 4: Fix DialContext context propagation**

In `docker.go`, replace the `DialContext` func:
```go
DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return dialer.DialContext(ctx, "unix", socketPath)
},
```

- [ ] **Step 5: Check Docker response status codes**

In `docker.go`, after `c.client.Do(req)`, check `resp.StatusCode`:
```go
if resp.StatusCode != http.StatusOK {
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return nil, fmt.Errorf("docker API error: %d %s", resp.StatusCode, string(body))
}
```
Apply to both container list and log fetch.

- [ ] **Step 6: Check scanner.Err in parseAddressesFromLogs**

In `docker.go` `parseAddressesFromLogs`, after the scan loop:
```go
if err := scanner.Err(); err != nil {
	return nil, fmt.Errorf("read log stream: %w", err)
}
```

- [ ] **Step 7: Improve stripANSI with regex**

In `docker.go`, replace the manual `stripANSI` with:
```go
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*(\x07|\x1b\\)|\x1b[()][AB0]`)

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}
```
Add `"regexp"` to imports.

- [ ] **Step 8: Per-target health path**

In `healthcheck.go`, remove the hardcoded `/alive` path. The `targets` map values should be full URLs (including path). Update `Check` to use the URL as-is. In `config.go`, parse `name=full_url` format for health targets. If existing config uses base URLs only, append nothing — the caller provides the full URL.

- [ ] **Step 9: respondJSON buffer-first**

In `respond.go`:
```go
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(data); err != nil {
		log.Printf("json encode error: %v", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(buf.Bytes())
}
```
Add `"bytes"` to imports.

- [ ] **Step 10: Remove dead code**

In `healthcheck.go`, remove the `Validate` function (lines ~61-65) if it exists and is never called.

- [ ] **Step 11: getEnv with LookupEnv**

In `config.go`, change `getEnv`:
```go
func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
```

- [ ] **Step 12: Drop SignCount column**

In `store.go`:
- Remove `SignCount` field from `User` struct
- Remove `sign_count` from SELECT/INSERT/UPDATE queries
- Remove the scan into `&user.SignCount`
In `auth.go`:
- Remove `user.SignCount = ...` after `FinishLogin`
In `store.go` `initSchema`: add migration to drop the column. Since SQLite may not support `DROP COLUMN` in older versions, use safe migration:
```sql
CREATE TABLE IF NOT EXISTS users_new (id BLOB PRIMARY KEY, username TEXT NOT NULL UNIQUE, credentials BLOB);
INSERT OR IGNORE INTO users_new (id, username, credentials) SELECT id, username, credentials FROM users;
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;
```
Guard with a check: only run if `sign_count` column exists (PRAGMA table_info).

- [ ] **Step 13: Run go mod tidy**

Run: `cd backend && go mod tidy`

- [ ] **Step 14: Verify build**

Run: `cd backend && go vet ./... && go build -o /dev/null ./cmd/server`
Expected: no errors.

---

## Task 6: Frontend Critical Fixes (Layer 4, part 1)

**Files:**
- Modify: `frontend/src/routes/+page.svelte`
- Modify: `frontend/src/lib/components/LoginCard.svelte`
- Modify: `frontend/src/app.html`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/lib/types.ts`

**Interfaces:**
- Consumes: existing stores, api functions
- Produces: race-free data loading, FOUC-free theme, safe bufToB64, typed credentials

- [ ] **Step 1: Read all frontend files**

Read:
- `frontend/src/routes/+page.svelte`
- `frontend/src/lib/components/LoginCard.svelte`
- `frontend/src/lib/components/ThemeSwitcher.svelte`
- `frontend/src/lib/api.ts`
- `frontend/src/lib/stores.ts`
- `frontend/src/lib/types.ts`
- `frontend/src/app.html`

- [ ] **Step 2: Fix loadDevices race condition**

In `+page.svelte`:
1. Add a module-level sequence counter:
```ts
let loadSeq = 0;
```
2. In `loadDevices` (and similarly `loadServices`, `loadContainers`, `loadSimplex`):
```ts
async function loadDevices() {
	const seq = ++loadSeq;
	devicesLoading.set(true);
	devicesError.set(null);
	try {
		const data = await getDevices();
		if (seq !== loadSeq) return;
		devices.set(data);
		lastUpdated.set(new Date());
	} catch (e) {
		if (seq !== loadSeq) return;
		devicesError.set(e instanceof Error ? e.message : String(e));
	} finally {
		if (seq === loadSeq) devicesLoading.set(false);
	}
}
```
3. Use separate sequence counters per data type (`loadDevicesSeq`, `loadServicesSeq`, etc.)
4. In `onMount`, gate `loadDevices()` on auth being known:
```ts
let authKnown = false;
// after loadAuth() resolves:
authKnown = true;
await loadDevices();
```
Or simpler: remove the unconditional `loadDevices()` from `onMount` and rely on the `$effect` that fires when `authStatus.authenticated` changes. But ensure the effect also fires on initial load when already authenticated.

- [ ] **Step 3: Add AbortController on unmount**

In `+page.svelte`:
```ts
let abortController = new AbortController();
// pass signal to fetch calls in api.ts (update fetchJSON to accept optional signal)
onDestroy(() => {
	abortController.abort();
	clearInterval(devicesInterval);
	clearInterval(servicesInterval);
	clearInterval(containersInterval);
});
```
Update `api.ts` `fetchJSON` to accept an optional `AbortSignal`:
```ts
async function fetchJSON<T>(path: string, options?: RequestInit): Promise<T> {
	const res = await fetch(`${BASE}${path}`, options);
	// ...
}
```
Pass `{ signal }` in each api function.

- [ ] **Step 4: Fix bufToB64 chunked conversion**

In `LoginCard.svelte`, replace the `bufToB64` function:
```ts
function bufToB64(buf: ArrayBuffer): string {
	const bytes = new Uint8Array(buf);
	let binary = '';
	for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
	return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}
```

- [ ] **Step 5: Add checkStatus after addKey**

In `LoginCard.svelte` `addKey` function, after `authAddKeyFinish()`:
```ts
await checkStatus();
```

- [ ] **Step 6: FOUC fix — inline theme script in app.html**

In `app.html`, add this script in `<head>` before `%sveltekit.head%`:
```html
<script>
	const t = localStorage.getItem('theme');
	document.documentElement.setAttribute('data-theme',
		t === 'terminal' || t === 'minimal' || t === 'glass' ? t : 'terminal');
</script>
```

- [ ] **Step 7: fetchJSON content-type check**

In `api.ts` `fetchJSON`:
```ts
async function fetchJSON<T>(path: string, options?: RequestInit): Promise<T> {
	const res = await fetch(`${BASE}${path}`, options);
	if (!res.ok) throw new Error(`API error: ${res.status} ${res.statusText}`);
	const ct = res.headers.get('content-type') ?? '';
	if (!ct.includes('application/json')) throw new Error(`Expected JSON, got ${ct}`);
	return res.json() as Promise<T>;
}
```

- [ ] **Step 8: Add typed credential interfaces**

In `types.ts`, add:
```ts
export interface RegistrationCredentialJSON {
	id: string;
	rawId: string;
	type: string;
	response: {
		attestationObject: string;
		clientDataJSON: string;
	};
}

export interface AuthenticationCredentialJSON {
	id: string;
	rawId: string;
	type: string;
	response: {
		authenticatorData: string;
		clientDataJSON: string;
		signature: string;
		userHandle: string | null;
	};
}
```
Update `api.ts` functions `authRegisterFinish`, `authLoginFinish`, `authAddKeyFinish` to accept these types instead of `unknown`.

- [ ] **Step 9: Remove meta text-scale**

In `app.html`, remove:
```html
<meta name="text-scale" content="scale" />
```

- [ ] **Step 10: Verify typecheck**

Run: `cd frontend && yarn svelte-check --tsconfig ./tsconfig.json`
Expected: no errors.

---

## Task 7: Frontend Medium Fixes (Layer 4, part 2)

**Files:**
- Modify: `frontend/src/routes/+page.svelte`
- Modify: `frontend/src/lib/components/LoginCard.svelte`
- Modify: `frontend/src/lib/components/ThemeSwitcher.svelte`
- Modify: `frontend/src/lib/themes/global.css`
- Modify: `frontend/src/lib/themes/minimal.css`
- Modify: `frontend/src/lib/themes/glass.css`
- Modify: `frontend/src/lib/themes/terminal.css`
- Modify: `frontend/src/lib/stores.ts`
- Modify: `frontend/package.json`

**Interfaces:**
- Consumes: existing theme structure, stores
- Produces: a11y improvements, theme fixes, dead code removal

- [ ] **Step 1: Read current theme and component files**

Read all files listed above.

- [ ] **Step 2: Stale data on error**

In `+page.svelte`, change all error display blocks from:
```svelte
{#if $devicesError}
	<div class="error-block">...
```
to:
```svelte
{#if $devicesError && $devices.length === 0}
	<div class="error-block" role="alert">...
{:else if $devicesLoading && $devices.length === 0}
	<!-- skeletons -->
{:else}
	<div class="grid">...</div>
	{#if $devicesError}<div class="error-banner" role="alert">{$devicesError}</div>{/if}
{/if}
```
Apply same pattern to services, containers, simplex sections.

- [ ] **Step 3: logout error + disabled**

In `LoginCard.svelte` `logout`:
```ts
async function logout() {
	try {
		await authLogout();
		await checkStatus();
	} catch {
		error = 'Logout failed — session may still be active';
	}
}
```
Add `disabled={loading}` to the logout button.

- [ ] **Step 4: A11y fixes**

In `LoginCard.svelte`:
- Error containers: add `role="alert"`
- Auth buttons: add `aria-busy={loading}`

In `+page.svelte`:
- Logo: replace `<span role="button" tabindex="0">` with `<button class="logo" onclick={handleLogoClick}>`. Add CSS to reset button defaults:
```css
.logo {
	background: none;
	border: none;
	padding: 0;
	font: inherit;
	color: inherit;
	cursor: pointer;
}
```

In `ThemeSwitcher.svelte`:
- Add `aria-pressed={$theme === t.id}` to each theme button

- [ ] **Step 5: prefers-reduced-motion**

In `global.css`, add at the end:
```css
@media (prefers-reduced-motion: reduce) {
	*, *::before, *::after {
		animation-duration: 0.01ms !important;
		animation-iteration-count: 1 !important;
		transition-duration: 0.01ms !important;
	}
}
```

- [ ] **Step 6: Theme color fixes**

In `minimal.css`:
- Change `--color-text-secondary: #888880` to `--color-text-secondary: #6a6a62`

In `glass.css`:
- Change `--font-mono: 'Outfit', sans-serif` to `--font-mono: 'JetBrains Mono', monospace`
- Keep `--font-body: 'Outfit', sans-serif` (or add it if not present)
- Add `--color-skeleton: rgba(255, 255, 255, 0.08)` and use it in skeleton styles (or update `--color-surface` for skeletons)

- [ ] **Step 7: Font loading optimization**

In each theme CSS file (`terminal.css`, `minimal.css`, `glass.css`), remove the `@import url('https://fonts.googleapis.com/...')` line. In `app.html` `<head>`, add conditional font loading — but since theme is set before render (Step 6 of Task 6), we can't conditionally load in HTML. Instead, move all font imports to a single `@import` in `global.css` but use `font-display: swap` (add `&display=swap` to Google Fonts URLs). This is the pragmatic fix — full conditional loading requires self-hosting.

Alternatively, self-host fonts in `frontend/static/fonts/` and `@font-face` with `font-display: swap`. If time-constrained, just add `&display=swap` to existing `@import` URLs.

- [ ] **Step 8: WebAuthn support check**

In `LoginCard.svelte`, at the start of `register()`, `login()`, `addKey()`:
```ts
if (typeof PublicKeyCredential === 'undefined') {
	error = 'WebAuthn is not supported in this browser';
	return;
}
```

- [ ] **Step 9: Error clear on re-show**

In `LoginCard.svelte`:
```ts
$effect(() => {
	if (showAuth) error = '';
});
```

- [ ] **Step 10: Skeleton keys**

In `+page.svelte`, change `{#each Array(4) as _}` to `{#each Array(4) as _, i (i)}`.

- [ ] **Step 11: transition: all → specific**

In `ThemeSwitcher.svelte` and `LoginCard.svelte`:
- Replace `transition: all 0.2s ease` with `transition: background 0.2s ease, color 0.2s ease, border-color 0.2s ease`

- [ ] **Step 12: Remove dead deps**

In `package.json`, remove:
- `@sveltejs/adapter-auto`
- `vite-plugin-resolve`
Run: `cd frontend && yarn install`

- [ ] **Step 13: Verify typecheck + build**

Run: `cd frontend && yarn svelte-check --tsconfig ./tsconfig.json && yarn build`
Expected: no errors.

---

## Task 8: Infra & CI (Layer 5)

**Files:**
- Modify: `docker-compose.yml`
- Modify: `backend/Dockerfile`
- Modify: `frontend/Dockerfile`
- Modify: `frontend/Caddyfile`
- Modify: `.github/workflows/deploy.yml`

**Interfaces:**
- Consumes: nothing
- Produces: hardened Docker, Caddy, CI

- [ ] **Step 1: Read all infra files**

Read:
- `docker-compose.yml`
- `backend/Dockerfile`
- `frontend/Dockerfile`
- `frontend/Caddyfile`
- `.github/workflows/deploy.yml`

- [ ] **Step 2: Compose healthchecks**

In `docker-compose.yml`:
```yaml
services:
  backend:
    image: ghcr.io/1-holopsicon-1/mydashboard/backend:latest
    env_file: .env
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - dashboard-db:/app/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8081/health"]
      interval: 10s
      timeout: 3s
      retries: 5
  frontend:
    image: ghcr.io/1-holopsicon-1/mydashboard/frontend:latest
    ports:
      - "8090:8080"
    depends_on:
      backend:
        condition: service_healthy
    restart: unless-stopped
volumes:
  dashboard-db:
```

- [ ] **Step 3: Backend Dockerfile — non-root + CGO_ENABLED=0**

In `backend/Dockerfile`, update the build stage and final stage:
```dockerfile
# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server ./cmd/server

# Final stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=builder /server .
RUN adduser -D -u 10001 app && chown -R app /app
USER app
EXPOSE 8081
CMD ["./server"]
```

- [ ] **Step 4: Frontend Dockerfile — non-root**

In `frontend/Dockerfile`, ensure Caddy runs as non-root. If using `caddy:alpine`:
```dockerfile
FROM node:24-alpine AS builder
WORKDIR /app
COPY package.json yarn.lock .yarnrc.yml ./
COPY .yarn .yarn
RUN yarn install --immutable
COPY . .
RUN yarn build

FROM caddy:2.8-alpine
COPY Caddyfile /etc/caddy/Caddyfile
COPY --from=builder /app/build /srv
EXPOSE 8080
USER caddy
```
Note: verify Caddy config paths are writable by `caddy` user. If not, adjust `Caddyfile` to use `/tmp/caddy` for admin endpoint or disable admin.

- [ ] **Step 5: Caddy security headers**

In `frontend/Caddyfile`:
```caddyfile
:8080 {
	header {
		Content-Security-Policy "default-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; script-src 'self'; frame-ancestors 'none';"
		X-Frame-Options "DENY"
		X-Content-Type-Options "nosniff"
		Referrer-Policy "strict-origin-when-cross-origin"
	}
	header /api/auth/* Cache-Control "no-store"
	encode gzip zstd

	handle /api/* {
		reverse_proxy backend:8081
	}
	handle {
		root * /srv
		try_files {path} /index.html
		file_server
	}
}
```

- [ ] **Step 6: CI test gate + environment + cache**

In `.github/workflows/deploy.yml`, add a `test` job before `build-and-push`:
```yaml
test:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.26'
    - run: cd backend && go vet ./...
    - run: cd backend && go test ./...
    - uses: actions/setup-node@v4
      with:
        node-version: '24'
    - run: cd frontend && yarn install --immutable
    - run: cd frontend && yarn svelte-check --tsconfig ./tsconfig.json
```
In `build-and-push` job, add `needs: [test]`.
In build steps, add `cache-from: type=gha` and `cache-to: type=gha,mode=max`.
In `deploy` job, add:
```yaml
deploy:
  needs: [build-and-push]
  environment: production
  permissions:
    contents: read
    deployments: write
```

- [ ] **Step 7: docker compose v2**

In `.github/workflows/deploy.yml`, replace `docker-compose` with `docker compose` (v2 plugin).

- [ ] **Step 8: Verify**

Run: `docker compose config`
Expected: valid compose file, no errors.

---

## Task 9: Docs Sync (Layer 6)

**Files:**
- Modify: `README.md`
- Modify: `.env.example`
- Modify: `backend/.env.example`

**Interfaces:**
- Consumes: final env var names from Tasks 2-5
- Produces: accurate documentation

- [ ] **Step 1: Read current docs**

Read `README.md`, `.env.example`, `backend/.env.example`.

- [ ] **Step 2: Consolidate .env.example**

Ensure root `.env.example` includes ALL variables (mirror `backend/.env.example`):
- `LISTEN_ADDR`
- `TAILSCALE_API_KEY`
- `TAILSCALE_TAILNET`
- `VAULTWARDEN_URL`
- `DOCKER_SOCKET`
- `CONTAINER_FILTERS`
- `SESSION_SECRET` (with comment: "Required in production. Auto-generated in dev.")
- `WEBAUTHN_RP_ID` (with comment: "Your domain for production")
- `WEBAUTHN_ORIGIN` (with comment: "Your origin URL for production. CORS origin derived from this.")
- `DB_PATH`

Remove duplicate/contradictory entries between root and backend `.env.example`.

- [ ] **Step 3: Update README**

In `README.md`:
- Add note under Auth section: "In production, `SESSION_SECRET` must be set explicitly — otherwise the server will refuse to start."
- Add note: "CORS allowed origin is automatically derived from `WEBAUTHN_ORIGIN`."
- Add Docker socket security note: "The backend mounts `/var/run/docker.sock` for container monitoring. This grants host-level access. For hardened deployments, consider [docker-socket-proxy](https://github.com/Tecnativa/docker-socket-proxy) to expose only read-only endpoints."
- Ensure env vars table matches the consolidated `.env.example`

- [ ] **Step 4: Verify consistency**

Cross-check that every env var in `.env.example` is documented in README's env table, and vice versa.

---

## Verification Summary

After all tasks complete, run full verification:

```bash
cd backend && go vet ./... && go build -o /dev/null ./cmd/server
cd frontend && yarn svelte-check --tsconfig ./tsconfig.json && yarn build
docker compose config
git status  # confirm db files untracked, .idea untracked
git diff --stat  # scope check
```