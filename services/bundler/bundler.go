package bundler

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"gopkg.in/yaml.v3"
)

const (
	manifestFileName   = "manifest.yaml"
	artifactsTarPrefix = "artifacts"
)

// Build assembles a bundle from the provided directory and writes the tar.zst archive to Output.
func Build(ctx context.Context, cfg BuildConfig) (*Manifest, error) {
	if cfg.ArtifactsDir == "" {
		return nil, errors.New("artifacts directory is required")
	}
	if cfg.Output == "" {
		return nil, errors.New("output path is required")
	}
	if cfg.Signer == nil {
		return nil, errors.New("signer is required")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	info, err := os.Stat(cfg.ArtifactsDir)
	if err != nil {
		return nil, fmt.Errorf("stat artifacts dir: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("artifacts dir %q is not a directory", cfg.ArtifactsDir)
	}

	entries, err := collectArtifacts(ctx, cfg.ArtifactsDir)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, errors.New("no artifacts found to bundle")
	}

	images, err := readImagesFile(cfg.ImagesFile)
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	manifest := &Manifest{
		Version:          "1",
		CreatedAt:        cfg.Now().UTC().Truncate(time.Second),
		Signer:           cfg.Signer.Recipient(),
		SigningPublicKey: cfg.Signer.PublicKeyBase64(),
		Images:           images,
		Artifacts:        entries,
	}

	payload, err := manifest.SigningBytes()
	if err != nil {
		return nil, fmt.Errorf("marshal manifest for signing: %w", err)
	}
	sig, err := cfg.Signer.Sign(payload)
	if err != nil {
		return nil, fmt.Errorf("sign manifest: %w", err)
	}
	manifest.Signature = sig

	manifestBytes, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}

	if err := writeBundle(cfg.Output, manifestBytes, cfg.ArtifactsDir, entries); err != nil {
		return nil, err
	}

	fmt.Fprintf(cfg.Stdout, "wrote bundle %s (%d artifacts)\n", cfg.Output, len(entries))
	return manifest, nil
}

func collectArtifacts(ctx context.Context, root string) ([]ManifestArtifact, error) {
	var artifacts []ManifestArtifact
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("relative path for %q: %w", path, err)
		}
		rel = filepath.ToSlash(rel)

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open %q: %w", path, err)
		}
		hash := sha256.New()
		size, err := io.Copy(hash, file)
		file.Close()
		if err != nil {
			return fmt.Errorf("hash %q: %w", path, err)
		}

		sha := hex.EncodeToString(hash.Sum(nil))
		artifacts = append(artifacts, ManifestArtifact{
			Path:   rel,
			Kind:   inferKind(rel),
			Size:   size,
			SHA256: sha,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return artifacts, nil
}

func readImagesFile(path string) ([]string, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read images file: %w", err)
	}
	var images []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		images = append(images, trimmed)
	}
	return images, nil
}

func writeBundle(output string, manifest []byte, artifactsDir string, entries []ManifestArtifact) error {
	dir := filepath.Dir(output)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create output dir: %w", err)
		}
	}

	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer file.Close()

	encoder, err := zstd.NewWriter(file)
	if err != nil {
		return fmt.Errorf("zstd writer: %w", err)
	}
	defer encoder.Close()

	tw := tar.NewWriter(encoder)
	defer tw.Close()

	manifestHeader := &tar.Header{
		Name:     manifestFileName,
		Mode:     0o644,
		Size:     int64(len(manifest)),
		ModTime:  time.Now().UTC(),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(manifestHeader); err != nil {
		return fmt.Errorf("write manifest header: %w", err)
	}
	if _, err := tw.Write(manifest); err != nil {
		return fmt.Errorf("write manifest body: %w", err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(artifactsDir, filepath.FromSlash(entry.Path))
		info, err := os.Stat(fullPath)
		if err != nil {
			return fmt.Errorf("stat %q: %w", entry.Path, err)
		}
		file, err := os.Open(fullPath)
		if err != nil {
			return fmt.Errorf("open %q: %w", entry.Path, err)
		}

		header := &tar.Header{
			Name:     filepath.ToSlash(filepath.Join(artifactsTarPrefix, entry.Path)),
			Mode:     int64(info.Mode().Perm()),
			Size:     info.Size(),
			ModTime:  info.ModTime(),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(header); err != nil {
			file.Close()
			return fmt.Errorf("write header for %q: %w", entry.Path, err)
		}
		if _, err := io.Copy(tw, file); err != nil {
			file.Close()
			return fmt.Errorf("copy %q: %w", entry.Path, err)
		}
		file.Close()
	}

	return nil
}

func inferKind(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".iso"):
		return "iso"
	case strings.HasSuffix(lower, ".wim"):
		return "wim"
	case strings.HasSuffix(lower, ".img"):
		return "disk-image"
	case strings.HasSuffix(lower, ".qcow2"):
		return "qcow2"
	case strings.HasSuffix(lower, ".vhd") || strings.HasSuffix(lower, ".vhdx"):
		return "vhd"
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		return "tar.gz"
	case strings.HasSuffix(lower, ".tar"):
		return "tar"
	case strings.HasSuffix(lower, ".zip"):
		return "zip"
	default:
		return "file"
	}
}

// Import extracts, verifies, uploads and registers bundle contents with the API and S3.
func Import(ctx context.Context, cfg ImportConfig) (*Manifest, error) {
	if cfg.BundlePath == "" {
		return nil, errors.New("bundle file is required")
	}
	if cfg.APIBaseURL == "" {
		return nil, errors.New("api base url is required")
	}
	if cfg.S3 == nil {
		return nil, errors.New("s3 client is required")
	}
	if cfg.Signer == nil {
		return nil, errors.New("signer is required")
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	bundleFile, err := os.Open(cfg.BundlePath)
	if err != nil {
		return nil, fmt.Errorf("open bundle: %w", err)
	}
	defer bundleFile.Close()

	decoder, err := zstd.NewReader(bundleFile)
	if err != nil {
		return nil, fmt.Errorf("zstd reader: %w", err)
	}
	defer decoder.Close()

	tr := tar.NewReader(decoder)
	tempDir, err := os.MkdirTemp("", "goosed-bundle-*")
	if err != nil {
		return nil, fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	var (
		manifestBytes []byte
		files         = map[string]string{}
	)

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar entry: %w", err)
		}

		name := filepath.Clean(header.Name)
		if header.Typeflag == tar.TypeDir {
			target := filepath.Join(tempDir, name)
			if !strings.HasPrefix(target, tempDir) {
				return nil, fmt.Errorf("invalid directory entry %q", name)
			}
			if err := os.MkdirAll(target, 0o755); err != nil {
				return nil, fmt.Errorf("mkdir %q: %w", name, err)
			}
			continue
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}

		if name == manifestFileName {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("read manifest: %w", err)
			}
			manifestBytes = data
			continue
		}

		targetPath := filepath.Join(tempDir, name)
		if !strings.HasPrefix(targetPath, tempDir) {
			return nil, fmt.Errorf("invalid entry path %q", name)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return nil, fmt.Errorf("mkdir %q: %w", filepath.Dir(targetPath), err)
		}
		file, err := os.Create(targetPath)
		if err != nil {
			return nil, fmt.Errorf("create temp file for %q: %w", name, err)
		}
		if _, err := io.Copy(file, tr); err != nil {
			file.Close()
			return nil, fmt.Errorf("write temp file for %q: %w", name, err)
		}
		file.Close()

		files[name] = targetPath
	}

	if len(manifestBytes) == 0 {
		return nil, errors.New("bundle missing manifest.yaml")
	}

	var manifest Manifest
	if err := yaml.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}
	if manifest.Version != "1" {
		return nil, fmt.Errorf("unsupported manifest version %q", manifest.Version)
	}
	if manifest.Signature == "" {
		return nil, errors.New("manifest missing signature")
	}

	payload, err := manifest.SigningBytes()
	if err != nil {
		return nil, fmt.Errorf("marshal manifest for verification: %w", err)
	}
	if err := cfg.Signer.Verify(payload, manifest.Signature, manifest.SigningPublicKey); err != nil {
		return nil, fmt.Errorf("verify manifest signature: %w", err)
	}

	fmt.Fprintf(cfg.Stdout, "verified manifest signed at %s\n", manifest.CreatedAt.Format(time.RFC3339))

	baseURL := strings.TrimRight(cfg.APIBaseURL, "/")
	for _, art := range manifest.Artifacts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		relative := filepath.ToSlash(filepath.Clean(art.Path))
		tarPath := filepath.ToSlash(filepath.Join(artifactsTarPrefix, relative))
		tempPath, ok := files[tarPath]
		if !ok {
			return nil, fmt.Errorf("artifact %q missing from archive", relative)
		}

		if err := validateArtifact(tempPath, art); err != nil {
			return nil, err
		}

		artifactURL, err := registerArtifact(ctx, cfg.HTTPClient, baseURL, art)
		if err != nil {
			return nil, err
		}

		bucket, key, err := parseS3URL(artifactURL)
		if err != nil {
			return nil, err
		}

		file, err := os.Open(tempPath)
		if err != nil {
			return nil, fmt.Errorf("open %q for upload: %w", relative, err)
		}
		if err := cfg.S3.PutObject(ctx, bucket, key, file, art.Size, art.SHA256); err != nil {
			file.Close()
			return nil, fmt.Errorf("upload %q: %w", relative, err)
		}
		file.Close()

		fmt.Fprintf(cfg.Stdout, "uploaded %s (%d bytes)\n", relative, art.Size)
	}

	return &manifest, nil
}

func validateArtifact(path string, art ManifestArtifact) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %q: %w", art.Path, err)
	}
	defer file.Close()

	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return fmt.Errorf("hash %q: %w", art.Path, err)
	}
	if size != art.Size {
		return fmt.Errorf("size mismatch for %q: expected %d got %d", art.Path, art.Size, size)
	}
	computed := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(computed, art.SHA256) {
		return fmt.Errorf("sha256 mismatch for %q", art.Path)
	}
	return nil
}

func registerArtifact(ctx context.Context, client *http.Client, baseURL string, art ManifestArtifact) (string, error) {
	body := map[string]any{
		"kind":   art.Kind,
		"sha256": art.SHA256,
		"meta": map[string]any{
			"path": art.Path,
			"size": art.Size,
		},
		"mode": "register",
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal artifact request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/artifacts", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("post artifact: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("artifact register failed: %s", strings.TrimSpace(string(data)))
	}

	var response struct {
		Artifact struct {
			URL string `json:"url"`
		} `json:"artifact"`
		S3 struct {
			Bucket string `json:"bucket"`
			Key    string `json:"key"`
		} `json:"s3"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode artifact response: %w", err)
	}
	if response.S3.Bucket != "" && response.S3.Key != "" {
		return fmt.Sprintf("s3://%s/%s", response.S3.Bucket, response.S3.Key), nil
	}
	if response.Artifact.URL == "" {
		return "", errors.New("api response missing artifact url")
	}
	return response.Artifact.URL, nil
}

func parseS3URL(url string) (string, string, error) {
	if !strings.HasPrefix(url, "s3://") {
		return "", "", fmt.Errorf("unsupported artifact url %q", url)
	}
	trimmed := strings.TrimPrefix(url, "s3://")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid s3 url %q", url)
	}
	bucket := parts[0]
	key := parts[1]
	if bucket == "" || key == "" {
		return "", "", fmt.Errorf("invalid s3 url %q", url)
	}
	return bucket, key, nil
}
