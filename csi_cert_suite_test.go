package csi_cert_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCsiCert(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CsiCert Suite")
}
