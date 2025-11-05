package bundler

import (
	"time"

	"gopkg.in/yaml.v3"
)

// Manifest represents the signed metadata included in bundles.
type Manifest struct {
	Version          string             `yaml:"version"`
	CreatedAt        time.Time          `yaml:"created_at"`
	Signer           string             `yaml:"signer,omitempty"`
	SigningPublicKey string             `yaml:"signing_public_key,omitempty"`
	Signature        string             `yaml:"signature,omitempty"`
	Images           []string           `yaml:"images,omitempty"`
	Artifacts        []ManifestArtifact `yaml:"artifacts"`
}

// SigningBytes marshals the manifest without its signature for signing/verification.
func (m Manifest) SigningBytes() ([]byte, error) {
	clone := m
	clone.Signature = ""
	return yaml.Marshal(clone)
}

// ManifestArtifact describes a single file within the bundle.
type ManifestArtifact struct {
	Path   string `yaml:"path"`
	Kind   string `yaml:"kind"`
	Size   int64  `yaml:"size"`
	SHA256 string `yaml:"sha256"`
}
