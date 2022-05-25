package errors

import "fmt"

func Ctx(message string, e error) error {
	return fmt.Errorf("%s: %w", message, e)
}
