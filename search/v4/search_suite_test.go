package search_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSearchBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Search Ladder Suite")
}
