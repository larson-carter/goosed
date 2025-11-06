package tftp

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pin/tftp"

	"goosed/services/pxe-stack/internal/config"
)

func NewServer(cfg config.TFTPConfig, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.Default()
	}
	return &Server{cfg: cfg, logger: logger}
}

func (s *Server) Run(ctx context.Context, ready *atomic.Bool) error {
	srv := tftp.NewServer(s.readHandler, nil)
	srv.SetTimeout(time.Duration(s.cfg.TimeoutSec) * time.Second)

	addr := s.cfg.Address
	if addr == "" {
		addr = ":69"
	}

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", addr, err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	ready.Store(true)

	done := make(chan struct{})
	go func() {
		srv.Serve(conn)
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		srv.Shutdown()
		<-done
		return nil
	}
}

func (s *Server) readHandler(filename string, rf io.ReaderFrom) error {
	clean := filepath.Clean(filename)
	for strings.HasPrefix(clean, string(filepath.Separator)) {
		clean = strings.TrimPrefix(clean, string(filepath.Separator))
	}
	path := filepath.Join(s.cfg.RootDir, clean)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := rf.ReadFrom(f); err != nil {
		return err
	}
	s.logger.Printf("INFO served %s via TFTP", filename)
	return nil
}
