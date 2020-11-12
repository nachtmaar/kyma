package testing

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/ory/oathkeeper-maester/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	. "github.com/onsi/gomega/types"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apigatewayv1alpha1 "github.com/kyma-incubator/api-gateway/api/v1alpha1"

	eventingv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/constants"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/object"
	"github.com/kyma-project/kyma/components/eventing-controller/utils"
)

//
// string matchers
//

func HaveSubscriptionName(name string) GomegaMatcher {
	return WithTransform(func(s eventingv1alpha1.Subscription) string { return s.Name }, Equal(name))
}

func HaveSubscriptionFinalizer(finalizer string) GomegaMatcher {
	return WithTransform(func(s eventingv1alpha1.Subscription) []string { return s.ObjectMeta.Finalizers }, ContainElement(finalizer))
}

func IsAnEmptySubscription() GomegaMatcher {
	return WithTransform(func(s eventingv1alpha1.Subscription) bool {
		emptySub := eventingv1alpha1.Subscription{}
		return reflect.DeepEqual(s, emptySub)
	}, BeTrue())
}

//
// APIRule matchers
//

func HaveNotEmptyAPIRule() GomegaMatcher {
	return WithTransform(func(a apigatewayv1alpha1.APIRule) types.UID {
		return a.UID
	}, Not(BeEmpty()))
}

func HaveNotEmptyHost() GomegaMatcher {
	return WithTransform(func(a apigatewayv1alpha1.APIRule) bool {
		return a.Spec.Service != nil && a.Spec.Service.Host != nil
	}, BeTrue())
}

func HaveAPIRuleSpec(ruleMethods []string, accessStrategy string) GomegaMatcher {
	return WithTransform(func(a apigatewayv1alpha1.APIRule) []apigatewayv1alpha1.Rule {
		return a.Spec.Rules
	}, ContainElement(
		MatchFields(IgnoreExtras|IgnoreMissing, Fields{
			"Methods":          ConsistOf(ruleMethods),
			"AccessStrategies": ConsistOf(haveAPIRuleAccessStrategies(accessStrategy)),
		}),
	))
}

func haveAPIRuleAccessStrategies(accessStrategy string) GomegaMatcher {
	return WithTransform(func(a *v1alpha1.Authenticator) string {
		return a.Name
	}, Equal(accessStrategy))
}

func HaveAPIRuleOwnersRefs(uid types.UID) GomegaMatcher {
	return WithTransform(func(a apigatewayv1alpha1.APIRule) []metav1.OwnerReference {
		return a.ObjectMeta.OwnerReferences
	}, ConsistOf(MatchFields(IgnoreExtras|IgnoreMissing, Fields{
		"UID": Equal(uid),
	})))
}

//
// Subscription matchers
//

func HaveNoneEmptyAPIRuleName() GomegaMatcher {
	return WithTransform(func(s eventingv1alpha1.Subscription) string {
		return s.Status.APIRuleName
	}, Not(BeEmpty()))
}

func HaveAPIRuleName(name string) GomegaMatcher {
	return WithTransform(func(s eventingv1alpha1.Subscription) bool {
		return s.Status.APIRuleName == name
	}, BeTrue())
}

func HaveSubscriptionReady() GomegaMatcher {
	return WithTransform(func(s eventingv1alpha1.Subscription) bool {
		return s.Status.Ready
	}, BeTrue())
}

// TODO: replace this with matchers
func HaveValidAPIRule(s eventingv1alpha1.Subscription) GomegaMatcher {
	return WithTransform(func(apiRule apigatewayv1alpha1.APIRule) bool {
		hasOwnRef, hasRule := false, false
		for _, or := range apiRule.OwnerReferences {
			if or.Name == s.Name && or.UID == s.UID {
				hasOwnRef = true
				break
			}
		}
		if !hasOwnRef {
			return false
		}
		// --

		sURL, err := url.ParseRequestURI(s.Spec.Sink)
		if err != nil {
			return false
		}

		sURLSplitArr := strings.Split(sURL.Host, ".")
		if len(sURLSplitArr) != 5 {
			return false
		}
		_, subscriberSvcName := sURLSplitArr[1], sURLSplitArr[0]
		if sURL.Path == "" {
			sURL.Path = "/"
		}
		for _, rule := range apiRule.Spec.Rules {
			if rule.Path == sURL.Path {
				if len(rule.Methods) != 2 {
					break
				}
				acceptableMethods := []string{http.MethodPost, http.MethodOptions}
				if !(containsString(acceptableMethods, rule.Methods[0]) && containsString(acceptableMethods, rule.Methods[1])) {
					break
				}
				if len(rule.AccessStrategies) != 1 {
					break
				}
				accessStrategy := rule.AccessStrategies[0]
				if accessStrategy.Handler.Name != object.OAuthHandlerName {
					break
				}
				hasRule = true
				break
			}
		}
		if !hasRule {
			return false
		}

		if *apiRule.Spec.Gateway != constants.ClusterLocalAPIGateway {
			return false
		}

		expectedLabels := map[string]string{
			constants.ControllerIdentityLabelKey: constants.ControllerIdentityLabelValue,
			constants.ControllerServiceLabelKey:  subscriberSvcName,
		}
		if !reflect.DeepEqual(expectedLabels, apiRule.Labels) {
			return false
		}

		if subscriberSvcName != *apiRule.Spec.Service.Name {
			return false
		}
		svcPort, err := utils.GetPortNumberFromURL(*sURL)
		if err != nil {
			return false
		}

		if svcPort != *apiRule.Spec.Service.Port {
			return false
		}

		return true
	}, BeTrue())
}

func containsString(list []string, contains string) bool {
	for _, elem := range list {
		if elem == contains {
			return true
		}
	}
	return false
}

func HaveCondition(condition eventingv1alpha1.Condition) GomegaMatcher {
	return WithTransform(func(s eventingv1alpha1.Subscription) []eventingv1alpha1.Condition { return s.Status.Conditions }, ContainElement(MatchFields(IgnoreExtras|IgnoreMissing, Fields{
		"Type":    Equal(condition.Type),
		"Reason":  Equal(condition.Reason),
		"Message": Equal(condition.Message),
	})))
}

func HaveEvent(event v1.Event) GomegaMatcher {
	return WithTransform(func(l v1.EventList) []v1.Event { return l.Items }, ContainElement(MatchFields(IgnoreExtras|IgnoreMissing, Fields{
		"Reason":  Equal(event.Reason),
		"Message": Equal(event.Message),
		"Type":    Equal(event.Type),
	})))
}

func IsK8sUnprocessableEntity() GomegaMatcher {
	return WithTransform(func(err *errors.StatusError) metav1.StatusReason { return err.ErrStatus.Reason }, Equal(metav1.StatusReasonInvalid))
}
