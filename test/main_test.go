package test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAllSuites(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "All Test Suites")
}
