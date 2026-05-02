# Contributing

Thanks for taking a look at mdori.

## Development

Run the default checks before sending a change:

```sh
make good
```

That formats Go files, runs `go vet ./...`, and runs `go test ./...`.

To build the demo site locally:

```sh
make site
```

The generated `_site/` directory is ignored and should not be committed.

## Golden tests

Renderer fixtures live under `internal/mdori/testdata`.

If a rendering change intentionally updates the expected HTML, refresh the
golden files with:

```sh
UPDATE_GOLDEN=1 go test ./internal/mdori
```

Review the resulting diffs before committing them.

## Vendored browser assets

Browser runtime assets live under `internal/mdori/assets`. Keep third-party
license files next to the assets they cover, and update
`internal/mdori/assets/README.md` when changing vendored versions.
