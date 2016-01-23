import React from 'react';
import Form from './form.jsx';
import Results from './results.jsx';

export default class App extends React.Component {
  render() {
    if (false) {
      return (<Results url="https://example.invalid" />);
    } else {
      return (<Form />);
    }
  }
}
