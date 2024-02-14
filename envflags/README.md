# envflags

## Description
`envflags` aims to make command line and configuration a little easier.
- Back command line flags up wih environment variables and defaults
- Use any standard or custom type as a flag/config value
- Meant to work hand-in-hand with `flag`
- Built-in support for all `flags` supported types with some useful extras like `slog.Level` control.
- Easy to add support for custom types with a few lines of code.

## Quickstart

```
import "github.com/efixler/envflags"

var flags flag.FlagSet 

envflags.EnvPrefix = "MYAPP_"

portValue := envflags.NewInt("PORT", 8080)
flags.Var(portValue, "port", "The port to use")

logLevel := envflags.NewLogLevel("LOG_LEVEL", slog.LevelWarn)
flags.Var(logLevel, "log-level", "Set the log level [debug|error|info|warn]")

flags.Parse(os.Args[1:])

// .Get() returns the typed resolved value, an int in the first case
server.Port = portValue.Get()

var level slog.Level
// and here it's a slog.Level
level = logLevel.Get()
```

### Adding Environment Variable Names to Usage
`envars` provides a utility function that will add environment variable specs to usage
entries, while also adding a flag to a `flags.FlagSet`. 

Instead of calling `flags.Var()` as in the example above, do:
```
portValue.AddTo(&flags, "port", "The port to use")
```

The output of `myapp -h` will then include something like this:

```
...
  -port
      The port to use
      Environment: MYAPP_PORT (default 8080)
```

## Custom Types

To map a flag/environment variable to a custom type you just need to:

1. Write a function to convert a string into your custom type
2. Write a function to instantiate the `envflag.Value` for that type  

You can implement (1) as an anonymous function in the context of (2)

```
type MyType struct {
    Prefix string
    Suffix string
}

func NewMyType(env string, defaultValue MyType) *envflags.Value[MyType] {
    converter := func(s string) (MyType, error) {
        value := MyType{}
        splits := strings.SplitAfterN(s, ":", 2)
        if len(splits) < 2 {
            return value, fmt.Errorf("invalid input %q", s)
        }
        value.Prefix = matches[0]
        value.Suffix = matches[1]
        return value
    }
    return envflags.NewEnvFlag(env, defaultValue, converter)
}
```

Implement `fmt.Stringer` on your custom type so it shows up properly when `flags`
displays defaults.

## Hints and Details

Pass a value of `""` as the `env` to ignore the environment and just use command-line flags.

## Bugs and Suggestions?

Open an issue in this repo! Feedback welcome.
