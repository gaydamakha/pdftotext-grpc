package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"gitlab.com/gaydamakha/ter-grpc/cmd"
)

func main() {
	app := &cli.App{
		Name:  "ter",
		Usage: "I dunno for a while",
		Commands: []*cli.Command{
			&cmd.Serve,
			&cmd.Upload,
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "enables debug logging",
			},
		},
	}

	app.Run(os.Args)
}
