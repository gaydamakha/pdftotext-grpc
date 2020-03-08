package server

import (
	"io"
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

type ServerGRPC struct {
	logger      zerolog.Logger
	server      *grpc.Server
	port        int
	certificate string
	key         string
}

type ServerGRPCConfig struct {
	Certificate string
	Key         string
	Port        int
}

// var filename string

func NewServerGRPC(cfg ServerGRPCConfig) (s ServerGRPC, err error) {
	s.logger = zerolog.New(os.Stdout).
		With().
		Str("from", "server").
		Logger()

	if cfg.Port == 0 {
		err = errors.Errorf("Port must be specified")
		return
	}

	s.port = cfg.Port
	s.certificate = cfg.Certificate
	s.key = cfg.Key

	return
}

func (s *ServerGRPC) Listen() (err error) {
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
	messaging.RegisterPdftotextServiceServer(s.server, s)

	err = s.server.Serve(listener)
	if err != nil {
		err = errors.Wrapf(err, "errored listening for grpc connections")
		return
	}

	return
}

// UploadPdfAndGetTextt implements the UploadPdfAndGetText method of the PdftotextService
// interface which is responsible for receiving a stream of
// chunks that form a complete file.
func (s *ServerGRPC) UploadPdfAndGetText(stream messaging.PdftotextService_UploadPdfAndGetTextServer) (err error) {
	// while there are messages coming
	fn := "pdftotext.pdf"
	file, err := os.Create(fn)
	if err != nil {
		err = errors.Wrapf(err,
			"failed to create file %s",
			fn)
		return
	}

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				goto END
			}

			return errors.Wrapf(err,
				"failed unexpectadely while reading chunks from stream")
		}
		_, err = file.Write(chunk.Content)
		if err != nil {
			return errors.Wrapf(err,
				"failed to write into file %s",
				fn)
		}
	}

END:
	s.logger.Info().Msg("upload received")
	txtfn := "pdftotext.txt"
	_, err = exec.Command("pdftotext", fn, txtfn).Output()
	if err != nil {
		err = errors.Wrapf(err,
			"pdftotext didn't worked")
		return
	}

	//open recently created file
	//pdftotext.txt is the default filename
	txtfile, err := os.Open("pdftotext.txt")
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

func (s *ServerGRPC) Close() {
	if s.server != nil {
		s.server.Stop()
	}

	return
}

func (s *ServerGRPC) UploadPdf(stream messaging.PdftotextService_UploadPdfServer) (err error) {
	return
}

func (s *ServerGRPC) GetText(id *messaging.Id, stream messaging.PdftotextService_GetTextServer) (err error) {
	return
}
