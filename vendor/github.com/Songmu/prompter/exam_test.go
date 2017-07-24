package prompter_test

import (
	"fmt"

	"github.com/Songmu/prompter"
)

func ExamplePrompter() {
	input := (&prompter.Prompter{
		Choices:    []string{"aa", "bb", "cc"},
		Default:    "aa",
		Message:    "please select",
		IgnoreCase: true,
	}).Prompt()
	fmt.Println("aa")
	fmt.Printf("your input is %s.", input)
	// Output:
	// please select (aa/bb/cc) [aa]: aa
	// your input is aa.
}

func ExampleChoose() {
	lang := prompter.Choose("Which language do you like the most?", []string{"Perl", "Golang", "Scala", "Ruby"}, "Perl")
	fmt.Println("Perl")
	fmt.Printf("Great! You like %s!", lang)
	// Output:
	// Which language do you like the most? (Perl/Golang/Scala/Ruby) [Perl]: Perl
	// Great! You like Perl!
}

func ExamplePrompt() {
	answer := prompter.Prompt("Enter your twitter ID", "")
	_ = answer
	fmt.Println("Songmu")
	fmt.Printf("Hi Songmu!")
	// Output:
	// Enter your twitter ID: Songmu
	// Hi Songmu!
}

func ExamplePassword() {
	passwd := prompter.Password("Enter your password")
	_ = passwd
	fmt.Println("****")
	fmt.Print("I got your password! :P")
	// Output:
	// Enter your password: ****
	// I got your password! :P
}

func ExampleYN() {
	if prompter.YN("Do you like sushi?", true) {
		fmt.Println("y")
		fmt.Print("Nice! Let's go sushi bar!")
	}
	// Output:
	// Do you like sushi? (y/n) [y]: y
	// Nice! Let's go sushi bar!
}

func ExampleYesNo() {
	if !prompter.YesNo("Do you like beer?", false) {
		fmt.Println("no")
		fmt.Print("Oops!")
	}
	// Output:
	// Do you like beer? (yes/no) [no]: no
	// Oops!
}

