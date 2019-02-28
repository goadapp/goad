package types

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

const (
	MAX_REQUEST_COUNT = math.MaxInt32
	nano              = 1000000000
)

var supportedRegions = []string{
	"us-east-1",      // N. Virginia
	"us-east-2",      // Ohio
	"us-west-1",      // N.California
	"us-west-2",      // Oregon
	"eu-west-1",      // Ireland
	"eu-central-1",   // Frankfurt
	"ap-northeast-1", // Tokyo
	"ap-northeast-2", // Seoul
	"ap-southeast-1", // Singapore
	"ap-southeast-2", // Sydney
	"sa-east-1",      // Sao Paulo
}

// TestConfig type
type TestConfig struct {
	URL         string
	Concurrency int
	Requests    int
	Timelimit   int
	Timeout     int
	Regions     []string
	Method      string
	Body        string
	Headers     []string
	Output      string
	Settings    string
	RunDocker   bool
	Lambdas     int
	RunnerPath  string
}

func (c *TestConfig) Check() error {
	concurrencyLimit := 25000 * len(c.Regions)
	if c.Concurrency < 1 || c.Concurrency > concurrencyLimit {
		return fmt.Errorf("Invalid concurrency (use 1 - %d)", concurrencyLimit)
	}
	if (c.Requests < 1 && c.Timelimit <= 0) || c.Requests > MAX_REQUEST_COUNT {
		return errors.New(fmt.Sprintf("Invalid total requests (use 1 - %d)", MAX_REQUEST_COUNT))
	}
	if c.Timelimit > 3600 {
		return errors.New("Invalid maximum execution time in seconds (use 0 - 3600)")
	}
	if c.Timeout < 1 || c.Timeout > 100 {
		return errors.New("Invalid timeout (1s - 100s)")
	}
	for _, region := range c.Regions {
		supportedRegionFound := false
		for _, supported := range supportedRegions {
			if region == supported {
				supportedRegionFound = true
			}
		}
		if !supportedRegionFound {
			return fmt.Errorf("Unsupported region: %s. Supported regions are: %s.", region, strings.Join(supportedRegions, ", "))
		}
	}
	for _, v := range c.Headers {
		header := strings.Split(v, ":")
		if len(header) < 2 {
			return fmt.Errorf("Header %s not valid. Make sure your header is of the form \"Header: value\"", v)
		}
	}
	return nil
}
