package prompter

import (
	"fmt"
	"testing"
)

func TestMsg(t *testing.T) {
	p := &Prompter{
		Choices: []string{"aa", "bb", "cc"},
		Default: "aa",
		Message: "plaase select",
	}

	if p.msg() != "plaase select (aa/bb/cc) [aa]: " {
		t.Errorf("something went wrong")
	}

	if p.errorMsg() != "# Enter `aa`, `bb` or `cc`" {
		t.Errorf("something went wrong")
	}

	if !p.inputIsValid("aa") {
		t.Errorf("something went wrong")
	}

	if p.inputIsValid("AA") {
		t.Errorf("something went wrong")
	}

	input := p.Prompt()
	if input != "aa" {
		fmt.Printf("%s\n", input)
		t.Errorf("something went wrong")
	}
}
