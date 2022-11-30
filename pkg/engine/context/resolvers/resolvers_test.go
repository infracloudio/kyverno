package resolvers

import (
	"context"
	"reflect"
	"testing"
	"time"

	"gotest.tools/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
	corev1listers "k8s.io/client-go/listers/core/v1"
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
	selector, err := GetCacheSelector()
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
	informerResolver, err := NewInformerBasedResolver(kubeResourceInformer.Core().V1().ConfigMaps().Lister())
	assert.NilError(t, err)
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
	informerResolver, err := NewInformerBasedResolver(kubeResourceInformer.Core().V1().ConfigMaps().Lister())
	assert.NilError(t, err)
	kubeResourceInformer.Start(make(<-chan struct{}))
	time.Sleep(10 * time.Second)
	_, err = informerResolver.Get(ctx, "default", "myconfigmap1")
	assert.Equal(t, err.Error(), "configmap \"myconfigmap1\" not found")
}

func TestNewInformerBasedResolver(t *testing.T) {
	type args struct {
		lister corev1listers.ConfigMapLister
	}
	client := newEmptyFakeClient()
	informer := initialiseInformer(client)
	lister := informer.Core().V1().ConfigMaps().Lister()
	tests := []struct {
		name    string
		args    args
		want    ConfigmapResolver
		wantErr bool
	}{{
		name:    "nil shoud return an error",
		wantErr: true,
	}, {
		name: "not nil",
		args: args{lister},
		want: &informerBasedResolver{lister},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewInformerBasedResolver(tt.args.lister)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewInformerBasedResolver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewInformerBasedResolver() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewClientBasedResolver(t *testing.T) {
	type args struct {
		client kubernetes.Interface
	}
	client := newEmptyFakeClient()
	tests := []struct {
		name    string
		args    args
		want    ConfigmapResolver
		wantErr bool
	}{{
		name:    "nil shoud return an error",
		wantErr: true,
	}, {
		name: "not nil",
		args: args{client},
		want: &clientBasedResolver{client},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewClientBasedResolver(tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClientBasedResolver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClientBasedResolver() = %v, want %v", got, tt.want)
			}
		})
	}
}

type dummyResolver struct{}

func (c dummyResolver) Get(context.Context, string, string) (*corev1.ConfigMap, error) {
	return nil, nil
}

func TestNewResolverChain(t *testing.T) {
	type args struct {
		resolvers []ConfigmapResolver
	}
	tests := []struct {
		name    string
		args    args
		want    ConfigmapResolver
		wantErr bool
	}{{
		name:    "nil shoud return an error",
		wantErr: true,
	}, {
		name:    "empty list shoud return an error",
		args:    args{[]ConfigmapResolver{}},
		wantErr: true,
	}, {
		name:    "one nil in the list shoud return an error",
		args:    args{[]ConfigmapResolver{dummyResolver{}, nil}},
		wantErr: true,
	}, {
		name: "no nil",
		args: args{[]ConfigmapResolver{dummyResolver{}, dummyResolver{}, dummyResolver{}}},
		want: resolverChain{dummyResolver{}, dummyResolver{}, dummyResolver{}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewResolverChain(tt.args.resolvers...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewResolverChain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewResolverChain() = %v, want %v", got, tt.want)
			}
		})
	}
}
