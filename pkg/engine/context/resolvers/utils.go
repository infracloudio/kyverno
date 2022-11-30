package resolvers

import (
	"errors"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
)

func selectorForCache() (labels.Selector, error) {
	expectedLabels := "cache.kyverno.io/enabled"
	selector := labels.Everything()
	requirement, err := labels.NewRequirement(expectedLabels, selection.Exists, nil)
	if err != nil {
		return nil, err
	}
	selector = selector.Add(*requirement)
	return selector, err
}

func ResourceInformer(kubeClient kubernetes.Interface, resyncPeriod time.Duration) (kubeinformers.SharedInformerFactory, error) {
	if kubeClient == nil {
		return nil, errors.New("kubeClient cannot be nil")
	}
	selector, err := selectorForCache()
	if err != nil {
		return nil, err
	}
	labelOptions := kubeinformers.WithTweakListOptions(func(opts *metav1.ListOptions) {
		opts.LabelSelector = selector.String()
	})
	kubeResourceInformer := kubeinformers.NewSharedInformerFactoryWithOptions(kubeClient, resyncPeriod, labelOptions)
	return kubeResourceInformer, nil
}

func ConfigMapLister(kubeResourceInformer kubeinformers.SharedInformerFactory) corev1listers.ConfigMapLister {
	if kubeResourceInformer == nil {
		return nil
	}
	return kubeResourceInformer.Core().V1().ConfigMaps().Lister()
}

func ResolverChain(client kubernetes.Interface, lister corev1listers.ConfigMapLister) (ConfigmapResolver, error) {
	var resolvers []ConfigmapResolver
	if lister != nil {
		resolver, err := NewInformerBasedResolver(lister)
		if err != nil {
			resolvers = append(resolvers, resolver)
		}
	}
	if client != nil {
		resolver, err := NewClientBasedResolver(client)
		if err != nil {
			resolvers = append(resolvers, resolver)
		}
	}
	return NewResolverChain(resolvers...)
}
