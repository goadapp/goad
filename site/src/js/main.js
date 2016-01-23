import React from 'react';
import ReactDOM from 'react-dom';
window.React = React;
window.ReactDOM = ReactDOM;

import Form from './components/form.jsx';
ReactDOM.render(<Form />, document.getElementById("form"));

const binaries = [
  { os: "OS X", architecture: 64, url: "#mac" },
  { os: "Linux", architecture: 32, url: "#linux32" },
  { os: "Linux", architecture: 64, url: "#linux64" },
  { os: "Windows", architecture: 32, url: "#windows32" },
  { os: "Windows", architecture: 64, url: "#windows64" },
];

import Downloads from './components/downloads.jsx';
ReactDOM.render(<Downloads binaries={binaries} />, document.getElementById("downloads"));
