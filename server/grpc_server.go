package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"gitlab.com/gaydamakha/ter-grpc/messaging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	_ "google.golang.org/grpc/encoding/gzip"
)

type workerRequest struct {
	txtfn string
	err   error
}

type ServerGRPC struct {
	logger         zerolog.Logger
	server         *grpc.Server
	port           int
	certificate    string
	key            string
	chunkSize      int
	compress       bool
	workers        []workerClientGRPC
	workerCount    int
	workermtx      *sync.RWMutex
	nbWorkers      int
	incomingFolder string
	outgoingFolder string
	requests       map[string]chan workerRequest
	reqmtx         *sync.RWMutex
}

type ServerGRPCConfig struct {
	Certificate string
	Key         string
	Port        int
	ChunkSize   int
	Compress    bool
	AdWorkers   []string
}

func NewServerGRPC(cfg ServerGRPCConfig) (s ServerGRPC, err error) {
	s.logger = zerolog.New(os.Stdout).
		With().
		Timestamp().
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
	s.nbWorkers = len(cfg.AdWorkers)
	s.workerCount = 0
	s.incomingFolder = "/tmp/pdftotext/incoming/"
	s.outgoingFolder = "/tmp/pdftotext/outgoing/"
	s.workermtx = &sync.RWMutex{}
	s.reqmtx = &sync.RWMutex{}
	s.requests = make(map[string]chan workerRequest)

	if len(cfg.AdWorkers) == 0 {
		err = errors.Errorf("Workers addresses must be specified")
	}

	for _, adWorker := range cfg.AdWorkers {
		grpcWorkerClient, err := newWorkerClientGRPC(workerClientGRPCConfig{
			Address:         adWorker,
			ChunkSize:       s.chunkSize,
			RootCertificate: s.certificate,
			Compress:        s.compress,
		})
		if err != nil {
			//TODO: replace by return?
			panic(err)
		}
		s.logger.Info().Msg(fmt.Sprintf("Server successfully added %s as a worker", adWorker))
		s.workers = append(s.workers, grpcWorkerClient)
	}

	err = os.MkdirAll(s.incomingFolder, 0777)
	if err != nil {
		return
	}
	err = os.MkdirAll(s.outgoingFolder, 0777)
	if err != nil {
		return
	}

	s.logger.Info().Msg("Server successfully configured")

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

	s.logger.Info().Msg("Serving...")

	err = s.server.Serve(listener)
	if err != nil {
		err = errors.Wrapf(err, "errored listening for grpc connections")
		return
	}

	return
}

// UploadPdfAndGetText implements the UploadPdfAndGetText method of the PdftotextService
// interface which is responsible for receiving a stream of
// chunks that form a complete file.
func (s *ServerGRPC) UploadPdfAndGetText(stream messaging.PdftotextService_UploadPdfAndGetTextServer) (err error) {
	uuid := uuid.New().String()
	fn := s.incomingFolder + "pdftotext" + uuid + ".pdf"

	file, err := messaging.ReceiveFile(stream, fn)
	if err != nil {
		return
	}

	s.logger.Info().Msg("upload received: processing the text")
	txtfn := s.outgoingFolder + "pdftotext" + uuid+ ".txt"
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
	uuid := uuid.New().String()
	fn := s.incomingFolder + "pdftotext" + uuid + ".pdf"
	file, err := messaging.ReceiveFile(stream, fn)
	if err != nil {
		return
	}
	defer file.Close()

	s.logger.Info().Msg(fmt.Sprintf("%s: upload from client received", uuid))

	//fetch current worker number and update shared worker count
	s.workermtx.RLock()
	currentwrk := s.workerCount
	s.workermtx.RUnlock()

	s.workermtx.Lock()
	// Increment workerCount even if error occurs
	s.workerCount++
	// Come back to the first worker if it was the last
	s.workerCount %= s.nbWorkers
	s.workermtx.Unlock()

	reschan := make(chan workerRequest)
	go s.workers[currentwrk].PdfToTextFile(context.Background(), fn, s.outgoingFolder, reschan)

	s.reqmtx.Lock()
	s.requests[uuid] = reschan
	s.reqmtx.Unlock()

	stream.SendAndClose(&messaging.IdAndStatus{
		Uuid:    uuid,
		Message: "File is received and will be processed soon",
		Code:    messaging.StatusCode_Ok,
	})
	if err != nil {
		err = errors.Wrapf(err,
			"failed to send status code")
		s.logger.Error().Err(err)
		return
	}

	//Be clean.
	file.Close()
	if os.Remove(fn) != nil {
		err = errors.Wrapf(err,
			"failed to remove tmp pdf file")
		s.logger.Error().Err(err)
		return
	}

	return
}

// GetText implements GetText method of PdftotextService. It returns a text file in the form of stream,
// giving the id.
func (s *ServerGRPC) GetText(id *messaging.Id, stream messaging.PdftotextService_GetTextServer) (err error) {
	s.reqmtx.RLock()
	//TODO: check if key exists
	reqchan := s.requests[id.Uuid]
	s.reqmtx.RUnlock()

	//Wait for the client to finish the file processing and return the result filename
	result := <-reqchan
	err = result.err
	if err != nil {
		//TODO: maybe send a error code? to test
		s.logger.Error().Err(err)
		return
	}

	s.logger.Info().Msg(fmt.Sprintf("%s: sending a text..", id.Uuid))
	err = messaging.SendFile(stream, s.chunkSize, result.txtfn, true)
	if err != nil {
		s.logger.Error().Err(err)
		return
	}
	s.logger.Info().Msg(fmt.Sprintf("%s: text sent", id.Uuid))

	// If all is ok, delete this request from the map
	s.reqmtx.Lock()
	delete(s.requests, id.Uuid)
	s.reqmtx.Unlock()

	return

}

func (s *ServerGRPC) Close() {
	if s.server != nil {
		s.server.Stop()
	}
}
