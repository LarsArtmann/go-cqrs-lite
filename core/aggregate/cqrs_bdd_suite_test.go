package aggregate_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCQRSBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CQRS BDD Suite")
}
