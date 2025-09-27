package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/mcpjungle/mcpjungle/internal/version"
)

// getServerVersion attempts to fetch the server version from the configured server.
// Returns the version string and a boolean indicating success.
func getServerVersion() (string, bool) {
	// Try to get server metadata with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	metadata, err := apiClient.GetServerMetadata(ctx)
	if err != nil {
		return "", false
	}

	return metadata.Version, true
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		// We want the extra newline for proper formatting
		cmd.Print(asciiArt) //nolint:staticcheck
		
		// Display CLI version
		cliVersion := version.GetVersion()
		cmd.Printf("CLI Version: %s\n", cliVersion)
		
		// Try to fetch server version
		serverVersion, ok := getServerVersion()
		if ok {
			cmd.Printf("Server Version: %s (server url: %s)\n", serverVersion, apiClient.BaseURL())
		} else {
			cmd.Printf("Server Version: Couldn't retrieve Server version at this time. (server url: %s)\n", apiClient.BaseURL())
		}
	},
	Annotations: map[string]string{
		"group": string(subCommandGroupBasic),
		"order": "7",
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.Flags().BoolP("version", "v", false, "Display version information")
}
