package forms

import (
	"reflect"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// DefaultValidator implements a validator with lazy initialization
type DefaultValidator struct {
	once     sync.Once
	validate *validator.Validate
}

var _ binding.StructValidator = &DefaultValidator{}

// ValidateStruct validates whether the fields of a struct satisfy validation constraints
// specified via struct tags. It returns an error if validation fails.
func (v *DefaultValidator) ValidateStruct(obj interface{}) error {

	if kindOfData(obj) == reflect.Struct {

		v.lazyinit()

		if err := v.validate.Struct(obj); err != nil {
			return err
		}
	}

	return nil
}

// Engine returns the underlying validator engine. It ensures the validator
// is initialized before returning it.
func (v *DefaultValidator) Engine() interface{} {
	v.lazyinit()
	return v.validate
}

// lazyinit performs one-time initialization of the validator
func (v *DefaultValidator) lazyinit() {
	v.once.Do(func() {

		v.validate = validator.New()
		v.validate.SetTagName("binding")

		// add any custom validations etc. here

	})
}

// kindOfData returns the reflection Kind of the passed data
// If the data is a pointer, it returns the Kind of the referenced value
func kindOfData(data interface{}) reflect.Kind {

	value := reflect.ValueOf(data)
	valueType := value.Kind()

	if valueType == reflect.Ptr {
		valueType = value.Elem().Kind()
	}
	return valueType
}
