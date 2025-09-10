package env

import (
	"reflect"
	"time"
)

// helpers for time type.
func isDurationType(t reflect.Type) bool {
	return t == reflect.TypeFor[time.Duration]()
}

func isTimeType(t reflect.Type) bool {
	return t == reflect.TypeFor[time.Time]()
}

func isTimePtrType(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr && isTimeType(t.Elem())
}
