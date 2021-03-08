package utils

import "fmt"

//WrapError return fmt.Errorf("%s: %w", msg, err)
func WrapError(msg string, err error) error {
	return fmt.Errorf("%s: %w", msg, err)
}
