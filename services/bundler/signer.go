package bundler

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"filippo.io/age"
	"github.com/btcsuite/btcutil/bech32"
)

const (
	envAgeSecretKey = "AGE_SECRET_KEY"
	envAgePublicKey = "AGE_PUBLIC_KEY"
)

// Signer handles signing and verifying manifest payloads using an age-derived Ed25519 key pair.
type Signer struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	recipient  string
}

// NewSignerFromEnv initialises a Signer using AGE_SECRET_KEY/AGE_PUBLIC_KEY environment variables.
// At least AGE_SECRET_KEY or AGE_PUBLIC_KEY must be provided. AGE_PUBLIC_KEY is expected to be a base64-encoded
// Ed25519 public key derived from the age secret key seed.
func NewSignerFromEnv() (*Signer, error) {
	secret := strings.TrimSpace(os.Getenv(envAgeSecretKey))
	pub := strings.TrimSpace(os.Getenv(envAgePublicKey))

	if secret == "" && pub == "" {
		return nil, fmt.Errorf("%s or %s must be set", envAgeSecretKey, envAgePublicKey)
	}

	var (
		privateKey ed25519.PrivateKey
		publicKey  ed25519.PublicKey
		recipient  string
	)

	if secret != "" {
		seed, err := decodeAgeSecretKey(secret)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", envAgeSecretKey, err)
		}
		privateKey = ed25519.NewKeyFromSeed(seed)
		publicKey = ed25519.PublicKey(privateKey[ed25519.SeedSize:])

		if identity, err := age.ParseX25519Identity(secret); err == nil {
			if r := identity.Recipient(); r != nil {
				recipient = r.String()
			}
		}
	}

	if pub != "" {
		decoded, err := base64.StdEncoding.DecodeString(pub)
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", envAgePublicKey, err)
		}
		if l := len(decoded); l != ed25519.PublicKeySize {
			return nil, fmt.Errorf("%s must decode to %d bytes, got %d", envAgePublicKey, ed25519.PublicKeySize, l)
		}
		if publicKey == nil {
			publicKey = ed25519.PublicKey(decoded)
		} else if !bytes.Equal(publicKey, decoded) {
			return nil, errors.New("AGE_PUBLIC_KEY does not match AGE_SECRET_KEY")
		}
	}

	if publicKey == nil {
		return nil, errors.New("no public key available for signer")
	}

	return &Signer{
		privateKey: privateKey,
		publicKey:  publicKey,
		recipient:  recipient,
	}, nil
}

// Sign produces a base64-encoded Ed25519 signature for the provided payload.
func (s *Signer) Sign(payload []byte) (string, error) {
	if s == nil {
		return "", errors.New("nil signer")
	}
	if len(s.privateKey) == 0 {
		return "", errors.New("signer configured without private key")
	}
	sig := ed25519.Sign(s.privateKey, payload)
	return base64.StdEncoding.EncodeToString(sig), nil
}

// Verify checks the supplied base64 signature against the payload. If the manifest embeds a
// signing public key, provide it via manifestPublicKey. The signer validates it matches the
// configured public key if present.
func (s *Signer) Verify(payload []byte, signature, manifestPublicKey string) error {
	if s == nil {
		return errors.New("nil signer")
	}
	sigBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature))
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length %d", len(sigBytes))
	}

	key := s.publicKey
	if manifestPublicKey != "" {
		decoded, err := base64.StdEncoding.DecodeString(manifestPublicKey)
		if err != nil {
			return fmt.Errorf("decode manifest public key: %w", err)
		}
		if l := len(decoded); l != ed25519.PublicKeySize {
			return fmt.Errorf("manifest public key must be %d bytes, got %d", ed25519.PublicKeySize, l)
		}
		if key != nil && !bytes.Equal(key, decoded) {
			return errors.New("manifest signed by unexpected key")
		}
		if key == nil {
			key = ed25519.PublicKey(decoded)
		}
	}

	if key == nil {
		return errors.New("no public key available for verification")
	}
	if !ed25519.Verify(key, payload, sigBytes) {
		return errors.New("signature verification failed")
	}
	return nil
}

// PublicKeyBase64 returns the configured Ed25519 public key in base64 form.
func (s *Signer) PublicKeyBase64() string {
	if s == nil || len(s.publicKey) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(s.publicKey)
}

// Recipient returns the age recipient string if the signer was initialised with AGE_SECRET_KEY.
func (s *Signer) Recipient() string {
	if s == nil {
		return ""
	}
	return s.recipient
}

func decodeAgeSecretKey(raw string) ([]byte, error) {
	hrp, data, err := bech32.Decode(raw)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(hrp, "age-secret-key-") {
		return nil, fmt.Errorf("unexpected hrp %q", hrp)
	}
	decoded, err := bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return nil, err
	}
	if len(decoded) != ed25519.SeedSize {
		return nil, fmt.Errorf("unexpected seed length %d", len(decoded))
	}
	return decoded, nil
}
