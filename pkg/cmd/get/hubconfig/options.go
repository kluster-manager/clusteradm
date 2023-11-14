// Copyright Contributors to the Open Cluster Management project
package hubconfig

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

// Options: The structure holding all the command-line options
type Options struct {
	//ClusteradmFlags: The generic options from the clusteradm cli-runtime.
	ClusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags

	//Values below are input from flags
	//The token generated on the hub to access it from the cluster
	token string
	//The external hub apiserver url (https://<host>:<port>)
	hubAPIServer string
	//The hub ca-file(optional)
	caFile string
	//The base64 encoded hub ca-data(optional)
	caDataEnc string

	//The file to output the resources will be sent to the file.
	outputFile string
	// By default, The installing registration agent will be starting registration using
	// the external endpoint from --hub-apiserver instead of looking for the internal
	// endpoint from the public cluster-info.
	forceHubInClusterEndpointLookup bool
	hubInClusterEndpoint            string

	//Values below are tempoary data
	//HubCADate: data in hub ca file
	HubCADate []byte
	// hub config
	HubConfig *clientcmdapiv1.Config

	//AgentNamespace: the namespace to deploy the agent
	agentNamespace string

	//Values below are used to fill in yaml files
	values Values

	Streams genericclioptions.IOStreams
}

// Values: The values used in the template
type Values struct {
	//Hub: Hub information
	Hub Hub
}

// Hub: The hub values for the template
type Hub struct {
	//APIServer: The API Server external URL
	APIServer string
	//KubeConfig: The kubeconfig of the bootstrap secret to connect to the hub
	KubeConfig string
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ClusteradmFlags: clusteradmFlags,
		Streams:         streams,
	}
}
