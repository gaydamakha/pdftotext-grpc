package server

import (
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

type ServerGRPC struct {
	logger      zerolog.Logger
	server      *grpc.Server
	port        int
	certificate string
	key         string
	chunkSize   int
	compress    bool
}

type ServerGRPCConfig struct {
	Certificate string
	Key         string
	Port        int
	ChunkSize   int
	Compress    bool
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
	s.compress = cfg.Compress

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

	//TODO: check if compression is possible

	// if s.compress {
	// 	grpcOpts = append(grpcOpts,
	// 		grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	// }

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

	//open recently created file
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

//UploadPdf implements UploadPdf method of PdftotextService. It receives a pdf file in the form of stream,
// transforms it into the pdf file and returns an ID of the file.
func (s *ServerGRPC) UploadPdf(stream messaging.PdftotextService_UploadPdfServer) (err error) {
	// while there are messages coming
	fn := "pdftotext.pdf"
    file, err := messaging.ReceiveFile(stream, fn)
    if err != nil {
        return
    }
    defer file.Close()

    uuid := uuid.New().String()

	s.logger.Info().Msg("upload received")
	txtfn := "pdftotext" + uuid + ".txt"
	_, err = exec.Command("pdftotext", fn, txtfn).Output()
	if err != nil {
		err = errors.Wrapf(err,
			"pdftotext didn't worked")
		return
	}

	stream.SendAndClose(&messaging.IdAndStatus{
		Uuid:    uuid,
		Message: "File received and processed with success",
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

	return
}

// GetText implements GetText method of PdftotextService. It returns a text file in the form of stream,
// giving the id.
func (s *ServerGRPC) GetText(id *messaging.Id, stream messaging.PdftotextService_GetTextServer) (err error) {
    txtfn := "pdftotext" + id.Uuid + ".txt"
	//TODO: maybe send a error code? to test
    err = messaging.SendFile(stream, s.chunkSize, txtfn, true)
    if err != nil {
        return
    }
	s.logger.Info().Msg("text sent")

	return
}

func (s *ServerGRPC) Close() {
	if s.server != nil {
		s.server.Stop()
	}
}
