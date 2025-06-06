package dcb

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDCB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DCB Test Suite")
}
