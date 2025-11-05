# Building and Importing Air-Gap Bundles

Bundle content includes container images, ISOs/WIMs, drivers, and a signed metadata manifest. Use `goosectl` to build bundles on a connected workstation and import them into an isolated environment.

## Export signing keys

Export an **age** secret key (`AGE-SECRET-KEY-...`) before building. To avoid placing the private key on the import host, derive a verifier public key (base64 Ed25519) once and store it securely:

```bash
export AGE_SECRET_KEY=$(cat ~/.config/goosed/age.key)
export AGE_PUBLIC_KEY=$(go run - <<'EOF'
package main

import (
        "fmt"

        "goosed/services/bundler"
)

func main() {
        signer, err := bundler.NewSignerFromEnv()
        if err != nil {
                panic(err)
        }
        fmt.Println(signer.PublicKeyBase64())
}
EOF
)
```

`AGE_SECRET_KEY` is required for signing; either `AGE_SECRET_KEY` **or** `AGE_PUBLIC_KEY` must be present when importing.

## Build a bundle

```bash
go run ./services/bundler/cmd/goosectl bundles build --artifacts-dir ./artifacts --images-file ./images.txt --output ./bundle-$(date +%Y%m%d).tar.zst
```

## Import a bundle

```bash
export AGE_PUBLIC_KEY=<base64-ed25519-from-above>   # or reuse AGE_SECRET_KEY
export S3_ENDPOINT=https://seaweedfs.example.local:8333
export S3_ACCESS_KEY=...
export S3_SECRET_KEY=...
export S3_REGION=us-east-1
export S3_DISABLE_TLS=false

go run ./services/bundler/cmd/goosectl bundles import --file ./bundle-20251104.tar.zst --api https://api.goose.local
```

## Offline workflow checklist

1. On a connected workstation, run the build command to generate the `bundle-*.tar.zst` archive.
2. Transfer the bundle and the `AGE_PUBLIC_KEY` value to the air-gapped environment (keep the secret key offline).
3. Configure the air-gapped host with S3 credentials, `AGE_PUBLIC_KEY`, and the API URL, then run the import command.
4. `goosectl` verifies the signed manifest, uploads each object via the S3 API with checksums, and registers it through `POST /v1/artifacts` in register-only mode.
5. Confirm availability with `GET /v1/artifacts` or the dashboard before scheduling installs.

For uploading base media prior to bundling, follow the [SeaweedFS ISO guide](seaweedfs-iso-upload.md).
