package cli

import (
	"github.com/hellolib/agent-notify/internal/common"
	"github.com/spf13/cobra"
)

func newKimiCmd(streams Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kimi",
		Short: "Manage Kimi Code hook integration",
	}
	cmd.AddCommand(newKimiPrintHooksCmd(streams), newKimiInstallHooksCmd())
	return cmd
}

func newKimiPrintHooksCmd(streams Streams) *cobra.Command {
	var binaryPath string

	cmd := &cobra.Command{
		Use:   "print-hooks",
		Short: "Print Kimi hook settings TOML",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrintKimiHooks(streams, firstNonEmpty(binaryPath))
		},
	}
	cmd.Flags().StringVar(&binaryPath, "binary", common.ResolveBinaryPath(""), "agent-notify binary path")
	return cmd
}

func newKimiInstallHooksCmd() *cobra.Command {
	var binaryPath string
	var scope string

	cmd := &cobra.Command{
		Use:   "install-hooks",
		Short: "Install Kimi hook settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstallKimiHooks(scope, firstNonEmpty(binaryPath))
		},
	}
	cmd.Flags().StringVar(&binaryPath, "binary", common.ResolveBinaryPath(""), "agent-notify binary path")
	cmd.Flags().StringVar(&scope, "scope", "user", "install scope")
	return cmd
}
