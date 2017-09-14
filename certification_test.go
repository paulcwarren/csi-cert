package csi_cert_test

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/paulcwarren/csi-cert"
)

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
	conn, err := grpc.Dial(certFixture.ControllerAddress, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	csiControllerClient := csi.NewControllerClient(conn)

	capabilityRequest := &csi.ControllerGetCapabilitiesRequest{
		Version: version,
	}
	capabilityResp, err := csiControllerClient.ControllerGetCapabilities(ctx, capabilityRequest)
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

func VolumeID(volID *csi.VolumeHandle) GomegaMatcher {
	return WithTransform(func(entry *csi.ListVolumesResponse_Result_Entry) *csi.VolumeHandle {
		return entry.GetVolumeInfo().GetHandle()
	}, Equal(volID))
}

var _ = Describe("CSI Certification", func() {

	var (
		err                 error
		conn                *grpc.ClientConn
		csiControllerClient csi.ControllerClient
		certFixture         csi_cert.CertificationFixture
		ctx                 context.Context
		version             *csi.Version
	)

	getContollerCapabilites()

	BeforeEach(func() {
		fileName := os.Getenv("FIXTURE_FILENAME")
		Expect(fileName).NotTo(Equal(""))
		certFixture, err = csi_cert.LoadCertificationFixture(fileName)
		Expect(err).NotTo(HaveOccurred())
		ctx = context.TODO()
		conn, err = grpc.Dial(certFixture.ControllerAddress, grpc.WithInsecure())
		Expect(err).NotTo(HaveOccurred())
		csiControllerClient = csi.NewControllerClient(conn)
	})

	AfterEach(func() {
		err := conn.Close()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when the plugin's capabilities are fetched", func() {
		var (
			capabilityRequest *csi.ControllerGetCapabilitiesRequest
			capabilityResp    *csi.ControllerGetCapabilitiesResponse
			capabilityErr     error
		)

		JustBeforeEach(func() {
			capabilityRequest = &csi.ControllerGetCapabilitiesRequest{
				Version: version,
			}
			capabilityResp, capabilityErr = csiControllerClient.ControllerGetCapabilities(ctx, capabilityRequest)
		})

		It("should succeed", func() {
			Expect(capabilityErr).NotTo(HaveOccurred())
			Expect(capabilityResp).NotTo(BeNil())
		})

		if hasCreateDeleteCapability {
			Context("given it supports the CREATE_DELETE_VOLUME capability", func() {
				Context("when a volume is created", func() {
					var (
						volName       string
						vc            []*csi.VolumeCapability
						request       *csi.CreateVolumeRequest
						createVolResp *csi.CreateVolumeResponse
						volHandle     *csi.VolumeHandle
					)

					BeforeEach(func() {
						volName = fmt.Sprintf("new-volume-%s", randomString(10))
						vc = []*csi.VolumeCapability{{AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{}}}}
						request = &csi.CreateVolumeRequest{
							Version:            version,
							Name:               volName,
							VolumeCapabilities: vc,
						}
					})

					JustBeforeEach(func() {
						createVolResp, err = csiControllerClient.CreateVolume(ctx, request)
					})

					AfterEach(func() {
						volHandle := createVolResp.GetResult().GetVolumeInfo().GetHandle()
						deleteRequest := &csi.DeleteVolumeRequest{
							Version:      version,
							VolumeHandle: volHandle,
						}
						_, err := csiControllerClient.DeleteVolume(ctx, deleteRequest)
						Expect(err).NotTo(HaveOccurred())
					})

					It("should succeed", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(createVolResp).NotTo(BeNil())
						Expect(createVolResp.GetError()).To(BeNil())
						Expect(createVolResp.GetResult().GetVolumeInfo().GetHandle().GetId()).NotTo(BeNil())
					})

					if hasListVolumesCapability {
						Context("given it supports the LIST_VOLUMES capability", func() {
							Context("when volumes are listed", func() {
								var (
									listReq  *csi.ListVolumesRequest
									listResp *csi.ListVolumesResponse
								)
								JustBeforeEach(func() {
									listReq = &csi.ListVolumesRequest{
										Version: version,
									}
									listResp, err = csiControllerClient.ListVolumes(ctx, listReq)
								})

								It("should include the volume just created", func() {
									volHandle = createVolResp.GetResult().GetVolumeInfo().GetHandle()
									Expect(err).NotTo(HaveOccurred())
									Expect(listResp).NotTo(BeNil())
									Expect(listResp.GetResult().GetEntries()).To(ContainElement(VolumeID(volHandle)))
								})
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
								VolumeCapabilities: []*csi.VolumeCapability{{AccessType: &csi.VolumeCapability_Mount{
									Mount: &csi.VolumeCapability_MountVolume{
										MountFlags: []string{""},
									},
								}}},
							}
							validateVolumeResp, err = csiControllerClient.ValidateVolumeCapabilities(ctx, validateVolumeRequest)
						})

						It("should succeed", func() {
							Expect(err).NotTo(HaveOccurred())
							Expect(validateVolumeResp).NotTo(BeNil())
							Expect(validateVolumeResp.GetResult()).NotTo(BeNil())
							Expect(validateVolumeResp.GetError()).To(BeNil())
						})

						//TODO: Add some real test to test the volume can acutally do things they claimed
					})

					Context("given the volume is created for a second time", func() {
						var (
							anotherCreateVolResp *csi.CreateVolumeResponse
						)

						JustBeforeEach(func() {
							anotherCreateVolResp, err = csiControllerClient.CreateVolume(ctx, request)
						})

						It("should succeed", func() {
							Expect(err).NotTo(HaveOccurred())
							Expect(anotherCreateVolResp).NotTo(BeNil())
							Expect(anotherCreateVolResp.GetError()).To(BeNil())
							Expect(anotherCreateVolResp.GetResult()).To(Equal(createVolResp.GetResult()))
						})
					})

					Context("with an invalid request (no volume name)", func() {
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
						Context("given it supports the PUBLISH_UNPUBLISH_VOLUME capability", func() {
							Context("when a volume is published", func() {
								var (
									publishRequest *csi.ControllerPublishVolumeRequest
									publishResp    *csi.ControllerPublishVolumeResponse
								)

								JustBeforeEach(func() {
									volHandle = createVolResp.GetResult().VolumeInfo.GetHandle()
									publishRequest = &csi.ControllerPublishVolumeRequest{
										Version:      version,
										VolumeHandle: volHandle,
										Readonly:     false,
									}
									publishResp, err = csiControllerClient.ControllerPublishVolume(ctx, publishRequest)
								})

								AfterEach(func() {
									unpublishRequest := &csi.ControllerUnpublishVolumeRequest{
										Version:      version,
										VolumeHandle: volHandle,
									}
									_, err := csiControllerClient.ControllerUnpublishVolume(ctx, unpublishRequest)
									Expect(err).NotTo(HaveOccurred())
								})

								It("should succeed", func() {
									Expect(err).NotTo(HaveOccurred())
									Expect(publishResp).NotTo(BeNil())
									Expect(publishResp.GetResult().GetPublishVolumeInfo()).NotTo(BeNil())
								})

								Context("when it is published for a second time", func() {
									var (
										anotherPublishResp *csi.ControllerPublishVolumeResponse
									)

									JustBeforeEach(func() {
										volHandle = createVolResp.GetResult().VolumeInfo.GetHandle()
										anotherPublishResp, err = csiControllerClient.ControllerPublishVolume(ctx, publishRequest)
									})

									It("should succeed", func() {
										Expect(err).NotTo(HaveOccurred())
										Expect(anotherPublishResp).NotTo(BeNil())
										Expect(anotherPublishResp.GetResult().GetPublishVolumeInfo()).NotTo(BeNil())
										Expect(anotherPublishResp.GetResult()).To(Equal(publishResp.GetResult()))
									})
								})

								NodeTests(conn)

								Context("when it is unpublished", func() {
									var (
										unpublishResp    *csi.ControllerUnpublishVolumeResponse
										unpublishRequest *csi.ControllerUnpublishVolumeRequest
									)

									BeforeEach(func() {
										unpublishRequest = &csi.ControllerUnpublishVolumeRequest{
											Version:      version,
											VolumeHandle: volHandle,
										}
									})

									JustBeforeEach(func() {
										unpublishResp, err = csiControllerClient.ControllerUnpublishVolume(ctx, unpublishRequest)
									})

									It("should succeed", func() {
										Expect(err).NotTo(HaveOccurred())
										Expect(unpublishResp).NotTo(BeNil())
										Expect(unpublishResp.GetResult()).NotTo(BeNil())
									})

									Context("when it is unpublished for the second time", func() {
										var (
											anotherUnpublishResp *csi.ControllerUnpublishVolumeResponse
										)

										JustBeforeEach(func() {
											anotherUnpublishResp, err = csiControllerClient.ControllerUnpublishVolume(ctx, unpublishRequest)
										})

										It("should succeed", func() {
											Expect(err).NotTo(HaveOccurred())
											Expect(anotherUnpublishResp).NotTo(BeNil())
											Expect(anotherUnpublishResp.GetResult()).NotTo(BeNil())
											Expect(anotherUnpublishResp.GetResult()).To(Equal(unpublishResp.GetResult()))
										})

									})

								})
							})
						})
					} else {
						NodeTests(conn)
					}

					Context("when a volume is deleted", func() {
						var (
							volHandle     *csi.VolumeHandle
							deleteRequest *csi.DeleteVolumeRequest
							deleteResp    *csi.DeleteVolumeResponse
						)

						JustBeforeEach(func() {
							volHandle = createVolResp.GetResult().GetVolumeInfo().GetHandle()
							deleteRequest = &csi.DeleteVolumeRequest{
								Version:      version,
								VolumeHandle: volHandle,
							}
							deleteResp, err = csiControllerClient.DeleteVolume(ctx, deleteRequest)
						})

						It("should succeeed", func() {
							Expect(err).NotTo(HaveOccurred())
							Expect(deleteResp).NotTo(BeNil())
							Expect(deleteResp.GetError()).To(BeNil())
						})

						Context("when it is deleted for a second time", func() {
							var (
								anotherDeleteResp *csi.DeleteVolumeResponse
							)
							JustBeforeEach(func() {
								anotherDeleteResp, err = csiControllerClient.DeleteVolume(ctx, deleteRequest)
							})

							It("should succeeed", func() {
								Expect(err).NotTo(HaveOccurred())
								Expect(anotherDeleteResp).NotTo(BeNil())
								Expect(anotherDeleteResp.GetError()).To(BeNil())
								Expect(anotherDeleteResp.GetResult()).To(Equal(deleteResp.GetResult()))
							})
						})

						Context("with a invalid volume name", func() {
							JustBeforeEach(func() {
								volHandle = &csi.VolumeHandle{Id: ""}
								deleteRequest = &csi.DeleteVolumeRequest{
									Version:      version,
									VolumeHandle: volHandle,
								}
								deleteResp, err = csiControllerClient.DeleteVolume(ctx, deleteRequest)
							})

							It("should fail with an error", func() {
								Expect(err).NotTo(HaveOccurred())
								Expect(deleteResp).NotTo(BeNil())
								respError := deleteResp.GetError()
								Expect(respError).NotTo(BeNil())
								Expect(respError.GetDeleteVolumeError().GetErrorCode()).To(Equal(csi.Error_DeleteVolumeError_INVALID_VOLUME_HANDLE))
							})
						})
					})
				})
			})
		}
	})

	if hasGetCapacityCapability {
		Context("given it supports the GET_CAPACITY capability", func() {
			Context("when the capacity of the storage is fetched", func() {
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
					capResp, err = csiControllerClient.GetCapacity(ctx, capRequest)
				})

				It("should succeed and return a valid capacity", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(capResp).NotTo(BeNil())
					Expect(capResp.GetError()).To(BeNil())
					Expect(capResp.GetResult()).NotTo(BeNil())
					Expect(capResp.GetResult().GetAvailableCapacity()).NotTo(BeNil())
				})
			})

		})

	}

})

func NodeTests(conn *grpc.ClientConn) {
	// TODO: PublishNode, UnpublishNode should be tested here before deleteing volume
	Context("when a volume is node published", func() {
		var (
			err           error
			ctx           context.Context
			csiNodeClient csi.NodeClient
			nodePubReq    *csi.NodePublishVolumeRequest
			nodePubResp   *csi.NodePublishVolumeResponse
			volName       string
			volHandle     *csi.VolumeHandle
			targetPath    string
			volCapability *csi.VolumeCapability
			readOnly      bool
		)
		BeforeEach(func() {
			fileName := os.Getenv("FIXTURE_FILENAME")
			certFixture, err := csi_cert.LoadCertificationFixture(fileName)
			if err != nil {
				panic(err)
			}
			conn, err := grpc.Dial(certFixture.NodeAddress, grpc.WithInsecure())
			if err != nil {
				panic(err)
			}
			csiNodeClient = csi.NewNodeClient(conn)
			ctx = context.TODO()
			volName = fmt.Sprintf("node-volume-%s", randomString(5))
			targetPath = fmt.Sprintf("/tmp/_mounts/%s", volName)
			osErr := os.MkdirAll("/tmp/_mounts", os.ModePerm)
			Expect(osErr).NotTo(HaveOccurred())
			volHandle = &csi.VolumeHandle{Id: volName}
			volCapability = &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{MountFlags: []string{}}},
			}
		})

		JustBeforeEach(func() {
			nodePubReq = &csi.NodePublishVolumeRequest{
				Version:           version,
				VolumeHandle:      volHandle,
				PublishVolumeInfo: &csi.PublishVolumeInfo{Values: map[string]string{}},
				TargetPath:        targetPath,
				VolumeCapability:  volCapability,
				Readonly:          readOnly,
			}

			nodePubResp, err = csiNodeClient.NodePublishVolume(ctx, nodePubReq)
		})

		It("should succeed", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(nodePubResp).NotTo(BeNil())
			Expect(nodePubResp.GetError()).To(BeNil())
			Expect(nodePubResp.GetResult()).NotTo(BeNil())
		})

		Context("given the volume is node published a second time", func() {
			var (
				anotherNodePubResp *csi.NodePublishVolumeResponse
			)

			JustBeforeEach(func() {
				anotherNodePubResp, err = csiNodeClient.NodePublishVolume(ctx, nodePubReq)
			})

			It("should succeed", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(anotherNodePubResp).NotTo(BeNil())
				Expect(anotherNodePubResp.GetError()).To(BeNil())
				Expect(anotherNodePubResp.GetResult()).To(Equal(nodePubResp.GetResult()))
			})
		})

		Context("with an invalid request (no volume id)", func() {
			BeforeEach(func() {
				volHandle = nil
			})

			It("should fail with an error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(nodePubResp.GetError()).NotTo(BeNil())
			})
		})

		Context("with an invalid request (no volume capability)", func() {
			BeforeEach(func() {
				volCapability = nil
			})

			It("should fail with an error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(nodePubResp.GetError()).NotTo(BeNil())
			})
		})

		Context("with an invalid request (empty volume capability)", func() {
			BeforeEach(func() {
				volCapability = &csi.VolumeCapability{}
			})

			It("should fail with an error", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(nodePubResp.GetError()).NotTo(BeNil())
			})
		})

		Context("when a volume is node unpublished", func() {
			var (
				nodeUnpubReq  *csi.NodeUnpublishVolumeRequest
				nodeUnpubResp *csi.NodeUnpublishVolumeResponse
			)

			BeforeEach(func() {
				nodeUnpubReq = &csi.NodeUnpublishVolumeRequest{
					Version:      version,
					VolumeHandle: volHandle,
					TargetPath:   targetPath,
				}
			})

			JustBeforeEach(func() {
				nodeUnpubResp, err = csiNodeClient.NodeUnpublishVolume(ctx, nodeUnpubReq)
			})

			It("should succeed", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(nodeUnpubResp.GetError()).To(BeNil())
			})

			Context("given the volume is node unpublished a second time", func() {
				BeforeEach(func() {
					nodeUnpubReq = &csi.NodeUnpublishVolumeRequest{
						Version:      version,
						VolumeHandle: volHandle,
						TargetPath:   targetPath,
					}
				})

				It("should succeed", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(nodeUnpubResp.GetError()).To(BeNil())
				})
			})
			Context("with an invalid request (no volume id)", func() {
				BeforeEach(func() {
					nodeUnpubReq = &csi.NodeUnpublishVolumeRequest{
						Version:    version,
						TargetPath: targetPath,
					}
				})

				It("should fail with an error", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(nodeUnpubResp.GetError()).NotTo(BeNil())
				})
			})

			Context("with an invalid request (no target path)", func() {
				BeforeEach(func() {
					nodeUnpubReq = &csi.NodeUnpublishVolumeRequest{
						Version:      version,
						VolumeHandle: volHandle,
					}
				})

				It("should fail with an error", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(nodeUnpubResp.GetError()).NotTo(BeNil())
				})
			})
		})
	})
}
