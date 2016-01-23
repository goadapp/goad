import React from 'react';

export default class Results extends React.Component {
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

  render() {
    return (
      <div className="panel panel-results test-results">
        <div className="panel-heading">
          <h3 className="panel-title">$ goad -n 1000 -c 10 -m GET {this.props.url}</h3>
        </div>
        <div className="panel-body">
          <pre>{this.results()}<span className="blinking-cursor">▊</span></pre>
        </div>
      </div>
    );
  }
}
