var gulp = require("gulp");
var gutil = require("gulp-util");
var webpack = require("webpack");
var WebpackDevServer = require("webpack-dev-server");
var webpackConfig = require("./webpack.config.js");
var rimraf = require("rimraf");

gulp.task("default", ["webpack-dev-server"]);

gulp.task("dist", function (callback) {
	var config = Object.create(webpackConfig);
  config.plugins.push(
    new webpack.optimize.DedupePlugin(),
		new webpack.optimize.UglifyJsPlugin()
  );

	rimraf("dist/*", function () {
		webpack(config, function(err, stats) {
			if(err) throw new gutil.PluginError("webpack:build", err);
			gutil.log("[dist]", stats.toString({
				colors: true
			}));
			callback();
		});
	});
});

gulp.task("webpack-dev-server", function (callback) {
	var config = Object.create(webpackConfig);
	config.entry.push(
		"webpack-dev-server/client?http://localhost:8080",
		"webpack/hot/dev-server"
	);
	config.devtool = "eval";
	config.debug = true;
  config.plugins.push(new webpack.HotModuleReplacementPlugin());

	new WebpackDevServer(webpack(config), {
    contentBase: __dirname + "/dist",
    hot: true,
		stats: {
			colors: true
		}
	}).listen(8080, "localhost", function(err) {
		if(err) throw new gutil.PluginError("webpack-dev-server", err);
		gutil.log("[webpack-dev-server]", "http://localhost:8080/webpack-dev-server/index.html");
	});
});
