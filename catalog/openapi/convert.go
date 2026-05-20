package openapi

import "github.com/larsartmann/go-cqrs-lite/catalog/internal/caseutil"

func toKebab(s string) string {
	return caseutil.ToKebab(s)
}

func toPascal(s string) string {
	return caseutil.ToPascal(s)
}
