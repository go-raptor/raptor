# Changelog

## Unreleased

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
