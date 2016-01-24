# Goad

Goad is an AWS Lambda powered, highly distributed,
load testing tool built in Go for the 2016 [Gopher Gala][].

**Go + Load** ⇒ **Goad**


## Libraries and tools used

### Go libraries

* [AWS SDK for Go][]
* [Gorilla WebSocket][]
* [Termbox][]
* [UUID][]

### Goad.io site

* [WebPack][]
* [React][]
* [Babel][]
* [SCSS/Sass][]

## How it works

Goad takes full advantage of the power of Amazon Lambdas for distributed load testing. You can use goad to launch HTTP loads from up to four AWS regions at once. Each lambda can handle hundreds of concurrent connections, we estimate that Goad should be able to achieve peak loads of up to **10,000 requests/second**.

![Goad diagram](./site/src/img/diagram.png)


## License & Copyright

MIT License. Copyright 2016 [Joao Cardoso][], [Matias Korhonen][], [Rasmus Sten][], and [Stephen Sykes][].

See the LICENSE file for more details.

[AWS SDK for Go]: http://aws.amazon.com/sdk-for-go/
[Gorilla WebSocket]: https://github.com/gorilla/websocket
[Termbox]: https://github.com/nsf/termbox-go
[UUID]: https://github.com/satori/go.uuid

[WebPack]: https://webpack.github.io/
[React]: https://facebook.github.io/react/
[Babel]: https://babeljs.io/
[SCSS/Sass]: http://sass-lang.com/

[Gopher Gala]: http://gophergala.com/
[Joao Cardoso]: https://twitter.com/jcxplorer
[Matias Korhonen]: https://twitter.com/matiaskorhonen
[Rasmus Sten]: https://twitter.com/pajp
[Stephen Sykes]: https://twitter.com/sdsykes
