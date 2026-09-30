# Changelog

## Unreleased

### Added

- `Context.ParamInt64` and `Context.QueryInt64` parse numeric path and query parameters, returning a `400` `errs.Error` that names the parameter when it's missing, malformed or out of range.
- `Config.AppString`, `AppInt`, `AppInt64`, `AppBool` and `AppDuration` read the `app:` section with a default for a missing or empty key, and return an error naming the key for a malformed value.
- `Resources.AppContext()` is cancelled once in-flight requests have drained, before services clean up, so background work and database calls can stop on shutdown. `Resources.ShuttingDown()` reports that shutdown has begun.
- A built-in `HealthController` (registered unless the app has its own). `Live` answers `{"status":"ok"}`. `Ready` answers `503` once shutdown begins, or when a connector that implements `Ping(context.Context) error` can't reach the database within 2 seconds. Route them yourself, e.g. `/healthz` and `/readyz`.
- Test helpers `raptor.JSONBody`, `raptor.DecodeJSON[T]` and `raptor.WithCookie`.
- Raptor's "Unhandled error in handler" and "Panic recovered in handler" lines carry `request_id` when the requestid middleware has set one.

## v4.5.0 — 2026-09-30

### Upgrading

- `Bind` and `BindWith` require `Content-Type: application/json` or an `application/*+json` type whenever the request has a body; other bodies get `415`. A request without a body skips the check and decodes as empty, as before. Clients posting JSON without the header (e.g. `curl -d` without `-H 'Content-Type: application/json'`) must add it. To accept any body, decode `ctx.Request().Body` yourself.
- Malformed JSON in `Bind`/`BindWith` is a `400` (an `errs.Error` wrapping the decode error) instead of a logged `500`.
- `File`, `FileFromDir`, `Attachment` and `Inline` return `errs.ErrNotFound` for a missing path, a directory or any non-regular file instead of writing a bare 404 and returning nil. The body changes from empty to `{"code":404,"message":"Not Found"}`. Return the error, or handle it with `errors.Is(err, errs.ErrNotFound)`: a handler that ignores it (`ctx.File(p); return nil`) now answers an empty `200` for a missing file, where it used to answer `404`.
- Error responses carry `Cache-Control: no-store`, `Content-Type: application/json` and `X-Content-Type-Options: nosniff`, and drop `ETag`, `Last-Modified`, `Expires`, `CDN-Cache-Control`, `Surrogate-Control`, `Content-Length` and `Content-Disposition`. `Content-Encoding` is dropped too, unless a middleware substituted the writer (a compressing middleware encodes the error as well and keeps it). Headers describing the error itself, such as `Retry-After`, `Allow` and `WWW-Authenticate`, are kept. To send a cacheable error, write it yourself with `ctx.JSON`.
- An `errs.Error` with a status outside 400–599 is sent as a `500` and logged.
- A `*raptor.Context` used after its handler returned has no request or response writer: `Request()` returns nil, and most methods (`Param`, `QueryParam`, `Cookie`, `RealIP`, `Bind`, `JSON`, …) panic with a nil pointer dereference. In a goroutine that panic is unrecovered and ends the process; before, such code silently read or wrote another request's data. Pass goroutines what they need (`ctx.Request().Context()`, parsed values), never the Context.

### Security

- Error responses kept a `Content-Type` the handler had set, e.g. `text/html`, while the message may echo request input (the built-in 404 echoes the path). They are now always JSON with `nosniff`.
- Caching headers set for a successful response leaked onto errors, so a CDN could cache a 404 for as long as the file would have been cached. Errors are now `no-store`, and `Expires`, `CDN-Cache-Control` and `Surrogate-Control` are dropped along with `Cache-Control`'s old value.
- Multipart temp files were left on disk when a middleware replaced the request (`ctx.SetRequest(r.WithContext(...))`) before the form was parsed. Raptor now removes them.
- File serving skips FIFOs, sockets and devices; a FIFO in a served directory could hang the request.
- `Attachment` and `Inline` encode the filename per RFC 6266 (`filename*` for non-ASCII), so the header never carries raw non-ASCII bytes or line breaks.
- Config values whose keys contain `pass`, `credential`, `private`, `salt` or `dsn` are masked in startup logs.

### Performance

- `core.Response` implements `io.ReaderFrom`, so `File`, `FileFromDir`, `FileFromRoot`, `Stream` and `http.ServeContent(ctx.Response(), …)` use net/http's sendfile path: an 8 MB file over loopback takes 35% less time (2.78 ms → 1.80 ms) and 82% fewer bytes allocated.
- The 404/405 fallback probes only methods that routes use: 4.9 µs → 3.3 µs and 52 → 33 allocs per unmatched request in the benchmark app.
- `ctx.RealIP()` is computed once per request instead of once per caller.
- JSON responses encode into pooled buffers: one allocation fewer per response.

### Added

- `Context.FileFromRoot(root *os.Root, name string)` serves from a root opened once, e.g. in `Setup`.

### Fixed

- `Run` shuts down services and closes the database when the server fails, instead of exiting from the serve goroutine. A second Ctrl-C during shutdown exits immediately.

### Docs

- README: why `read_timeout`/`write_timeout` default to off, and per-handler deadlines with `http.ResponseController`.

## v4.4.0 — 2026-09-25

### Fixed

- An `errs.Error` whose attrs cannot be encoded (an `error` value, a `time.Duration`, a func or a channel) no longer answers an empty `200`. The response is retried without `attrs`, keeping the status and message; invalid UTF-8 in the message (the built-in 404 echoes the request path, so a scanner probing `/x%ff` triggers this) is replaced with U+FFFD rather than turning the error into a 500. A pre-encoded `{"code":500,"message":"Internal Server Error"}` remains the last resort. Encoding failures are logged with the original error.
- `raptor.NewTestApp(..., raptor.WithConfig(&config.Config{AppConfig: ...}))` no longer panics with "assignment to entry in nil map".

### Added

- `DatabaseConfig.SSLMode` (`database.ssl_mode`, `DATABASE_SSL_MODE`), passed to the Postgres connectors as `sslmode`. **Default: `prefer`** — TLS when the server offers it, plaintext otherwise, so local databases keep working and managed ones get encryption. Set `disable` for the previous behavior, or `verify-full` for managed databases. Requires connectors `pgx` / `bun/postgres` v1.2.0+; older connectors ignore it and keep `sslmode=disable`.
- `Context.BindWith(v, opts...)` decodes with `encoding/json/v2` options, e.g. `json.RejectUnknownMembers(true)`.
- `raptor.WithRemoteAddr(addr)` test request option. Every httptest request otherwise comes from `192.0.2.1:1234`, so a suite shares one client IP and one bucket in any per-IP middleware. The port is optional, and IPv6 addresses may be bracketed or bare.

### Docs

- README: current version and Go 1.27 requirement; `encoding/json/v2` semantics of `Bind`/`Data`; test helpers with a two-user example (compiled in `v4/example_test.go`); file serving and why request-derived paths go through `FileFromDir`; catch-all routes on `/` replacing the 404/405 fallback; `ssl_mode`.

## v4.3.2 — 2026-08-22

### Changed

- Requires Go 1.27. `Bind`, `Data` and `JSON` now use `encoding/json/v2`:
  - member names match case-sensitively; a mis-cased key is silently dropped;
  - duplicate member names, invalid UTF-8 and trailing data are rejected;
  - nil slices and maps encode as `[]` and `{}`;
  - `omitempty` omits empty JSON values; `omitzero` omits Go zero values;
  - `time.Duration` (no default encoding) and `format:` tag options fail.

## v4.3.1 — 2026-08-15

### Changed

- YAML parsing moved from `gopkg.in/yaml.v3` to its maintained fork `go.yaml.in/yaml/v3`. Config files are unaffected.

## v4.3.0 — 2026-07-18

### Security

- 500 responses no longer echo internal error strings or panic values to clients. Deliberate `errs.*` errors keep their messages; everything else is redacted to a generic body and logged server-side — panics now with a full stack trace. `http.ErrAbortHandler` is re-panicked per the `net/http` contract.
- Request bodies are limited to **8 MB by default** (`server.max_body_bytes`, `SERVER_MAX_BODY_BYTES`; explicit `0` disables). The limit is enforced centrally, covering JSON binding, form/multipart parsing, and wrapped `net/http` handlers; oversized bodies return `413`.
- New `Context.FileFromDir(dir, name)` serves files confined to a root directory via `os.Root`, immune to `..` traversal, absolute paths, and symlink escapes. `Context.File` is documented as unsafe for untrusted input.
- Client-IP extraction hardened: the all-trusted X-Forwarded-For fallback now returns a validated, normalized IP, and `server.trusted_proxies` (`SERVER_TRUSTED_PROXIES`, CIDR list) lets you trust load balancers outside the default loopback/link-local/private ranges. Invalid entries fail startup.
- Configuration logging masks URL-embedded credentials (e.g. DSNs) in addition to password/token/secret-like keys.

### Fixed

- **Startup panic** when two routes shared a path shape with different parameter names (e.g. `GET /things/{id}` + `POST /things/{slug}`).
- 404/405 handling no longer shadows wildcard or `ANY` routes: `GET /users/search` now reaches `GET /users/{id}` when only `POST /users/search` exists, and `ANY` routes receive every method. 405 responses still carry a correct `Allow` header, now computed by probing the router.
- `UseStd` middleware that substitutes the `http.ResponseWriter` (compression, metrics wrappers) is no longer silently bypassed.
- `Setup()` hooks run **after** dependency injection for services, controllers, and middlewares, so injected fields are usable during setup.
- Dependency injection: duplicate service type names across packages are rejected at startup instead of silently overwriting; fields are matched by exact type; unexported fields of a service type produce a clear error instead of a reflect panic.
- `errs.Error.WithCause` returns a copy instead of mutating shared sentinels like `errs.ErrNotFound` (data race and cross-request contamination).
- Route method `"*"` now behaves as `ANY` instead of registering a dead pattern.
- Routes YAML: invalid entries (non-string method handlers, method-named keys with nested maps, invalid path values) now return errors instead of being silently dropped; keys parse in sorted order so route lists are deterministic.
- Graceful shutdown drains in-flight requests **before** tearing down services, and closes the database connector when it implements `io.Closer`.
- `http.Server` errors are routed to `slog` via `ErrorLog`; the startup banner prints only after the listener is successfully bound, and bind errors surface synchronously.

### Added

- `raptor.Use`, `raptor.UseOnly`, `raptor.UseExcept` aliases (previously only reachable via the `core` package).
- `Raptor.Shutdown()` for programmatic graceful shutdown.
- `Server.Listen()`/`Server.Serve()` split; `Server.Address()` reports the actually bound address (useful with port `0`).
- `Content-Length` set on buffered responses larger than 2 KB (JSON, string, blob), avoiding chunked encoding; smaller responses already get it from `net/http`.
- First test suite for the framework (router, DI lifecycle, middleware, context, IP extraction, config, errs) and request benchmarks.

### Changed

- **Behavior:** default `max_body_bytes` is now 8 MB (was unlimited). Explicitly configured values, including `0`, are honored.
- **Behavior:** non-`errs.Error` errors returned from handlers produce a generic 500 body (previously the raw error string).
- **API:** `server.NewServer` takes a `*slog.Logger` third argument.
- Config loading warns when dev and prod files are both present (dev wins) and when environment variable values fail to parse.
