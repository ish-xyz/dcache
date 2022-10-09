package utils

import "github.com/go-playground/validator"

func Validate(objects ...interface{}) error {

	validate := validator.New()

	for _, obj := range objects {
		err := validate.Struct(obj)
		if err != nil {
			return err
		}
	}

	return nil
}
