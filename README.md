# Goad

Goad is an AWS Lambda powered, highly distributed,
load testing tool built in Go for the 2016 [Gopher Gala][].

![Go + Load ⇒ Goad](./site/src/img/go-plus-load.png)

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
goad -n 1000 -c 5 https://example.com
```

## How it works

Goad takes full advantage of the power of Amazon Lambdas for distributed load testing. You can use goad to launch HTTP loads from up to four AWS regions at once. Each lambda can handle hundreds of concurrent connections, we estimate that Goad should be able to achieve peak loads of up to **100,000 concurrent requests**.

![Goad diagram](./site/src/img/diagram.png)

## How it was built

### Go CLI and server

* [AWS SDK for Go][]
* [Gorilla WebSocket][]
* [Termbox][]
* [UUID][]
* [bindata][]

### Lambda workers

AWS Lambda instances are bootstrapped using node.js but the actual work on the Lambda instances is performed by a Go process. The HTTP
requests are distributed among multiple Lambda instances each running multiple concurrent goroutines, in order to achieve the desired
concurrency level with high throughput.

### [Goad.io][] site

Because we we wanted to use React and ES6 to build the online demo, we opted to use a Node.js-based toolchain for the website. As far as we could tell, none of static site builders built with Go have built in support for an asset pipeline that would support ES6 modules out of the box or even easily…

We use WebSockets and React to present the results of the demo load test every few seconds as more results come in.

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
