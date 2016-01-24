# Goad

Goad is an AWS Lambda powered, highly distributed,
load testing tool built in Go for the 2016 [Gopher Gala][].

![Go + Load ⇒ Goad](./site/src/img/go-plus-load.png)

Goad allows you to load test your websites from all over the world whilst costing you the tiniest fractions of a penny by using AWS Lambda in multiple regions simultaneously.

You can run Goad from your machine using your own AWS credentials, and it will automatically create the AWS resources you need and execute your test, and display the results broken down by region. This way you can see how fast your website is from all the major regions of the world.

If you just want to try Goad out, visit the [Goad.io website](https://goad.io) and enter the address of the site you want to test.

![goad CLI interface](./site/src/img/cli.gif)

## Installation

### Binary

The easiest way is to download a pre-built binary from [Goad.io] or from the [GitHub Releases][] page.

### From source

To build the Goad CLI from scratch, make sure you have a working Go 1.5 workspace ([instructions](https://golang.org/doc/install)), then:


1. Fetch the project with `go get`:

  ```sh
  go get github.com/gophergala2016/goad
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
$ goad -h
Usage of goad:
  -c uint
      number of concurrent requests (default 10)
  -m string
      HTTP method (default "GET")
  -n uint
      number of total requests to make (default 1000)
  -r string
      AWS regions to run in (comma separated, no spaces) (default "us-east-1")
  -t uint
      request timeout in seconds (default 15)
  -u string
      URL to load test (required)

# For example:
$ goad -n 1000 -c 5 -u https://example.com
```

## How it works

Goad takes full advantage of the power of Amazon Lambdas and Go's concurrency for distributed load testing. You can use Goad to launch HTTP loads from up to four AWS regions at once. Each lambda can handle hundreds of concurrent connections, we estimate that Goad should be able to achieve peak loads of up to **100,000 concurrent requests**.

![Goad diagram](./site/src/img/diagram.png)

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

### [Goad.io][] site

We used React and ES6 to build the online demo, and we used a Node.js-based toolchain for the website. We wanted to use Go, but as far as we could tell none of static site builders built with Go have built in support for an asset pipeline that would support ES6 modules out of the box (or even easily). However, note that the website is mainly a demo and download location, and is not essential to use Goad.

The demo uses WebSockets and React to present the results of the demo load test every few seconds as more results come in.

As we wanted to prevent the site from being used as a DDoS tool, the online demo is very limited but hopefully enough to demonstrate the usefulness of the CLI version of Goad.

* [Bootstrap][] and the Darkly theme from [Bootswatch][]
* [WebPack][]
* [React][]
* [Babel][]
* [SCSS/Sass][]


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

[Bootstrap]: http://getbootstrap.com/
[Bootswatch]: https://bootswatch.com/
[WebPack]: https://webpack.github.io/
[React]: https://facebook.github.io/react/
[Babel]: https://babeljs.io/
[SCSS/Sass]: http://sass-lang.com/

[Gopher Gala]: http://gophergala.com/
[Joao Cardoso]: https://twitter.com/jcxplorer
[Matias Korhonen]: https://twitter.com/matiaskorhonen
[Rasmus Sten]: https://twitter.com/pajp
[Stephen Sykes]: https://twitter.com/sdsykes
