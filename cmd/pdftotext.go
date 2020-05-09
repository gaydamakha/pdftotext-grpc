package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/urfave/cli/v2"
	"gitlab.com/gaydamakha/ter-grpc/client"
)

var PdfToText = cli.Command{
	Name:   "pdftotext",
	Usage:  "extracts text from pdf file",
	Action: pdftotextAction,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "address",
			Value: "localhost:1313",
			Usage: "address of the server to connect to",
		},
		&cli.IntFlag{
			Name:  "chunk-size",
			Usage: "size of the chunk messages",
			Value: (1 << 12),
		},
		&cli.StringFlag{
			Name:  "file",
			Usage: "file to transform",
		},
		&cli.StringFlag{
			Name:  "result-fn",
			Usage: "path to the metrics result file",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "txt-dir",
			Usage: "directiry to store returned text files",
			Value: "./",
		},
		&cli.StringFlag{
			Name:  "root-certificate",
			Usage: "path of a certificate to add to the root CAs",
		},
		&cli.BoolFlag{
			Name:  "compress",
			Usage: "whether or not to enable payload compression",
		},
		&cli.BoolFlag{
			Name:  "bidirectional",
			Usage: "whether or not to enable bidirectional communication",
		},
		&cli.IntFlag{
			Name:  "iters",
			Usage: "number of times to transform the file (testing option)",
			Value: 1,
		},
	},
}

func pdftotextAction(c *cli.Context) (err error) {
	var (
		chunkSize       = c.Int("chunk-size")
		address         = c.String("address")
		file            = c.String("file")
		rootCertificate = c.String("root-certificate")
		compress        = c.Bool("compress")
		iters           = c.Int("iters")
		txtDir          = c.String("txt-dir")
		resultfn        = c.String("result-fn")
		bi              = c.Bool("bidirectional")
		stats           client.Stats
		clt             *client.ClientGRPC
		errg            *errgroup.Group
	)

	errg, _ = errgroup.WithContext(context.Background())

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
		TxtDir:          txtDir,
	})
	must(err)
	clt = &grpcClient
	defer clt.Close()

	// Here the "iters" goroutines are launched to simulate a simultaneous connection of multiple clients
	stats.StartedAt = time.Now()
	if bi {
		// The file will be processed by some of the worker
		for i := 1; i <= iters; i++ {
			errg.Go(func() error {
				return clt.PdfToTextFileBi(context.Background(), file)
			})
		}
	} else {
		for i := 1; i <= iters; i++ {
			errg.Go(func() error {
				return clt.PdfToTextFile(context.Background(), file)
			})
		}
	}

	//Wait for others goroutines to finish or for a error (if any occurs)
	err = errg.Wait()
	if err != nil {
		must(err)
	}
	stats.FinishedAt = time.Now()
	result := stats.FinishedAt.Sub(stats.StartedAt).Seconds()
	if resultfn != "" {
		err = ioutil.WriteFile(resultfn, []byte(fmt.Sprintf("%f", result)), 0644)
	} else {
		fmt.Println(result)
	}

	return
}
