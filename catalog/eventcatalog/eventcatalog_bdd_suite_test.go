package eventcatalog_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEventCatalogBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EventCatalog BDD Suite")
}
