package cmd

import "fmt"

var (
	version      string
	gitCommit    string
	gitTreeState string
	buildDate    string
	buildTime    string
)

// PrintVersion prints version details which are injected via ldflags
// during build time
func PrintVersion() {
	fmt.Println("Build Version\t", version)
	fmt.Println("Git Commit\t", gitCommit)
	fmt.Println("Git State\t", gitTreeState)
	fmt.Println("Build Date\t", buildDate)
	fmt.Println("Build Time\t", buildTime)
}
