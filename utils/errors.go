package utils

import "fmt"

//Wrap return fmt.Errorf("%s: %w", msg, err)
func Wrap(msg string, err error) error {
	return fmt.Errorf("%s: %w", msg, err)
}
