package ask

import (
	"fmt"
	"github.com/GeertJohan/go.linenoise"
)

func Ask(question string) (bool, error) {
	return ask(question, false, false)
}

func Askf(questionFormat string, a ...interface{}) (bool, error) {
	return ask(fmt.Sprintf(questionFormat, a...), false, false)
}

func DefaultAsk(def bool, question string) (bool, error) {
	return ask(question, true, def)
}

func DefaultAskf(def bool, questionFormat string, a ...interface{}) (bool, error) {
	return ask(fmt.Sprintf(questionFormat, a...), true, def)
}

func MustAsk(question string) bool {
	res, err := ask(question, false, false)
	if err != nil {
		panic(err)
	}
	return res
}

func MustAskf(questionFormat string, a ...interface{}) bool {
	res, err := ask(fmt.Sprintf(questionFormat, a...), false, false)
	if err != nil {
		panic(err)
	}
	return res
}

func MustDefaultAsk(def bool, question string) bool {
	res, err := ask(question, true, def)
	if err != nil {
		panic(err)
	}
	return res
}

func MustDefaultAskf(def bool, questionFormat string, a ...interface{}) bool {
	res, err := ask(fmt.Sprintf(questionFormat, a...), true, def)
	if err != nil {
		panic(err)
	}
	return res
}

func ask(question string, useDef bool, def bool) (bool, error) {

	// build prompt
	y := "y"
	n := "n"
	if useDef {
		if def {
			y = "Y"
		} else {
			n = "N"
		}
	}
	prompt := fmt.Sprintf("%s, [%s/%s] ", question, y, n)

	// loop until we have an answer
	for {
		line, err := linenoise.Line(prompt)
		if err != nil {
			return false, err
		}
		switch line {
		case "y", "Y", "yes":
			return true, nil
		case "n", "N", "no":
			return false, nil
		default:
			if useDef && len(line) == 0 {
				// use default
				return def, nil
			}
			fmt.Println("Inavlid answer. Please give 'y' or 'n'.")
			continue
		}
	}
}
