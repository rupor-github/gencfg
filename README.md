
gencfg
========

Combination of importable module and a CLI tool on top of it to simplify dealing
with YAML configuration files in Go code.

Rather than using 3rd party packages to handle configuration today
it's preferable to use Go built in support for marshaling and struct tags.
Taking into account requirements of local development and testing it's difficult
to prevent logic of configuration from spreading across multiple source files
and sometimes modules. This module provides tooling to solve this problem to a
degree allowing most of the logic (including setting defaults) to be in the
configuration template (possibly embedded into resulting binary). 

It introduces an ability to use [Go Templating engine](https://golang.org/pkg/text/template/)
with added support for [slim-sprig lib](https://go-task.github.io/slim-sprig/)
and some "proprietary" extensions to derive "values" in the configuration. This
makes it possible to both generate configuration files and to create
configurations for production usage, development and testing in a coherent way
from single template.

## Template variables defined by project

	.Name (string) - name of the YAML node value is being assigned to
	.ProjectDir (string) - used to expand relative paths in configuration, could be passed in
	.Hostname (string) - Go's os.Hostname()
    .IPv4 (string) - IPv4 address of local host, not loopback address
	.Containerized (bool) - true if code is executed in container
    .Testing (bool) - true if expansion happens when code is being run with "go test"
    .CPUs (int) - Go's runtime.NumCPU()
    .OS (string) - Go's runtime.GOOS
    .ARCH (string) - Go's runtime.GOARCH
    .Agguments (map[strting]string) - could be passed to Process() using .WithArgument(name,value) calls

## Functions defined by project

    joinPath - Joins any number of arguments into a path. The same as Go's filepath.Join.
    freeLocalPort - takes no arguments, returns free unique local port to be used for testing. For running tests in parallel implementation keeps global port map.

## Command line tool

    ❯ gencfg -h

    NAME:
       gencnf - generate configuration file from template

    USAGE:
       gencnf [options] TEMPLATE [DESTINATION]

    OPTIONS:
       --project-dir value, -d value  Project directory to use for expansion (default is current directory)
       --help, -h                     show help (default: false)
       --version, -v                  print the version (default: false)

## Some examples

Reading database user/password from environment and if not set from Vault:

    db:
        username: '{{ default "user" (env "DB_USERNAME") }}'
        password: '{{ default "pass" (env "DB_PASSWORD") }}'

Setting parameters for logging from environment:

    logging:
        level: info
        # do not use "log timestamps" when running inside docker, rely on journald and docker logs to maintain timestamps
        use_timestamp: "{{ not .Containerized }}"

## Additional functionality provided by `gencfg` importable module

Since application configuration structures defined in Go by creating struct
types with declarative tags it makes sense to build on that as much as
possible. Presently this module offers two additional declarative capabilities:
"Sanitize" and "Validate" which used similarly to how L tags assigned to
configuration struct fields.

## Sanitizing configuration values

`gencfg` module has additional capability of sanitizing configuration values.
You can set "sanitize" tag on a configuration field and call Sanitize() function
after your configuration unmarshaled into the struct in your code.

    type SomeConfig struct {
        WorkerPoolSize uint32 `yaml:"worker_pool_size"`
        TempDir        string `yaml:"temp_dir" sanitize:"path_clean"`
    }

    cfg := &SomeConfig{}
    // Code to initialize cfg structure goes here (read from file, unmarshal, set...)
    .........

    // sanitize will call filepath.Clean() on TempDir value and assign the resulting cleaned path back to the TempDir field.
	if err := gencfg.Sanitize(&cfg); err != nil {
		// Prossing sanitization errors here
        .........
	}

Recognized actions called on fields with tags so path will be reset if
necessary. Sanitize supports comma-separated list of actions and calls them in
definition order.

Presently defined actions:

    path_clean - same as calling filepath.Clean(value) on the configuration field
    path_toslash - same as calling filepath.ToSlash(value) on the configuration field
    path_abs - same as calling filepath.Abs(value) on the configuration field
    assure_dir_exists - will call os.MkdirAll(value), not changing field itself
    assure_dir_exists_for_file - will call os.MkdirAll(filepath.Dir(value)), not changing field itself

## Validating configuration values

`gencfg` module has additional capability of validating configuration values
using code from [validator](https://pkg.go.dev/github.com/go-playground/validator/v10) project.

Note, that when supported tags aren't sufficient you could pass in custom
function to perform additional checks.

    type SomeConfig struct {
        WorkerPoolSize uint32 `yaml:"worker_pool_size" validate:"required"`
        TempDir        string `yaml:"temp_dir" sanitize:"path_clean,assure_dir_exists" validate:"required,gt=1"`
    }

    // To perform "unusual" specific checking, most likely involves cross-field, cross embedded stuctures validation
    func additionalChecks(sl validator.StructLevel) {

        c := sl.Current().Interface().(SomeConfig)

        if WorkerPoolSize == 999 {
            sl.ReportError(c.WorkerPoolSize, "WorkerPoolSize", "", "do not like size must be 666", "")
        }
        ..........
    }
    ..........

    cfg := &SomeConfig{}
    // Code to initialize cfg structure goes here (read from file, unmarshal, set...)
    .........

	if err := gencfg.Sanitize(&cfg); err != nil {
		// Processing sanitization errors here
        .........
	}
	if err := gencfg.Validate(&cfg, gencfg.WithAdditionalChecks(additionalChecks)); err != nil {
		// Processing validation errors here
        .........
	}

Validate() returns all found problems at once, it would not fail on each
consecutive violation encountered. Please, read
[documentation](https://pkg.go.dev/github.com/go-playground/validator/v10#readme-baked-in-validations)
for more details on available checks.
