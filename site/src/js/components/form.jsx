import React from "react";
var validUrl = require('valid-url');

export default class Form extends React.Component {
  constructor(props) {
    super(props);
    this.state = {url: ""};
  }

  handleURLChange(e) {
    const url = e.target.value;
    this.setState({url: url});
  }

  handleSubmit(e) {
    e.preventDefault();

    var self = this;

    // TODO: Use a real API URL
    if (validUrl.isWebUri(this.state.url)) {
      self.props.onUpdate(self.state)
    }
  }

  render() {
    var formGroupClass = "form-group";
    var helpBlock;
    if (this.state.url == null || this.state.url == "") {
      // No validation state
    } else if (validUrl.isWebUri(this.state.url)) {
      formGroupClass = `${formGroupClass} has-success`
    } else {
      formGroupClass = `${formGroupClass} has-error`
      helpBlock = <span className="help-block">Please enter a fully qualified valid URL…</span>;
    }

    return (
      <form onSubmit={this.handleSubmit.bind(this)}>
        <div className={formGroupClass}>
          <label className="control-label" htmlFor="url">URL</label>
          <input type="url" className="form-control" id="url" placeholder="https://…" onChange={this.handleURLChange.bind(this)}/>
          {helpBlock}
        </div>
        <button type="submit" className="btn btn-danger">Start test</button>
        <hr/>
        <h4>Demo settings</h4>
        <p>Download the CLI tool for full control.</p>
        <div className="form-group">
          <label>Maximum concurrency</label>
          <p className="form-control-static">5</p>
        </div>
        <div className="form-group">
          <label>Total number of requests</label>
          <p className="form-control-static">1000</p>
        </div>
        <div className="form-group">
          <label>Timeout after</label>
          <p className="form-control-static">5 seconds</p>
        </div>
      </form>
    );
  }
}
