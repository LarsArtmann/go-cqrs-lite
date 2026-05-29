package stream_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestStreamBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stream BDD Suite")
}
