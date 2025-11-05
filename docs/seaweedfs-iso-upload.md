# Uploading Rocky Linux ISOs to SeaweedFS

The PXE blueprints expect Rocky Linux media to be available from the SeaweedFS S3 gateway. Follow the steps below to place the ISO in the canonical location (`artifacts/rocky/9/Rocky-9.4-x86_64-minimal.iso`) inside the `goosed-artifacts` bucket.

## 1. Collect your SeaweedFS S3 credentials

1. Retrieve the endpoint, access key, and secret key used by the cluster. For the developer Helm values the defaults are:
   - **Endpoint**: `http://localhost:8333`
   - **Access key**: `goosed`
   - **Secret key**: `supersecret`
   - **Bucket**: `goosed-artifacts`
2. Export them for the CLI you plan to use:

   ```bash
   export S3_ENDPOINT=http://localhost:8333
   export S3_ACCESS_KEY=goosed
   export S3_SECRET_KEY=supersecret
   export S3_BUCKET=goosed-artifacts
   ```

Adjust the values if you overrode them in `deploy/helm/umbrella/values-*.yaml`.

## 2. Create the bucket (first time only)

SeaweedFS creates buckets lazily. If this is a new environment, run one of the commands below to create the `goosed-artifacts` bucket:

```bash
aws --endpoint-url "$S3_ENDPOINT" s3 mb "s3://$S3_BUCKET"
# or
mc alias set goose "$S3_ENDPOINT" "$S3_ACCESS_KEY" "$S3_SECRET_KEY"
mc mb goose/$S3_BUCKET
```

## 3. Upload the ISO

1. Download the Rocky Linux ISO on your workstation (for example `Rocky-9.4-x86_64-minimal.iso`).
2. Upload it to the canonical key expected by the blueprints:

   ```bash
   aws --endpoint-url "$S3_ENDPOINT" \
     s3 cp ./Rocky-9.4-x86_64-minimal.iso \
     "s3://$S3_BUCKET/artifacts/rocky/9/Rocky-9.4-x86_64-minimal.iso"
   ```

   Using `mc`:

   ```bash
   mc cp ./Rocky-9.4-x86_64-minimal.iso \
     goose/$S3_BUCKET/artifacts/rocky/9/Rocky-9.4-x86_64-minimal.iso
   ```

3. (Optional) Upload matching kernel and initrd images alongside the ISO under `artifacts/rocky/9/` if you plan to override the defaults in the blueprint.

## 4. Verify the upload

List the object to confirm it is readable from the cluster:

```bash
aws --endpoint-url "$S3_ENDPOINT" s3 ls "s3://$S3_BUCKET/artifacts/rocky/9/"
# or
mc ls goose/$S3_BUCKET/artifacts/rocky/9/
```

You should see the ISO in the output. Once present, the PXE stack will be able to hand the artifact to VMware Fusion guests through the normal Kickstart flow.
