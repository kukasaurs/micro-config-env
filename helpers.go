package env

import (
	"reflect"
	"time"
)

// helpers for time type and array.
func isDurationType(t reflect.Type) bool {
	return t == reflect.TypeFor[time.Duration]()
}

func isTimeType(t reflect.Type) bool {
	return t == reflect.TypeFor[time.Time]()
}

func isTimePtrType(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr && isTimeType(t.Elem())
}

func isZeroArray(v reflect.Value) bool {
    for i := 0; i < v.Len(); i++ {
        if !v.Index(i).IsZero() {
            return false
        }
    }
    return true
}
