import React from 'react';
var URI = require("urijs");

export default class Results extends React.Component {
  requestResults() {
    // curl -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" \
    // -H "Host: api.goad.io" -H "Origin: https://api.goad.io" \
    //-H "Sec-Websocket-Version: 13" -H "Sec-Websocket-Key: 1" \
    // "https://api.goad.io/goad?url=http://dll.nu&c=5&tot=1000&timeout=5"

    var wsURI = URI("wss://api.goad.io/goad").query({
      url: this.props.url,
      c: 5,
      tot: 1000,
      timeout: 5
    });

    var socket = new WebSocket(wsURI.toString());

    socket.onopen = event => {
      this.setState({
        socketOpen: true
      });
      this.socket = socket;
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
          <pre>Waiting for results...<span className="blinking-cursor">▊</span></pre>
        </div>
      </div>
    );
  }
}
