package prompter

import (
	"regexp"
	"testing"
)

func TestPrompt(t *testing.T) {
	if Prompt("enter!", "ent") != "ent" {
		t.Errorf("something went wrong")
	}
}

func TestYN(t *testing.T) {
	if !YN("enter!", true) {
		t.Errorf("something went wrong")
	}
}

func TestYesNo(t *testing.T) {
	if YesNo("enter!", false) {
		t.Errorf("something went wrong")
	}
}

func TestPassword(t *testing.T) {
	if Password("enter!") != "" {
		t.Errorf("something went wrong")
	}
}

func TestChoose(t *testing.T) {
	if Choose("enter!", []string{"Perl", "Golang"}, "Perl") != "Perl" {
		t.Errorf("something went wrong")
	}
}

func TestRegexp(t *testing.T) {
	if Regexp("enter!", regexp.MustCompile(`\A(?:Perl|Golang)\z`), "Perl") != "Perl" {
		t.Errorf("something went wrong")
	}
}
