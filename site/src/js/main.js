import React from 'react';
import ReactDOM from 'react-dom';
import App from './components/app.jsx';
import Downloads from './components/downloads.jsx';

window.React = React;
window.ReactDOM = ReactDOM;

ReactDOM.render(<App />, document.getElementById("demo-app"));

const binaries = [
  { os: "OS X", architecture: 64, url: "https://github.com/gophergala2016/goad/releases#mac" },
  { os: "Linux", architecture: 32, url: "https://github.com/gophergala2016/goad/releases#linux32" },
  { os: "Linux", architecture: 64, url: "https://github.com/gophergala2016/goad/releases#linux64" },
  { os: "Windows", architecture: 32, url: "https://github.com/gophergala2016/goad/releases#windows32" },
  { os: "Windows", architecture: 64, url: "https://github.com/gophergala2016/goad/releases#windows64" },
];

ReactDOM.render(<Downloads binaries={binaries} />, document.getElementById("downloads"));
