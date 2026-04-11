When you change Go code, before stopping, test what you changed, run go fmt, go fix, and golangci-lint.

When you need context in tests, use t.Context().

When debugging, generating random files, logs, or doing whatever temporary stuff, use the debug directory.

Do not use dot imports (`. "maragu.dev/gomponents/html"`) — the linter forbids them. Use qualified imports (`"maragu.dev/gomponents/html"` then `html.P(...)`) instead.

The gomponents `html` package exports a `Nav` function, so avoid naming your own functions `Nav` in packages that import it.

`html.StyleAttr` is deprecated. Use `html.Style` instead.

Constructor functions that perform I/O (e.g. network calls, auth) should accept a `context.Context` parameter. In cobra commands, use `cmd.Context()` as the parent context rather than `context.Background()`.
