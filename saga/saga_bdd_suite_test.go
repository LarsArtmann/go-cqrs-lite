package saga_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSagaBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Saga BDD Suite")
}
