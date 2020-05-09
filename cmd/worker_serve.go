package cmd

import (
	"github.com/urfave/cli/v2"
	"gitlab.com/gaydamakha/ter-grpc/worker"
)

var WorkerServe = cli.Command{
	Name:   "worker-serve",
	Usage:  "initiates a gRPC server",
	Action: workerServeAction,
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

func workerServeAction(c *cli.Context) (err error) {
	var (
		port        = c.Int("port")
		key         = c.String("key")
		certificate = c.String("certificate")
		wrk         *worker.WorkerServerGRPC
	)

	grpcWorkerServer, err := worker.NewWorkerServerGRPC(worker.WorkerServerGRPCConfig{
		Port:        port,
		Certificate: certificate,
		Key:         key,
	})
	must(err)
	wrk = &grpcWorkerServer

	err = wrk.Listen()
	must(err)
	defer wrk.Close()

	return
}
