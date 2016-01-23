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
  Speed index:  Hahahaha

Response time histogram:
  0.987 [1]     |
  1.188 [2]     |
  1.389 [3]     |
  1.590 [18]    |∎∎
  1.790 [85]    |∎∎∎∎∎∎∎∎∎∎∎
  1.991 [244]   |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  2.192 [284]   |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  2.393 [304]   |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  2.594 [50]    |∎∎∎∎∎∎
  2.795 [5]     |
  2.996 [4]     |

Latency distribution:
  10% in 1.7607 secs.
  25% in 1.9770 secs.
  50% in 2.0961 secs.
  75% in 2.2385 secs.
  90% in 2.3681 secs.
  95% in 2.4451 secs.
  99% in 2.5393 secs.

Status code distribution:
  [200] 1000 responses`;
  }

  render() {
    return (
      <div className="panel panel-info test-results">
        <div className="panel-heading">
          <h3 className="panel-title">$ goad -n 1000 -c 10 -m GET {this.props.url}</h3>
        </div>
        <div className="panel-body">
          <pre>{this.results()}</pre>
        </div>
      </div>
    );
  }
}
