package cmd

import (
	flag "github.com/spf13/pflag"
	"os"
	"path/filepath"
)

// ParseFlags parses command line flags and returns the path to the config
// file along with version bool (used to print version)
func ParseFlags() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		os.Exit(1)
	}
	defaultConfpath := filepath.Join(home, "opensearch_config.yaml")

	confUsage := "provide path to config file, omit for default"
	versionUsage := "use -v or --version to display build version"

	configPath := flag.StringP("config", "c", defaultConfpath, confUsage)
	versionFlag := flag.BoolP("version", "v", false, versionUsage)
	flag.Parse()

	return *configPath, *versionFlag
}
