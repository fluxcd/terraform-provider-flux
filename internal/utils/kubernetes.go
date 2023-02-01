package utils

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	imageautov1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
	imagereflectv1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	notificationv1 "github.com/fluxcd/notification-controller/api/v1beta1"
	runclient "github.com/fluxcd/pkg/runtime/client"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/pkg/ssa"
)

func KubeConfig(rcg genericclioptions.RESTClientGetter, opts *runclient.Options) (*rest.Config, error) {
	cfg, err := rcg.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("kubernetes configuration load failed: %w", err)
	}
	// avoid throttling request when some Flux CRDs are not registered
	cfg.QPS = opts.QPS
	cfg.Burst = opts.Burst
	return cfg, nil
}

func ResourceManager(rcg genericclioptions.RESTClientGetter, opts *runclient.Options) (*ssa.ResourceManager, error) {
	cfg, err := KubeConfig(rcg, opts)
	if err != nil {
		return nil, err
	}
	restMapper, err := rcg.ToRESTMapper()
	if err != nil {
		return nil, err
	}
	kubeClient, err := client.New(cfg, client.Options{Mapper: restMapper, Scheme: NewScheme()})
	if err != nil {
		return nil, err
	}
	kubePoller := polling.NewStatusPoller(kubeClient, restMapper, polling.Options{})
	return ssa.NewResourceManager(kubeClient, kubePoller, ssa.Owner{
		Field: "flux",
		Group: "fluxcd.io",
	}), nil
}

func NewScheme() *apiruntime.Scheme {
	scheme := apiruntime.NewScheme()
	_ = apiextensionsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = networkingv1.AddToScheme(scheme)
	_ = sourcev1.AddToScheme(scheme)
	_ = kustomizev1.AddToScheme(scheme)
	_ = helmv2.AddToScheme(scheme)
	_ = notificationv1.AddToScheme(scheme)
	_ = imagereflectv1.AddToScheme(scheme)
	_ = imageautov1.AddToScheme(scheme)
	return scheme
}

func KubeClient(rcg genericclioptions.RESTClientGetter, opts *runclient.Options) (client.WithWatch, error) {
	cfg, err := rcg.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	cfg.QPS = opts.QPS
	cfg.Burst = opts.Burst
	kubeClient, err := client.NewWithWatch(cfg, client.Options{Scheme: NewScheme()})
	if err != nil {
		return nil, fmt.Errorf("kubernetes client initialization failed: %w", err)
	}
	return kubeClient, nil
}

type RESTClientGetter struct {
	clientconfig clientcmd.ClientConfig
}

func NewRestClientGetter(clientconfig clientcmd.ClientConfig) *RESTClientGetter {
	return &RESTClientGetter{clientconfig: clientconfig}
}

func (r *RESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	return r.clientconfig.ClientConfig()
}

func (r *RESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	restconfig, err := r.clientconfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	dc, err := discovery.NewDiscoveryClientForConfig(restconfig)
	if err != nil {
		return nil, err
	}
	return memory.NewMemCacheClient(dc), nil
}

func (r *RESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	dc, err := r.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(dc), nil
}

func (r *RESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return r.clientconfig
}
