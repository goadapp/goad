package version

import "strings"

// Version describes the Goad version.
var (
	version string
	build   string
)

// Version returns the version
func Version() string {
	return version
}

// Build returns the build number
func Build() string {
	if len(build) >= 8 {
		return build[0:8]
	}
	return build
}

// String returns a composed string of version and build number
func String() string {
	return Version() + "-" + Build()
}

// LambdaVersion returns a version string that can be used as a Lambda function
// alias.
func LambdaVersion() string {
	return "v" + strings.Replace(String(), ".", "-", -1)
}
