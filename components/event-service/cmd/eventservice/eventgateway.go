package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kyma-project/kyma/components/event-service/internal/events/subscribed"

	subscriptions "github.com/kyma-project/kyma/components/event-bus/generated/push/clientset/versioned"
	"github.com/kyma-project/kyma/components/event-service/internal/events/bus"
	"github.com/kyma-project/kyma/components/event-service/internal/externalapi"
	"github.com/kyma-project/kyma/components/event-service/internal/httptools"
	log "github.com/sirupsen/logrus"
	core "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const (
	shutdownTimeout = time.Minute
)

func main() {
	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)

	log.Info("Starting event-service")

	options := parseArgs()
	log.Infof("Options: %s", options)

	bus.Init(options.sourceID, options.eventsTargetURLV1, options.eventsTargetURLV2)

	subscriptionsClient, namespacesClient, e := initK8sResourcesClients()

	if e != nil {
		log.Error("Unable to create Events Client.", e.Error())
		return
	}

	eventsClient := subscribed.NewEventsClient(subscriptionsClient, namespacesClient)

	externalHandler := externalapi.NewHandler(options.maxRequestSize, eventsClient)

	if options.requestLogging {
		externalHandler = httptools.RequestLogger("External handler: ", externalHandler)
	}

	externalSrv := &http.Server{
		Addr:         ":" + strconv.Itoa(options.externalAPIPort),
		Handler:      externalHandler,
		ReadTimeout:  time.Duration(options.requestTimeout) * time.Second,
		WriteTimeout: time.Duration(options.requestTimeout) * time.Second,
	}

	// start the HTTP server
	go start(externalSrv)

	// shutdown the HTTP server gracefully
	shutdown(externalSrv, shutdownTimeout)
}

func start(server *http.Server) {
	if server == nil {
		log.Error("cannot start a nil HTTP server")
		return
	}

	if err := server.ListenAndServe(); err != nil {
		log.Error(err)
	}
}

func shutdown(server *http.Server, timeout time.Duration) {
	if server == nil {
		log.Info("cannot shutdown a nil HTTP server")
		return
	}

	shutdownSignal := make(chan os.Signal, 1)
	defer close(shutdownSignal)

	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-shutdownSignal

	log.Infof("HTTP server shutdown with timeout: %s\n", timeout)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("HTTP server shutdown error: %v\n", err)
	} else {
		log.Info("HTTP server shutdown successful")
	}
}

func initK8sResourcesClients() (subscribed.SubscriptionsGetter, subscribed.NamespacesClient, error) {
	k8sConfig, err := loadKubeConfig()
	if err != nil {
		return nil, nil, err
	}

	subscriptionsClient, err := subscriptions.NewForConfig(k8sConfig)
	if err != nil {
		return nil, nil, err
	}

	coreClient, err := core.NewForConfig(k8sConfig)
	if err != nil {
		return nil, nil, err
	}

	namespacesClient := coreClient.Namespaces()

	return subscriptionsClient.EventingV1alpha1(), namespacesClient, nil
}

func loadKubeConfig() (*rest.Config, error) {
	if _, err := os.Stat(clientcmd.RecommendedHomeFile); os.IsNotExist(err) {
		cfg, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		return nil, err
	}
	return kubeConfig, nil
}
