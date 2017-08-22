package cli

import (
	"bytes"
	"io"
	"io/ioutil"
	"sort"
	"testing"

	"github.com/goadapp/goad/goad/types"
	"github.com/stretchr/testify/assert"
)

const testDataFile = "testdata/test-config.ini"

var expectedRegions = []string{"us-east-1", "eu-west-1"}
var expectedHeader = []string{"cache-control: no-cache", "auth-token: YOUR-SECRET-AUTH-TOKEN", "base64-header: dGV4dG8gZGUgcHJ1ZWJhIA=="}

func TestLoadStandardConfig(t *testing.T) {
	iniFile = testDataFile
	config := parseSettings()
	applyExtendedConfiguration(config)
	assertConfigContent(config, t)
}

func assertConfigContent(config *types.TestConfig, t *testing.T) {
	assert := assert.New(t)
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
	assert.Equal("default-runner", config.RunnerPath, "Should load runner path configuration")
}

func TestSaveConfig(t *testing.T) {
	writer := bytes.NewBuffer(make([]byte, 0))
	writeConfigStream(writer)
	assertMatchTemplate(t, writer)
}

func assertMatchTemplate(t *testing.T, reader io.Reader) {
	assert := assert.New(t)
	expected := Must(ioutil.ReadFile("testdata/default.ini"))
	actual := Must(ioutil.ReadAll(reader))
	assert.Equal(len(expected), len(actual), "should be the same amount of bytes")
	assert.Equal(expected, actual, "Should exactly be that template")
}

func Must(value []byte, err error) []byte {
	if err != nil {
		panic(1)
	}
	return value
}
