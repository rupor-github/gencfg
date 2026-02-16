package gencfg_test

import (
	"fmt"

	"github.com/rupor-github/gencfg"
)

func ExampleProcess() {
	input := []byte(`
name: hello
greeting: "{{ .Name }}"
`)
	output, err := gencfg.Process(input)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Print(string(output))
	// Output:
	// name: hello
	// greeting: "greeting"
}

func ExampleProcess_withArgument() {
	input := []byte(`
env: '{{ index .Arguments "env" }}'
`)
	output, err := gencfg.Process(input,
		gencfg.WithArgument("env", "production"),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Print(string(output))
	// Output:
	// env: 'production'
}

func ExampleProcess_withDoNotExpandField() {
	input := []byte(`
expand_me: "{{ .Name }}"
keep_me: "{{ .Name }}"
`)
	output, err := gencfg.Process(input,
		gencfg.WithDoNotExpandField("keep_me"),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Print(string(output))
	// Output:
	// expand_me: "expand_me"
	// keep_me: "{{ .Name }}"
}

func ExampleSanitize() {
	type Config struct {
		Path string `yaml:"path" sanitize:"path_clean"`
	}
	cfg := &Config{Path: "/some//messy/../path/./here"}
	if err := gencfg.Sanitize(cfg); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(cfg.Path)
	// Output:
	// /some/path/here
}

func ExampleValidate() {
	type Config struct {
		Host string `validate:"required"`
		Port int    `validate:"required,min=1,max=65535"`
	}
	cfg := &Config{Host: "localhost", Port: 8080}
	if err := gencfg.Validate(cfg); err != nil {
		fmt.Println("validation failed:", err)
		return
	}
	fmt.Println("valid")
	// Output:
	// valid
}

func ExampleValidate_error() {
	type Config struct {
		Host string `validate:"required"`
		Port int    `validate:"required,min=1,max=65535"`
	}
	cfg := &Config{Host: "", Port: 0}
	if err := gencfg.Validate(cfg); err != nil {
		fmt.Println("validation failed")
		return
	}
	fmt.Println("valid")
	// Output:
	// validation failed
}
