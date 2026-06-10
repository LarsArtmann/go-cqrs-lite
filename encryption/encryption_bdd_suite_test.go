package encryption_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEncryptionBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Encryption BDD Suite")
}
