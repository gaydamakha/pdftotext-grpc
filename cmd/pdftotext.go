package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/urfave/cli/v2"
	"gitlab.com/gaydamakha/ter-grpc/client"
	"golang.org/x/net/context"
)

var PdfToText = cli.Command{
	Name:   "pdftotext",
	Usage:  "extracts text from pdf file",
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
			Usage: "file to transform",
		},
		&cli.StringFlag{
			Name:  "root-certificate",
			Usage: "path of a certificate to add to the root CAs",
		},
		&cli.BoolFlag{
			Name:  "compress",
			Usage: "whether or not to enable payload compression",
		},
		&cli.IntFlag{
			Name:  "iters",
			Usage: "number of times to transform the file (testing option)",
			Value: 1,
		},
	},
}

func uploadAction(c *cli.Context) (err error) {
	var (
		chunkSize          = c.Int("chunk-size")
		address            = c.String("address")
		file               = c.String("file")
		rootCertificate    = c.String("root-certificate")
		compress           = c.Bool("compress")
		iters              = c.Int("iters")
		statBegin, statEnd client.Stats
		clt                *client.ClientGRPC
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
	statBegin, err = clt.PdfToTextFile(context.Background(), file, "1")
	must(err)
	for i := 2; i < iters; i++ {
		_, err := clt.PdfToTextFile(context.Background(), file, strconv.Itoa(i))
		must(err)
	}
	if iters == 1 {
		statEnd = statBegin
	} else {
		statEnd, err = clt.PdfToTextFile(context.Background(), file, strconv.Itoa(iters))
		must(err)
	}
	defer clt.Close()

	fmt.Printf("Started at: %d\n", statBegin.StartedAt.UnixNano())
	fmt.Printf("Finished at: %d\n", statEnd.FinishedAt.UnixNano())
	fmt.Printf("Time elapsed: %d\n", statEnd.FinishedAt.Sub(statBegin.StartedAt).Nanoseconds())

	return
}
