// Code generated by injection-gen. DO NOT EDIT.

package client

import (
	"context"

	versioned "istio.io/client-go/pkg/clientset/versioned"
	rest "k8s.io/client-go/rest"
	injection "knative.dev/pkg/injection"
	logging "knative.dev/pkg/logging"
)

func init() {
	injection.Default.RegisterClient(withClient)
}

// Key is used as the key for associating information with a context.Context.
type Key struct{}

func withClient(ctx context.Context, cfg *rest.Config) context.Context {
	return context.WithValue(ctx, Key{}, versioned.NewForConfigOrDie(cfg))
}

// Get extracts the versioned.Interface client from the context.
func Get(ctx context.Context) versioned.Interface {
	untyped := ctx.Value(Key{})
	if untyped == nil {
		logging.FromContext(ctx).Panic(
			"Unable to fetch istio.io/client-go/pkg/clientset/versioned.Interface from context.")
	}
	return untyped.(versioned.Interface)
}
