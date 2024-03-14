package scratch

import (
	"io"
	"log"
)

var _ io.Writer = (*streamWrapper)(nil)

// streamWrapper wraps a logger to adapt it as an io.Writer.
type streamWrapper struct {
	log    *log.Logger
	prefix string
}

func wrapLogger(prefix string, logger *log.Logger) io.Writer {
	return &streamWrapper{
		log:    logger,
		prefix: prefix,
	}
}

func (s *streamWrapper) Write(p []byte) (n int, err error) {
	s.log.Print(s.prefix + string(p))
	return len(p), nil
}
