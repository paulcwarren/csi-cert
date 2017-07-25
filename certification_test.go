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

func HasCapability(client csi.ControllerClient, hasCap csi.ControllerServiceCapability_RPC_Type) bool {
	ctx := context.TODO()
	capRequest := &csi.ControllerGetCapabilitiesRequest{
		Version: &csi.Version{
			Major: 0,
			Minor: 0,
			Patch: 1,
		},
	}
	capResp, err := client.ControllerGetCapabilities(ctx, capRequest)
	Expect(err).NotTo(HaveOccurred())
	capResult := capResp.GetResult()
	Expect(capResult).NotTo(BeNil())

	capabilitySupported := false

	for _, capability := range capResult.Capabilities {
		if capability.GetRpc().GetType() == hasCap {
			capabilitySupported = true
		}
	}
	return capabilitySupported
}

func HasCapabilityAddDeleteVolume(client csi.ControllerClient) bool {
	return HasCapability(client, csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME)
}

func HasCapabilityPublishUnpublishVolume(client csi.ControllerClient) bool {
	return HasCapability(client, csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME)
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
		err                              error
		testLogger                       lager.Logger
		conn                             *grpc.ClientConn
		csiClient                        csi.ControllerClient
		certFixture                      csi_cert.CertificationFixture
		ctx                              context.Context
		capabilityAddDeleteVolume        bool
		capabilityPublishUnpublishVolume bool
		version                          *csi.Version
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
		version = &csi.Version{
			Major: 0,
			Minor: 0,
			Patch: 1,
		}
		capabilityAddDeleteVolume = HasCapabilityAddDeleteVolume(csiClient)
		capabilityPublishUnpublishVolume = HasCapabilityPublishUnpublishVolume(csiClient)
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
				Version: version,
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
			volName       string
			vc            []*csi.VolumeCapability
			request       *csi.CreateVolumeRequest
			createVolResp *csi.CreateVolumeResponse
			volID         *csi.VolumeID
		)

		BeforeEach(func() {
			volName = fmt.Sprintf("new-volume-%s", randomString(10))
			vc = []*csi.VolumeCapability{{Value: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{}}}}
			request = &csi.CreateVolumeRequest{
				Version:            version,
				Name:               volName,
				VolumeCapabilities: vc,
			}
		})

		JustBeforeEach(func() {
			createVolResp, err = csiClient.CreateVolume(ctx, request)
		})

		It("should respond to a CreateVolume request", func() {
			if capabilityAddDeleteVolume {
				Expect(err).NotTo(HaveOccurred())
				Expect(createVolResp).NotTo(BeNil())
				Expect(createVolResp.GetError()).To(BeNil())
				Expect(createVolResp.GetResult().GetVolumeInfo().GetId()).NotTo(BeNil())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(createVolResp.GetError()).NotTo(BeNil())
			}
		})

		Context("given the volume already exists", func() {
			BeforeEach(func() {
				createVolResp, err = csiClient.CreateVolume(ctx, request)
				if capabilityAddDeleteVolume {
					Expect(err).NotTo(HaveOccurred())
					Expect(createVolResp).NotTo(BeNil())
					Expect(createVolResp.GetError()).To(BeNil())
				}
			})
			It("should succeed", func() {
				if capabilityAddDeleteVolume {
					Expect(err).NotTo(HaveOccurred())
					Expect(createVolResp).NotTo(BeNil())
					Expect(createVolResp.GetError()).To(BeNil())
				}
			})
		})

		Context("given an invalid request (no volume name)", func() {
			BeforeEach(func() {
				volName = ""
				request = &csi.CreateVolumeRequest{
					Version:            version,
					Name:               volName,
					VolumeCapabilities: vc,
				}
			})

			It("should fail with an error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(createVolResp.GetError()).NotTo(BeNil())
			})
		})

		Context("#ControllerPublishVolume", func() {
			var (
				publishRequest *csi.ControllerPublishVolumeRequest
				publishResp    *csi.ControllerPublishVolumeResponse
			)

			JustBeforeEach(func() {
				volID = createVolResp.GetResult().VolumeInfo.GetId()
				publishRequest = &csi.ControllerPublishVolumeRequest{
					Version:  version,
					VolumeId: volID,
					Readonly: false,
				}
				publishResp, err = csiClient.ControllerPublishVolume(ctx, publishRequest)
			})

			It("should respond to a ControllerPublishVolume request", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(publishResp).NotTo(BeNil())
				Expect(publishResp.GetResult().GetPublishVolumeInfo()).NotTo(BeNil())
			})

			Context("#ControllerUnpublishVolume", func() {
				var (
					unpublishRequest *csi.ControllerUnpublishVolumeRequest
					unpublishResp    *csi.ControllerUnpublishVolumeResponse
				)

				BeforeEach(func() {
					unpublishRequest = &csi.ControllerUnpublishVolumeRequest{
						Version:  version,
						VolumeId: volID,
					}
				})

				JustBeforeEach(func() {
					unpublishResp, err = csiClient.ControllerUnpublishVolume(ctx, unpublishRequest)
				})

				It("should respond to a ControllerUnpublishVolume request", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(unpublishResp).NotTo(BeNil())
					Expect(unpublishResp.GetResult()).NotTo(BeNil())
				})
			})

			Context("#DeleteVolume", func() {
				var (
					volID         *csi.VolumeID
					deleteRequest *csi.DeleteVolumeRequest
					deleteResp    *csi.DeleteVolumeResponse
				)

				Context("given a existent volume", func() {
					JustBeforeEach(func() {
						volID = createVolResp.GetResult().GetVolumeInfo().GetId()
						deleteRequest = &csi.DeleteVolumeRequest{
							Version:  version,
							VolumeId: volID,
						}
						deleteResp, err = csiClient.DeleteVolume(ctx, deleteRequest)
					})

					It("should respond to a DeleteVolume request", func() {
						if capabilityAddDeleteVolume {
							Expect(err).NotTo(HaveOccurred())
							Expect(deleteResp).NotTo(BeNil())
							Expect(deleteResp.GetError()).To(BeNil())
						} else {
							Expect(err).NotTo(HaveOccurred())
							Expect(deleteResp.GetError()).NotTo(BeNil())
						}
					})
				})

				Context("given a non existent volume", func() {
					JustBeforeEach(func() {
						volID = &csi.VolumeID{Values: map[string]string{"volume_name": "non_existent_volume"}}
						deleteRequest = &csi.DeleteVolumeRequest{
							Version:  version,
							VolumeId: volID,
						}
						deleteResp, err = csiClient.DeleteVolume(ctx, deleteRequest)
					})

					It("should fail with an error", func() {
						if capabilityAddDeleteVolume {
							Expect(err).NotTo(HaveOccurred())
							Expect(deleteResp).NotTo(BeNil())
							respError := deleteResp.GetError()
							Expect(respError).NotTo(BeNil())
							Expect(respError.GetDeleteVolumeError().GetErrorCode()).To(Equal(csi.Error_DeleteVolumeError_VOLUME_DOES_NOT_EXIST))
						} else {
							Expect(err).NotTo(HaveOccurred())
							Expect(deleteResp.GetError()).NotTo(BeNil())
						}
					})
				})

				Context("given a invalid volume", func() {
					JustBeforeEach(func() {
						volID = &csi.VolumeID{Values: map[string]string{"volume_name": ""}}
						deleteRequest = &csi.DeleteVolumeRequest{
							Version:  version,
							VolumeId: volID,
						}
						deleteResp, err = csiClient.DeleteVolume(ctx, deleteRequest)
					})

					It("should fail with an error", func() {
						if capabilityAddDeleteVolume {
							Expect(err).NotTo(HaveOccurred())
							Expect(deleteResp).NotTo(BeNil())
							respError := deleteResp.GetError()
							Expect(respError).NotTo(BeNil())
							Expect(respError.GetDeleteVolumeError().GetErrorCode()).To(Equal(csi.Error_DeleteVolumeError_INVALID_VOLUME_ID))
						} else {
							Expect(err).NotTo(HaveOccurred())
							Expect(deleteResp.GetError()).NotTo(BeNil())
						}
					})
				})
			})
		})
	})

})
