// Package cmd implements the vsno command line.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cliVersion   = "dev"
	kubeconfig   string
	kubeContext  string
	namespace    string
	outputFormat string
)

var rootCmd = &cobra.Command{
	Use:   "vsno",
	Short: "Run Image Based Upgrade tests on virtual SNOs inside an existing OpenShift cluster",
	Long: `vsno creates virtual single-node OpenShift clusters (vSNOs) as OpenShift
Virtualization VMs on a bare-metal worker of a cluster you already have, and
drives a lifecycle-agent Image Based Upgrade (IBU) between them.

Start with:  vsno precheck`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		switch outputFormat {
		case "table", "json":
			return nil
		default:
			return fmt.Errorf("invalid --output %q (want table or json)", outputFormat)
		}
	},
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&kubeconfig, "kubeconfig", "", "path to the host cluster kubeconfig (default: $KUBECONFIG, then ~/.kube/config)")
	pf.StringVar(&kubeContext, "context", "", "kubeconfig context to use")
	pf.StringVarP(&namespace, "namespace", "n", "vsno", "namespace that holds the vSNO environment")
	pf.StringVarP(&outputFormat, "output", "o", "table", "output format: table or json")
}

// SetVersion is called from main with the build-time version.
func SetVersion(v string) {
	cliVersion = v
	rootCmd.Version = v
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
