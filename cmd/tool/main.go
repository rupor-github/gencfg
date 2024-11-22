package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/urfave/cli/v2"

	"github.com/rupor-github/gencfg"
	"github.com/rupor-github/gencfg/misc"
)

const errorCode = 1

func main() {

	cli.AppHelpTemplate = `
NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}TEMPLATE [DESTINATION]{{end}}
   {{if .VisibleFlags}}
OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
NOTE:
   when run with redirected Stdout tool would not ask for Vault credentials even if new token is needed.
`
	app := &cli.App{
		Name:    misc.AppName,
		Usage:   "generate configuration file from template",
		Version: misc.GetVersion() + " (" + runtime.Version() + ")",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "project-dir",
				Aliases: []string{"d"},
				Usage:   "Project directory to use for expansion (default is current directory)",
			},
		},
		Action: func(cCtx *cli.Context) error {

			tmplPath := cCtx.Args().Get(0)
			if len(tmplPath) == 0 {
				return cli.Exit(errors.New("no template file has been specified"), errorCode)
			}
			tmplPath, err := filepath.Abs(tmplPath)
			if err != nil {
				return cli.Exit(fmt.Errorf("normalizing template path failed: %w", err), errorCode)
			}
			tmpl, err := os.ReadFile(tmplPath)
			if err != nil {
				return cli.Exit(fmt.Errorf("unable to open template file: %w", err), errorCode)
			}
			if len(tmpl) == 0 {
				return cli.Exit(errors.New("template file is empty"), errorCode)
			}
			cnf, err := gencfg.Process(tmpl, gencfg.WithRootDir(cCtx.String("project-dir")))
			if err != nil {
				return cli.Exit(fmt.Errorf("unable to generate configuration: %w", err), errorCode)
			}
			cnfFile := os.Stdout
			cnfPath := cCtx.Args().Get(1)
			if len(cnfPath) != 0 {
				cnfFile, err = os.Create(cnfPath)
				if err != nil {
					return cli.Exit(fmt.Errorf("unable to create output file: %w", err), errorCode)
				}
				defer cnfFile.Close()
			}
			_, err = io.Copy(cnfFile, bytes.NewBuffer(cnf))
			if err != nil {
				return cli.Exit(fmt.Errorf("unable to write output file: %w", err), errorCode)
			}
			return nil
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
