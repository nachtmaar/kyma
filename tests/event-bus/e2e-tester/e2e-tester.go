package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/avast/retry-go"
	api "github.com/kyma-project/kyma/components/event-bus/api/publish"
	subApis "github.com/kyma-project/kyma/components/event-bus/api/push/eventing.kyma-project.io/v1alpha1"
	eaClientSet "github.com/kyma-project/kyma/components/event-bus/generated/ea/clientset/versioned"
	subscriptionClientSet "github.com/kyma-project/kyma/components/event-bus/generated/push/clientset/versioned"
	"github.com/kyma-project/kyma/components/event-bus/test/util"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	eventType               = "test-e2e"
	eventTypeVersion        = "v1"
	subscriptionName        = "test-sub"
	headersSubscriptionName = "headers-test-sub"
	eventActivationName     = "test-ea"
	srcID                   = "test.local"

	success = 0
	fail    = 1

	idHeader               = "ce-id"
	timeHeader             = "ce-time"
	contentTypeHeader      = "content-type"
	sourceHeader           = "ce-source"
	eventTypeHeader        = "ce-type"
	eventTypeVersionHeader = "ce-eventtypeversion"
	customHeader           = "ce-xcustomheader"

	ceSourceIDHeaderValue         = "override-source-ID"
	ceEventTypeHeaderValue        = "override-event-type"
	contentTypeHeaderValue        = "application/json"
	ceEventTypeVersionHeaderValue = "override-event-type-version"
	customHeaderValue             = "Ce-X-custom-header-value"
)

var (
	clientK8S *kubernetes.Clientset
	eaClient  *eaClientSet.Clientset
	subClient *subscriptionClientSet.Clientset
)

//Unexportable struct, encapsulates subscriber resource parameters
type subscriberDetails struct {
	subscriberImage               string
	subscriberNamespace           string
	subscriberEventEndpointURL    string
	subscriberResultsEndpointURL  string
	subscriberStatusEndpointURL   string
	subscriberShutdownEndpointURL string
	subscriber3EventEndpointURL   string
	subscriber3ResultsEndpointURL string
	subscriber3StatusEndpointURL  string
}

//Unexportable struct, encapsulates publisher details
type publisherDetails struct {
	publishEventEndpointURL  string
	publishStatusEndpointURL string
}

func main() {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	var subDetails subscriberDetails
	var pubDetails publisherDetails

	//Initialise publisher struct
	flags.StringVar(&pubDetails.publishEventEndpointURL, "publish-event-uri", "http://event-bus-publish:8080/v1/events", "publish service events endpoint `URL`")
	flags.StringVar(&pubDetails.publishStatusEndpointURL, "publish-status-uri", "http://event-bus-publish:8080/v1/status/ready", "publish service status endpoint `URL`")

	//Initialise subscriber
	flags.StringVar(&subDetails.subscriberImage, "subscriber-image", "", "subscriber Docker `image` name")
	flags.StringVar(&subDetails.subscriberNamespace, "subscriber-ns", "default", "k8s `namespace` in which subscriber test app is running")
	if err := flags.Parse(os.Args[1:]); err != nil {
		panic(err)
	}

	initSubscriberUrls(&subDetails)

	if flags.NFlag() == 0 || subDetails.subscriberImage == "" {

		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flags.PrintDefaults()
		os.Exit(1)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("Error in getting cluster config: %v\n", err)
		shutdown(fail, &subDetails)
	}

	log.Println("Create the clientK8S")
	clientK8S, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("Failed to create a ClientSet: %v\n", err)
		shutdown(fail, &subDetails)
	}

	log.Println("Create a test event activation")
	eaClient, err = eaClientSet.NewForConfig(config)
	if err != nil {
		log.Printf("Error in creating event activation client: %v\n", err)
		shutdown(fail, &subDetails)
	}
	if !createEventActivation(subDetails.subscriberNamespace) {
		log.Println("Error: Cannot create the event activation")
		shutdown(fail, &subDetails)
	}

	log.Println("Create a test subscriptions")
	subClient, err = subscriptionClientSet.NewForConfig(config)
	if err != nil {
		log.Printf("Error in creating subscription client: %v\n", err)
		shutdown(fail, &subDetails)
	}
	if !createSubscription(subDetails.subscriberNamespace, subscriptionName, subDetails.subscriberEventEndpointURL) {
		log.Println("Error: Cannot create Kyma subscription")
		shutdown(fail, &subDetails)
	}

	log.Println("Create a headers test subscription")
	subClient, err = subscriptionClientSet.NewForConfig(config)
	if err != nil {
		log.Printf("Error in creating headers subscription client: %v\n", err)
		shutdown(fail, &subDetails)
	}
	if !createSubscription(subDetails.subscriberNamespace, headersSubscriptionName, subDetails.subscriber3EventEndpointURL) {
		log.Println("Error: Cannot create Kyma headers subscription")
		shutdown(fail, &subDetails)
	}

	log.Println("Create Subscriber")
	if err := createSubscriber(util.SubscriberName, subDetails.subscriberNamespace, subDetails.subscriberImage); err != nil {
		log.Printf("Create Subscriber failed: %v\n", err)
	}

	log.Println("Check Subscriber Status")
	if !subDetails.checkSubscriberStatus() {
		log.Println("Error: Cannot connect to Subscriber")
		shutdown(fail, &subDetails)
	}

	log.Println("Check Subscriber 3 Status")
	if !subDetails.checkSubscriber3Status() {
		log.Println("Error: Cannot connect to Subscriber 3")
		shutdown(fail, &subDetails)
	}

	log.Println("Check Publisher Status")
	if !pubDetails.checkPublisherStatus() {
		log.Println("Error: Cannot connect to Publisher")
		shutdown(fail, &subDetails)
	}

	log.Println("Check Kyma subscription ready Status")
	if !subDetails.checkSubscriptionReady(subscriptionName) {
		log.Println("Error: Kyma Subscription not ready")
		shutdown(fail, &subDetails)
	}

	log.Println("Check Kyma headers subscription ready Status")
	if !subDetails.checkSubscriptionReady(headersSubscriptionName) {
		log.Println("Error: Kyma Subscription not ready")
		shutdown(fail, &subDetails)
	}

	log.Println("Publish an event")
	err = retry.Do(func() error {
		_, err := publishTestEvent(pubDetails.publishEventEndpointURL)
		return err
	})
	if err != nil {
		log.Printf("Error: Publish event failed: %v\n", err)
		shutdown(fail, &subDetails)
	}

	log.Println("Try to read the response from subscriber server")
	if err := subDetails.checkSubscriberReceivedEvent(); err != nil {
		log.Printf("Error: Cannot get the test event from subscriber: %v\n", err)
		shutdown(fail, &subDetails)
	}

	log.Println("Publish headers event")
	err = retry.Do(func() error {
		_, err := publishHeadersTestEvent(pubDetails.publishEventEndpointURL)
		return err
	})
	if err != nil {
		log.Printf("Error: Publish headers event failed: %v\n", err)
		shutdown(fail, &subDetails)
	}

	log.Println("Try to read the response from subscriber 3 server")
	if err := subDetails.checkSubscriberReceivedEventHeaders(); err != nil {
		log.Printf("Error: Cannot get the test event from subscriber 3: %v\n", err)
		shutdown(fail, &subDetails)
	}

	log.Println("Successfully finished")
	shutdown(success, &subDetails)
}

// Initialize subscriber urls
func initSubscriberUrls(subDetails *subscriberDetails) {
	subDetails.subscriberEventEndpointURL = "http://" + util.SubscriberName + "." + subDetails.subscriberNamespace + ":9000/v1/events"
	subDetails.subscriberResultsEndpointURL = "http://" + util.SubscriberName + "." + subDetails.subscriberNamespace + ":9000/v1/results"
	subDetails.subscriberStatusEndpointURL = "http://" + util.SubscriberName + "." + subDetails.subscriberNamespace + ":9000/v1/status"
	subDetails.subscriberShutdownEndpointURL = "http://" + util.SubscriberName + "." + subDetails.subscriberNamespace + ":9000/shutdown"
	subDetails.subscriber3EventEndpointURL = "http://" + util.SubscriberName + "." + subDetails.subscriberNamespace + ":9000/v3/events"
	subDetails.subscriber3ResultsEndpointURL = "http://" + util.SubscriberName + "." + subDetails.subscriberNamespace + ":9000/v3/results"
	subDetails.subscriber3StatusEndpointURL = "http://" + util.SubscriberName + "." + subDetails.subscriberNamespace + ":9000/v3/status"
}

func shutdown(code int, subscriber *subscriberDetails) {
	log.Println("Send shutdown request to Subscriber")
	if _, err := http.Post(subscriber.subscriberShutdownEndpointURL, "application/json", strings.NewReader(`{"shutdown": "true"}`)); err != nil {
		log.Printf("Warning: Shutdown Subscriber failed: %v", err)
	}
	log.Println("Delete Subscriber deployment")
	deletePolicy := metav1.DeletePropagationForeground
	gracePeriodSeconds := int64(0)

	if err := clientK8S.AppsV1().Deployments(subscriber.subscriberNamespace).Delete(util.SubscriberName,
		&metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds, PropagationPolicy: &deletePolicy}); err != nil {
		log.Printf("Warning: Delete Subscriber Deployment failed: %v", err)
	}
	log.Println("Delete Subscriber service")
	if err := clientK8S.CoreV1().Services(subscriber.subscriberNamespace).Delete(util.SubscriberName,
		&metav1.DeleteOptions{GracePeriodSeconds: &gracePeriodSeconds}); err != nil {
		log.Printf("Warning: Delete Subscriber Service failed: %v", err)
	}
	if subClient != nil {
		log.Printf("Delete test subscription: %v\n", subscriptionName)
		if err := subClient.EventingV1alpha1().Subscriptions(subscriber.subscriberNamespace).Delete(subscriptionName,
			&metav1.DeleteOptions{PropagationPolicy: &deletePolicy}); err != nil {
			log.Printf("Warning: Delete Subscription failed: %v", err)
		}
	}
	if subClient != nil {
		log.Printf("Delete headers test subscription: %v\n", subscriptionName)
		if err := subClient.EventingV1alpha1().Subscriptions(subscriber.subscriberNamespace).Delete(headersSubscriptionName,
			&metav1.DeleteOptions{PropagationPolicy: &deletePolicy}); err != nil {
			log.Printf("Warning: Delete Subscription failed: %v", err)
		}
	}
	if eaClient != nil {
		log.Printf("Delete test event activation: %v\n", eventActivationName)
		if err := eaClient.ApplicationconnectorV1alpha1().EventActivations(subscriber.subscriberNamespace).Delete(eventActivationName, &metav1.DeleteOptions{PropagationPolicy: &deletePolicy}); err != nil {
			log.Printf("Warning: Delete Event Activation failed: %v", err)
		}
	}
	os.Exit(code)
}

func publishTestEvent(publishEventURL string) (*api.Response, error) {
	payload := fmt.Sprintf(
		`{"source-id": "%s","event-type":"%s","event-type-version":"%s","event-time":"2018-11-02T22:08:41+00:00","data":"test-event-1"}`, srcID, eventType, eventTypeVersion)
	log.Printf("event to be published: %v\n", payload)
	res, err := http.Post(publishEventURL, "application/json", strings.NewReader(payload))
	if err != nil {
		log.Printf("Post request failed: %v\n", err)
		return nil, err
	}
	dumpResponse(res)
	if err := verifyStatusCode(res, 200); err != nil {
		return nil, err
	}
	respObj := &api.Response{}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Println(err)
		}
	}()
	err = json.Unmarshal(body, &respObj)
	if err != nil {
		log.Printf("Unmarshal error: %v", err)
		return nil, err
	}
	log.Printf("Publish response object: %+v", respObj)
	if len(respObj.EventID) == 0 {
		return nil, fmt.Errorf("empty respObj.EventID")
	}
	return respObj, err
}

func publishHeadersTestEvent(publishEventURL string) (*api.Response, error) {
	payload := fmt.Sprintf(
		`{"source-id": "%s","event-type":"%s","event-type-version":"%s","event-time":"2018-11-02T22:08:41+00:00","data":"headers-test-event"}`, srcID, eventType, eventTypeVersion)
	log.Printf("event to be published: %v\n", payload)

	client := &http.Client{}
	req, err := http.NewRequest("POST", publishEventURL, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Add(sourceHeader, ceSourceIDHeaderValue)
	req.Header.Add(eventTypeHeader, ceEventTypeHeaderValue)
	req.Header.Add(eventTypeVersionHeader, ceEventTypeVersionHeaderValue)
	req.Header.Add(customHeader, customHeaderValue)
	res, err := client.Do(req)

	if err != nil {
		log.Printf("Post request failed: %v\n", err)
		return nil, err
	}
	dumpResponse(res)
	if err := verifyStatusCode(res, 200); err != nil {
		return nil, err
	}
	respObj := &api.Response{}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Print(err)
		}
	}()
	err = json.Unmarshal(body, &respObj)
	if err != nil {
		log.Printf("Unmarshal error: %v", err)
		return nil, err
	}
	log.Printf("Publish response object: %+v", respObj)
	if len(respObj.EventID) == 0 {
		return nil, fmt.Errorf("empty respObj.EventID")
	}
	return respObj, err
}

func (subDetails *subscriberDetails) checkSubscriberReceivedEvent() error {
	return retry.Do(func() error {
		res, err := http.Get(subDetails.subscriberResultsEndpointURL)
		if err != nil {
			return err
		}
		dumpResponse(res)
		if err := verifyStatusCode(res, 200); err != nil {
			return err
		}
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		var resp string
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return err
		}
		log.Printf("Subscriber response: %s\n", resp)
		defer func() {
			if err := res.Body.Close(); err != nil {
				log.Println(err)
			}
		}()
		if len(resp) == 0 {
			return errors.New("no event received by subscriber")
		}
		if resp != "test-event-1" {
			return fmt.Errorf("wrong response: %s, want: %s", resp, "test-event-1")
		}
		return nil
	})
}

func (subDetails *subscriberDetails) checkSubscriberReceivedEventHeaders() error {
	return retry.Do(func() error {
		res, err := http.Get(subDetails.subscriber3ResultsEndpointURL)
		if err != nil {
			log.Printf("Get request failed: %v\n", err)
			return err
		}
		dumpResponse(res)
		if err := verifyStatusCode(res, 200); err != nil {
			log.Printf("Get request failed: %v", err)
			return err
		}
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		var resp map[string][]string
		if err := json.Unmarshal(body, &resp); err != nil {
			return err
		}
		log.Printf("Subscriber 3 response: %v\n", resp)
		defer func() {
			if err := res.Body.Close(); err != nil {
				log.Println(err)
			}
		}()
		if len(resp) == 0 {
			return errors.New("no event received by subscriber")
		}

		lowerResponseHeaders := make(map[string][]string)
		for k := range resp {
			lowerResponseHeaders[strings.ToLower(k)] = resp[k]
		}

		var testDataSets = []struct {
			headerKey           string
			headerExpectedValue string
		}{
			{headerKey: sourceHeader, headerExpectedValue: srcID},
			{headerKey: eventTypeHeader, headerExpectedValue: eventType},
			{headerKey: eventTypeVersionHeader, headerExpectedValue: eventTypeVersion},
			{headerKey: contentTypeHeader, headerExpectedValue: contentTypeHeaderValue},
			{headerKey: customHeader, headerExpectedValue: customHeaderValue},
		}

		for _, testData := range testDataSets {
			// panic: runtime error: index out of range
			// main.(*subscriberDetails).checkSubscriberReceivedEventHeaders.func1(0x0, 0x0)
			// /go/src/github.com/kyma-project/kyma/tests/event-bus/e2e-tester/e2e-tester.go:435 +0xb37
			if lowerResponseHeaders[testData.headerKey][0] != testData.headerExpectedValue {
				if _, ok := lowerResponseHeaders[testData.headerKey]; !ok {
					log.Printf("map %+v does not contain key %v\n", lowerResponseHeaders, testData.headerKey)
					panic("foo")
					// return fmt.Errorf("map %+v does not contain key %v", lowerResponseHeaders, testData.headerKey)
				}
				return fmt.Errorf("wrong response: %s, want: %s", lowerResponseHeaders[testData.headerKey][0], testData.headerExpectedValue)
			}
		}

		if lowerResponseHeaders[idHeader][0] == "" {
			return fmt.Errorf("wrong response: %s, can't be empty", lowerResponseHeaders[idHeader][0])
		}
		if lowerResponseHeaders[timeHeader][0] == "" {
			return fmt.Errorf("wrong response: %s, can't be empty", lowerResponseHeaders[timeHeader][0])
		}
		return nil
	})
}

func dumpResponse(resp *http.Response) {
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	_, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Fatal(err)
	}
}

func checkStatusCode(res *http.Response, expectedStatusCode int) bool {
	if res.StatusCode != expectedStatusCode {
		log.Printf("Status code is wrong, have: %d, want: %d\n", res.StatusCode, expectedStatusCode)
		return false
	}
	return true
}

func checkPublishStatus(statusEndpointURL string) error {
	res, err := http.Get(statusEndpointURL)
	if err != nil {
		return err
	}
	return verifyStatusCode(res, http.StatusOK)
}

func verifyStatusCode(res *http.Response, expectedStatusCode int) error {
	if res.StatusCode != expectedStatusCode {
		return fmt.Errorf("status code is wrong, have: %d, want: %d", res.StatusCode, expectedStatusCode)
	}
	return nil
}

func isPodReady(pod *apiv1.Pod) bool {
	for _, cs := range pod.Status.ContainerStatuses {
		if !cs.Ready {
			return false
		}
	}
	return true
}

func createSubscriber(subscriberName string, subscriberNamespace string, sbscrImg string) error {
	if _, err := clientK8S.AppsV1().Deployments(subscriberNamespace).Get(subscriberName, metav1.GetOptions{}); err != nil {
		log.Println("Create Subscriber deployment")
		if _, err := clientK8S.AppsV1().Deployments(subscriberNamespace).Create(util.NewSubscriberDeployment(sbscrImg)); err != nil {
			log.Printf("Create Subscriber deployment: %v\n", err)
			return err
		}
		log.Println("Create Subscriber service")
		if _, err := clientK8S.CoreV1().Services(subscriberNamespace).Create(util.NewSubscriberService()); err != nil {
			log.Printf("Create Subscriber service failed: %v\n", err)
			return err
		}

		// wait until pod is ready
		return retry.Do(func() error {
			var podReady bool
			var podNotReady error
			pods, err := clientK8S.CoreV1().Pods(subscriberNamespace).List(metav1.ListOptions{LabelSelector: "app=" + util.SubscriberName})
			if err != nil {
				return err
			}
			// check if pod is ready
			for _, pod := range pods.Items {
				if podReady = isPodReady(&pod); !podReady {
					podNotReady = fmt.Errorf("Subscriber pod not ready: %+v;", pod)
					break
				}
			}
			if !podReady {
				return podNotReady
			}
			log.Println("Subscriber created")
			return nil
		})
	}
	return nil
}

// TODO: why is there a retry loop at all?
// we need to wait until the api server is reachable and accepting requests
func createEventActivation(subscriberNamespace string) bool {
	err := retry.Do(func() error {
		_, err := eaClient.ApplicationconnectorV1alpha1().EventActivations(subscriberNamespace).Create(util.NewEventActivation(eventActivationName, subscriberNamespace, srcID))
		if err == nil {
			return nil
		}
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("error in creating event activation - %v", err)
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
	return err == nil
}

func createSubscription(subscriberNamespace string, subName string, subscriberEventEndpointURL string) bool {
	_, err := subClient.EventingV1alpha1().Subscriptions(subscriberNamespace).Create(util.NewSubscription(subName, subscriberNamespace, subscriberEventEndpointURL, eventType, "v1", srcID))
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			log.Printf("Error in creating subscription: %v\n", err)
			return false
		}
	}
	return true
}

func (subDetails *subscriberDetails) checkSubscriberStatus() bool {
	err := retry.Do(func() error {
		if res, err := http.Get(subDetails.subscriberStatusEndpointURL); err != nil {
			return err
		} else if !checkStatusCode(res, http.StatusOK) {
			return fmt.Errorf("Subscriber Server Status request returns: %v", res)
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	return err == nil
}

func (subDetails *subscriberDetails) checkSubscriber3Status() bool {
	err := retry.Do(func() error {
		if res, err := http.Get(subDetails.subscriber3StatusEndpointURL); err != nil {
			return err
		} else if !checkStatusCode(res, http.StatusOK) {
			return fmt.Errorf("Subscriber 3 Server Status request returns: %v", res)
		}
		return nil
	})
	if err != nil {
		fmt.Print(err)
	}
	return err == nil
}

func (pubDetails *publisherDetails) checkPublisherStatus() bool {
	err := retry.Do(func() error {
		return checkPublishStatus(pubDetails.publishStatusEndpointURL)
	})
	if err != nil {
		log.Println(err)
	}

	return err == nil
}

func (subDetails *subscriberDetails) checkSubscriptionReady(subscriptionName string) bool {
	err := retry.Do(func() error {
		var isReady bool
		activatedCondition := subApis.SubscriptionCondition{Type: subApis.Ready, Status: subApis.ConditionTrue}
		kySub, err := subClient.EventingV1alpha1().Subscriptions(subDetails.subscriberNamespace).Get(subscriptionName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if isReady = kySub.HasCondition(activatedCondition); !isReady {
			return fmt.Errorf("Subscription %v is not ready yet", subscriptionName)
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	return err == nil
}
