import React from 'react';
import Form from './form.jsx';
import Results from './results.jsx';

export default class App extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      url: "",
      submitted: false
    };
  }

  handleUpdate(data) {
    this.setState({
      submitted: true,
      url: data.url
    });
  }

  render() {
    if (this.state.submitted) {
      return (<Results url={this.state.url} />);
    } else {
      return (<Form onUpdate={this.handleUpdate.bind(this)} />);
    }
  }
}
