# package-surface self-test fixture

Used by `bash scripts/check-package-surface.sh --self-test`, which copies this
directory four times and mutates one copy per case:

| Case     | Mutation                                   | Expected |
| -------- | ------------------------------------------ | -------- |
| clean    | none                                       | rc 0     |
| newgod   | 400 symbols appended for an unknown package | rc 1     |
| growth   | one symbol appended to baselined `big`     | rc 1     |
| shrink   | every `big` symbol removed                 | rc 0     |

The `.go` files exist so the production-file half of the ratchet has real input;
they live under `testdata/` so the Go toolchain never builds them.
