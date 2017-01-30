var spawn = require("child_process").spawn;

exports.handler = function(event, context) {
    child = spawn(event.file, event.args);

    child.stdout.on("data", function (data) {
        console.log(data.toString())
    });
    child.stderr.on("data", function (data) {
        console.error(data.toString())
    });
    child.on("close", function(code) {
        if (code == 0) {
            context.succeed("Process complete!");
        } else {
            context.fail("Process failed with exit code: " + code);
        }
    });
};
