import React from 'react';
import ReactDOM from 'react-dom';
import App from './components/app.jsx';
import Downloads from './components/downloads.jsx';

window.React = React;
window.ReactDOM = ReactDOM;

ReactDOM.render(<App />, document.getElementById("demo-app"));

const binaries = [
  { os: "OS X", architecture: 64, url: "#mac" },
  { os: "Linux", architecture: 32, url: "#linux32" },
  { os: "Linux", architecture: 64, url: "#linux64" },
  { os: "Windows", architecture: 32, url: "#windows32" },
  { os: "Windows", architecture: 64, url: "#windows64" },
];

ReactDOM.render(<Downloads binaries={binaries} />, document.getElementById("downloads"));
