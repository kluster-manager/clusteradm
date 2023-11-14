// Copyright Contributors to the Open Cluster Management project
package hubconfig

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

var example = `
# Prints hub config
%[1]s get hubconfig --hub-token <tokenID.tokenSecret> --hub-apiserver <hub_apiserver_url>
# Prints hub config while the hub provided no valid CA data in kube-public namespace
%[1]s get hubconfig --hub-token <tokenID.tokenSecret> --hub-apiserver <hub_apiserver_url> --ca-file <ca-file>
`

// NewCmd ...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "hubconfig",
		Short:        "prints hub config to be used by a spoke cluster",
		Long:         "prints hub config to be used by a spoke cluster",
		Example:      fmt.Sprintf(example, helpers.GetExampleHeader()),
		SilenceUsage: true,
		PreRun: func(c *cobra.Command, args []string) {
			helpers.DryRunMessage(o.ClusteradmFlags.DryRun)
		},
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.complete(c, args); err != nil {
				return err
			}
			if err := o.validate(); err != nil {
				return err
			}
			if err := o.run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&o.token, "hub-token", "", "The token to access the hub")
	cmd.Flags().StringVar(&o.hubAPIServer, "hub-apiserver", "", "The api server url to the hub")
	cmd.Flags().StringVar(&o.caFile, "ca-file", "", "the file path to hub ca, optional")
	cmd.Flags().StringVar(&o.caDataEnc, "ca-data-enc", "", "the base64 encoded hub ca data, optional")
	cmd.Flags().StringVar(&o.outputFile, "output-file", "", "The generated resources will be copied in the specified file")
	cmd.Flags().BoolVar(&o.forceHubInClusterEndpointLookup, "force-internal-endpoint-lookup", false,
		"If true, the installed klusterlet agent will be starting the cluster registration process by "+
			"looking for the internal endpoint from the public cluster-info in the hub cluster instead of from --hub-apiserver.")
	cmd.Flags().StringVar(&o.agentNamespace, "agent-namespace", "", "The agent namespace")

	return cmd
}
