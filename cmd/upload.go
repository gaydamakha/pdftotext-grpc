package cmd

import (
	"errors"
	"fmt"

	"gitlab.com/gaydamakha/ter-grpc/client"
	"golang.org/x/net/context"
	"gopkg.in/urfave/cli.v2"
)

var Upload = cli.Command{
	Name:   "upload",
	Usage:  "uploads a file",
	Action: uploadAction,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "address",
			Value: "localhost:1313",
			Usage: "address of the server to connect to",
		},
		&cli.IntFlag{
			Name:  "chunk-size",
			Usage: "size of the chunk messages (grpc only)",
			Value: (1 << 12),
		},
		&cli.StringFlag{
			Name:  "file",
			Usage: "file to upload",
		},
		&cli.StringFlag{
			Name:  "root-certificate",
			Usage: "path of a certificate to add to the root CAs",
		},
		&cli.BoolFlag{
			Name:  "compress",
			Usage: "whether or not to enable payload compression",
		},
	},
}

func uploadAction(c *cli.Context) (err error) {
	var (
		chunkSize       = c.Int("chunk-size")
		address         = c.String("address")
		file            = c.String("file")
		rootCertificate = c.String("root-certificate")
		compress        = c.Bool("compress")
		clt             *client.ClientGRPC
	)

	if address == "" {
		must(errors.New("address"))
	}

	if file == "" {
		must(errors.New("file must be set"))
	}

	grpcClient, err := client.NewClientGRPC(client.ClientGRPCConfig{
		Address:         address,
		RootCertificate: rootCertificate,
		Compress:        compress,
		ChunkSize:       chunkSize,
	})
	must(err)
	clt = &grpcClient

	stat, err := clt.UploadFile(context.Background(), file)
	must(err)
	defer clt.Close()

	fmt.Printf("Started at: %d\n", stat.StartedAt.UnixNano())
	fmt.Printf("Finished at: %d\n", stat.FinishedAt.UnixNano())
	fmt.Printf("Time elapsed: %d\n", stat.FinishedAt.Sub(stat.StartedAt).Nanoseconds())

	return
}
