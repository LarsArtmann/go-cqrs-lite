# import-cycle self-test fixture

`bash scripts/check-cycles.sh --self-test` feeds each `.graph` through the real
parse path via `PKGCYCLES_GRAPH` (a pre-captured `go list -f` dump). Shapes:

| Case       | Graph                    | Expected |
| ---------- | ------------------------ | -------- |
| acyclic    | a -> b -> c              | rc 0     |
| two-node   | a <-> b                  | rc 1     |
| three-node | a -> b -> c -> a         | rc 1     |
| empty      | generated at test time   | rc 1     |

The empty case is the important one: a shell gate whose loop never executes is
a green gate over nothing, so an empty graph must refuse rather than pass.
