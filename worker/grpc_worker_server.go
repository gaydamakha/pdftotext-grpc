package worker

import (
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"gitlab.com/gaydamakha/ter-grpc/messaging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	_ "google.golang.org/grpc/encoding/gzip"
)

type WorkerServerGRPC struct {
	logger      zerolog.Logger
	server      *grpc.Server
	port        int
	certificate string
	key         string
	chunkSize   int
}

type WorkerServerGRPCConfig struct {
	Certificate string
	Key         string
	Port        int
	ChunkSize   int
}

func NewWorkerServerGRPC(cfg WorkerServerGRPCConfig) (s WorkerServerGRPC, err error) {
	s.logger = zerolog.New(os.Stdout).
		With().
		Str("from", "worker_server").
		Logger()

	if cfg.Port == 0 {
		err = errors.Errorf("Port must be specified")
		return
	}

	switch {
	case cfg.ChunkSize == 0:
		err = errors.Errorf("ChunkSize must be specified")
		return
	case cfg.ChunkSize > (1 << 22):
		err = errors.Errorf("ChunkSize must be < than 4MB")
		return
	default:
		s.chunkSize = cfg.ChunkSize
	}

	s.port = cfg.Port
	s.certificate = cfg.Certificate
	s.key = cfg.Key

	s.logger.Info().Msg("Worker server successfully configured...")

	return
}

func (s *WorkerServerGRPC) Listen() (err error) {
	var (
		listener  net.Listener
		grpcOpts  = []grpc.ServerOption{}
		grpcCreds credentials.TransportCredentials
	)

	listener, err = net.Listen("tcp", ":"+strconv.Itoa(s.port))
	if err != nil {
		err = errors.Wrapf(err,
			"failed to listen on port %d",
			s.port)
		return
	}

	if s.certificate != "" && s.key != "" {
		grpcCreds, err = credentials.NewServerTLSFromFile(
			s.certificate, s.key)
		if err != nil {
			err = errors.Wrapf(err,
				"failed to create tls grpc server using cert %s and key %s",
				s.certificate, s.key)
			return
		}

		grpcOpts = append(grpcOpts, grpc.Creds(grpcCreds))
	}

	s.server = grpc.NewServer(grpcOpts...)
	messaging.RegisterPdftotextWorkerServer(s.server, s)

	s.logger.Info().Msg("Serving...")

	err = s.server.Serve(listener)
	if err != nil {
		err = errors.Wrapf(err, "error listening for grpc connections")
		return
	}

	return
}

// UploadPdfAndGetText implements the UploadPdfAndGetText method of the PdftotextWorker
// interface which is responsible for receiving a stream of
// chunks that form a complete file.
func (s *WorkerServerGRPC) UploadPdfAndGetText(stream messaging.PdftotextWorker_UploadPdfAndGetTextServer) (err error) {
	fn := "pdftotext.pdf"
	s.logger.Info().Msg("receiving the upload...")

    file, err := messaging.ReceiveFile(stream, fn)
    if err != nil {
        return
    }

	s.logger.Info().Msg("upload received: processing the text")
	txtfn := "pdftotext.txt"
	_, err = exec.Command("pdftotext", fn, txtfn).Output()
	if err != nil {
		err = errors.Wrapf(err,
			"pdftotext didn't worked")
		return
	}

	// open recently created file
	txtfile, err := os.Open(txtfn)
	if err != nil {
		err = errors.Wrapf(err,
			"can't open result file")
		return
	}

	// read the result content
	text, err := ioutil.ReadAll(txtfile)
	if err != nil {
		err = errors.Wrapf(err,
			"can't read from result file")
		return
	}

	// once the transmission finished, send the
	// confirmation and the text if nothing went wrong
	err = stream.SendAndClose(&messaging.TextAndStatus{
		Message: "File received with success",
		Text:    text,
		Code:    messaging.StatusCode_Ok,
	})
	if err != nil {
		err = errors.Wrapf(err,
			"failed to send status code")
		return
	}

	//Be clean.
	file.Close()
	if os.Remove(fn) != nil {
		err = errors.Wrapf(err,
			"failed to remove tmp pdf file")
		return
	}
	txtfile.Close()
	if os.Remove(txtfn) != nil {
		err = errors.Wrapf(err,
			"failed to remove tmp txt file")
		return
	}

	return
}

func (s *WorkerServerGRPC) Close() {
	if s.server != nil {
		s.server.Stop()
	}
}
