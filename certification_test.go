package csi_cert_test

import (
	"math/rand"
	"os"
	"time"

	"golang.org/x/net/context"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/types"
	"google.golang.org/grpc"

	"fmt"

	csi "github.com/jeffpak/csi"
	csi_cert "github.com/paulcwarren/csi-cert"
)

func HasCapability(capResp *csi.ControllerGetCapabilitiesResponse, hasCap csi.ControllerServiceCapability_RPC_Type) bool {
	Expect(capResp).NotTo(BeNil())
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

var hasCreateDeleteCapability = false
var hasPublishUnpublishCapability = false
var hasListVolumesCapability = false
var hasGetCapacityCapability = false

var version = &csi.Version{
	Major: 0,
	Minor: 0,
	Patch: 1,
}

func getContollerCapabilites() {
	ctx := context.TODO()
	fileName := os.Getenv("FIXTURE_FILENAME")
	certFixture, err := csi_cert.LoadCertificationFixture(fileName)
	if err != nil {
		panic(err)
	}
	conn, err := grpc.Dial(certFixture.DriverAddress, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	csiClient := csi.NewControllerClient(conn)

	capabilityRequest := &csi.ControllerGetCapabilitiesRequest{
		Version: version,
	}
	capabilityResp, err := csiClient.ControllerGetCapabilities(ctx, capabilityRequest)
	if err != nil {
		panic(err)
	}

	capabilityResult := capabilityResp.GetResult()

	for _, capability := range capabilityResult.Capabilities {
		capType := capability.GetRpc().GetType()
		switch capType {
		case csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME:
			hasCreateDeleteCapability = true
		case csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME:
			hasPublishUnpublishCapability = true
		case csi.ControllerServiceCapability_RPC_LIST_VOLUMES:
			hasListVolumesCapability = true
		case csi.ControllerServiceCapability_RPC_GET_CAPACITY:
			hasGetCapacityCapability = true
		}
	}
}

func VolumeID(volID *csi.VolumeID) GomegaMatcher {
	return WithTransform(func(entry *csi.ListVolumesResponse_Result_Entry) *csi.VolumeID {
		return entry.GetVolumeInfo().GetId()
	}, Equal(volID))
}

var _ = Describe("Certify with: ", func() {

	var (
		err         error
		testLogger  lager.Logger
		conn        *grpc.ClientConn
		csiClient   csi.ControllerClient
		certFixture csi_cert.CertificationFixture
		ctx         context.Context
		version     *csi.Version
	)

	getContollerCapabilites()

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

	})

	AfterEach(func() {
		conn.Close()
	})

	Context("#ControllerGetcapabilities", func() {
		var (
			capabilityRequest *csi.ControllerGetCapabilitiesRequest
			capabilityResp    *csi.ControllerGetCapabilitiesResponse
			capabilityErr     error
		)

		JustBeforeEach(func() {
			capabilityRequest = &csi.ControllerGetCapabilitiesRequest{
				Version: version,
			}
			capabilityResp, capabilityErr = csiClient.ControllerGetCapabilities(ctx, capabilityRequest)
		})

		It("should respond to a ControllerGetCapabilities request", func() {
			Expect(capabilityErr).NotTo(HaveOccurred())
			Expect(capabilityResp).NotTo(BeNil())
		})

		if hasCreateDeleteCapability {
			Context("#CreateVolume", func() {
				var (
					volName       string
					vc            []*csi.VolumeCapability
					request       *csi.CreateVolumeRequest
					createVolResp *csi.CreateVolumeResponse
					volID         *csi.VolumeID
					//listReq  *csi.ListVolumesRequest
					//listResp *csi.ListVolumesResponse
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

				//AfterEach(func() {
				//		volID = createVolResp.GetResult().GetVolumeInfo().GetId()
				//		deleteRequest := &csi.DeleteVolumeRequest{
				//			Version:  version,
				//			VolumeId: volID,
				//		}
				//		csiClient.DeleteVolume(ctx, deleteRequest)
				//
				//	listReq = &csi.ListVolumesRequest{
				//		Version: version,
				//	}
				//	listResp, err = csiClient.ListVolumes(ctx, listReq)
				//	fmt.Printf("%#v",listResp.GetResult().GetEntries())
				//	fmt.Println("-----------------------\n")
				//
				//	//		//Expect(err2).NotTo(HaveOccurred())
				////		//Expect(deleteResp).NotTo(BeNil())
				////		//Expect(deleteResp.GetError()).To(BeNil())
				//})

				It("should respond to a CreateVolume request", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(createVolResp).NotTo(BeNil())
					Expect(createVolResp.GetError()).To(BeNil())
					Expect(createVolResp.GetResult().GetVolumeInfo().GetId()).NotTo(BeNil())
				})

				if hasListVolumesCapability {
					Context("when we list volumes", func() {
						var (
							listReq  *csi.ListVolumesRequest
							listResp *csi.ListVolumesResponse
						)
						JustBeforeEach(func() {
							listReq = &csi.ListVolumesRequest{
								Version: version,
							}
							listResp, err = csiClient.ListVolumes(ctx, listReq)
						})

						It("should list the volume just created", func() {
							volID = createVolResp.GetResult().GetVolumeInfo().GetId()
							Expect(err).NotTo(HaveOccurred())
							Expect(listResp).NotTo(BeNil())
							Expect(listResp.GetResult().GetEntries()).To(ContainElement(VolumeID(volID)))
						})
					})
				}

				Context("when a volume's capabilities are validated", func() {
					var (
						validateVolumeRequest *csi.ValidateVolumeCapabilitiesRequest
						validateVolumeResp    *csi.ValidateVolumeCapabilitiesResponse
					)

					JustBeforeEach(func() {
						volInfo := createVolResp.GetResult().VolumeInfo
						validateVolumeRequest = &csi.ValidateVolumeCapabilitiesRequest{
							Version:    version,
							VolumeInfo: volInfo,
							VolumeCapabilities: []*csi.VolumeCapability{{Value: &csi.VolumeCapability_Mount{
								Mount: &csi.VolumeCapability_MountVolume{
									MountFlags: []string{""},
								},
							}}},
						}
						validateVolumeResp, err = csiClient.ValidateVolumeCapabilities(ctx, validateVolumeRequest)
					})

					It("should succeed", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(validateVolumeResp).NotTo(BeNil())
						Expect(validateVolumeResp.GetResult()).NotTo(BeNil())
						Expect(validateVolumeResp.GetError()).To(BeNil())
					})

					//TODO: Add some real test to test the volume can acutally do things they claimed
				})

				Context("given the volume already exists", func() {
					BeforeEach(func() {
						createVolResp, err = csiClient.CreateVolume(ctx, request)
						Expect(err).NotTo(HaveOccurred())
						Expect(createVolResp).NotTo(BeNil())
						Expect(createVolResp.GetError()).To(BeNil())
					})
					//AfterEach(func() {
					//	volID = createVolResp.GetResult().GetVolumeInfo().GetId()
					//	deleteRequest := &csi.DeleteVolumeRequest{
					//		Version:  version,
					//		VolumeId: volID,
					//	}
					//	csiClient.DeleteVolume(ctx, deleteRequest)
					//	//Expect(err2).NotTo(HaveOccurred())
					//	//Expect(deleteResp).NotTo(BeNil())
					//	//Expect(deleteResp.GetError()).To(BeNil())
					//})
					It("should succeed", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(createVolResp).NotTo(BeNil())
						Expect(createVolResp.GetError()).To(BeNil())
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

				if hasPublishUnpublishCapability {
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
									Expect(err).NotTo(HaveOccurred())
									Expect(deleteResp).NotTo(BeNil())
									Expect(deleteResp.GetError()).To(BeNil())
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
									Expect(err).NotTo(HaveOccurred())
									Expect(deleteResp).NotTo(BeNil())
									respError := deleteResp.GetError()
									Expect(respError).NotTo(BeNil())
									Expect(respError.GetDeleteVolumeError().GetErrorCode()).To(Equal(csi.Error_DeleteVolumeError_VOLUME_DOES_NOT_EXIST))
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
									Expect(err).NotTo(HaveOccurred())
									Expect(deleteResp).NotTo(BeNil())
									respError := deleteResp.GetError()
									Expect(respError).NotTo(BeNil())
									Expect(respError.GetDeleteVolumeError().GetErrorCode()).To(Equal(csi.Error_DeleteVolumeError_INVALID_VOLUME_ID))
								})
							})
						})
					})
				}
			})
		}
	})

	if hasGetCapacityCapability {
		Context("#GetCapacity", func() {
			var (
				capRequest *csi.GetCapacityRequest
				capResp    *csi.GetCapacityResponse
			)

			BeforeEach(func() {
				capRequest = &csi.GetCapacityRequest{
					Version: version,
				}
			})

			JustBeforeEach(func() {
				capResp, err = csiClient.GetCapacity(ctx, capRequest)
			})

			It("should respond to a ControllerGetCapabilities request", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(capResp).NotTo(BeNil())
				Expect(capResp.GetError()).To(BeNil())
				Expect(capResp.GetResult()).NotTo(BeNil())
				Expect(capResp.GetResult().GetTotalCapacity()).NotTo(BeNil())
			})
		})
	}

})
