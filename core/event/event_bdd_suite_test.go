package event_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEventBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Event BDD Suite")
}
