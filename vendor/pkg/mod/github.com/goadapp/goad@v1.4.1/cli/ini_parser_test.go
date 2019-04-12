package cli

import (
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

var expectedRegions = []string{"us-east-1", "eu-west-1"}
var expectedHeader = []string{"cache-control: no-cache", "auth-token: YOUR-SECRET-AUTH-TOKEN", "base64-header: dGV4dG8gZGUgcHJ1ZWJhIA=="}

func TestLoadStandardConfig(t *testing.T) {
	assert := assert.New(t)
	delete := createTemporaryConfigFile()
	defer delete()
	config := parseSettingsFile("goad.ini")
	assert.Equal("http://file-config.com/", config.URL, "Should load the URL")
	assert.Equal("GET", config.Method, "Should load the request method")
	assert.Equal("Hello world", config.Body, "Should load the request body")
	assert.Equal(9, config.Timeout, "Should load the request timeout")
	assert.Equal(7, config.Concurrency, "Should load the concurrency setting")
	assert.Equal(107, config.Requests, "Should load the request count")
	assert.Equal(13, config.Timelimit, "Should load the execution timelimit")
	assert.Equal(expectedRegions, config.Regions, "Should load the regions")
	assert.Equal("test-result.json", config.Output, "Should load the output file")
	sort.Strings(expectedHeader)
	sort.Strings(config.Headers)
	assert.Equal(expectedHeader, config.Headers, "Should load the output file")
}

func createTemporaryConfigFile() func() {
	file := "goad.ini"
	os.Link("testdata/test-config.ini", file)
	return func() {
		os.Remove(file)
	}
}
