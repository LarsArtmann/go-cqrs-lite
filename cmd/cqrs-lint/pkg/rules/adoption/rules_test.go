package adoption_test

import (
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

var (
	_ = analyzer.BuildContextFromSource
	_ = ruletest.RunDetector
	_ = adoption.NewF001Detector
)
