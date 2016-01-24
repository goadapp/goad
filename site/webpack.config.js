var webpack = require("webpack");
var path = require("path");
var HtmlWebpackPlugin = require("html-webpack-plugin");
var autoprefixer = require("autoprefixer");
var precss = require("precss");
var ExtractTextPlugin = require('extract-text-webpack-plugin');
var CopyWebpackPlugin = require('copy-webpack-plugin');

module.exports = {
  entry: [
    "bootstrap-loader/extractStyles",
    "./src/js/main.js",
  ],
  output: {
    path: path.join(__dirname, "dist"),
    filename: "js/[name].[hash].js",
    chunkFilename: "[chunkhash].js"
  },
  module: {
    loaders: [
      {
        test: /\.jsx?$/,
        exclude: /(node_modules|bower_components)/,
        loader: "react-hot!babel?cacheDirectory,presets[]=react,presets[]=es2015",
        include: __dirname + "/src",
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
      }
    ]
  },
  plugins: [
    new ExtractTextPlugin('styles.css', { allChunks: true }),
    new HtmlWebpackPlugin({
      template: __dirname + "/src/index.html.slim",
      hash: true,
      filename: "index.html",
      inject: "body"
    }),
    new CopyWebpackPlugin([
       { from: __dirname + "/src/img/favicon-16.png", to: "assets" },
       { from: __dirname + "/src/img/favicon-32.png", to: "assets" },
       { from: __dirname + "/src/img/cli.gif", to: "assets" },
       { from: __dirname + "/src/img/go-plus-load.png", to: "assets" },
       { from: __dirname + "/src/img/diagram.svg)", to: "assets" },
       { from: __dirname + "/src/img/diagram.png)", to: "assets" },
    ])
  ],
  postcss: function() {
    return [autoprefixer, precss];
  }
};
