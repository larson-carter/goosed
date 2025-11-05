package bundler

import (
	"io"
	"net/http"
	"time"

	gos3 "goosed/pkg/s3"
)

// BuildConfig configures bundle creation.
type BuildConfig struct {
	ArtifactsDir string
	ImagesFile   string
	Output       string
	Signer       *Signer
	Now          func() time.Time
	Stdout       io.Writer
}

// ImportConfig configures bundle import operations.
type ImportConfig struct {
	BundlePath string
	APIBaseURL string
	HTTPClient *http.Client
	S3         *gos3.Client
	Signer     *Signer
	Now        func() time.Time
	Stdout     io.Writer
}
