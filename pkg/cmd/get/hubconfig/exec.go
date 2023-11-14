// Copyright Contributors to the Open Cluster Management project
package hubconfig

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/klog/v2"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	sdkhelpers "open-cluster-management.io/sdk-go/pkg/helpers"
)

const bootstrapHubKubeconfig = "bootstrap-hub-kubeconfig"

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	if cmd.Flags() == nil {
		return fmt.Errorf("no flags have been set: hub-apiserver, hub-token and cluster-name is required")
	}

	if o.token == "" {
		return fmt.Errorf("token is missing")
	}
	if o.hubAPIServer == "" {
		return fmt.Errorf("hub-server is missing")
	}
	if o.agentNamespace == "" {
		return fmt.Errorf("agent-namespace is missing")
	}

	klog.V(1).InfoS("join options:", "dry-run", o.ClusteradmFlags.DryRun, "api-server", o.hubAPIServer, "output", o.outputFile)

	o.values = Values{
		Hub: Hub{
			APIServer: o.hubAPIServer,
		},
	}

	// if --ca-file is set, read ca data
	if o.caFile != "" {
		cabytes, err := os.ReadFile(o.caFile)
		if err != nil {
			return err
		}
		o.HubCADate = cabytes
	} else if o.caDataEnc != "" {
		cabytes, err := base64.StdEncoding.DecodeString(o.caDataEnc)
		if err != nil {
			return err
		}
		o.HubCADate = cabytes
	}

	// code logic of building hub client in join process:
	// 1. use the token and insecure to fetch the ca data from cm in kube-public ns
	// 2. if not found, assume using an authorized ca.
	// 3. use the ca and token to build a secured client and call hub

	//Create an unsecure bootstrap
	bootstrapExternalConfigUnSecure := o.createExternalBootstrapConfig()
	//create external client from the bootstrap
	externalClientUnSecure, err := helpers.CreateClientFromClientcmdapiv1Config(bootstrapExternalConfigUnSecure)
	if err != nil {
		return err
	}
	//Create the kubeconfig for the internal client
	o.HubConfig, err = o.createClientcmdapiv1Config(externalClientUnSecure, bootstrapExternalConfigUnSecure)
	if err != nil {
		return err
	}

	return nil
}

func (o *Options) validate() error {
	err := o.setKubeconfig()
	if err != nil {
		return err
	}
	return nil
}

func (o *Options) run() error {
	// get managed cluster externalServerURL
	kubeClient, err := o.ClusteradmFlags.KubectlFactory.KubernetesClientSet()
	if err != nil {
		klog.Errorf("Failed building kube client: %v", err)
		return err
	}

	_, err = kubeClient.CoreV1().Namespaces().Get(context.TODO(), o.agentNamespace, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = kubeClient.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: o.agentNamespace,
				Annotations: map[string]string{
					"workload.openshift.io/allowed": "management",
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	_, err = kubeClient.CoreV1().Secrets(o.agentNamespace).Get(context.TODO(), bootstrapHubKubeconfig, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = kubeClient.CoreV1().Secrets(o.agentNamespace).Create(context.TODO(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      bootstrapHubKubeconfig,
				Namespace: o.agentNamespace,
			},
			StringData: map[string]string{
				"kubeconfig": o.values.Hub.KubeConfig,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if len(o.outputFile) > 0 {
		sh, err := os.OpenFile(o.outputFile, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(sh, "%s", o.values.Hub.KubeConfig)
		if err != nil {
			return err
		}
		if err := sh.Close(); err != nil {
			return err
		}
	}
	return err
}

// Create bootstrap with token but without CA
func (o *Options) createExternalBootstrapConfig() clientcmdapiv1.Config {
	return clientcmdapiv1.Config{
		// Define a cluster stanza based on the bootstrap kubeconfig.
		Clusters: []clientcmdapiv1.NamedCluster{
			{
				Name: "hub",
				Cluster: clientcmdapiv1.Cluster{
					Server:                o.hubAPIServer,
					InsecureSkipTLSVerify: true,
				},
			},
		},
		// Define auth based on the obtained client cert.
		AuthInfos: []clientcmdapiv1.NamedAuthInfo{
			{
				Name: "bootstrap",
				AuthInfo: clientcmdapiv1.AuthInfo{
					Token: o.token,
				},
			},
		},
		// Define a context that connects the auth info and cluster, and set it as the default
		Contexts: []clientcmdapiv1.NamedContext{
			{
				Name: "bootstrap",
				Context: clientcmdapiv1.Context{
					Cluster:   "hub",
					AuthInfo:  "bootstrap",
					Namespace: "default",
				},
			},
		},
		CurrentContext: "bootstrap",
	}
}

func (o *Options) createClientcmdapiv1Config(externalClientUnSecure *kubernetes.Clientset,
	bootstrapExternalConfigUnSecure clientcmdapiv1.Config) (*clientcmdapiv1.Config, error) {
	var err error
	// set hub in cluster endpoint
	if o.forceHubInClusterEndpointLookup {
		o.hubInClusterEndpoint, err = sdkhelpers.GetAPIServer(externalClientUnSecure)
		if err != nil {
			if !errors.IsNotFound(err) {
				return nil, err
			}
		}
	}

	bootstrapConfig := bootstrapExternalConfigUnSecure.DeepCopy()
	bootstrapConfig.Clusters[0].Cluster.InsecureSkipTLSVerify = false
	bootstrapConfig.Clusters[0].Cluster.Server = o.hubAPIServer
	if o.HubCADate != nil {
		// directly set ca-data if --ca-file is set
		bootstrapConfig.Clusters[0].Cluster.CertificateAuthorityData = o.HubCADate
	} else {
		// get ca data from externalClientUnsecure, ca may empty(cluster-info exists with no ca data)
		ca, err := sdkhelpers.GetCACert(externalClientUnSecure)
		if err != nil {
			return nil, err
		}
		bootstrapConfig.Clusters[0].Cluster.CertificateAuthorityData = ca
	}

	return bootstrapConfig, nil
}

func (o *Options) setKubeconfig() error {
	// replace apiserver if the flag is set, the apiserver value should not be set
	// to in-cluster endpoint until preflight check is finished
	if o.forceHubInClusterEndpointLookup {
		o.HubConfig.Clusters[0].Cluster.Server = o.hubInClusterEndpoint
	}

	bootstrapConfigBytes, err := yaml.Marshal(o.HubConfig)
	if err != nil {
		return err
	}

	o.values.Hub.KubeConfig = string(bootstrapConfigBytes)
	return nil
}
