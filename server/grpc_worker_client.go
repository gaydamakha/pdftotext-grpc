package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"gitlab.com/gaydamakha/ter-grpc/messaging"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	_ "google.golang.org/grpc/encoding/gzip"
)

type workerClientGRPC struct {
	logger    zerolog.Logger
	conn      *grpc.ClientConn
	client    messaging.PdftotextWorkerClient
	chunkSize int
}

type workerClientGRPCConfig struct {
	Address         string
	ChunkSize       int
	RootCertificate string
	Compress        bool
}

func newWorkerClientGRPC(cfg workerClientGRPCConfig) (c workerClientGRPC, err error) {
	var (
		grpcOpts  = []grpc.DialOption{}
		grpcCreds credentials.TransportCredentials
	)

	if cfg.Address == "" {
		err = errors.Errorf("address must be specified")
		return
	}

	if cfg.Compress {
		grpcOpts = append(grpcOpts,
			grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	}

	if cfg.RootCertificate != "" {
		grpcCreds, err = credentials.NewClientTLSFromFile(cfg.RootCertificate, "localhost")
		if err != nil {
			err = errors.Wrapf(err,
				"failed to create grpc tls client via root-cert %s",
				cfg.RootCertificate)
			return
		}

		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(grpcCreds))
	} else {
		grpcOpts = append(grpcOpts, grpc.WithInsecure())
	}

	switch {
	case cfg.ChunkSize == 0:
		err = errors.Errorf("ChunkSize must be specified")
		return
	case cfg.ChunkSize > (1 << 22):
		err = errors.Errorf("ChunkSize must be < than 4MB")
		return
	default:
		c.chunkSize = cfg.ChunkSize
	}

	c.logger = zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("from", fmt.Sprintf("worker_client %s", cfg.Address)).
		Logger()

	c.conn, err = grpc.Dial(cfg.Address, grpcOpts...)
	if err != nil {
		err = errors.Wrapf(err,
			"failed to start grpc connection with address %s",
			cfg.Address)
		return
	}

	c.client = messaging.NewPdftotextWorkerClient(c.conn)

	return
}

func (c *workerClientGRPC) PdfToTextFile(ctx context.Context, f string, dir string, reschan chan workerRequest) {
	var (
		status *messaging.TextAndStatus
		result workerRequest
	)

	result = workerRequest{}

	// Open a stream-based connection with the
	// gRPC server
	c.logger.Info().Msg("creating upload stream to worker...")

	stream, err := c.client.UploadPdfAndGetText(ctx)
	if err != nil {
		result.err = errors.Wrapf(err,
			"failed to create upload stream for file %s",
			f)
		reschan <- result
		return
	}
	defer stream.CloseSend()

	c.logger.Info().Msg("sending a file to worker...")

	err = messaging.SendFile(stream, c.chunkSize, f, false)
	if err != nil {
		result.err = err
		reschan <- result
		return
	}

	c.logger.Info().Msg("file sent to worker: receiving the result")

	status, err = stream.CloseAndRecv()
	if err != nil {
		result.err = errors.Wrapf(err,
			"failed to receive upstream status response")
		reschan <- result
		return
	}

	c.logger.Info().Msg("received!")

	if status.Code != messaging.StatusCode_Ok {
		result.err = errors.Errorf(
			"upload failed - msg: %s",
			status.Message)
		reschan <- result
		return
	}

	fn := filepath.Base(f)
	txtfn := dir + strings.TrimSuffix(fn, path.Ext(fn)) + ".txt"
	err = ioutil.WriteFile(txtfn, status.Text, 0644)
	if err != nil {
		result.err = errors.Wrapf(err,
			"failed to create result file %s",
			txtfn)
		reschan <- result
		return
	}

	result.txtfn = txtfn
	reschan <- result
	return
}

func (c *workerClientGRPC) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
