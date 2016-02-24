package helpers

import "fmt"

// StringsliceFlag allows you to pass a flag more than one time
type StringsliceFlag []string

func (s *StringsliceFlag) String() string {
	return fmt.Sprintf("%s", *s)
}

// Set appends each instance of the flag passed
func (s *StringsliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}
