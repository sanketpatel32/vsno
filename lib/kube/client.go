// Package kube builds Kubernetes clients for the host cluster.
package kube

import (
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Clients bundles the clients vsno uses against the host cluster.
type Clients struct {
	Config *rest.Config
	Kube   kubernetes.Interface
	Dyn    dynamic.Interface
	// Context is the kubeconfig context in use, for display.
	Context string
}

// New loads a kubeconfig the same way oc/kubectl do: explicit path, then
// $KUBECONFIG, then ~/.kube/config.
func New(kubeconfig, context string) (*Clients, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		rules.ExplicitPath = kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: context}
	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)

	cfg, err := cc.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("loading kubeconfig: %w", err)
	}
	raw, err := cc.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("reading kubeconfig: %w", err)
	}
	ctxName := raw.CurrentContext
	if context != "" {
		ctxName = context
	}

	k, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes client: %w", err)
	}
	d, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}
	return &Clients{Config: cfg, Kube: k, Dyn: d, Context: ctxName}, nil
}
