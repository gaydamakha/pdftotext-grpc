package main

import (
	"os"

	"gitlab.com/gaydamakha/ter-grpc/cmd"
	"gopkg.in/urfave/cli.v2"
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