package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"

	cli "github.com/urfave/cli/v3"

	"github.com/rupor-github/gencfg"
	"github.com/rupor-github/gencfg/misc"
)

const errorCode = 1

func main() {

	app := &cli.Command{
		Name:    misc.AppName,
		Usage:   "generate configuration file from template",
		Version: misc.GetVersion() + " (" + runtime.Version() + ")",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "project-dir",
				Aliases: []string{"d"},
				Usage:   "Project directory to use for expansion (default is current directory)",
			},
			&cli.StringSliceFlag{
				Name:    "literal",
				Aliases: []string{"l"},
				Usage:   "Name of the field(s) not to be treated as template",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {

			tmplPath := cmd.Args().Get(0)
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

			options := make([]func(*gencfg.ProcessingOptions), 0, 16)
			options = append(options, gencfg.WithRootDir(cmd.String("project-dir")))
			for _, literal := range cmd.StringSlice("literal") {
				options = append(options, gencfg.WithDoNotExpandField(literal))
			}

			cnf, err := gencfg.Process(tmpl, options...)
			if err != nil {
				return cli.Exit(fmt.Errorf("unable to generate configuration: %w", err), errorCode)
			}
			cnfFile := os.Stdout
			cnfPath := cmd.Args().Get(1)
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

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
