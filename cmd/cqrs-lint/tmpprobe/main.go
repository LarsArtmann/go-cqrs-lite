package main

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

func main() {
	actx, err := analyzer.BuildContext("/tmp/cqrsfix")
	fmt.Printf("err=%v\n", err)
	if actx != nil {
		fmt.Printf("packages=%d gofiles=%d loaderrors=%d\n", len(actx.Packages), len(actx.GoFiles), len(actx.LoadErrors))
		for i, le := range actx.LoadErrors {
			if i > 2 {
				break
			}
			fmt.Printf("  loaderror: module=%q pkg=%q errs=%v\n", le.Module, le.PkgPath, le.Errors)
		}
		for _, p := range actx.Packages {
			fmt.Printf("  pkg=%s errors=%d\n", p.PkgPath, len(p.Errors))
			for j, e := range p.Errors {
				if j > 1 {
					break
				}
				fmt.Printf("    %v\n", e)
			}
		}
	}
}
