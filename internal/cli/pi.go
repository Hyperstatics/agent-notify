package cli

import (
	"github.com/hellolib/agent-notify/internal/common"
	"github.com/spf13/cobra"
)

func newPiCmd(streams Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pi",
		Short: "Manage Pi extension integration",
	}
	cmd.AddCommand(newPiPrintHooksCmd(streams), newPiInstallHooksCmd())
	return cmd
}

func newPiPrintHooksCmd(streams Streams) *cobra.Command {
	var binaryPath string
	cmd := &cobra.Command{
		Use:   "print-hooks",
		Short: "Print the Pi extension source",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrintPiHooks(streams, firstNonEmpty(binaryPath))
		},
	}
	cmd.Flags().StringVar(&binaryPath, "binary", common.ResolveBinaryPath(""), "agent-notify binary path")
	return cmd
}

func newPiInstallHooksCmd() *cobra.Command {
	var binaryPath string
	var scope string
	cmd := &cobra.Command{
		Use:   "install-hooks",
		Short: "Install the Pi extension",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstallPiHooks(scope, firstNonEmpty(binaryPath))
		},
	}
	cmd.Flags().StringVar(&binaryPath, "binary", common.ResolveBinaryPath(""), "agent-notify binary path")
	cmd.Flags().StringVar(&scope, "scope", "user", "install scope")
	return cmd
}
