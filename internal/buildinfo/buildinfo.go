package buildinfo

var (
	Version        = "dev"
	GitHubClientID = ""
	BuildTime      = ""
	GitCommit      = ""
)

func IsProduction() bool {
	return Version != "dev" && GitHubClientID != ""
}
