import React from "react";
var validUrl = require('valid-url');
var request = require('superagent');

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
      // request.post("http://httpbin.org/post").send({url: this.state.url}).end(function(err, res){
      //   if (typeof(self.props.onUpdate) == "function") {
      //     self.props.onUpdate(err, res, self.state)
      //   }
      // });
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
      helpBlock = <span class="help-block">Please enter a fully qualified valid URL…</span>;
    }

    return (
      <form onSubmit={this.handleSubmit.bind(this)}>
        <div className={formGroupClass}>
          <label className="control-label" htmlFor="url">URL</label>
          <input type="url" className="form-control" id="url" placeholder="https://…" onChange={this.handleURLChange.bind(this)}/>
          {helpBlock}
        </div>
        <div className="form-group">
          <label>Target load</label>
          <p className="form-control-static">10 / second (download the CLI tool to use a higher load)</p>
        </div>
        <button type="submit" className="btn btn-danger">Start test</button>
      </form>
    );
  }
}
