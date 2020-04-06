package messaging

import (
	"io"
	"os"

	"github.com/pkg/errors"
)

type ChunkReceiver interface {
    Recv() (*Chunk, error)
}

type ChunkSender interface {
    Send(*Chunk) error
}

//This function receives a file by reading the stream.
//As a pointer to the file is returned, it's up to the caller to remove/close this file.
func ReceiveFile(stream ChunkReceiver, filename string) (file *os.File, err error) {
    //TODO: create a directory if not exist
    file, err = os.Create(filename)
	if err != nil {
		err = errors.Wrapf(err,
			"failed to create file %s",
			filename)
		return nil, err
	}

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return file, nil
			}

			return nil, errors.Wrapf(err,
				"failed unexpectadely while reading chunks from stream")
		}
		_, err = file.Write(chunk.Content)
		if err != nil {
			return nil, errors.Wrapf(err,
				"failed to write into file %s",
				filename)
		}
	}
}

//This function sends a file by stream. If file needs to be removed,
//the toremove parameter should be set to true
func SendFile(
    stream ChunkSender,
    chunkSize int,
    filename string,
    toremove bool) (err error) {
    var (
        writing = true
        buf     []byte
        n       int
        file    *os.File
    )
    // Get a file handle for the file we want to process
	file, err = os.Open(filename)
	if err != nil {
		err = errors.Wrapf(err,
			"failed to open file %s",
			filename)
		return
	}

    // Allocate a buffer with `chunkSize` as the capacity
    // and length (making a 0 array of the size of `chunkSize`)
	buf = make([]byte, chunkSize)
	for writing {
		// put as many bytes as `chunkSize` into the
		// buf array.
		n, err = file.Read(buf)
		if err != nil {
			if err == io.EOF {
				writing = false
				err = nil
				continue
			}

			err = errors.Wrapf(err,
				"errored while copying from file to buf")
			return
		}

		err = stream.Send(&Chunk{
			// because we might've read less than
			// `chunkSize` we want to only send up to
			// `n` (amount of bytes read).
			// note: slicing (`:n`) won't copy the
			// underlying data, so this as fast as taking
			// a "pointer" to the underlying storage.
			Content: buf[:n],
		})
		if err != nil {
			err = errors.Wrapf(err,
				"failed to send chunk via stream")
			return
		}
	}

    file.Close()
    if toremove {
	    if os.Remove(filename) != nil {
            err = errors.Wrapf(err,
            "failed to remove tmp file")
	        return
	    }
    }

    return
}

