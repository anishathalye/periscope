# Development

Periscope uses the standard [Go toolchain][golang] for development.

[golang]: https://go.dev/

## Testing

You can run the tests with:

```bash
go test ./...
```

## Formatting

You can run the code formatter with:

```bash
go fmt ./...
```

## Static analysis

You can run Go's built-in `vet` tool with:

```bash
go vet ./...
```

This project additionally uses the [staticcheck] tool. You can install it with:

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

You can run staticcheck with:

```bash
staticcheck -f stylish ./...
```

[staticcheck]: https://staticcheck.dev/

## Building and installing

You can build and install the `psc` binary locally with `go install ./cmd/psc`.

## Continuous integration

Testing and static analysis is [run in CI][ci-test]. Additionally, building and publishing binaries is [run in CI][ci-release].

[ci-test]: .github/workflows/ci.yml
[ci-release]: .github/workflows/release.yml
