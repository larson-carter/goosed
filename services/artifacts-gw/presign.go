package artifactsgw

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	gos3 "goosed/pkg/s3"
)

const (
	defaultTTLSeconds = 300
	maxTTLSeconds     = 3600
)

// Server exposes helpers for presigning artifact downloads.
type Server struct {
	bucket string
	s3     *gos3.Client
}

// NewServer configures a Server using the provided S3 client and bucket.
func NewServer(bucket string, client *gos3.Client) (*Server, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return nil, errors.New("bucket is required")
	}
	if client == nil {
		return nil, errors.New("s3 client is required")
	}
	return &Server{bucket: bucket, s3: client}, nil
}

// RegisterHandlers attaches HTTP routes for presign features.
func (s *Server) RegisterHandlers(mux *http.ServeMux) error {
	if s == nil {
		return errors.New("nil server")
	}
	if mux == nil {
		return errors.New("nil mux")
	}

	mux.HandleFunc("/v1/presign/get", s.handleGetPresign)
	return nil
}

func (s *Server) handleGetPresign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key == "" {
		http.Error(w, "missing key query parameter", http.StatusBadRequest)
		return
	}

	ttlSeconds := defaultTTLSeconds
	if raw := strings.TrimSpace(r.URL.Query().Get("ttl")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			http.Error(w, "invalid ttl", http.StatusBadRequest)
			return
		}
		if parsed > maxTTLSeconds {
			parsed = maxTTLSeconds
		}
		ttlSeconds = parsed
	}

	url, err := s.s3.PresignGet(r.Context(), s.bucket, key, time.Duration(ttlSeconds)*time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("presign: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// When fronted by ingress-nginx ensure Range headers reach the backend:
	// nginx.ingress.kubernetes.io/configuration-snippet: |
	//   proxy_set_header Range $http_range;
	//   proxy_set_header If-Range $http_if_range;
	_ = json.NewEncoder(w).Encode(map[string]string{"url": url})
}
