package controllers

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	. "github.com/onsi/gomega/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	eventingv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	"github.com/kyma-project/kyma/components/eventing-controller/controllers/testing"
	bebtypes "github.com/kyma-project/kyma/components/eventing-controller/pkg/ems2/api/events/types"

	// gcp auth etc.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// +kubebuilder:scaffold:imports
)

const (
	subscriptionNamespacePrefix = "test-"
	subscriptionID              = "test-subs-1"
)

var _ = Describe("Subscription", func() {
	var namespaceName string

	// enable me for debugging
	// SetDefaultEventuallyTimeout(time.Minute)
	// SetDefaultEventuallyPollingInterval(time.Second)

	BeforeEach(func() {
		namespaceName = getUniqueNamespaceName()
		// we need to reset the http requests which the mock captured
		beb.Reset()
	})

	AfterEach(func() {
		// detailed request logs
		logf.Log.V(1).Info("beb requests", "number", len(beb.Requests))

		i := 0
		for req, payloadObject := range beb.Requests {
			reqDescription := fmt.Sprintf("method: %q, url: %q, payload object: %+v", req.Method, req.RequestURI, payloadObject)
			fmt.Printf("request[%d]: %s\n", i, reqDescription)
			i++
		}

		// print all subscriptions in the namespace for debugging purposes
		if err := printSubscriptions(namespaceName); err != nil {
			logf.Log.Error(err, "error while printing subscriptions")
		}
	})

	Context("When creating a valid Subscription", func() {
		It("Should reconcile the Subscription", func() {
			subscriptionName := "test-valid-subscription-1"
			ctx := context.Background()
			givenSubscription := fixtureValidSubscription(subscriptionName, namespaceName)
			ensureSubscriptionCreated(givenSubscription, ctx)
			subscriptionLookupKey := types.NamespacedName{Name: subscriptionName, Namespace: namespaceName}

			By("Setting a finalizer")
			var subscription = &eventingv1alpha1.Subscription{}
			getSubscription(subscription, subscriptionLookupKey, ctx).Should(And(
				Not(BeNil()),
				haveName(subscriptionName),
				haveFinalizer(FinalizerName),
			))

			By("Setting a Subscribed condition")
			condition := eventingv1alpha1.MakeCondition(eventingv1alpha1.ConditionSubscribed, eventingv1alpha1.ConditionReasonSubscriptionCreated, v1.ConditionTrue)
			getSubscription(subscription, subscriptionLookupKey, ctx).Should(And(
				Not(BeNil()),
				haveName(subscriptionName),
				haveFinalizer(FinalizerName),
				haveCondition(condition),
			))

			By("Creating a BEB Subscription")
			var bebSubscription bebtypes.Subscription
			Eventually(func() bool {
				for r, payloadObject := range beb.Requests {
					if testing.IsBebSubscriptionCreate(r, *beb.BebConfig) {
						bebSubscription = payloadObject.(bebtypes.Subscription)
						receivedSubscriptionName := bebSubscription.Name
						// ensure the correct subscription was created
						return subscriptionName == receivedSubscriptionName
					}
				}
				return false
			}).Should(BeTrue())

			By("Emitting some k8s events")
			var subscriptionEvents = v1.EventList{}
			// TODO: adjust to event we want to have
			event := v1.Event{
				Reason:  string(eventingv1alpha1.ConditionReasonSubscriptionActive),
				Message: "",
				Type:    v1.EventTypeNormal,
			}
			Eventually(func() v1.EventList {
				err := k8sClient.List(ctx, &subscriptionEvents, client.InNamespace(namespaceName))
				if err != nil {
					return v1.EventList{}
				}
				return subscriptionEvents

			}).Should(haveEvent(event))

		})
	})

	Context("When deleting a valid Subscription", func() {
		It("Should reconcile the Subscription", func() {
			By(fmt.Sprintf("Using unique namespace name: %s", namespaceName))

			subscriptionName := "test-delete-valid-subscription-1"
			ctx := context.Background()
			givenSubscription := fixtureValidSubscription(subscriptionName, namespaceName)
			ensureSubscriptionCreated(givenSubscription, ctx)
			subscriptionLookupKey := types.NamespacedName{Name: subscriptionName, Namespace: namespaceName}

			// ensure subscription is given
			var subscription = &eventingv1alpha1.Subscription{}
			getSubscription(subscription, subscriptionLookupKey, ctx).Should(And(
				Not(BeNil()),
				haveName(subscriptionName),
				haveFinalizer(FinalizerName),
			))

			By("Creating a BEB Subscription")
			var bebSubscription bebtypes.Subscription
			processedBebRequests := 0
			Eventually(func() bool {
				for r, payloadObject := range beb.Requests {
					if testing.IsBebSubscriptionCreate(r, *beb.BebConfig) {
						bebSubscription = payloadObject.(bebtypes.Subscription)
						receivedSubscriptionName := bebSubscription.Name
						// ensure the correct subscription was created
						return subscriptionName == receivedSubscriptionName
					}
					processedBebRequests++
				}
				return false
			}).Should(BeTrue())

			By("Deleting the Subscription")
			Expect(k8sClient.Delete(ctx, subscription)).Should(BeNil())

			By("Deleting the BEB Subscription")
			Eventually(func() bool {
				i := -1
				for r, _ := range beb.Requests {
					i++
					// only consider requests against beb after the subscription creation request
					if i <= processedBebRequests {
						continue
					}
					if testing.IsBebSubscriptionDelete(r) {
						receivedSubscriptionName := testing.GetRestAPIObject(r.URL)
						// ensure the correct subscription was created
						return subscriptionName == receivedSubscriptionName
					}
				}
				return false
			}).Should(BeTrue())

			By("Removing the finalizer")
			getSubscription(subscription, subscriptionLookupKey, ctx).Should(BeNil())

			By("Emitting some k8s events")

			// TODO(nachtmaar):
		})
	})

	DescribeTable("Schema tests",

		func(subscription *eventingv1alpha1.Subscription) {
			ctx := context.Background()
			subscription.Namespace = namespaceName

			By("Letting the APIServer reject the custom resource")
			ensureSubscriptionCreationFails(subscription, ctx)
		},
		Entry("filter missing",
			func() *eventingv1alpha1.Subscription {
				subscription := fixtureValidSubscription("schema-filter-missing", "")
				subscription.Spec.Filter = nil
				return subscription
			}()),
		Entry("protocolsettings missing",
			func() *eventingv1alpha1.Subscription {
				subscription := fixtureValidSubscription("schema-filter-missing", "")
				subscription.Spec.ProtocolSettings = nil
				return subscription
			}()),
		Entry("protocolsettings.webhookauth missing",
			func() *eventingv1alpha1.Subscription {
				subscription := fixtureValidSubscription("schema-filter-missing", "")
				subscription.Spec.ProtocolSettings.WebhookAuth = nil
				return subscription
			}()),
	)

})

// fixtureValidSubscription returns a valid subscription
func fixtureValidSubscription(name, namespace string) *eventingv1alpha1.Subscription {
	return &eventingv1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Subscription",
			APIVersion: "eventing.kyma-project.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: eventingv1alpha1.SubscriptionSpec{
			Id:       subscriptionID,
			Protocol: "BEB",
			ProtocolSettings: &eventingv1alpha1.ProtocolSettings{
				ContentMode:     eventingv1alpha1.ProtocolSettingsContentModeBinary,
				ExemptHandshake: true,
				Qos:             "AT-LEAST_ONCE",
				WebhookAuth: &eventingv1alpha1.WebhookAuth{
					Type:         "oauth2",
					GrantType:    "client_credentials",
					ClientId:     "xxx",
					ClientSecret: "xxx",
					TokenUrl:     "https://oauth2.xxx.com/oauth2/token",
					Scope:        []string{"guid-identifier"},
				},
			},
			Sink: "https://webhook.xxx.com",
			Filter: &eventingv1alpha1.BebFilters{
				Dialect: "beb",
				Filters: []*eventingv1alpha1.BebFilter{
					{
						EventSource: &eventingv1alpha1.Filter{
							Type:     "exact",
							Property: "source",
							Value:    "/default/kyma/myinstance",
						},
						EventType: &eventingv1alpha1.Filter{
							Type:     "exact",
							Property: "type",
							Value:    "kyma.ev2.poc.event1.v1",
						},
					},
				},
			},
		},
	}
}

// TODO: document
func getSubscription(subscription *eventingv1alpha1.Subscription, lookupKey types.NamespacedName, ctx context.Context) AsyncAssertion {
	return Eventually(func() *eventingv1alpha1.Subscription {
		if err := k8sClient.Get(ctx, lookupKey, subscription); err != nil {
			return nil
		}
		return subscription
	}, time.Second*10, time.Second)
}

// ensureSubscriptionCreated creates a Subscription in the k8s cluster. If a custom namespace is used, it will be created as well.
func ensureSubscriptionCreated(subscription *eventingv1alpha1.Subscription, ctx context.Context) {
	if subscription.Namespace != "default " {
		// create testing namespace
		namespace := fixtureNamespace(subscription.Namespace)
		if namespace.Name != "default" {
			Expect(k8sClient.Create(ctx, namespace)).Should(Or(
				// ignore if namespaces is already created
				// isK8sAlreadyExistsError(),
				BeNil(),
			))
		}
	}
	// create subscription
	Expect(k8sClient.Create(ctx, subscription)).Should(BeNil())
}

// ensureSubscriptionCreationFails creates a Subscription in the k8s cluster and ensures that it is reject because of invalid schema
func ensureSubscriptionCreationFails(subscription *eventingv1alpha1.Subscription, ctx context.Context) {
	if subscription.Namespace != "default " {
		namespace := fixtureNamespace(subscription.Namespace)
		if namespace.Name != "default" {
			Expect(k8sClient.Create(ctx, namespace)).Should(BeNil())
		}
	}
	Expect(k8sClient.Create(ctx, subscription)).Should(
		And(
			// prevent nil-pointer stacktrace
			Not(BeNil()),
			isK8sUnprocessableEntity(),
		),
	)
}

func fixtureNamespace(name string) *v1.Namespace {
	namespace := v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return &namespace
}

// TODO: add subscription prefix or move to subscription package
// TODO: move matchers  to extra file ?
func haveName(name string) GomegaMatcher {
	return WithTransform(func(s *eventingv1alpha1.Subscription) string { return s.Name }, Equal(name))
}

func haveFinalizer(finalizer string) GomegaMatcher {
	return WithTransform(func(s *eventingv1alpha1.Subscription) []string { return s.ObjectMeta.Finalizers }, ContainElement(finalizer))
}

func haveCondition(condition eventingv1alpha1.Condition) GomegaMatcher {
	return WithTransform(func(s *eventingv1alpha1.Subscription) []eventingv1alpha1.Condition { return s.Status.Conditions }, ContainElement(MatchFields(IgnoreExtras|IgnoreMissing, Fields{
		"Type":    Equal(condition.Type),
		"Reason":  Equal(condition.Reason),
		"Message": Equal(condition.Message),
	})))
}

func haveEvent(event v1.Event) GomegaMatcher {
	return WithTransform(func(l v1.EventList) []v1.Event { return l.Items }, ContainElement(MatchFields(IgnoreExtras|IgnoreMissing, Fields{
		"Reason":  Equal(event.Reason),
		"Message": Equal(event.Message),
		"Type":    Equal(event.Type),
	})))
}

func isK8sUnprocessableEntity() GomegaMatcher {
	// TODO: also check for status code 422
	//  <*errors.StatusError | 0xc0001330e0>: {
	//     ErrStatus: {
	//         TypeMeta: {Kind: "", APIVersion: ""},
	//         ListMeta: {
	//             SelfLink: "",
	//             ResourceVersion: "",
	//             Continue: "",
	//             RemainingItemCount: nil,
	//         },
	//         Status: "Failure",
	//         Message: "Subscription.eventing.kyma-project.io \"test-valid-subscription-1\" is invalid: spec.filter: Invalid value: \"null\": spec.filter in body must be of type object: \"null\"",
	//         Reason: "Invalid",
	//         Details: {
	//             Name: "test-valid-subscription-1",
	//             Group: "eventing.kyma-project.io",
	//             Kind: "Subscription",
	//             UID: "",
	//             Causes: [
	//                 {
	//                     Type: "FieldValueInvalid",
	//                     Message: "Invalid value: \"null\": spec.filter in body must be of type object: \"null\"",
	//                     Field: "spec.filter",
	//                 },
	//             ],
	//             RetryAfterSeconds: 0,
	//         },
	//         Code: 422,
	//     },
	// }
	return WithTransform(func(err *errors.StatusError) metav1.StatusReason { return err.ErrStatus.Reason }, Equal(metav1.StatusReasonInvalid))
}

// printSubscriptions prints all subscriptions in the given namespace
func printSubscriptions(namespace string) error {
	// print subscription details
	ctx := context.TODO()
	subscriptionList := eventingv1alpha1.SubscriptionList{}
	if err := k8sClient.List(ctx, &subscriptionList, client.InNamespace(namespace)); err != nil {
		logf.Log.V(1).Info("error while getting subscription list", "error", err)
		return err
	}
	logf.Log.V(1).Info("subscriptions", "subscriptions", subscriptionList)
	return nil
}

func generateTestSuiteID() int {
	var seededRand = rand.New(
		rand.NewSource(time.Now().UnixNano()))
	return seededRand.Int()
}

func getUniqueNamespaceName() string {
	testSuiteID := generateTestSuiteID()
	namespaceName := fmt.Sprintf("%s%d", subscriptionNamespacePrefix, testSuiteID)
	return namespaceName
}
