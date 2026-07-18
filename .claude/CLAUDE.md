# go-isnow

The isnow date/time pattern language as an importable Go library: `Parse` a pattern into an immutable `Pattern`, test membership with `Holds`/`Is`, and derive occurrences with `Next`/`Prev`. This is the library split out of [tsvsheet/isnow.go](https://github.com/tsvsheet/isnow.go), which remains the CLI consuming it; the language itself (grammar, specification, conformance corpus) lives in [tsvsheet/isnow](https://github.com/tsvsheet/isnow).

- The public surface is intentionally tiny — `Parse`, `Pattern` and its query methods, `Is`, `Code`, and the four cross-implementation error sentinels. Everything else stays unexported; do not widen the API without a real external consumer.
- `internal/isnowgrammar/` is the committed, machine-generated ANTLR parser tree, regenerated via `make grammars` from the sibling grammar repo (needs Docker). Never hand-edit it; it is excluded from the vet/staticcheck/coverage gates in [Makefile.local](Makefile.local).
- The conformance corpus in the sibling `tsvsheet/isnow` repo is the behavior oracle; `make check` fetches it in CI. Language semantics are corpus-frozen — a semantics change starts in the grammar repo, never here.
- `Makefile`, `.golangci.yaml`, `.editorconfig`, `.gitignore`, and `.github/` are distributed by nicerobot/tools.repository — never edit them in-tree; repo-local divergence goes in `Makefile.local`.
- Docs live in [tsvsheet/docs.go-isnow](https://github.com/tsvsheet/docs.go-isnow); this README stays exactly badges + docs link.
