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
        socketOpen: true,
        error: null
      });
      this.socket = socket;
    };

    socket.onclose = event => {
      this.setState({
        socketOpen: false,
        error: null
      });
    }

    socket.onerror = event => {
      this.setState({
        socketOpen: socket.readyState == WebSocket.OPEN,
        error: true,
        errorData: event.data
      });
    }

    socket.onmessage = this.handleMessage.bind(this)
  }

  handleMessage(event) {
    const data = JSON.parse(event.data);
    this.setState({
      data: data
    });
  }

  componentWillMount() {
    this.requestResults();
  }

  sendToSocket(data) {
    if (this.state && this.state.socketOpen) {
      this.socket.send(JSON.stringify(data));
    }
  }

  resultsHandler() {
    if (this.state) {
      if (this.state.socketOpen || this.state.data) {
        if (this.state.data) {
          return this.resultsFormatter();
        } else {
          return "Waiting for results..."
        }
      } else {
        if (this.state.error == null) {
          return "Waiting for socket...";
        } else {
          return "Socket error ";
        }
      }
    } else {
      return "Loading..."
    }
  }

  resultsFormatter() {
    if (this.state.data) {
      // return `Summary`;

      const data = this.state.data;
      var regions = ["us-east-1", "us-west-2", "eu-west-1", "ap-northeast-1"].map(name => {
        if (data.Regions[name]) {
          return this.formatRegionData(name, data.Regions[name]);
        }
      })

      return regions.join("\n").trim();
    } else {
      return "No results";
    }
  }

  // Region: us-east-1
  //   TotReqs   TotBytes  AveTime   AveReq/s Ave1stByte
  //      1000   18323910    0.18s      54.83      0.18s

  formatRegionData(region, data) {
    return `Region: ${region}

        Total Reqs               ${data["total-reqs"]}
        Total Bytes              ${data["tot-bytes-read"]}
        Average Time             ${data["ave-time-for-req"]}
        Average Req/s            ${data["ave-req-per-sec"]}
        Average Time To 1st Byte ${data["ave-time-to-first"]}
    `
  }

  render() {
    var cursor = <span />;
    var socketClass = "float-right glyphicon glyphicon-remove-sign";

    if (this.state) {
      if (this.state.socketOpen) {
        socketClass = "float-right text-success glyphicon glyphicon-flash";
        cursor = <span className="blinking-cursor">▊</span>;
      } else {
        if (this.state.data) {
          socketClass = "float-right text-info glyphicon glyphicon-flash";
        } else {
          socketClass = "float-right text-danger glyphicon glyphicon-remove-sign";
        }
      }
    }

    var socket = <span className={socketClass} aria-hidden="true"></span>;

    return (
      <div className="panel panel-results test-results">
        <div className="panel-heading">
          <h3 className="panel-title">$ goad -n 1000 -c 10 -m GET {this.props.url} {socket}</h3>
        </div>
        <div className="panel-body">
          <pre>{this.resultsHandler()}{cursor}</pre>
        </div>
      </div>
    );
  }
}
