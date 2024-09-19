// Copyright Contributors to the Open Cluster Management project
package clusterinfo

import (
	"k8s.io/cli-runtime/pkg/genericiooptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	//KubectlFlags: The generic options from the kubectl cli-runtime.
	KubectlFlags cmdutil.Factory

	Streams genericiooptions.IOStreams

	clusterName         string
	storeInConfigMap    string
	createNamespace     bool
	storeInClusterClaim string

	client client.Client
}

func newOptions(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *Options {
	return &Options{
		KubectlFlags: clusteradmFlags.KubectlFactory,
		Streams:      streams,
	}
}
