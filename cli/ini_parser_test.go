package main

import (
	"os"
	"testing"

	"github.com/goadapp/goad"
	"github.com/naoina/toml"
	"github.com/stretchr/testify/assert"
)

var expectedRegions = []string{"eu-west-1", "us-east-1"}

func TestReadSettings(t *testing.T) {
	assert := assert.New(t)

	f, err := os.Open("testdata/test-config.toml")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var config goad.TestConfig
	if err := toml.NewDecoder(f).Decode(&config); err != nil {
		panic(err)
	}
	assert.Equal("http://file-config.com/", config.URL, "Should load the URL")
	assert.Equal("GET", config.Method, "Should load the request method")
	assert.Equal("Hello world", config.Body, "Should load the request body")
	assert.Equal(9, config.Timeout, "Should load the request timeout")
	assert.Equal(7, config.Concurrency, "Should load the concurrency setting")
	assert.Equal(107, config.Requests, "Should load the request count")
	assert.Equal(13, config.Timelimit, "Should load the execution timelimit")
	assert.Equal(expectedRegions, config.Regions, "Should load the regions")
	assert.Equal("my-aws-profile", config.AwsProfile, "Should load the AWS profile name")
	assert.Equal("test-result.json", config.Output, "Should load the output file")
	assert.Equal([]string{"cache-control: no-cache", "auth-token: YOUR-SECRET-AUTH-TOKEN"}, config.Headers, "Should load the output file")
}

func TestLoadStandardConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode to prevent fail in editor")
	}
	assert := assert.New(t)
	createTemporaryConfigFile()
	defer deleteTemporaryConfigFile()
	config := aggregateConfiguration()
	assert.Equal("http://file-config.com/", config.URL, "should have loaded from the default file in cwd")
	assert.Equal(expectedRegions, config.Regions, "should not have overwritten the regions with cli default")
}

func createTemporaryConfigFile() {
	os.Link("testdata/test-config.toml", "goad.ini")
}

func deleteTemporaryConfigFile() {
	os.Remove("goad.ini")
}
