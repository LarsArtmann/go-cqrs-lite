traced: not the journal — sqlite stores exact JSON strings. the loss is
the default CBOR codec (go-codec TimeUnixDynamic = float64 unix seconds,
~256ns/hop; measured 165ns single-hop e2e, your 611ns is multi-hop).
pinned by systemtest/time_fidelity_test.go (json codec leg exact 0ns),
ADR-0056 amended with tolerance guidance, upstream default flip filed:
larsartmann/go-codec#4. closing as traced+pinned, the real fix rides there
