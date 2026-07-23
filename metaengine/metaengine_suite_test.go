package metaengine_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMetaengine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metaengine Suite")
}
