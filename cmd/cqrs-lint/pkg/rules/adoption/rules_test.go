package adoption_test

import (
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/adoption"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

var (
	_ = analyzer.BuildContextFromSource
	_ = ruletest.RunDetector
	_ = adoption.NewF001Detector
)
