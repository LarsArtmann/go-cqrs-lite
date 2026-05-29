package query_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestQueryCoreBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Query Core BDD Suite")
}
