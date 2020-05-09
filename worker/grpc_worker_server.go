package worker

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"

	"github.com/google/uuid"
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
}

type WorkerServerGRPCConfig struct {
	Certificate string
	Key         string
	Port        int
}

func NewWorkerServerGRPC(cfg WorkerServerGRPCConfig) (s WorkerServerGRPC, err error) {
	s.logger = zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("from", "worker_server").
		Logger()

	if cfg.Port == 0 {
		err = errors.Errorf("Port must be specified")
		s.logger.Error().Err(err)
		return
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
		s.logger.Error().Err(err)
		return
	}

	if s.certificate != "" && s.key != "" {
		grpcCreds, err = credentials.NewServerTLSFromFile(
			s.certificate, s.key)
		if err != nil {
			err = errors.Wrapf(err,
				"failed to create tls grpc server using cert %s and key %s",
				s.certificate, s.key)
			s.logger.Error().Err(err)
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
		s.logger.Error().Err(err)
		return
	}

	return
}

// UploadPdfAndGetText implements the UploadPdfAndGetText method of the PdftotextWorker
// interface which is responsible for receiving a stream of
// chunks that form a complete file.
func (s *WorkerServerGRPC) UploadPdfAndGetText(stream messaging.PdftotextWorker_UploadPdfAndGetTextServer) (err error) {
	uuid := uuid.New().String()
	fn := "pdftotext" + uuid + ".pdf"
	s.logger.Info().Msg(fmt.Sprintf("%s: receiving the upload...", uuid))

	file, err := messaging.ReceiveFile(stream, fn)
	if err != nil {
		return
	}

	s.logger.Info().Msg(fmt.Sprintf("%s: upload received: processing the text", uuid))
	txtfn := "pdftotext" + uuid + ".txt"
	_, err = exec.Command("pdftotext", fn, txtfn).Output()
	if err != nil {
		err = errors.Wrapf(err,
			"pdftotext didn't worked")
		s.logger.Error().Err(err)
		return
	}

	// open recently created file
	txtfile, err := os.Open(txtfn)
	if err != nil {
		err = errors.Wrapf(err,
			"can't open result file")
		s.logger.Error().Err(err)
		return
	}

	// read the result content
	text, err := ioutil.ReadAll(txtfile)
	if err != nil {
		err = errors.Wrapf(err,
			"can't read from result file")
		return
	}

	s.logger.Info().Msg(fmt.Sprintf("%s: file processed: sending the file", uuid))

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
		s.logger.Error().Err(err)
		return
	}

	s.logger.Info().Msg(fmt.Sprintf("%s: file sent", uuid))

	//Be clean.
	file.Close()
	if os.Remove(fn) != nil {
		err = errors.Wrapf(err,
			"failed to remove tmp pdf file")
		s.logger.Error().Err(err)
		return
	}
	txtfile.Close()
	if os.Remove(txtfn) != nil {
		err = errors.Wrapf(err,
			"failed to remove tmp txt file")
		s.logger.Error().Err(err)
		return
	}

	return
}

func (s *WorkerServerGRPC) Close() {
	if s.server != nil {
		s.server.Stop()
	}
}
