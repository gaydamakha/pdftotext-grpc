package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/urfave/cli/v2"
	"gitlab.com/gaydamakha/ter-grpc/client"
	"golang.org/x/net/context"
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
		&cli.StringFlag{
			Name:  "result-fn",
			Usage: "path to the result file",
			Value: "",
		},
	},
}

func pdftotextAction(c *cli.Context) (err error) {
	var (
		chunkSize          = c.Int("chunk-size")
		address            = c.String("address")
		file               = c.String("file")
		rootCertificate    = c.String("root-certificate")
		compress           = c.Bool("compress")
		iters              = c.Int("iters")
		resultfn           = c.String("result-fn")
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
	defer clt.Close()

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

	result := statEnd.FinishedAt.Sub(statBegin.StartedAt).String()
	if resultfn != "" {
		err = ioutil.WriteFile(resultfn, []byte(result), 0644)
	} else {
		fmt.Println(result)
	}

	return
}
