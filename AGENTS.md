# AGENTS.md

## Commands

```bash
make build
make run
make test
make lint
go test -run TestHandleBucketsView ./internal/app/s3manager/  # single test
podman compose up  # two RustFS instances at localhost:9000 and :9002; app at localhost:8080
```

## Architecture

There is no database; S3 is the sole source of truth.

**Backend** (`main.go` + `internal/app/s3manager/`): Gorilla mux routes requests to handler factory functions. Each handler receives its dependencies (S3 client, templates, `Options`) via closure, not global state. `S3Instances` is the ordered, immutable list of configured instances, selected by a `{instance}` URL prefix that `WithInstance` resolves before handing the matching client to a single-instance handler; the few handlers that need more than the client (the two views and `HandleCheckPublicAccess`) take `S3Instances` and call `resolveInstance` themselves. The `S3` interface (`s3.go`) wraps the minio-go operations the handlers use. The feature flags the handlers care about are bundled into `s3manager.Options`, which `main.go` fills in; it includes the server side encryption, built once at start-up (`NewServerSideEncryption`) and also sent with reads so that SSE-C objects can be read back.

**Frontend** (`web/`): Server-rendered HTML using three Go templates (`layout.html.tmpl`, `buckets.html.tmpl`, `bucket.html.tmpl`) styled with [BeerCSS](https://www.beercss.com/) (Material Design 3). No JS framework — plain `fetch` plus BeerCSS's `ui()` helper for dialogs, menus and snackbars. `layout.html.tmpl` owns the page shell, the shared `appbar-actions` template (instance switcher, light/dark toggle) and the shared `toast`/`request` helpers; each page template renders its own `<header>` and `<main>`.

Static assets and templates are embedded into the binary via `//go:embed`, and BeerCSS is vendored rather than loaded from a CDN so the app works offline. The icon font is a subset — see `web/static/css/ICONS.md` before adding a new `<i>` icon.

### Non-obvious behavior

- `HandleBucketView` has two listing paths, picked by `cursorPagingPossible`: the default name-ascending view is listed one page at a time through S3's own `StartAfter`/`MaxKeys` paging (`listObjectPage`), which is why it has no total count; every other view (other sorts, search, `All`, versions) needs the whole prefix and falls back to a scan capped at `maxScanObjects` (`listAllObjects`).
- Objects are only deleted through the bulk endpoint. In the bulk endpoints, a key ending in `/` stands for the whole folder.
- Objects served inline get a content type that can't run scripts, and binary ones the browser can't display are downloaded instead (`inlineContentType`).
- `versionId` is honored only when `SHOW_VERSIONS` is on.
- The environment variables are documented in the README.

## Testing

Tests live alongside source files: each handler file has a corresponding `_test.go` in package `s3manager_test`, and the few tests of unexported helpers use `*_internal_test.go` in package `s3manager`. Handler tests use the `S3` mock in `mocks/s3.go`, which moq generates — regenerate with `go generate ./...` if the interface changes. Tests use `github.com/matryer/is` for assertions.
