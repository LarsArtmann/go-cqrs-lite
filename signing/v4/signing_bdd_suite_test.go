package signing_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSigningBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Signing BDD Suite")
}
