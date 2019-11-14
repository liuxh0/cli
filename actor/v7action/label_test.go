package v7action_test

import (
	"errors"

	. "code.cloudfoundry.org/cli/actor/v7action"
	"code.cloudfoundry.org/cli/actor/v7action/v7actionfakes"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv3"
	"code.cloudfoundry.org/cli/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Labels", func() {
	var (
		actor                     *Actor
		fakeCloudControllerClient *v7actionfakes.FakeCloudControllerClient
		fakeSharedActor           *v7actionfakes.FakeSharedActor
		fakeConfig                *v7actionfakes.FakeConfig
		warnings                  Warnings
		executeErr                error
		resourceName              string
		spaceGUID                 string
		orgGUID                   string
		labels                    map[string]types.NullString
	)

	BeforeEach(func() {
		fakeCloudControllerClient = new(v7actionfakes.FakeCloudControllerClient)
		fakeSharedActor = new(v7actionfakes.FakeSharedActor)
		fakeConfig = new(v7actionfakes.FakeConfig)
		actor = NewActor(fakeCloudControllerClient, fakeConfig, fakeSharedActor, nil, nil)
	})

	Context("UpdateApplicationLabelsByApplicationName", func() {
		JustBeforeEach(func() {
			warnings, executeErr = actor.UpdateApplicationLabelsByApplicationName(resourceName, spaceGUID, labels)
		})

		When("there are no client errors", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetApplicationsReturns(
					[]ccv3.Application{ccv3.Application{GUID: "some-guid"}},
					ccv3.Warnings([]string{"warning-1", "warning-2"}),
					nil,
				)
				fakeCloudControllerClient.UpdateResourceMetadataReturns(
					ccv3.ResourceMetadata{},
					ccv3.Warnings{"set-app-labels-warnings"},
					nil,
				)
			})

			It("sets the app labels", func() {
				Expect(fakeCloudControllerClient.UpdateResourceMetadataCallCount()).To(Equal(1))
				resourceType, appGUID, sentMetadata := fakeCloudControllerClient.UpdateResourceMetadataArgsForCall(0)
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(resourceType).To(BeEquivalentTo("app"))
				Expect(appGUID).To(BeEquivalentTo("some-guid"))
				Expect(sentMetadata.Labels).To(BeEquivalentTo(labels))
			})

			It("aggregates warnings", func() {
				Expect(warnings).To(ConsistOf("warning-1", "warning-2", "set-app-labels-warnings"))
			})
		})

		When("there are client errors", func() {
			When("GetApplications fails", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetApplicationsReturns(
						[]ccv3.Application{ccv3.Application{GUID: "some-guid"}},
						ccv3.Warnings([]string{"warning-failure-1", "warning-failure-2"}),
						errors.New("get-apps-error"),
					)
				})

				It("returns the error and all warnings", func() {
					Expect(executeErr).To(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-failure-1", "warning-failure-2"))
					Expect(executeErr).To(MatchError("get-apps-error"))
				})
			})

			When("UpdateApplication fails", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetApplicationsReturns(
						[]ccv3.Application{ccv3.Application{GUID: "some-guid"}},
						ccv3.Warnings([]string{"warning-1", "warning-2"}),
						nil,
					)
					fakeCloudControllerClient.UpdateResourceMetadataReturns(
						ccv3.ResourceMetadata{},
						ccv3.Warnings{"set-app-labels-warnings"},
						errors.New("update-application-error"),
					)
				})

				It("returns the error and all warnings", func() {
					Expect(executeErr).To(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-1", "warning-2", "set-app-labels-warnings"))
					Expect(executeErr).To(MatchError("update-application-error"))
				})
			})

		})
	})

	Context("UpdateDomainLabelsByDomainName", func() {
		JustBeforeEach(func() {
			warnings, executeErr = actor.UpdateDomainLabelsByDomainName(resourceName, labels)
		})

		When("there are no client errors", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetDomainsReturns(
					[]ccv3.Domain{ccv3.Domain{GUID: "some-guid"}},
					ccv3.Warnings([]string{"warning-1", "warning-2"}),
					nil,
				)
				fakeCloudControllerClient.UpdateResourceMetadataReturns(
					ccv3.ResourceMetadata{},
					ccv3.Warnings{"warning-updating-metadata"},
					nil,
				)
			})

			It("sets the domain labels", func() {
				Expect(fakeCloudControllerClient.UpdateResourceMetadataCallCount()).To(Equal(1))
				resourceType, domainGUID, sentMetadata := fakeCloudControllerClient.UpdateResourceMetadataArgsForCall(0)
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(resourceType).To(BeEquivalentTo("domain"))
				Expect(domainGUID).To(BeEquivalentTo("some-guid"))
				Expect(sentMetadata.Labels).To(BeEquivalentTo(labels))
			})

			It("aggregates warnings", func() {
				Expect(warnings).To(ConsistOf("warning-1", "warning-2", "warning-updating-metadata"))
			})
		})

		When("there are client errors", func() {
			When("fetching the domain fails", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetDomainsReturns(
						[]ccv3.Domain{ccv3.Domain{GUID: "some-guid"}},
						ccv3.Warnings([]string{"warning-failure-1", "warning-failure-2"}),
						errors.New("get-domains-error"),
					)
				})

				It("returns the error and all warnings", func() {
					Expect(executeErr).To(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-failure-1", "warning-failure-2"))
					Expect(executeErr).To(MatchError("get-domains-error"))
				})
			})

			When("updating the domain fails", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetDomainsReturns(
						[]ccv3.Domain{ccv3.Domain{GUID: "some-guid"}},
						ccv3.Warnings([]string{"warning-1", "warning-2"}),
						nil,
					)
					fakeCloudControllerClient.UpdateResourceMetadataReturns(
						ccv3.ResourceMetadata{},
						ccv3.Warnings{"warning-updating-metadata"},
						errors.New("update-domain-error"),
					)
				})
				It("returns the error and all warnings", func() {
					Expect(executeErr).To(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-1", "warning-2", "warning-updating-metadata"))
					Expect(executeErr).To(MatchError("update-domain-error"))
				})
			})
		})
	})

	Context("UpdateOrganizationLabelsByOrganizationName", func() {
		JustBeforeEach(func() {
			warnings, executeErr = actor.UpdateOrganizationLabelsByOrganizationName(resourceName, labels)
		})

		When("there are no client errors", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetOrganizationsReturns(
					[]ccv3.Organization{ccv3.Organization{GUID: "some-guid"}},
					ccv3.Warnings([]string{"warning-1", "warning-2"}),
					nil,
				)
				fakeCloudControllerClient.UpdateResourceMetadataReturns(
					ccv3.ResourceMetadata{},
					ccv3.Warnings{"set-org"},
					nil,
				)
			})

			It("sets the org labels", func() {
				Expect(fakeCloudControllerClient.UpdateResourceMetadataCallCount()).To(Equal(1))
				resourceType, orgGUID, sentMetadata := fakeCloudControllerClient.UpdateResourceMetadataArgsForCall(0)
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(resourceType).To(BeEquivalentTo("org"))
				Expect(orgGUID).To(BeEquivalentTo("some-guid"))
				Expect(sentMetadata.Labels).To(BeEquivalentTo(labels))
			})

			It("aggregates warnings", func() {
				Expect(warnings).To(ConsistOf("warning-1", "warning-2", "set-org"))
			})
		})

		When("there are client errors", func() {
			When("fetching the organization fails", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetOrganizationsReturns(
						[]ccv3.Organization{ccv3.Organization{GUID: "some-guid"}},
						ccv3.Warnings([]string{"warning-failure-1", "warning-failure-2"}),
						errors.New("get-orgs-error"),
					)
				})

				It("returns the error and all warnings", func() {
					Expect(executeErr).To(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-failure-1", "warning-failure-2"))
					Expect(executeErr).To(MatchError("get-orgs-error"))
				})
			})

			When("updating the organization fails", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetOrganizationsReturns(
						[]ccv3.Organization{ccv3.Organization{GUID: "some-guid"}},
						ccv3.Warnings([]string{"warning-1", "warning-2"}),
						nil,
					)
					fakeCloudControllerClient.UpdateResourceMetadataReturns(
						ccv3.ResourceMetadata{},
						ccv3.Warnings{"set-org"},
						errors.New("update-orgs-error"),
					)
				})
				It("returns the error and all warnings", func() {
					Expect(executeErr).To(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-1", "warning-2", "set-org"))
					Expect(executeErr).To(MatchError("update-orgs-error"))
				})
			})
		})
	})

	Context("UpdateRouteLabels", func() {
		JustBeforeEach(func() {
			warnings, executeErr = actor.UpdateRouteLabels("sub.example.com/my-route/path", "space-guid", labels)
		})

		When("there are no client errors", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetDomainsReturns(
					[]ccv3.Domain{
						{Name: "domain-name", GUID: "domain-guid"},
					},
					ccv3.Warnings{"get-domains-warning"},
					nil,
				)

				fakeCloudControllerClient.GetSpacesReturns(
					[]ccv3.Space{
						{Name: "space-name", GUID: "space-guid"},
					},
					ccv3.Warnings{"get-spaces-warning"},
					nil,
				)

				fakeCloudControllerClient.GetRoutesReturns(
					[]ccv3.Route{
						{GUID: "route-guid", SpaceGUID: "space-guid", DomainGUID: "domain-guid", Host: "hostname", URL: "hostname.domain-name", Path: "/the-path"},
					},
					ccv3.Warnings{"get-route-warning-1", "get-route-warning-2"},
					nil,
				)

				fakeCloudControllerClient.UpdateResourceMetadataReturns(
					ccv3.ResourceMetadata{},
					ccv3.Warnings{"set-route-warning"},
					nil,
				)
			})

			It("sets the route labels", func() {
				Expect(fakeCloudControllerClient.UpdateResourceMetadataCallCount()).To(Equal(1))
				resourceType, routeGUID, sentMetadata := fakeCloudControllerClient.UpdateResourceMetadataArgsForCall(0)
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(resourceType).To(BeEquivalentTo("route"))
				Expect(routeGUID).To(BeEquivalentTo("route-guid"))
				Expect(sentMetadata.Labels).To(BeEquivalentTo(labels))
			})

			It("aggregates warnings", func() {
				Expect(warnings).To(ConsistOf("get-domains-warning", "get-route-warning-1", "get-route-warning-2", "get-spaces-warning", "set-route-warning"))
			})
		})

		When("there are client errors", func() {
			When("fetching the route fails", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetDomainsReturns(
						nil,
						ccv3.Warnings{"get-domains-warning"},
						errors.New("get-domain-error"),
					)
				})

				It("returns the error and all warnings", func() {
					Expect(executeErr).To(HaveOccurred())
					Expect(warnings).To(ConsistOf("get-domains-warning"))
					Expect(executeErr).To(MatchError("get-domain-error"))
				})
			})

			When("updating the route fails", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetDomainsReturns(
						[]ccv3.Domain{
							{Name: "domain-name", GUID: "domain-guid"},
						},
						ccv3.Warnings{"get-domains-warning"},
						nil,
					)

					fakeCloudControllerClient.GetSpacesReturns(
						[]ccv3.Space{
							{Name: "space-name", GUID: "space-guid"},
						},
						ccv3.Warnings{"get-spaces-warning"},
						nil,
					)

					fakeCloudControllerClient.GetRoutesReturns(
						[]ccv3.Route{
							{GUID: "route-guid", SpaceGUID: "space-guid", DomainGUID: "domain-guid", Host: "hostname", URL: "hostname.domain-name", Path: "/the-path"},
						},
						ccv3.Warnings{"get-route-warning-1", "get-route-warning-2"},
						nil,
					)

					fakeCloudControllerClient.UpdateResourceMetadataReturns(
						ccv3.ResourceMetadata{},
						ccv3.Warnings{"set-route-warning"},
						errors.New("update-route-error"),
					)
				})

				It("returns the error and all warnings", func() {
					Expect(executeErr).To(HaveOccurred())
					Expect(warnings).To(ConsistOf("get-domains-warning", "get-route-warning-1", "get-route-warning-2", "get-spaces-warning", "set-route-warning"))
					Expect(executeErr).To(MatchError("update-route-error"))
				})
			})
		})
	})

	Context("UpdateSpaceLabelsBySpaceName", func() {
		JustBeforeEach(func() {
			warnings, executeErr = actor.UpdateSpaceLabelsBySpaceName(resourceName, orgGUID, labels)
		})

		When("there are no client errors", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetSpacesReturns(
					[]ccv3.Space{ccv3.Space{GUID: "some-guid"}},
					ccv3.Warnings([]string{"warning-1", "warning-2"}),
					nil,
				)
				fakeCloudControllerClient.UpdateResourceMetadataReturns(
					ccv3.ResourceMetadata{},
					ccv3.Warnings{"set-space-metadata"},
					nil,
				)
			})

			It("sets the space labels", func() {
				Expect(fakeCloudControllerClient.UpdateResourceMetadataCallCount()).To(Equal(1))
				resourceType, spaceGUID, sentMetadata := fakeCloudControllerClient.UpdateResourceMetadataArgsForCall(0)
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(resourceType).To(BeEquivalentTo("space"))
				Expect(spaceGUID).To(BeEquivalentTo("some-guid"))
				Expect(sentMetadata.Labels).To(BeEquivalentTo(labels))
			})

			It("aggregates warnings", func() {
				Expect(warnings).To(ConsistOf("warning-1", "warning-2", "set-space-metadata"))
			})
		})

		When("there are client errors", func() {
			When("fetching the space fails", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetSpacesReturns(
						[]ccv3.Space{ccv3.Space{GUID: "some-guid"}},
						ccv3.Warnings([]string{"warning-failure-1", "warning-failure-2"}),
						errors.New("get-spaces-error"),
					)
				})

				It("returns the error and all warnings", func() {
					Expect(executeErr).To(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-failure-1", "warning-failure-2"))
					Expect(executeErr).To(MatchError("get-spaces-error"))
				})
			})

			When("updating the space fails", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetSpacesReturns(
						[]ccv3.Space{ccv3.Space{GUID: "some-guid"}},
						ccv3.Warnings([]string{"warning-1", "warning-2"}),
						nil,
					)
					fakeCloudControllerClient.UpdateResourceMetadataReturns(
						ccv3.ResourceMetadata{},
						ccv3.Warnings{"set-space"},
						errors.New("update-space-error"),
					)
				})
				It("returns the error and all warnings", func() {
					Expect(executeErr).To(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-1", "warning-2", "set-space"))
					Expect(executeErr).To(MatchError("update-space-error"))
				})
			})
		})
	})

	Context("UpdateStackLabelsByStackName", func() {
		JustBeforeEach(func() {
			warnings, executeErr = actor.UpdateStackLabelsByStackName(resourceName, labels)
		})

		When("there are no client errors", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetStacksReturns(
					[]ccv3.Stack{ccv3.Stack{GUID: "some-guid"}},
					ccv3.Warnings([]string{"warning-1", "warning-2"}),
					nil,
				)
				fakeCloudControllerClient.UpdateResourceMetadataReturns(
					ccv3.ResourceMetadata{},
					ccv3.Warnings{"set-stack-metadata"},
					nil,
				)
			})

			It("sets the stack labels", func() {
				Expect(fakeCloudControllerClient.UpdateResourceMetadataCallCount()).To(Equal(1))
				resourceType, stackGUID, sentMetadata := fakeCloudControllerClient.UpdateResourceMetadataArgsForCall(0)
				Expect(executeErr).ToNot(HaveOccurred())
				Expect(resourceType).To(BeEquivalentTo("stack"))
				Expect(stackGUID).To(BeEquivalentTo("some-guid"))
				Expect(sentMetadata.Labels).To(BeEquivalentTo(labels))
			})

			It("aggregates warnings", func() {
				Expect(warnings).To(ConsistOf("warning-1", "warning-2", "set-stack-metadata"))
			})
		})

		When("there are client errors", func() {
			When("fetching the stack fails", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetStacksReturns(
						[]ccv3.Stack{ccv3.Stack{GUID: "some-guid"}},
						ccv3.Warnings([]string{"warning-failure-1", "warning-failure-2"}),
						errors.New("get-stacks-error"),
					)
				})

				It("returns the error and all warnings", func() {
					Expect(executeErr).To(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-failure-1", "warning-failure-2"))
					Expect(executeErr).To(MatchError("get-stacks-error"))
				})
			})

			When("updating the stack fails", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetStacksReturns(
						[]ccv3.Stack{ccv3.Stack{GUID: "some-guid"}},
						ccv3.Warnings([]string{"warning-1", "warning-2"}),
						nil,
					)
					fakeCloudControllerClient.UpdateResourceMetadataReturns(
						ccv3.ResourceMetadata{},
						ccv3.Warnings{"set-stack"},
						errors.New("update-stack-error"),
					)
				})
				It("returns the error and all warnings", func() {
					Expect(executeErr).To(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-1", "warning-2", "set-stack"))
					Expect(executeErr).To(MatchError("update-stack-error"))
				})
			})
		})
	})

	Context("GetDomainLabels", func() {
		JustBeforeEach(func() {
			labels, warnings, executeErr = actor.GetDomainLabels(resourceName)
		})

		When("there are no client errors", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetDomainsReturns(
					[]ccv3.Domain{ccv3.Domain{GUID: "some-guid"}},
					ccv3.Warnings([]string{"warning-1", "warning-2"}),
					nil,
				)
			})

			When("there are no labels on a domain", func() {
				It("returns an empty map", func() {
					Expect(executeErr).NotTo(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-1", "warning-2"))
					Expect(labels).To(BeEmpty())
				})
			})

			When("there are labels", func() {
				var expectedLabels map[string]types.NullString

				BeforeEach(func() {
					expectedLabels = map[string]types.NullString{"key1": types.NewNullString("value1"), "key2": types.NewNullString("value2")}
					fakeCloudControllerClient.GetDomainsReturns(
						[]ccv3.Domain{ccv3.Domain{
							GUID: "some-guid",
							Metadata: &ccv3.Metadata{
								Labels: expectedLabels,
							}}},
						ccv3.Warnings([]string{"warning-1", "warning-2"}),
						nil,
					)
				})
				It("returns the labels", func() {
					Expect(executeErr).NotTo(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-1", "warning-2"))
					Expect(labels).To(Equal(expectedLabels))
				})
			})
		})

		When("there is a client error", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetDomainsReturns(
					[]ccv3.Domain{ccv3.Domain{GUID: "some-guid"}},
					ccv3.Warnings([]string{"warning-1", "warning-2"}),
					errors.New("get-domains-error"),
				)
			})
			When("GetDomainByName fails", func() {
				It("returns the error and all warnings", func() {
					Expect(executeErr).To(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-1", "warning-2"))
					Expect(executeErr).To(MatchError("get-domains-error"))
				})
			})
		})
	})

	Context("GetStackLabels", func() {
		JustBeforeEach(func() {
			labels, warnings, executeErr = actor.GetStackLabels(resourceName)
		})

		When("there are no client errors", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetStacksReturns(
					[]ccv3.Stack{ccv3.Stack{GUID: "some-guid"}},
					ccv3.Warnings([]string{"warning-1", "warning-2"}),
					nil,
				)
			})

			When("there are no labels on a stack", func() {
				It("returns an empty map", func() {
					Expect(executeErr).NotTo(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-1", "warning-2"))
					Expect(labels).To(BeEmpty())
				})
			})

			When("there are labels", func() {
				var expectedLabels map[string]types.NullString

				BeforeEach(func() {
					expectedLabels = map[string]types.NullString{"key1": types.NewNullString("value1"), "key2": types.NewNullString("value2")}
					fakeCloudControllerClient.GetStacksReturns(
						[]ccv3.Stack{ccv3.Stack{
							GUID: "some-guid",
							Metadata: &ccv3.Metadata{
								Labels: expectedLabels,
							}}},
						ccv3.Warnings([]string{"warning-1", "warning-2"}),
						nil,
					)
				})
				It("returns the labels", func() {
					Expect(executeErr).NotTo(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-1", "warning-2"))
					Expect(labels).To(Equal(expectedLabels))
				})
			})
		})

		When("there is a client error", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetStacksReturns(
					[]ccv3.Stack{ccv3.Stack{GUID: "some-guid"}},
					ccv3.Warnings([]string{"warning-1", "warning-2"}),
					errors.New("get-stacks-error"),
				)
			})
			When("GetStackByName fails", func() {
				It("returns the error and all warnings", func() {
					Expect(executeErr).To(HaveOccurred())
					Expect(warnings).To(ConsistOf("warning-1", "warning-2"))
					Expect(executeErr).To(MatchError("get-stacks-error"))
				})
			})
		})
	})
})
