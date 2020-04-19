package cmd

import (
	"strings"

	"github.com/urfave/cli/v2"
	"gitlab.com/gaydamakha/ter-grpc/server"
)

var Serve = cli.Command{
	Name:   "serve",
	Usage:  "initiates a gRPC server",
	Action: serveAction,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "workers",
			Usage:    "IP addresses of workers",
			Required: true,
		},
		&cli.IntFlag{
			Name:  "port",
			Usage: "port to bind to",
			Value: 1313,
		},
		&cli.IntFlag{
			Name:  "chunk-size",
			Usage: "size of the chunk messages",
			Value: (1 << 12),
		},
		&cli.StringFlag{
			Name:  "key",
			Usage: "path to TLS certificate",
		},
		&cli.StringFlag{
			Name:  "certificate",
			Usage: "path to TLS certificate",
		},
		&cli.BoolFlag{
			Name:  "compress",
			Usage: "whether or not to enable payload compression",
		},
	},
}

func serveAction(c *cli.Context) (err error) {
	var (
		port        = c.Int("port")
		key         = c.String("key")
		certificate = c.String("certificate")
		compress    = c.Bool("compress")
		chunkSize   = c.Int("chunk-size")
		adWorkers   = strings.Fields(c.String("workers"))
		srv         *server.ServerGRPC
	)

	grpcServer, err := server.NewServerGRPC(server.ServerGRPCConfig{
		Port:        port,
		Certificate: certificate,
		Key:         key,
		ChunkSize:   chunkSize,
		AdWorkers:   adWorkers,
		Compress:    compress,
	})
	must(err)
	srv = &grpcServer

	err = srv.Listen()
	must(err)
	defer srv.Close()

	return
}
