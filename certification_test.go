package csi_cert_test

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"golang.org/x/net/context"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"

	csi "github.com/jeffpak/csi"
	csi_cert "github.com/paulcwarren/csi-cert"
)

func HasCapabilityAddDeleteVolume(client csi.ControllerClient) bool {

	ctx := context.TODO()
	cap_request := &csi.ControllerGetCapabilitiesRequest{
		Version: &csi.Version{
			Major: 0,
			Minor: 0,
			Patch: 1,
		},
	}
	cap_resp, err := client.ControllerGetCapabilities(ctx, cap_request)
	Expect(err).NotTo(HaveOccurred())
	cap_result := cap_resp.GetResult()
	Expect(cap_result).NotTo(BeNil())

	create_delete_volume_supported := false

	for _, capability := range cap_result.Capabilities {
		if capability.GetRpc().GetType() == csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME {
			create_delete_volume_supported = true
		}
	}

	return create_delete_volume_supported
}

var isSeeded = false

func randomString(n int) string {
	if !isSeeded {
		rand.Seed(time.Now().UnixNano())
		isSeeded = true
	}
	runes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}

var _ = Describe("Certify with: ", func() {

	var (
		err               error
		testLogger        lager.Logger
		conn              *grpc.ClientConn
		csiClient         csi.ControllerClient
		certFixture       csi_cert.CertificationFixture
		ctx               context.Context
		capability_volume bool
	)

	BeforeEach(func() {
		testLogger = lagertest.NewTestLogger("CSI Certification")
		fileName := os.Getenv("FIXTURE_FILENAME")
		Expect(fileName).NotTo(Equal(""))
		certFixture, err = csi_cert.LoadCertificationFixture(fileName)
		Expect(err).NotTo(HaveOccurred())
		ctx = context.TODO()
		conn, err = grpc.Dial(certFixture.DriverAddress, grpc.WithInsecure())
		Expect(err).NotTo(HaveOccurred())
		csiClient = csi.NewControllerClient(conn)
		capability_volume = HasCapabilityAddDeleteVolume(csiClient)
	})

	AfterEach(func() {
		conn.Close()
	})

	Context("#ControllerGetcapabilities", func() {
		var (
			request *csi.ControllerGetCapabilitiesRequest
			resp    *csi.ControllerGetCapabilitiesResponse
		)

		BeforeEach(func() {
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
			volName string
			vc      []*csi.VolumeCapability
			request *csi.CreateVolumeRequest
			resp    *csi.CreateVolumeResponse
		)

		BeforeEach(func() {
			volName = fmt.Sprintf("new-volume-%s", randomString(10))
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
			if capability_volume {
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeNil())
				Expect(resp.GetError()).To(BeNil())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.GetError()).NotTo(BeNil())
			}
		})

		Context("given the volume already exists", func() {
			BeforeEach(func() {
				resp, err = csiClient.CreateVolume(ctx, request)
				if capability_volume {
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).NotTo(BeNil())
					Expect(resp.GetError()).To(BeNil())
				}
			})
			It("should succeed", func() {
				if capability_volume {
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).NotTo(BeNil())
					Expect(resp.GetError()).To(BeNil())
				}
			})
		})

		Context("given an invalid request (no volume name)", func() {
			BeforeEach(func() {
				volName = ""
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

			It("should fail with an error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.GetError()).NotTo(BeNil())
			})
		})
	})

	Context("#DeleteVolume", func() {
		var (
			vc             []*csi.VolumeCapability
			volName        string
			volID          *csi.VolumeID
			create_request *csi.CreateVolumeRequest
			delete_request *csi.DeleteVolumeRequest
			resp           *csi.DeleteVolumeResponse
		)

		BeforeEach(func() {
			volName = fmt.Sprintf("existing-volume-%s", randomString(10))
			volID = &csi.VolumeID{Values: map[string]string{"volume_name": volName}}
		})

		JustBeforeEach(func() {
			delete_request = &csi.DeleteVolumeRequest{
				Version: &csi.Version{
					Major: 0,
					Minor: 0,
					Patch: 1,
				},
				VolumeId: volID,
			}

			resp, err = csiClient.DeleteVolume(ctx, delete_request)
		})

		Context("given a existent volume", func() {
			BeforeEach(func() {
				if capability_volume {
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
				}
			})

			It("should respond to a DeleteVolume request", func() {
				if capability_volume {
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).NotTo(BeNil())
					Expect(resp.GetError()).To(BeNil())
				} else {
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.GetError()).NotTo(BeNil())
				}
			})
		})

		Context("given a non existent volume", func() {
			BeforeEach(func() {
				volID = &csi.VolumeID{Values: map[string]string{"volume_name": "non_existent_volume"}}
			})

			It("should fail with an error", func() {
				if capability_volume {
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).NotTo(BeNil())
					resp_error := resp.GetError()
					Expect(resp_error).NotTo(BeNil())
					Expect(resp_error.Value).To(Equal(csi.Error_DeleteVolumeError_VOLUME_DOES_NOT_EXIST))
				} else {
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.GetError()).NotTo(BeNil())
				}
			})
		})

		Context("given a invalid volume", func() {
			BeforeEach(func() {
				volID = &csi.VolumeID{Values: map[string]string{"volume_name": ""}}
			})

			It("should fail with an error", func() {
				if capability_volume {
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).NotTo(BeNil())
					resp_error := resp.GetError()
					Expect(resp_error).NotTo(BeNil())
					Expect(resp_error.Value).To(Equal(csi.Error_DeleteVolumeError_INVALID_VOLUME_ID))
				} else {
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.GetError()).NotTo(BeNil())
				}
			})
		})
	})
})
