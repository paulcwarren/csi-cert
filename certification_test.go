package csi_cert_test

import (
	"os"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"

	csi "github.com/jeffpak/csi"
	csi_cert "github.com/paulcwarren/csi-cert"
)

var _ = Describe("Certify with: ", func() {

	var (
		testLogger  lager.Logger
		clientConn  *grpc.ClientConn
		csiClient   csi.ControllerClient
		certFixture csi_cert.CertificationFixture
	)

	BeforeEach(func() {
		testLogger = lagertest.NewTestLogger("CSI Certification")
		fileName := os.Getenv("FIXTURE_FILENAME")
		Expect(fileName).NotTo(Equal(""))
		certFixture = csi_cert.LoadCertificationFixture(fileName)
		conn, err := grpc.Dial(certFixture.)
		Expect(err).NotTo(HaveOccurred())
		csiClient = csi.NewControllerClient()
	})

	Context("given a CSI client", func() {
		BeforeEach(func() {

		})
		AfterEach(func() {
			conn.Close()
		})
		It("should respond with Capabilities", func() {
			resp := csiClient.Capabilities(testEnv)
			Expect(resp.Capabilities).NotTo(BeNil())
		})
	})

})
