package resolvers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
)

type NamespacedResourceResolver[T any] interface {
	Get(context.Context, string, string) (T, error)
}

type ConfigmapResolver = NamespacedResourceResolver[*corev1.ConfigMap]

type informerBasedResolver struct {
	lister corev1listers.ConfigMapLister
}

func NewInformerBasedResolver(lister corev1listers.ConfigMapLister) ConfigmapResolver {
	return &informerBasedResolver{
		lister: lister,
	}
}

func (i *informerBasedResolver) Get(ctx context.Context, namespace, name string) (*corev1.ConfigMap, error) {
	return i.lister.ConfigMaps(namespace).Get(name)
}

type clientBasedResolver struct {
	kubeClient kubernetes.Interface
}

func NewClientBasedResolver(kubeClient kubernetes.Interface) ConfigmapResolver {
	return &clientBasedResolver{
		kubeClient: kubeClient,
	}
}

func (c *clientBasedResolver) Get(ctx context.Context, namespace, name string) (*corev1.ConfigMap, error) {
	return c.kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
}

type resolverChain []ConfigmapResolver

func NewResolverChain(resolver ...ConfigmapResolver) ConfigmapResolver {
	return resolverChain(resolver)
}

func (r resolverChain) Get(ctx context.Context, namespace, name string) (*corev1.ConfigMap, error) {
	// if CM is not found in informer cache, error will be stored in
	// lastErr variable and resolver chain will try to get CM using
	// Kubernetes client
	var lastErr error
	for _, resolver := range r {
		cm, err := resolver.Get(ctx, namespace, name)
		if err == nil {
			return cm, nil
		}
		lastErr = err
	}
	return nil, lastErr
}
