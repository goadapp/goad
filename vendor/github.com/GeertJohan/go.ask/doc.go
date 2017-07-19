// Package ask helps with writing simple yes/no cli-based questions.
// There are several methods, following these rules
//
// Methods starting with "Must" do not return an error.
// Instead they panic if an error occurs (pty not available? closed stdin/stdout?).
//
// Methods containing "Default" have an extra argument (placed first), which is returned
// when the user simply hits [enter].
//
// When user replies with "y", "Y" or "yes", true is returned
// When user replies with "n", "N" or "no", false is returned
//
// Methods with a trailing "f" wrap fmt.Sprintf
package ask
