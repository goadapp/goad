var webpack = require("webpack");
var path = require("path");
var HtmlWebpackPlugin = require("html-webpack-plugin");
var autoprefixer = require("autoprefixer");
var precss = require("precss");

module.exports = {
  entry: [
    "./src/js/main.js"
  ],
  output: {
    path: path.join(__dirname, "dist"),
    filename: "js/[name].[hash].js",
    chunkFilename: "[chunkhash].js"
  },
  module: {
    loaders: [
      {
        test: /\.js$/,
        loader: "react-hot!babel",
        include: __dirname + "/src"
      },
      {
        test: /\.scss$/,
        loader: "style-loader!css-loader!sass-loader!postcss-loader"
      },
      {
        test: /\.(png|jpg|gif)$/,
        loader: "file-loader?name=img/[name].[hash].[ext]"
      }
    ]
  },
  plugins: [
    new HtmlWebpackPlugin({
      template: __dirname + "/src/index.html",
      hash: true,
      filename: "index.html",
      inject: "body"
    })
  ],
  postcss: function () {
    return [autoprefixer, precss];
  }
};
