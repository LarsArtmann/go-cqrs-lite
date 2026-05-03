package projection_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestProjectionBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Projection BDD Suite")
}
