package routines

import (
	"regexp"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate
	once     sync.Once
)

func Init() {
	once.Do(func() {
		validate = validator.New()

		// Register custom validators here
		if err := RegisterCustomValidators(validate); err != nil {
			panic(err) // Or handle the error more gracefully
		}
		// Add more custom validators as needed
	})
}
func GetValidator() *validator.Validate {
	Init() // Ensure initialization
	return validate
}

// Custom validator function for hexadecimal string
func isHexadecimal(fl validator.FieldLevel) bool {
	hexString := fl.Field().String()

	// Regular expression for hexadecimal string
	hexRegex := regexp.MustCompile(`^[0-9a-fA-F]+$`)

	return hexRegex.MatchString(hexString)
}

func RegisterCustomValidators(validate *validator.Validate) error {
	// Register custom validator
	if err := validate.RegisterValidation("hexadecimalan", isHexadecimal); err != nil {
		return err
	}
	return nil
}
