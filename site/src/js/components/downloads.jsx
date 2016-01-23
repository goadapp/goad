import React from "react";
var platform = require("platform");

export default class Downloads extends React.Component {
  render() {
    const items = (this.props.binaries || []).map(function(binary) {
      var arch;
      switch(binary.architecture) {
        case 32:
          arch = "32-bit (x86)";
          break;
        case 64:
          arch = "64-bit (x86-64)"
          break;
      }

      var className = "list-group-item";
      console.log(platform);

      if (binary.os == "OS X" && platform.os.family == "OS X") {
        className = className + " list-group-item-success";
      } else if (binary.os == platform.os.family) {
        if (platform.os.architecture == binary.architecture) {
          className = className + " list-group-item-success";
        }
      }

      return <a className={className} key={binary.url} href={binary.url}>{binary.os} {arch}</a>
    });

    return (
      <div className="list-group">
        {items}
      </div>
    );
  }
}
