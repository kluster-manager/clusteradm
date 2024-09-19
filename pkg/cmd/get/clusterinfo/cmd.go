// Copyright Contributors to the Open Cluster Management project
package clusterinfo

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	clusteradmhelpers "open-cluster-management.io/clusteradm/pkg/helpers"
)

var example = `
# Get cluster-info.
%[1]s get cluster-info
`

// NewCmd...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "cluster-info",
		Short:        "get cluster-info",
		Example:      fmt.Sprintf(example, clusteradmhelpers.GetExampleHeader()),
		SilenceUsage: true,
		PreRunE: func(c *cobra.Command, args []string) error {
			clusteradmhelpers.DryRunMessage(clusteradmFlags.DryRun)
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.complete(c, args); err != nil {
				return err
			}
			if err := o.validate(args); err != nil {
				return err
			}
			if err := o.run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&o.clusterName, "cluster-name", "", "Name of cluster")
	cmd.Flags().StringVar(&o.storeInConfigMap, "store-in-configmap", "", "Stores cluster info into a configmap <ns>/<name> under clusterinfo.yaml key")
	cmd.Flags().BoolVar(&o.createNamespace, "create-namespace", o.createNamespace, "If true, create namespace for configmap")
	cmd.Flags().StringVar(&o.storeInClusterClaim, "store-in-clusterclaim", "", "Stores cluster info into a ClusterClaim <name>")

	return cmd
}
