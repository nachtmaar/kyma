package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/avast/retry-go"
	cloudevents "github.com/cloudevents/sdk-go"
	cloudeventshttp "github.com/cloudevents/sdk-go/pkg/cloudevents/transport/http"
	"go.uber.org/zap"
	"io/ioutil"
	"knative.dev/eventing/pkg/adapter"
	"knative.dev/eventing/pkg/kncloudevents"
	"knative.dev/pkg/source"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const startPort = 54321

var testsReceiveBrokenEvent = []struct {
	name                string
	giveMessage         func() (*cloudeventshttp.Message, error)
	wantResponseMessage string
	wantResponseCode    int
}{
	{
		name: "send empty message",
		giveMessage: func() (*cloudeventshttp.Message, error) {
			return &cloudeventshttp.Message{
				Header: map[string][]string{
					"content-type": {""},
				},
				Body: nil,
			}, nil
		},
		// expect empty body
		wantResponseMessage: "",
		wantResponseCode:    http.StatusBadRequest,
	},
	{
		name: "send event - structured - only specversion",
		giveMessage: func() (*cloudeventshttp.Message, error) {

			body, err := json.Marshal(map[string]string{
				// to get to the event handler, there must be at least an event version
				"specversion": "1.0",
			})
			if err != nil {
				return nil, err
			}
			return &cloudeventshttp.Message{
				Header: map[string][]string{
					"content-type": {cloudevents.ApplicationCloudEventsJSON},
				},
				Body: body,
			}, nil
		},
		wantResponseMessage: `{"error":"id: MUST be a non-empty string\nsource: REQUIRED\ntype: MUST be a non-empty string"}`,
		wantResponseCode:    http.StatusBadRequest,
	},
	{
		name: "send event - binary - only specversion",
		giveMessage: func() (*cloudeventshttp.Message, error) {
			return &cloudeventshttp.Message{
				Header: map[string][]string{
					"ce-specversion": {"1.0"},
				},
				Body: nil,
			}, nil
		},
		wantResponseMessage: `{"error":"id: MUST be a non-empty string\nsource: REQUIRED\ntype: MUST be a non-empty string"}`,
		wantResponseCode:    http.StatusBadRequest,
	},
	// extra test is required because message will not receive serverHTTP of adapter
	// `JsonDecodeV1` will fail parsing timestamp
	{
		name: "send event - structured - invalid time",
		giveMessage: func() (*cloudeventshttp.Message, error) {

			body, err := json.Marshal(map[string]string{
				// required fields
				"specversion": "1.0",
				"type":        "type",
				"source":      "foo",
				// optional fields
				"time": "foo",
			})
			if err != nil {
				return nil, err
			}
			return &cloudeventshttp.Message{
				Header: map[string][]string{
					"content-type": {cloudevents.ApplicationCloudEventsJSON},
				},
				Body: body,
			}, nil
		},
		wantResponseMessage: `{"error":"cannot convert \"foo\" to time.Time: not in RFC3339 format"}`,
		wantResponseCode:    http.StatusBadRequest,
	},
	// extra test is required because message will not receive serverHTTP of adapter
	// `JsonDecodeV1` will fail parsing timestamp
	{
		name: "send event - binary - invalid time",
		giveMessage: func() (*cloudeventshttp.Message, error) {

			body, err := json.Marshal(map[string]string{
			})
			if err != nil {
				return nil, err
			}
			return &cloudeventshttp.Message{
				Header: map[string][]string{
					// required fields
					"ce-specversion": {"1.0"},
					"ce-type":        {"type"},
					"ce-source":      {"foo"},
					// optional fields
					"ce-time": {"foo"},
				},
				Body: body,
			}, nil
		},
		wantResponseMessage: `{"error":"cannot convert \"foo\" to time.Time: not in RFC3339 format"}`,
		wantResponseCode:    http.StatusBadRequest,
	},
	{
		name: "send event - structured - invalid datacontenttype",
		giveMessage: func() (*cloudeventshttp.Message, error) {

			body, err := json.Marshal(map[string]string{
				// required fields
				"specversion": "1.0",
				"type":        "type",
				"source":      "foo",
				// optional fields
				"id":              "foo",
				"datacontenttype": "foo",
				"dataschema":      "foo",
				// invalid time is tested in test case before
				"subject": "foo",
			})
			if err != nil {
				return nil, err
			}
			return &cloudeventshttp.Message{
				Header: map[string][]string{
					"content-type": {cloudevents.ApplicationCloudEventsJSON},
				},
				Body: body,
			}, nil
		},
		wantResponseMessage: " todo ",
		wantResponseCode:    http.StatusBadRequest,
	},
	{
		name: "send event - binary - invalid datacontenttype",
		giveMessage: func() (*cloudeventshttp.Message, error) {

			body, err := json.Marshal(map[string]string{
			})
			if err != nil {
				return nil, err
			}
			return &cloudeventshttp.Message{
				Header: map[string][]string{
					// datacontenttype as per https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#311-http-content-type
					"content-type": {"foo"},
					// required fields
					"ce-id":          {"1"},
					"ce-specversion": {"1.0"},
					"ce-type":        {"type"},
					"ce-source":      {"foo"},
				},
				Body: body,
			}, nil
		},
		wantResponseMessage: "",
		wantResponseCode:    http.StatusOK,
	},
}

var testsValidCloudEvents = []struct {
	name         string
	giveEvent    func() cloudevents.Event
	giveEncoding cloudevents.HTTPEncoding
	// the expected status code and error message then sending to the adapter with cloudevents sdk
	// if none given, assume sending adapter was ok
	wantAdapterError string
}{
	{
		name: "accepts CE v1.0 binary",
		giveEvent: func() cloudevents.Event {
			event := cloudevents.NewEvent(cloudevents.VersionV1)
			_ = event.Context.SetType("foo")
			_ = event.Context.SetID("foo")
			// event.Context.SetSource("will be replaced by adapter anyways, but we need a valid event here")
			_ = event.Context.SetSource("foo")
			return event
		},
		giveEncoding: cloudevents.HTTPBinaryV1,
	},
	{
		name: "accepts CE v1.0 structured",
		giveEvent: func() cloudevents.Event {
			event := cloudevents.NewEvent(cloudevents.VersionV1)
			_ = event.Context.SetType("foo")
			_ = event.Context.SetID("foo")
			// event.Context.SetSource("will be replaced by adapter anyways, but we need a valid event here")
			_ = event.Context.SetSource("foo")
			return event
		},
		giveEncoding: cloudevents.HTTPStructuredV1,
	},
	{
		name: "declines CE < v1.0 binary",
		giveEvent: func() cloudevents.Event {
			event := cloudevents.NewEvent(cloudevents.VersionV03)
			_ = event.Context.SetSpecVersion(cloudevents.VersionV03)
			_ = event.Context.SetType("foo")
			_ = event.Context.SetID("foo")
			// event.Context.SetSource("will be replaced by adapter anyways, but we need a valid event here")
			_ = event.Context.SetSource("foo")
			return event
		},
		wantAdapterError: "error sending cloudevent: 400 Bad Request",
		giveEncoding:     cloudevents.HTTPBinaryV03,
	},
	{
		name: "declines CE < v1.0 structured",
		giveEvent: func() cloudevents.Event {
			event := cloudevents.NewEvent(cloudevents.VersionV03)
			_ = event.Context.SetSpecVersion(cloudevents.VersionV03)
			_ = event.Context.SetType("foo")
			_ = event.Context.SetID("foo")
			// event.Context.SetSource("will be replaced by adapter anyways, but we need a valid event here")
			_ = event.Context.SetSource("foo")
			return event
		},
		wantAdapterError: "error sending cloudevent: 400 Bad Request",
		giveEncoding:     cloudevents.HTTPStructuredV03,
	},
}

type handler struct {
	requests []*http.Request
}

func (h *handler) startSink(t *testing.T) string {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log("received sink request")
		if _, err := fmt.Fprintln(w, "Hello, cloudevents client"); err != nil {
			t.Error(err)
		}
		h.requests = append(h.requests, r)
	}))
	sinkURI := ts.URL

	return sinkURI
}

type config struct {
	sinkURI       string
	namespace     string
	metricsConfig string
	loggingConfig string
	source        string
	port          int
}

func (c config) GetSinkURI() string {
	return c.sinkURI
}

func (c config) GetNamespace() string {
	return c.namespace
}

func (c config) GetMetricsConfigJson() string {
	return c.metricsConfig
}

func (c config) GetLoggingConfigJson() string {
	return c.loggingConfig
}

func (c config) GetSource() string {
	return c.source
}

func (c config) GetPort() int {
	return c.port
}

// TestAdapter tests the http-adapter by
// - spinning up the adapter
// - sending a CE event
// - receiving the CE event enriched by application source from adapter using a mocked server in the test
// - the sinkURI is set to the mocked http server
func TestAdapter(t *testing.T) {
	t.Parallel()

	for idx, tt := range testsValidCloudEvents {
		// https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721#how-to-solve-this
		tt := tt
		idx := idx

		t.Logf("running test %s", tt.name)
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// receive channel for http.Request from sink
			handler := handler{
				requests: []*http.Request{},
			}

			sinkURI := handler.startSink(t)
			t.Logf("sink URI: %q", sinkURI)

			adapterPort := startPort + idx
			adapterURI := fmt.Sprintf("http://localhost:%d", adapterPort)

			c := config{
				sinkURI:   sinkURI,
				namespace: "foo",
				source:    "guenther",
				port:      adapterPort,
			}

			// start http-adapter
			startHttpAdapter(t, c, context.Background())

			waitAdapterReady(t, adapterURI)
			eventResponse, err := sendEvent(t, adapterURI, tt.giveEvent(), tt.giveEncoding)
			// TODO(nachtmaar):
			fmt.Println(eventResponse)
			t.Log("waiting for sink response")

			if tt.wantAdapterError != "" {
				if err == nil || err.Error() != tt.wantAdapterError {
					t.Fatalf("Expected the cloudevents error to be: %q, but got: %q", tt.wantAdapterError, err)
				} else {
					// done with testing
					return
				}
			}

			// TODO: validate sink request: trace headers etc ...
			if len(handler.requests) != 1 {
				t.Fatalf("Exactly one sink request expected, got: %d", len(handler.requests))
			}
			sinkRequest := handler.requests[0]

			t.Log("ensure source set on event")
			ensureSourceSet(t, sinkRequest, c.GetSource())

			t.Logf("test %q done", tt.name)
		})
	}
}

// TestAdapterReceiveBrokenEvent sends a broken event to the adapter
// and checks the response code & response message
func TestAdapterReceiveBrokenEvent(t *testing.T) {
	t.Parallel()

	for idx, tt := range testsReceiveBrokenEvent {
		// https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721#how-to-solve-this
		idx := idx
		tt := tt

		t.Logf("running test %s", tt.name)
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// receive channel for http.Request from sink
			handler := handler{
				requests: []*http.Request{},
			}

			sinkURI := handler.startSink(t)
			t.Logf("sink URI: %q", sinkURI)

			adapterPort := startPort + idx
			adapterURI := fmt.Sprintf("http://localhost:%d", adapterPort)

			c := config{
				sinkURI:   sinkURI,
				namespace: "foo",

				source: "guenther",
				port:   adapterPort,
			}

			// start http-adapter
			startHttpAdapter(t, c, context.Background())

			waitAdapterReady(t, adapterURI)
			message, err := tt.giveMessage()
			if err != nil {
				t.Fatal(err)
			}
			req := sendEventHttp(t, adapterURI, *message)

			// check response code
			if tt.wantResponseCode != req.StatusCode {
				t.Errorf("Expected response code: %d, got: %d", tt.wantResponseCode, req.StatusCode)
			}

			// check response message
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				t.Fatal(err)
			}
			responseMessage := string(body)
			if len(tt.wantResponseMessage) > 0 {
				t.Logf("received: %q\n", responseMessage)
				if tt.wantResponseMessage != responseMessage {
					t.Errorf("Expected response messages: %v, got: %q\n", tt.wantResponseMessage, responseMessage)
				}
			} else {
				if len(body) != 0 {
					t.Errorf("Expected empty body, got: %s\n", responseMessage)
				}
			}
		})
	}
}

// use readiness probe to ensure adapter is ready
func waitAdapterReady(t *testing.T, adapterURI string) {
	t.Helper()
	if err := retry.Do(
		func() error {
			resp, err := http.Get(adapterURI + readinessReadiness)
			if err != nil {
				return err
			}
			expectedStatusCode := 200
			if resp.StatusCode != expectedStatusCode {
				return fmt.Errorf("adapter is not ready, expected status code: %d, got: %d", expectedStatusCode, resp.StatusCode)
			}
			return err
		},
	); err != nil {
		t.Fatalf("timeout waiting for adapter readiness: %v", err)
	}
}

// startHttpAdapter starts the adapter with a cloudevents client configured with the test sink as target
func startHttpAdapter(t *testing.T, c config, ctx context.Context) *adapter.Adapter {
	sinkClient, err := kncloudevents.NewDefaultClient(c.GetSinkURI())
	if err != nil {
		t.Fatal("error building cloud event client", zap.Error(err))
	}
	statsReporter, err := source.NewStatsReporter()
	if err != nil {
		t.Errorf("error building statsreporter: %v", err)
	}
	// TODO(nachtmaar): validate metrics reporter called
	httpAdapter := NewAdapter(ctx, c, sinkClient, statsReporter)
	go func() {
		if err := httpAdapter.Start(ctx.Done()); err != nil {
			t.Errorf("start returned an error: %v", err)
		}
	}()
	return &httpAdapter
}

// ensureSourceSet checks that the http adapter sets the event source on the event which is sent to the sink
func ensureSourceSet(t *testing.T, sinkReponse *http.Request, wantEventSource string) {
	t.Helper()
	giveEventSource := sinkReponse.Header.Get("CE-Source")
	if giveEventSource != wantEventSource {
		t.Errorf("Adapter is supposed to set the event source to: %q, got: %q", wantEventSource, giveEventSource)
	}
}

// sendEventHttp sends an eventing to the given `adapterURI based on `message`
func sendEventHttp(t *testing.T, adapterURI string, message cloudeventshttp.Message) *http.Response {
	req, err := http.NewRequest("POST", adapterURI, bytes.NewBuffer(message.Body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header = message.Header
	c := http.Client{}

	t.Logf("sending request: %+v\n", req)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// sendEvent sends a cloudevent to the adapter
// returns an error when not getting status code 2xx
func sendEvent(t *testing.T, adapterURI string, event cloudevents.Event, encoding cloudevents.HTTPEncoding) (*cloudevents.Event, error) {
	t.Helper()
	transport, err := cloudevents.NewHTTPTransport(
		cloudevents.WithTarget(adapterURI),
		cloudevents.WithEncoding(encoding),
	)
	if err != nil {
		return nil, err
	}
	client, err := cloudevents.NewClient(transport)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	t.Logf("sending event to http adapter, event: %s", event)
	// NOTE: then using CE sdk to send an event we get error message and status code in one message: "error sending cloudevent: 400 Bad Request"
	_, eventResponse, err := client.Send(ctx, event)
	if err != nil {
		return nil, err
	}
	return eventResponse, nil
}

// TestAdapterShutdown testsValidCloudEvents that the adapter is shutdown properly when receiving a stop signal
func TestAdapterShutdown(t *testing.T) {
	timeout := time.Second * 10

	c := config{}

	ctx := context.Background()
	// used to simulate sending a stop signal
	ctx, cancelFunc := context.WithCancel(ctx)

	httpAdapter := NewAdapter(ctx, c, nil, nil)
	stopChannel := make(chan error)

	// start adapter
	go func() {
		t.Log("starting http adapter in goroutine")
		err := httpAdapter.Start(ctx.Done())
		stopChannel <- err
		t.Log("http adapter goroutine ends here")
	}()

	t.Log("simulate stop signal")
	// call close on internal ctx.Done() channel
	cancelFunc()

	t.Log("waiting for adapter to stop")

	select {
	case err := <-stopChannel:
		if err != nil {
			t.Fatalf("Expected adapter shutdown to return no error, got: %v\n", err)
		}
	case <-time.Tick(timeout):
		t.Fatalf("Expected adapter to shutdown after timeout: %v\n", timeout)
	}

	t.Log("waiting for adapter to stop [done]")
}