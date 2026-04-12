package buildinfo

import "runtime/debug"

var (
	CLIName               = "nexus"
	DaemonName            = "nexus-daemon"
	CLIVersion            = "0.0.0-dev"
	DaemonVersion         = "0.0.0-dev"
	Commit                = ""
	BuiltAt               = ""
	UpdatePublicKeyBase64 = ""
)

type Info struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	BuiltAt string `json:"builtAt"`
}

func CLI() Info {
	version := CLIVersion
	if version == "" || version == "0.0.0-dev" {
		if v := versionFromBuildSettings(); v != "" {
			version = v
		}
	}
	return Info{Name: CLIName, Version: version, Commit: Commit, BuiltAt: BuiltAt}
}

func Daemon() Info {
	version := DaemonVersion
	if version == "" || version == "0.0.0-dev" {
		if v := versionFromBuildSettings(); v != "" {
			version = v
		}
	}
	return Info{Name: DaemonName, Version: version, Commit: Commit, BuiltAt: BuiltAt}
}

func versionFromBuildSettings() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil {
		return ""
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return ""
}
