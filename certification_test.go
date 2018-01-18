package csi_cert_test

import (
	"fmt"
	"math/rand"
	"os"
	"time"
	"regexp"

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
	Minor: 1,
	Patch: 0,
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

	for _, capability := range capabilityResp.GetCapabilities() {
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

func VolumeID(volID string) GomegaMatcher {
	return WithTransform(func(entry *csi.ListVolumesResponse_Entry) string {
		return entry.GetVolumeInfo().GetId()
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

	Context("when the controller is probed", func(){
		var resp *csi.ControllerProbeResponse
		JustBeforeEach(func(){
			resp, err = csiControllerClient.ControllerProbe(ctx, &csi.ControllerProbeRequest{Version: version})
		})
		It("should succeed", func(){
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
		})
	})

	Context("when the controller is interrogated for its ID", func() {
		var csiIdentityClient csi.IdentityClient
		var identityConn *grpc.ClientConn

		BeforeEach(func() {
			identityConn, err = grpc.Dial(certFixture.ControllerAddress, grpc.WithInsecure())
			Expect(err).NotTo(HaveOccurred())
			csiIdentityClient = csi.NewIdentityClient(conn)
		})

		AfterEach(func() {
			err := identityConn.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return an array of supported versions", func() {
			res, err := csiIdentityClient.GetSupportedVersions(
				context.Background(),
				&csi.GetSupportedVersionsRequest{})

			By("checking response to have supported versions list")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.GetSupportedVersions()).NotTo(BeNil())
			Expect(len(res.GetSupportedVersions()) >= 1).To(BeTrue())

			By("checking each version")
			for _, version := range res.GetSupportedVersions() {
				Expect(version).NotTo(BeNil())
				Expect(version.GetMajor()).To(BeNumerically("<", 100))
				Expect(version.GetMinor()).To(BeNumerically("<", 100))
				Expect(version.GetPatch()).To(BeNumerically("<", 100))
			}
		})

		It("should return appropriate information", func() {
			req := &csi.GetPluginInfoRequest{
				Version: version,
			}
			res, err := csiIdentityClient.GetPluginInfo(context.Background(), req)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())

			By("verifying name size and characters")
			Expect(res.GetName()).ToNot(HaveLen(0))
			Expect(len(res.GetName())).To(BeNumerically("<=", 63))
			Expect(regexp.
			MustCompile("^[a-zA-Z][A-Za-z0-9-\\.\\_]{0,61}[a-zA-Z]$").
				MatchString(res.GetName())).To(BeTrue())
		})
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
						volumeId := createVolResp.GetVolumeInfo().GetId()
						deleteRequest := &csi.DeleteVolumeRequest{
							Version:  version,
							VolumeId: volumeId,
						}

						if volumeId != "" {
							_, err := csiControllerClient.DeleteVolume(ctx, deleteRequest)
							Expect(err).NotTo(HaveOccurred())
						}
					})

					It("should succeed", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(createVolResp).NotTo(BeNil())
						Expect(createVolResp.GetVolumeInfo().GetId()).NotTo(BeEmpty())
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
									volumeId := createVolResp.GetVolumeInfo().GetId()
									Expect(err).NotTo(HaveOccurred())
									Expect(listResp).NotTo(BeNil())
									Expect(listResp.GetEntries()).To(ContainElement(VolumeID(volumeId)))
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
							volInfo := createVolResp.GetVolumeInfo()
							validateVolumeRequest = &csi.ValidateVolumeCapabilitiesRequest{
								Version:  version,
								VolumeId: volInfo.GetId(),
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

						It("should succeed and return the same id and metadata", func() {
							Expect(err).NotTo(HaveOccurred())
							Expect(anotherCreateVolResp).NotTo(BeNil())
							Expect(anotherCreateVolResp).To(Equal(createVolResp))
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
							Expect(err).To(HaveOccurred())
						})
					})

					Context("given a node plugin", func() {
						var (
							err           error
							ctx           context.Context
							csiNodeClient csi.NodeClient
						)

						BeforeEach(func(){
							fileName := os.Getenv("FIXTURE_FILENAME")
							certFixture, err := csi_cert.LoadCertificationFixture(fileName)
							if err != nil {
								panic(err)
							}
							conn, err := grpc.Dial(certFixture.NodeAddress, grpc.WithInsecure())
							Expect(err).ToNot(HaveOccurred())
							csiNodeClient = csi.NewNodeClient(conn)
							ctx = context.Background()
						})

						Context("when a node is probed", func(){
							var resp *csi.NodeProbeResponse
							JustBeforeEach(func(){
								resp, err = csiNodeClient.NodeProbe(ctx,&csi.NodeProbeRequest{Version: version})
							})
							It("should succeed", func(){
								Expect(err).ToNot(HaveOccurred())
								Expect(resp).ToNot(BeNil())
							})
						})

						Context("when the node is interrogated for its ID", func() {
							var csiIdentityClient csi.IdentityClient
							var identityConn *grpc.ClientConn

							BeforeEach(func() {
								identityConn, err = grpc.Dial(certFixture.NodeAddress, grpc.WithInsecure())
								Expect(err).NotTo(HaveOccurred())
								csiIdentityClient = csi.NewIdentityClient(conn)
							})

							AfterEach(func() {
								err := identityConn.Close()
								Expect(err).NotTo(HaveOccurred())
							})

							It("should return an array of supported versions", func() {
								res, err := csiIdentityClient.GetSupportedVersions(
									context.Background(),
									&csi.GetSupportedVersionsRequest{})

								By("checking response to have supported versions list")
								Expect(err).NotTo(HaveOccurred())
								Expect(res.GetSupportedVersions()).NotTo(BeNil())
								Expect(len(res.GetSupportedVersions()) >= 1).To(BeTrue())

								By("checking each version")
								for _, version := range res.GetSupportedVersions() {
									Expect(version).NotTo(BeNil())
									Expect(version.GetMajor()).To(BeNumerically("<", 100))
									Expect(version.GetMinor()).To(BeNumerically("<", 100))
									Expect(version.GetPatch()).To(BeNumerically("<", 100))
								}
							})

							It("should return appropriate information", func() {
								req := &csi.GetPluginInfoRequest{
									Version: version,
								}
								res, err := csiIdentityClient.GetPluginInfo(context.Background(), req)
								Expect(err).NotTo(HaveOccurred())
								Expect(res).NotTo(BeNil())

								By("verifying name size and characters")
								Expect(res.GetName()).ToNot(HaveLen(0))
								Expect(len(res.GetName())).To(BeNumerically("<=", 63))
								Expect(regexp.
								MustCompile("^[a-zA-Z][A-Za-z0-9-\\.\\_]{0,61}[a-zA-Z]$").
									MatchString(res.GetName())).To(BeTrue())
							})
						})

						// TODO: PublishNode, UnpublishNode should be tested here before deleteing volume
						Context("when a volume is node published", func() {
							var (
								nodePubReq    *csi.NodePublishVolumeRequest
								nodePubResp   *csi.NodePublishVolumeResponse
								volName       string
								targetPath    string
								readOnly      bool

								volumeId          string
								volumeAttributes  map[string]string
								publishVolumeInfo map[string]string
								volCapability     *csi.VolumeCapability

								controllerPublishRequest *csi.ControllerPublishVolumeRequest
								controllerPublishResp    *csi.ControllerPublishVolumeResponse
							)

							BeforeEach(func() {
								volName = fmt.Sprintf("node-volume-%s", randomString(5))
								targetPath = fmt.Sprintf("/tmp/_mounts/%s", volName)
								osErr := os.MkdirAll("/tmp/_mounts", os.ModePerm)
								Expect(osErr).NotTo(HaveOccurred())

								volumeId = createVolResp.GetVolumeInfo().GetId()
								volumeAttributes = createVolResp.GetVolumeInfo().GetAttributes()
								publishVolumeInfo = map[string]string{}
								volCapability = &csi.VolumeCapability{
									AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{MountFlags: []string{}}},
								}
							})

							JustBeforeEach(func() {
								if hasPublishUnpublishCapability {
									controllerPublishRequest = &csi.ControllerPublishVolumeRequest{
										Version:  version,
										VolumeId: volumeId,
										Readonly: false,
									}
									controllerPublishResp, err = csiControllerClient.ControllerPublishVolume(ctx, controllerPublishRequest)
									Expect(err).NotTo(HaveOccurred())
									Expect(controllerPublishResp).NotTo(BeNil())
									publishVolumeInfo = controllerPublishResp.GetPublishVolumeInfo()
								}

								volumeId = createVolResp.GetVolumeInfo().GetId()
								volumeAttributes = createVolResp.GetVolumeInfo().GetAttributes()
								publishVolumeInfo = map[string]string{}

								nodePubReq = &csi.NodePublishVolumeRequest{
									Version:           version,
									VolumeId:          volumeId,
									VolumeAttributes:  volumeAttributes,
									PublishVolumeInfo: publishVolumeInfo,
									TargetPath:        targetPath,
									VolumeCapability:  volCapability,
									Readonly:          readOnly,
								}

								nodePubResp, err = csiNodeClient.NodePublishVolume(ctx, nodePubReq)
							})

							AfterEach(func() {
								if hasPublishUnpublishCapability {
									volumeId := createVolResp.GetVolumeInfo().GetId()
									unpublishRequest := &csi.ControllerUnpublishVolumeRequest{
										Version:  version,
										VolumeId: volumeId,
									}
									_, err := csiControllerClient.ControllerUnpublishVolume(ctx, unpublishRequest)
									Expect(err).NotTo(HaveOccurred())
								}
							})

							It("should succeed", func() {
								Expect(err).NotTo(HaveOccurred())
								Expect(nodePubResp).NotTo(BeNil())
							})

							Context("given the volume is node published a second time", func() {
								var (
									anotherNodePubResp *csi.NodePublishVolumeResponse
								)

								JustBeforeEach(func() {
									anotherNodePubResp, err = csiNodeClient.NodePublishVolume(ctx, nodePubReq)
								})

								It("should succeed and return the same response", func() {
									Expect(err).NotTo(HaveOccurred())
									Expect(anotherNodePubResp).NotTo(BeNil())
									Expect(anotherNodePubResp).To(Equal(nodePubResp))
								})
							})

							Context("with an invalid request (no volume id)", func() {
								JustBeforeEach(func() {
									nodePubReq = &csi.NodePublishVolumeRequest{
										Version:           version,
										VolumeId:          "",
										VolumeAttributes:  volumeAttributes,
										PublishVolumeInfo: publishVolumeInfo,
										TargetPath:        targetPath,
										VolumeCapability:  volCapability,
										Readonly:          readOnly,
									}

									_, err = csiNodeClient.NodePublishVolume(ctx, nodePubReq)
								})

								It("should fail with an error", func() {
									Expect(err).To(HaveOccurred())
								})
							})

							Context("with an invalid request (no volume capability)", func() {
								BeforeEach(func() {
									volCapability = nil
								})

								It("should fail with an error", func() {
									Expect(err).To(HaveOccurred())
								})
							})

							Context("with an invalid request (empty volume capability)", func() {
								BeforeEach(func() {
									volCapability = &csi.VolumeCapability{}
								})

								It("should fail with an error", func() {
									Expect(err).To(HaveOccurred())
								})
							})

							Context("when a volume is node unpublished", func() {
								var (
									nodeUnpubReq                    *csi.NodeUnpublishVolumeRequest
									nodeUnpubResp, anotherUnpubResp *csi.NodeUnpublishVolumeResponse
								)

								BeforeEach(func() {
									nodeUnpubReq = &csi.NodeUnpublishVolumeRequest{
										Version:    version,
										VolumeId:   volumeId,
										TargetPath: targetPath,
									}
								})

								JustBeforeEach(func() {
									nodeUnpubResp, err = csiNodeClient.NodeUnpublishVolume(ctx, nodeUnpubReq)
								})

								It("should succeed", func() {
									Expect(err).NotTo(HaveOccurred())
								})

								Context("given the volume is node unpublished a second time", func() {
									JustBeforeEach(func() {
										anotherUnpubResp, err = csiNodeClient.NodeUnpublishVolume(ctx, nodeUnpubReq)
									})

									It("should succeed", func() {
										Expect(err).NotTo(HaveOccurred())
										Expect(anotherUnpubResp).To(Equal(nodeUnpubResp))
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
										Expect(err).To(HaveOccurred())
									})
								})

								Context("with an invalid request (no target path)", func() {
									BeforeEach(func() {
										nodeUnpubReq = &csi.NodeUnpublishVolumeRequest{
											Version:  version,
											VolumeId: volumeId,
										}
									})

									It("should fail with an error", func() {
										Expect(err).To(HaveOccurred())
									})
								})
							})
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
									volumeId := createVolResp.GetVolumeInfo().GetId()
									publishRequest = &csi.ControllerPublishVolumeRequest{
										Version:  version,
										VolumeId: volumeId,
										Readonly: false,
									}
									publishResp, err = csiControllerClient.ControllerPublishVolume(ctx, publishRequest)
								})

								AfterEach(func() {
									volumeId := createVolResp.GetVolumeInfo().GetId()
									unpublishRequest := &csi.ControllerUnpublishVolumeRequest{
										Version:  version,
										VolumeId: volumeId,
									}
									_, err := csiControllerClient.ControllerUnpublishVolume(ctx, unpublishRequest)
									Expect(err).NotTo(HaveOccurred())
								})

								It("should succeed", func() {
									Expect(err).NotTo(HaveOccurred())
									Expect(publishResp).NotTo(BeNil())
								})

								Context("when it is published for a second time", func() {
									var (
										anotherPublishResp *csi.ControllerPublishVolumeResponse
									)

									JustBeforeEach(func() {
										anotherPublishResp, err = csiControllerClient.ControllerPublishVolume(ctx, publishRequest)
									})

									It("should succeed", func() {
										Expect(err).NotTo(HaveOccurred())
										Expect(anotherPublishResp).NotTo(BeNil())
										Expect(anotherPublishResp).To(Equal(publishResp))
									})
								})

								Context("when it is unpublished", func() {
									var (
										unpublishResp    *csi.ControllerUnpublishVolumeResponse
										unpublishRequest *csi.ControllerUnpublishVolumeRequest
									)

									BeforeEach(func() {
										volumeId := createVolResp.GetVolumeInfo().GetId()
										unpublishRequest = &csi.ControllerUnpublishVolumeRequest{
											Version:  version,
											VolumeId: volumeId,
										}
									})

									JustBeforeEach(func() {
										unpublishResp, err = csiControllerClient.ControllerUnpublishVolume(ctx, unpublishRequest)
									})

									It("should succeed", func() {
										Expect(err).NotTo(HaveOccurred())
										Expect(unpublishResp).NotTo(BeNil())
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
											Expect(anotherUnpublishResp).To(Equal(unpublishResp))
										})
									})
								})
							})
						})
					}

					Context("when a volume is deleted", func() {
						var (
							volumeId      string
							deleteRequest *csi.DeleteVolumeRequest
							deleteResp    *csi.DeleteVolumeResponse
						)

						JustBeforeEach(func() {
							volumeId = createVolResp.GetVolumeInfo().GetId()
							deleteRequest = &csi.DeleteVolumeRequest{
								Version:  version,
								VolumeId: volumeId,
							}
							deleteResp, err = csiControllerClient.DeleteVolume(ctx, deleteRequest)
						})

						It("should succeeed", func() {
							Expect(err).NotTo(HaveOccurred())
							Expect(deleteResp).NotTo(BeNil())
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
								Expect(anotherDeleteResp).To(Equal(deleteResp))
							})
						})

						Context("with a invalid volume id", func() {
							JustBeforeEach(func() {
								deleteRequest = &csi.DeleteVolumeRequest{
									Version:  version,
									VolumeId: "",
								}
								deleteResp, err = csiControllerClient.DeleteVolume(ctx, deleteRequest)
							})

							It("should fail with an error", func() {
								Expect(err).To(HaveOccurred())
								// TODO - the spec does not call out an error for this, but probably we should see a grpc InvalidArgument error?
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
					Expect(capResp.GetAvailableCapacity()).NotTo(BeNil())
				})
			})

		})

	}

})
