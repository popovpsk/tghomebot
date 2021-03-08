package utils

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
)

//ParseConfig fills config from env varibles
func ParseConfig(cfgPtr interface{}) error {
	ptrVal := reflect.ValueOf(cfgPtr)
	cfg := ptrVal.Elem()
	t := cfg.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := cfg.Field(i)
		if !fieldValue.CanSet() {
			fmt.Println(i)
			continue
		}
		envName, ok := getTagValue(&field.Tag, "env")
		if !ok {
			return fmt.Errorf("env tag is required")
		}
		env, ok := os.LookupEnv(envName)
		if !ok {
			return fmt.Errorf("environment variable %s is not set", envName)
		}
		parser, ok := defaultBuiltInParsers[field.Type.Kind()]
		if !ok {
			return fmt.Errorf("parser for env: %s not found", env)
		}
		val, err := parser(env)
		if err != nil {
			return err
		}

		fieldValue.Set(reflect.ValueOf(val))
	}
	return nil
}

func getTagValue(tag *reflect.StructTag, key string) (string, bool) {
	t := tag.Get(key)
	if t == "" {
		return "", false
	}
	return t, true
}

func getRequired(key string) (value string, err error) {

	var ok bool
	if value, ok = os.LookupEnv(key); ok {
		return
	}
	err = fmt.Errorf(`env: required environment variable "%q" is not set`, key)
	return
}

var defaultBuiltInParsers = map[reflect.Kind]func(v string) (interface{}, error){

	reflect.Bool: func(v string) (interface{}, error) {
		return strconv.ParseBool(v)
	},

	reflect.String: func(v string) (interface{}, error) {
		return v, nil
	},

	reflect.Int: func(v string) (interface{}, error) {
		i, err := strconv.ParseInt(v, 10, 32)
		return int(i), err
	},

	reflect.Int16: func(v string) (interface{}, error) {
		i, err := strconv.ParseInt(v, 10, 16)
		return int16(i), err
	},

	reflect.Int32: func(v string) (interface{}, error) {
		i, err := strconv.ParseInt(v, 10, 32)
		return int32(i), err
	},

	reflect.Int64: func(v string) (interface{}, error) {
		return strconv.ParseInt(v, 10, 64)
	},

	reflect.Int8: func(v string) (interface{}, error) {
		i, err := strconv.ParseInt(v, 10, 8)
		return int8(i), err
	},

	reflect.Uint: func(v string) (interface{}, error) {
		i, err := strconv.ParseUint(v, 10, 32)
		return uint(i), err
	},

	reflect.Uint16: func(v string) (interface{}, error) {
		i, err := strconv.ParseUint(v, 10, 16)
		return uint16(i), err
	},

	reflect.Uint32: func(v string) (interface{}, error) {
		i, err := strconv.ParseUint(v, 10, 32)
		return uint32(i), err
	},

	reflect.Uint64: func(v string) (interface{}, error) {
		i, err := strconv.ParseUint(v, 10, 64)
		return i, err
	},

	reflect.Uint8: func(v string) (interface{}, error) {
		i, err := strconv.ParseUint(v, 10, 8)
		return uint8(i), err
	},

	reflect.Float64: func(v string) (interface{}, error) {
		return strconv.ParseFloat(v, 64)
	},

	reflect.Float32: func(v string) (interface{}, error) {
		f, err := strconv.ParseFloat(v, 32)
		return float32(f), err
	},
}
