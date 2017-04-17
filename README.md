# Goad

<https://goad.io>

Goad is an AWS Lambda powered, highly distributed,
load testing tool built in Go for the 2016 [Gopher Gala][].

![Go + Load ⇒ Goad](https://goad.io/assets/go-plus-load.png)

Goad allows you to load test your websites from all over the world whilst costing you the tiniest fractions of a penny by using AWS Lambda in multiple regions simultaneously.

You can run Goad from your machine using your own AWS credentials. Goad will automatically create the AWS resources you need and execute your test, and display the results broken down by region. This way, you can see how fast your website is from the major regions of the world.

If you just want to try Goad out, visit the [Goad.io website](https://goad.io) and enter the address of the site you want to test.

![goad CLI interface](https://goad.io/assets/cli.gif)

## Installation

### Binary

The easiest way is to download a pre-built binary from [Goad.io] or from the [GitHub Releases][] page.

### From source

To build the Goad CLI from scratch, make sure you have a working Go 1.5 workspace ([instructions](https://golang.org/doc/install)), then:


1. Fetch the project with `go get`:

  ```sh
  go get github.com/goadapp/goad
  ```

2. Install Go [bindata][]:

  ```sh
  go get -u github.com/jteeuwen/go-bindata/...
  ```

3. Run make to build for all supported platforms

  ```sh
  make
  ```

  Alternatively, run append `osx`, `linux`, or `windows` to just build for one platform, for example:

  ```sh
  make osx
  ```

4. You'll find your `goad` binary in the `build` folder…

## Usage

### AWS credentials

Goad will read your credentials from `~/.aws/credentials` or from the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables ([more info](http://blogs.aws.amazon.com/security/post/Tx3D6U6WSFGOK2H/A-New-and-Standardized-Way-to-Manage-Credentials-in-the-AWS-SDKs)).

### CLI

```sh
# Get help:
$ goad --help
usage: goad --url=URL [<flags>]

An AWS Lambda powered load testing tool

Flags:
  -h, --help                   Show context-sensitive help (also try --help-long
                               and --help-man).
  -u, --url=URL                URL to load test
  -m, --method="GET"           HTTP method
  -b, --body=BODY              HTTP request body
  -c, --concurrency=10         Number of concurrent requests
  -n, --requests=1000          Total number of requests to make
  -N, --timelimit=3600         Seconds to max. to spend on benchmarking
  -t, --timeout=15             Request timeout in seconds
  -r, --region=REGION ...      AWS regions to run in (repeat flag to run in more
                               then one region)
  -p, --awsprofile=AWSPROFILE  AWS named profile to use
  -o, --output=OUTPUT          Optional path to JSON file for result storage
  -H, --header=HEADER ...      HTTP request header (repeat flag to add more then
                               one header)
  -s, --settings=.goad         Load settings from file (defaults to .goad in
                               cwd)
      --version                Show application version.

# For example:
$ goad -n 1000 -c 5 -u https://example.com
```

Note that sites such as https://google.com that employ redirects cannot be tested correctly at this time.

### Settings

Goad supports to load settings stored in a ini compatible ([toml][]) settings file. By default if looks
for a goad.ini file in the current working directory. Flags explicitly set on the commandline will
be overwritten.

```toml
url = "http://example.com/"
timeout = 15
concurrency = 10
requests = 100
timelimit = 60
method = "GET"
body = "Hello world"
regions = ["eu-west-1", "us-east-1"]
headers = ["cache-control: no-cache", "auth-token: YOUR-SECRET-AUTH-TOKEN"]
awsprofile = "my-aws-profile"
output = "test-result.json"
```

## How it works

Goad takes full advantage of the power of Amazon Lambdas and Go's concurrency for distributed load testing. You can use Goad to launch HTTP loads from up to four AWS regions at once. Each lambda can handle hundreds of concurrent connections, we estimate that Goad should be able to achieve peak loads of up to **100,000 concurrent requests**.

![Goad diagram](https://goad.io/assets/diagram.svg)

Running Goad will create the following AWS resources:

- An IAM Role for the lambda function.
- An IAM Role Policy that allows the lambda function to send messages to SQS, to publish logs and to spawn new lambda in case an individual lambda times out on a long running test.
- A lambda function.
- An SQS queue for the test.

A new SQS queue is created for each test run, and automatically deleted after the test is completed. The other AWS resources are reused in subsequent tests.

## How it was built

### Go CLI and server

* [AWS SDK for Go][]
* [Gorilla WebSocket][]
* [Termbox][]
* [UUID][]
* [bindata][]

### Goad executable

Written in pure Go, Goad takes care of instantiating all the AWS resources, collecting results and displaying them. Interestingly, it contains the executable of the Lambda worker, which is also written in Go.

There is also a webapi version, which the [Goad.io] website uses to execute its tests. This streams the results using WebSockets.

### Lambda workers

AWS Lambda instances are bootstrapped using node.js but the actual work on the Lambda instances is performed by a Go process. The HTTP
requests are distributed among multiple Lambda instances each running multiple concurrent goroutines, in order to achieve the desired
concurrency level with high throughput.

## License & Copyright

MIT License. Copyright 2016 [Joao Cardoso][], [Matias Korhonen][], [Rasmus Sten][], and [Stephen Sykes][].

See the LICENSE file for more details.

[Goad.io]: https://goad.io
[GitHub Releases]: https://github.com/gophergala2016/goad/releases

[AWS SDK for Go]: http://aws.amazon.com/sdk-for-go/
[Gorilla WebSocket]: https://github.com/gorilla/websocket
[Termbox]: https://github.com/nsf/termbox-go
[UUID]: https://github.com/satori/go.uuid
[bindata]: https://github.com/jteeuwen/go-bindata
[toml]: https://github.com/toml-lang/toml

[Gopher Gala]: http://gophergala.com/
[Joao Cardoso]: https://twitter.com/jcxplorer
[Matias Korhonen]: https://twitter.com/matiaskorhonen
[Rasmus Sten]: https://twitter.com/pajp
[Stephen Sykes]: https://twitter.com/sdsykes
