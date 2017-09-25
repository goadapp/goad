package main

import (
	"fmt"
	"regexp"

	"github.com/Songmu/prompter"
)

func main() {
	input := (&prompter.Prompter{
		Choices:    []string{"aa", "bb", "cc"},
		Default:    "aa",
		Message:    "plaase select",
		IgnoreCase: true,
	}).Prompt()
	fmt.Println("your input is " + input)

	input = (&prompter.Prompter{
		Message: "enter password",
		Regexp:  regexp.MustCompile(`.{8,}`),
		NoEcho:  true,
	}).Prompt()
	fmt.Println("your password is " + input)

	if prompter.YN("do you like sushi?", true) {
		fmt.Println("Nice!")
	} else {
		fmt.Println("It's Okay.")
	}

	if prompter.YesNo("do you like beer?", false) {
		fmt.Println("Nice!")
	} else {
		fmt.Println("It's Okay.")
	}

	passwd := prompter.Password("enter your password")
	fmt.Println("I got your password :P " + passwd)

	lang := prompter.Choose("Whitch language do you like the most?", []string{"Perl", "Golang", "Scala", "Ruby"}, "Perl")
	if lang == "Perl" {
		fmt.Println("So Nice!")
	} else {
		fmt.Println("I like also " + lang + " too.")
	}
}
