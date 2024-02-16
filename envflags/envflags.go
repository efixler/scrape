package envflags

import (
	"encoding"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	EnvPrefix        = ""
	EnvUsageTemplate = "\nEnvironment: %s"
)

type Value[T any] struct {
	flagValue  T
	converter  func(string) (T, error)
	isBoolFlag bool
	envName    string
}

func NewEnvFlagValue[T any](
	envName string,
	defaultValue T,
	converter func(string) (T, error),
) *Value[T] {
	envFlag := &Value[T]{
		converter: converter,
		envName:   envName,
	}
	envFlag.setDefault(envName, defaultValue)
	return envFlag
}

func (p *Value[T]) AddTo(flags *flag.FlagSet, name, usage string) {
	usage += p.envUsage()
	flags.Var(p, name, usage)
}

func (p Value[T]) EnvName() string {
	if p.envName == "" {
		return ""
	}
	return EnvPrefix + p.envName
}

func (p Value[T]) envUsage() string {
	if p.envName == "" {
		return ""
	}
	return fmt.Sprintf(EnvUsageTemplate, p.EnvName())
}

func (p Value[T]) IsBoolFlag() bool {
	return p.isBoolFlag
}

func (p *Value[T]) setDefault(envName string, defaultValue T) {
	if env := os.Getenv(EnvPrefix + envName); env != "" {
		converted, err := p.converter(env)
		if err == nil {
			//defaultValue = converted
			p.flagValue = converted
			return
		} else {
			slog.Warn("error converting environment variable, ignoring", "env", EnvPrefix+envName, "error", err, "default", defaultValue)
		}
	}
	p.flagValue = defaultValue
}

func (p Value[T]) String() string {
	return fmt.Sprintf("%v", p.flagValue)
}

func (p Value[T]) Get() T {
	return p.flagValue
}

func (p *Value[T]) Set(value string) error {
	if p.converter == nil {
		return fmt.Errorf("no converter for type %T", p.flagValue)
	}
	converted, err := p.converter(value)
	if err != nil {
		return err
	}
	p.flagValue = converted
	return nil
}

func NewString(env, defaultValue string) *Value[string] {
	converter := func(s string) (string, error) {
		return s, nil
	}
	pflag := NewEnvFlagValue(env, defaultValue, converter)
	return pflag
}

func NewBool(env string, defaultValue bool) *Value[bool] {
	eflag := NewEnvFlagValue(env, defaultValue, strconv.ParseBool)
	eflag.isBoolFlag = true
	return eflag
}

func NewInt(env string, defaultValue int) *Value[int] {
	pflag := NewEnvFlagValue(env, defaultValue, strconv.Atoi)
	return pflag
}

func NewDuration(env string, defaultValue time.Duration) *Value[time.Duration] {
	pflag := NewEnvFlagValue(env, defaultValue, time.ParseDuration)
	return pflag
}

func NewLogLevel(env string, defaultValue slog.Level) *Value[slog.Level] {
	converter := func(s string) (slog.Level, error) {
		switch strings.ToLower(s) {
		case "debug":
			return slog.LevelDebug, nil
		case "info":
			return slog.LevelInfo, nil
		case "warn":
			return slog.LevelWarn, nil
		case "error":
			return slog.LevelError, nil
		default:
			return slog.LevelInfo, fmt.Errorf("invalid log level: %s", s)
		}
	}
	pflag := NewEnvFlagValue(env, defaultValue, converter)
	return pflag
}

func NewUint64(env string, defaultValue uint64) *Value[uint64] {
	converter := func(s string) (uint64, error) {
		return strconv.ParseUint(s, 10, strconv.IntSize)
	}
	pflag := NewEnvFlagValue(env, defaultValue, converter)
	return pflag
}

func NewText[S encoding.TextUnmarshaler](env string, defaultValue S) *Value[S] {
	// It seems not to be possible to create a new S without reflection.
	// If we don't create a new instance of S and use defaultValue as the UnmarshalText target,
	// defaultValue will be modified by the UnmarshalText call. This is only materially a problem
	// when the Unmarshal fails, because it can still update the thing's value.
	defVal := reflect.ValueOf(defaultValue)
	if defVal.Kind() == reflect.Ptr {
		defVal = defVal.Elem()
	}
	defType := defVal.Type()
	converter := func(s string) (S, error) {
		text := reflect.New(defType).Interface().(S)
		if s == "" {
			return text, fmt.Errorf("empty string for text value")
		}
		if err := text.UnmarshalText([]byte(s)); err != nil {
			return text, err
		}
		return text, nil
	}
	pflag := NewEnvFlagValue(env, defaultValue, converter)
	return pflag
}
