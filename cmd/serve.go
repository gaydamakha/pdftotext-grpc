package cmd

import (
	"github.com/urfave/cli/v2"
	"gitlab.com/gaydamakha/ter-grpc/server"
)

var Serve = cli.Command{
	Name:   "serve",
	Usage:  "initiates a gRPC server",
	Action: serveAction,
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "port",
			Usage: "port to bind to",
			Value: 1313,
		},
		&cli.StringFlag{
			Name:  "key",
			Usage: "path to TLS certificate",
		},
		&cli.StringFlag{
			Name:  "certificate",
			Usage: "path to TLS certificate",
		},
	},
}

func serveAction(c *cli.Context) (err error) {
	var (
		port        = c.Int("port")
		key         = c.String("key")
		certificate = c.String("certificate")
		srv         *server.ServerGRPC
	)

	grpcServer, err := server.NewServerGRPC(server.ServerGRPCConfig{
		Port:        port,
		Certificate: certificate,
		Key:         key,
	})
	must(err)
	srv = &grpcServer

	err = srv.Listen()
	must(err)
	defer srv.Close()

	return
}
