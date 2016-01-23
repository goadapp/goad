import React from "react";

export default class Form extends React.Component {
  render() {
    return (
      <form>
        <div className="form-group">
          <label htmlFor="url">URL</label>
          <input type="url" className="form-control" id="url" placeholder="https://…" />
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
