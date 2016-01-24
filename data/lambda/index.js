var execFile = require("child_process").execFile;

exports.handler = function(event, context) {
    child = execFile(event.file, event.args, function(error) {
        context.done(error, "Process complete!");
    });
    child.stdout.on("data", console.log);
    child.stderr.on("data", console.error);
};
