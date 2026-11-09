package catalog_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCatalogBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Catalog BDD Suite")
}
