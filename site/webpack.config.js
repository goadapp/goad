var webpack = require("webpack");
var path = require("path");
var HtmlWebpackPlugin = require("html-webpack-plugin");
var autoprefixer = require("autoprefixer");
var precss = require("precss");

module.exports = {
  entry: [
    "bootstrap-loader",
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
        test: /\.css$/,
        loaders: [ 'style', 'css', 'postcss' ]
      },
      {
        test: /\.scss$/,
        loader: "style-loader!css-loader!sass-loader!postcss-loader"
      },
      {
        test: /bootstrap-sass\/assets\/javascripts\//,
        loader: 'imports?jQuery=jquery'
      },
      {
        test: /\.(png|jpg|gif|svg|eot|woff2?|ttf|otf|svg)$/,
        loader: "file-loader?name=assets/[name].[hash].[ext]"
      },
      {
        test: /\.html\.(slm|slim)$/,
        loader: 'html!slm'
      },
      {
        test: require.resolve("jquery"),
        loader: "imports?this=>window"
      }
    ]
  },
  plugins: [
    new HtmlWebpackPlugin({
      template: __dirname + "/src/index.html.slim",
      hash: true,
      filename: "index.html",
      inject: "body"
    })
  ],
  postcss: function() {
    return [autoprefixer, precss];
  }
};
