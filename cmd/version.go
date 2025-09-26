package cmd

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/spf13/cobra"
	"github.com/mcpjungle/mcpjungle/client"
	"github.com/mcpjungle/mcpjungle/cmd/config"
)

const defaultVersion = "dev"

// Version can be overridden at build time using:
// go build -ldflags="-X 'github.com/mcpjungle/mcpjungle/cmd.Version=v1.2.3'"
var Version = defaultVersion

// getVersion returns the CLI version string.
func getVersion() string {
	if Version != "" && Version != defaultVersion {
		return normalizeVersion(Version)
	}

	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return normalizeVersion(info.Main.Version)
	}

	return defaultVersion
}

// normalizeVersion ensures a consistent version format:
// - If version starts with a digit (e.g., "1.2.3"), prefix with 'v' → "v1.2.3"
// - Leave values starting with 'v' or non-semver strings untouched
func normalizeVersion(v string) string {
	if v == "" {
		return v
	}
	if v[0] >= '0' && v[0] <= '9' {
		return "v" + v
	}
	return v
}

// getServerVersion attempts to fetch the server version from the configured server.
// Returns empty string if unable to fetch.
func getServerVersion() string {
	// Load client configuration
	clientConfig := config.Load()
	if clientConfig.RegistryURL == "" {
		return ""
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Create MCPJungle client
	mcpClient := client.NewClient(clientConfig.RegistryURL, clientConfig.AccessToken, httpClient)

	// Try to get server metadata
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	metadata, err := mcpClient.GetServerMetadata(ctx)
	if err != nil {
		return ""
	}

	return metadata.Version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		// We want the extra newline for proper formatting
		fmt.Print(asciiArt) //nolint:staticcheck
		
		// Display CLI version
		cliVersion := getVersion()
		fmt.Printf("CLI Version: %s\n", cliVersion)
		
		// Try to fetch server version
		serverVersion := getServerVersion()
		if serverVersion != "" {
			fmt.Printf("Server Version: %s\n", serverVersion)
		} else {
			fmt.Printf("Server Version: Couldn't retrieve server version\n")
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
