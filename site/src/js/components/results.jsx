import React from 'react';

export default class Results extends React.Component {
  requestResults() {
    var socket = new WebSocket('ws://echo.websocket.org');

    socket.onopen = event => {
      this.setState({
        socketOpen: true
      });
      this.socket = socket;

      this.sendToSocket(this.props);
    };

    socket.onclose = event => {
      this.setState({
        socketOpen: false
      });
    }

    socket.onmessage = this.handleMessage.bind(this)
  }

  handleMessage(event) {
    console.log("Message", event)
  }

  componentWillMount() {
    this.requestResults();
  }

  results() {
    return `1000 / 1000 ∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎ 100.00 %

Summary:
  Total:        21.1307 secs.
  Slowest:      2.9959 secs.
  Fastest:      0.9868 secs.
  Average:      2.0827 secs.
  Requests/sec: 47.3246

Status code distribution:
  [200] 1000 responses`;
  }

  sendToSocket(data) {
    if (this.state && this.state.socketOpen) {
      this.socket.send(JSON.stringify(data));
    }
  }

  render() {
    var socketClass = "float-right text-danger glyphicon glyphicon-remove-sign";
    if (this.state && this.state.socketOpen) {
      socketClass = "float-right text-success glyphicon glyphicon-flash";
    }
    var socket = <span className={socketClass} aria-hidden="true"></span>;

    return (
      <div className="panel panel-results test-results">
        <div className="panel-heading">
          <h3 className="panel-title">$ goad -n 1000 -c 10 -m GET {this.props.url} {socket}</h3>
        </div>
        <div className="panel-body">
          <pre>{this.results()}<span className="blinking-cursor">▊</span></pre>
        </div>
      </div>
    );
  }
}
