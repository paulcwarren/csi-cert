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
	})

	AfterEach(func() {
		conn.Close()
	})

	Context("given a CSI client", func() {

		BeforeEach(func() {
			conn, err = grpc.Dial(certFixture.DriverAddress, grpc.WithInsecure())
			Expect(err).NotTo(HaveOccurred())
			csiClient = csi.NewControllerClient(conn)
		})

		Context("#ControllerGetcapabilities", func() {
			var (
				ctx     context.Context
				request *csi.ControllerGetCapabilitiesRequest
				resp    *csi.ControllerGetCapabilitiesResponse
			)
			BeforeEach(func() {
				ctx = context.TODO()
				request = &csi.ControllerGetCapabilitiesRequest{
					Version: &csi.Version{
						Major: 0,
						Minor: 0,
						Patch: 1,
					},
				}
			})
			JustBeforeEach(func() {
				resp, err = csiClient.ControllerGetCapabilities(ctx, request)
			})
			It("should respond to a ControllerGetCapabilities request", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeNil())
			})
		})

		Context("#CreateVolume", func() {
			var (
				ctx     context.Context
				volName string
				vc      []*csi.VolumeCapability
				request *csi.CreateVolumeRequest
				resp    *csi.CreateVolumeResponse
			)

			BeforeEach(func() {
				ctx = context.TODO()
				volName = "some-volume"
				vc = []*csi.VolumeCapability{{Value: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{}}}}
				request = &csi.CreateVolumeRequest{
					Version: &csi.Version{
						Major: 0,
						Minor: 0,
						Patch: 1,
					},
					Name:               volName,
					VolumeCapabilities: vc,
				}
			})
			JustBeforeEach(func() {
				resp, err = csiClient.CreateVolume(ctx, request)
			})

			It("should respond to a CreateVolume request", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeNil())
			})
		})

		Context("#DeleteVolume", func() {
			var (
				ctx            context.Context
				vc             []*csi.VolumeCapability
				volName        string
				volID          *csi.VolumeID
				create_request *csi.CreateVolumeRequest
				delete_request *csi.DeleteVolumeRequest
				resp           *csi.DeleteVolumeResponse
			)
			BeforeEach(func() {
				ctx = context.TODO()
				volID = &csi.VolumeID{Values: map[string]string{"volume_name": "abcd"}}

				volName = "abcd"
				vc = []*csi.VolumeCapability{{Value: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{}}}}
				create_request = &csi.CreateVolumeRequest{
					Version: &csi.Version{
						Major: 0,
						Minor: 0,
						Patch: 1,
					},
					Name:               volName,
					VolumeCapabilities: vc,
				}
				csiClient.CreateVolume(ctx, create_request)

				delete_request = &csi.DeleteVolumeRequest{
					Version: &csi.Version{
						Major: 0,
						Minor: 0,
						Patch: 1,
					},
					VolumeId: volID,
				}
			})
			JustBeforeEach(func() {
				resp, err = csiClient.DeleteVolume(ctx, delete_request)
			})
			It("should respond to a DeleteVolume request", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeNil())
			})
		})
	})
})
