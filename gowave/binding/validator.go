package binding

import (
	"errors"
	"github.com/go-playground/validator/v10"
	"reflect"
	"sync"
)

type StructValidator interface {
	ValidateStruct(obj any) error
	Validator() any
}

var Validator StructValidator = &defaultValidator{}

type defaultValidator struct {
	once      sync.Once
	validator *validator.Validate
}

func (d *defaultValidator) ValidateStruct(obj any) error {
	of := reflect.ValueOf(obj)
	switch of.Kind() {
	case reflect.Ptr:
		return d.ValidateStruct(of.Elem().Interface())
	case reflect.Struct:
		d.lazyInit()
		return d.validator.Struct(obj)
	case reflect.Slice:
		for i := 0; i < of.Len(); i++ {
			if err := d.ValidateStruct(of.Index(i).Interface()); err != nil {
				return err
			}
		}
	default:
		return errors.New("unsupported type for JSON validation: " + of.Kind().String())
	}
	return nil
}

func (d *defaultValidator) Validator() any {
	d.lazyInit()
	return d.validator
}

func (d *defaultValidator) lazyInit() {
	d.once.Do(func() {
		d.validator = validator.New()
	})
}
