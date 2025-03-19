// Package file implements a file service.
package file

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/go-orb/go-orb/util/metadata"
	orberrors "github.com/go-orb/go-orb/util/orberrors"
	file "github.com/go-orb/plugins/client/tests/proto/file"
	"github.com/google/uuid"
)

// Handler is the implementation of the file service.
type Handler struct {
}

// streamReceiver defines a generic interface for file streams.
type streamReceiver interface {
	Recv() (*file.FileChunk, error)
	Context() context.Context
}

// UploadFile implements the ORB interface for FileServiceHandler.
func (c *Handler) UploadFile(stream file.FileServiceUploadFileStream) error {
	return c.handleUploadFile(stream, func(totalSize int64, _ context.Context) error {
		// If we have received chunks, consider the upload complete and send response
		resp := &file.UploadResponse{
			Id:      uuid.New().String(),
			Success: true,
			Size:    totalSize,
		}

		return stream.CloseSend(resp)
	})
}

// AuthorizedUploadFile implements the ORB interface for FileServiceHandler.
func (c *Handler) AuthorizedUploadFile(stream file.FileServiceAuthorizedUploadFileStream) error {
	// Check for authorization metadata.
	ctx := stream.Context()
	md, ok := metadata.Incoming(ctx)

	if !ok || md["authorization"] != "Bearer pleaseHackMe" {
		return orberrors.ErrUnauthorized
	}

	return c.handleUploadFile(stream, func(totalSize int64, _ context.Context) error {
		// Add metadata to response.
		_, mdout := metadata.WithOutgoing(ctx)
		mdout["bytes-received"] = "true"
		mdout["total-size"] = "completed"

		// If we have received chunks, consider the upload complete and send response
		resp := &file.UploadResponse{
			Id:      uuid.New().String(),
			Success: true,
			Size:    totalSize,
		}

		return stream.CloseSend(resp)
	})
}

// handleUploadFile is a generic handler for processing file uploads.
func (c *Handler) handleUploadFile(stream streamReceiver, onComplete func(int64, context.Context) error) error {
	nullFile, err := os.OpenFile("/dev/null", os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	defer func() {
		_ = nullFile.Close() //nolint:errcheck
	}()

	totalSize := int64(0)
	chunkCount := 0
	responseWritten := false

	// Process stream and write to /dev/null.
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// End of stream, call completion function if not already done.
			if !responseWritten {
				return onComplete(totalSize, stream.Context())
			}

			return nil
		}

		if err != nil {
			return err
		}

		totalSize += int64(len(chunk.GetData()))
		chunkCount++
	}
}
