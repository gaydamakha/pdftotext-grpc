package client

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"gitlab.com/gaydamakha/ter-grpc/messaging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	_ "google.golang.org/grpc/encoding/gzip"
)

// ClientGRPC provides the implementation of a file
// uploader that streams chunks via protobuf-encoded
// messages.
type ClientGRPC struct {
	logger    zerolog.Logger
	conn      *grpc.ClientConn
	client    messaging.PdftotextServiceClient
	chunkSize int
	txtDir    string
	nbCalls   int64
	nbcmtx    *sync.RWMutex
}

type ClientGRPCConfig struct {
	Address         string
	ChunkSize       int
	RootCertificate string
	Compress        bool
	TxtDir          string
}

func NewClientGRPC(cfg ClientGRPCConfig) (c ClientGRPC, err error) {
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

	if cfg.TxtDir != "" {
		c.txtDir = cfg.TxtDir
		err = os.MkdirAll(c.txtDir, 0777)
		if err != nil {
			err = errors.Wrapf(err,
				"failed to created directory to store result files")
		}
	}

	c.logger = zerolog.New(os.Stdout).
		With().
		Str("from", "client").
		Logger()

	c.conn, err = grpc.Dial(cfg.Address, grpcOpts...)
	if err != nil {
		err = errors.Wrapf(err,
			"failed to start grpc connection with address %s",
			cfg.Address)
		return
	}

	c.client = messaging.NewPdftotextServiceClient(c.conn)

	c.nbCalls = 0
	c.nbcmtx = &sync.RWMutex{}

	return
}

func (c *ClientGRPC) PdfToTextFile(ctx context.Context, f string) (err error) {
	var (
		status *messaging.TextAndStatus
	)
	c.nbcmtx.Lock()
	c.nbCalls++
	i := strconv.Itoa(int(c.nbCalls))
	c.nbcmtx.Unlock()
	// Open a stream-based connection with the
	// gRPC server
	stream, err := c.client.UploadPdfAndGetText(ctx)
	if err != nil {
		err = errors.Wrapf(err,
			"failed to create upload stream for file %s",
			f)
		return
	}
	defer stream.CloseSend()

	err = messaging.SendFile(stream, c.chunkSize, f, false)
	if err != nil {
		return
	}

	status, err = stream.CloseAndRecv()
	if err != nil {
		err = errors.Wrapf(err,
			"failed to receive upstream status response")
		return
	}

	if status.Code != messaging.StatusCode_Ok {
		err = errors.Errorf(
			"upload failed - msg: %s",
			status.Message)
		return
	}

	fn := filepath.Base(f)
	txtfn := c.txtDir + strings.TrimSuffix(fn, path.Ext(fn)) + i + ".txt"
	err = ioutil.WriteFile(txtfn, status.Text, 0644)
	if err != nil {
		err = errors.Wrapf(err,
			"failed to create result file %s",
			txtfn)
		return
	}

	return
}

func (c *ClientGRPC) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *ClientGRPC) PdfToTextFileBi(ctx context.Context, f string) (err error) {
	var (
		status *messaging.IdAndStatus
	)
	c.nbcmtx.Lock()
	c.nbCalls++
	i := strconv.Itoa(int(c.nbCalls))
	c.nbcmtx.Unlock()

	// Open a stream-based connection with the
	// gRPC server
	stream, err := c.client.UploadPdf(ctx)
	if err != nil {
		err = errors.Wrapf(err,
			"failed to create upload stream for file %s",
			f)
		return
	}
	defer stream.CloseSend()

	err = messaging.SendFile(stream, c.chunkSize, f, false)
	if err != nil {
		return
	}

	status, err = stream.CloseAndRecv()
	if err != nil {
		err = errors.Wrapf(err,
			"failed to receive upstream status response")
		return
	}

	if status.Code != messaging.StatusCode_Ok {
		err = errors.Errorf(
			"upload failed - msg: %s",
			status.Message)
		return
	}

	downloadStream, err := c.client.GetText(ctx, &messaging.Id{
		Uuid: status.Uuid,
	})
	if err != nil {
		err = errors.Wrapf(err,
			"failed to create download stream for file %s",
			f)
		return
	}

	fn := filepath.Base(f)
	txtfn := c.txtDir + strings.TrimSuffix(fn, path.Ext(fn)) + i + ".txt"
	txtfile, err := messaging.ReceiveFile(downloadStream, txtfn)
	defer txtfile.Close()
	if err != nil {
		return
	}

	return
}
