# Releasing

Go modules are published by Git tags; there is no registry upload.

1. Update `Version` in `client.go` and move the "Unreleased" notes in `CHANGELOG.md` under the new
   version. Merge that to `main` through a pull request.
2. Create a GitHub release with the tag `v<version>` (for example `v0.1.0`) on `main`.
3. Ask the Go proxy to pick it up, so pkg.go.dev shows it within minutes:
   `GOPROXY=proxy.golang.org go list -m github.com/vaartix-mail/vaartix-go@v0.1.0`
   (or open https://pkg.go.dev/github.com/vaartix-mail/vaartix-go@v0.1.0 and click "Request").
