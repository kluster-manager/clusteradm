// Copyright Contributors to the Open Cluster Management project
package clusterinfo

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	clustermeta "kmodules.xyz/client-go/cluster"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

func NewClient(cfg *rest.Config) (client.Client, error) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = clusterv1alpha1.AddToScheme(scheme)

	ctrl.SetLogger(klog.NewKlogr())

	hc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}
	mapper, err := apiutil.NewDynamicRESTMapper(cfg, hc)
	if err != nil {
		return nil, err
	}

	return client.New(cfg, client.Options{
		Scheme: scheme,
		Mapper: mapper,
	})
}

func (o *Options) complete(cmd *cobra.Command, args []string) error {
	cfg, err := o.KubectlFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	crdClient, err := NewClient(cfg)
	if err != nil {
		return err
	}

	o.client = crdClient
	return nil
}

func (o *Options) validate(args []string) (err error) {
	if len(args) != 0 {
		return fmt.Errorf("there should be no argument")
	}

	if o.clusterName == "" {
		return fmt.Errorf("cluster name is required")
	}

	return nil
}

func (o *Options) run() error {
	cid, err := clustermeta.ClusterUID(o.client)
	if err != nil {
		return err
	}
	clusterManagers := clustermeta.DetectClusterManager(o.client).Strings()

	md := map[string]any{
		"uid":             cid,
		"name":            o.clusterName,
		"clusterManagers": clusterManagers,
	}
	capiInfo, err := clustermeta.DetectCAPICluster(o.client)
	if err != nil {
		return err
	}
	if capiInfo != nil {
		md["capi"] = capiInfo
	}

	data, err := yaml.Marshal(map[string]any{
		"clusterMetadata": md,
	})
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(o.Streams.Out, "%s\n", string(data))

	if o.storeInConfigMap != "" {
		ns, name, err := cache.SplitMetaNamespaceKey(o.storeInConfigMap)
		if err != nil {
			return err
		}

		if o.createNamespace {
			if err = o.client.Create(context.Background(), &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			}); client.IgnoreAlreadyExists(err) != nil {
				return err
			}
		}

		obj := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
		}
		result, err := controllerutil.CreateOrPatch(context.Background(), o.client, obj, func() error {
			if obj.Data == nil {
				obj.Data = make(map[string]string)
			}
			obj.Data["clusterinfo.yaml"] = string(data)
			return nil
		})
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(o.Streams.Out, "%s ConfigMap %s\n", string(result), o.storeInConfigMap)
	}

	if o.storeInClusterClaim != "" {
		obj := &clusterv1alpha1.ClusterClaim{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name: o.storeInClusterClaim,
			},
		}
		result, err := controllerutil.CreateOrPatch(context.Background(), o.client, obj, func() error {
			obj.Spec.Value = string(data)
			return nil
		})
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(o.Streams.Out, "%s ClusterClaim %s\n", string(result), o.storeInClusterClaim)
	}
	return nil
}
