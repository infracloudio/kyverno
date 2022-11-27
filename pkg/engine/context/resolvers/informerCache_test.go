package resolvers

import (
	"context"
	"testing"
	"time"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func newEmptyFakeClient() *kubefake.Clientset {
	return kubefake.NewSimpleClientset()
}

func createConfigMaps(ctx context.Context, kubeClient *kubefake.Clientset, configMap *corev1.ConfigMap) error {
	_, err := kubeClient.CoreV1().ConfigMaps("default").Create(
		ctx, configMap, metav1.CreateOptions{})
	return err
}

func initialiseInformer(kubeClient *kubefake.Clientset) kubeinformers.SharedInformerFactory {
	selector, err := selectorForCache()
	if err != nil {
		return nil
	}
	labelOptions := kubeinformers.WithTweakListOptions(func(opts *metav1.ListOptions) {
		opts.LabelSelector = selector.String()
	})
	kubeResourceInformer := kubeinformers.NewSharedInformerFactoryWithOptions(kubeClient, 15*time.Minute, labelOptions)
	return kubeResourceInformer
}

func Test_InformerCacheSuccess(t *testing.T) {
	kubeClient := newEmptyFakeClient()
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myconfigmap",
			Namespace: "default",
			Labels:    map[string]string{"cache.kyverno.io/enabled": "true"},
		},
		Data: map[string]string{"configmapkey": "key1"},
	}
	ctx := context.TODO()
	err := createConfigMaps(ctx, kubeClient, cm)
	assert.NilError(t, err, "error while creating configmap")
	kubeResourceInformer := initialiseInformer(kubeClient)
	informerResolver := NewInformerBasedResolver(kubeResourceInformer.Core().V1().ConfigMaps().Lister())
	kubeResourceInformer.Start(make(<-chan struct{}))
	time.Sleep(10 * time.Second)
	_, err = informerResolver.Get(ctx, "default", "myconfigmap")
	assert.NilError(t, err, "informer didn't have expected configmap")
}

func Test_InformerCacheFailure(t *testing.T) {
	kubeClient := newEmptyFakeClient()
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myconfigmap1",
			Namespace: "default",
		},
		Data: map[string]string{"configmapkey": "key1"},
	}
	ctx := context.TODO()
	err := createConfigMaps(ctx, kubeClient, cm)
	assert.NilError(t, err, "error while creating configmap")
	kubeResourceInformer := initialiseInformer(kubeClient)
	informerResolver := NewInformerBasedResolver(kubeResourceInformer.Core().V1().ConfigMaps().Lister())
	kubeResourceInformer.Start(make(<-chan struct{}))
	time.Sleep(10 * time.Second)
	_, err = informerResolver.Get(ctx, "default", "myconfigmap1")
	assert.Equal(t, err.Error(), "configmap \"myconfigmap1\" not found")
}
