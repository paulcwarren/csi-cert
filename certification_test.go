package csi_cert_test

import (
	"os"

	"golang.org/x/net/context"

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
		err         error
		testLogger  lager.Logger
		conn        *grpc.ClientConn
		csiClient   csi.ControllerClient
		certFixture csi_cert.CertificationFixture
	)

	BeforeEach(func() {
		testLogger = lagertest.NewTestLogger("CSI Certification")
		fileName := os.Getenv("FIXTURE_FILENAME")
		Expect(fileName).NotTo(Equal(""))
		certFixture, err = csi_cert.LoadCertificationFixture(fileName)
		Expect(err).NotTo(HaveOccurred())
		conn, err = grpc.Dial(certFixture.DriverAddress, grpc.WithInsecure())
		Expect(err).NotTo(HaveOccurred())
		csiClient = csi.NewControllerClient(conn)
	})
	AfterEach(func() {
		conn.Close()
	})

	Context("given a CSI client", func() {
		var (
			resp *csi.ControllerGetCapabilitiesResponse
		)

		It("should respond with Capabilities", func() {
			ctx := context.TODO()
			request := &csi.ControllerGetCapabilitiesRequest{
				Version: &csi.Version{
					Major: 0,
					Minor: 0,
					Patch: 1,
				},
			}
			resp, err = csiClient.ControllerGetCapabilities(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
		})
	})

})
