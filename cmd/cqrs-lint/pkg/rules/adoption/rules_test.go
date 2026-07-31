package adoption_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/adoption"
)

var (
	_ = analyzer.BuildContextFromSource
	_ = ruletest.RunDetector
	_ = adoption.NewF001Detector
)
