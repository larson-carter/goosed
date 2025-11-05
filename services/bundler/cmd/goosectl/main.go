package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	gos3 "goosed/pkg/s3"
	"goosed/services/bundler"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "goosectl",
		Short:         "Utility for managing goosed air-gap bundles",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newBundlesCommand())
	return cmd
}

func newBundlesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundles",
		Short: "Bundle build and import operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newBundlesBuildCommand())
	cmd.AddCommand(newBundlesImportCommand())
	return cmd
}

func newBundlesBuildCommand() *cobra.Command {
	var (
		artifactsDir string
		imagesFile   string
		output       string
	)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Create a signed bundle from an artifacts directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			signer, err := bundler.NewSignerFromEnv()
			if err != nil {
				return err
			}
			_, err = bundler.Build(ctx, bundler.BuildConfig{
				ArtifactsDir: artifactsDir,
				ImagesFile:   imagesFile,
				Output:       output,
				Signer:       signer,
				Stdout:       os.Stdout,
			})
			return err
		},
	}

	cmd.Flags().StringVar(&artifactsDir, "artifacts-dir", "", "Directory containing artifacts to include")
	cmd.Flags().StringVar(&imagesFile, "images-file", "", "Optional file listing container images to mirror")
	cmd.Flags().StringVar(&output, "output", "", "Destination bundle file (tar.zst)")
	_ = cmd.MarkFlagRequired("artifacts-dir")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

func newBundlesImportCommand() *cobra.Command {
	var (
		bundleFile string
		apiBaseURL string
	)

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import a signed bundle into the API and S3",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			signer, err := bundler.NewSignerFromEnv()
			if err != nil {
				return err
			}
			s3Client, err := gos3.NewClientFromEnv()
			if err != nil {
				return fmt.Errorf("s3 client: %w", err)
			}
			_, err = bundler.Import(ctx, bundler.ImportConfig{
				BundlePath: bundleFile,
				APIBaseURL: apiBaseURL,
				HTTPClient: nil,
				S3:         s3Client,
				Signer:     signer,
				Stdout:     os.Stdout,
			})
			return err
		},
	}

	cmd.Flags().StringVar(&bundleFile, "file", "", "Path to the bundle tar.zst")
	cmd.Flags().StringVar(&apiBaseURL, "api", "", "Base URL of the goosed API (e.g. https://api.example.com)")
	_ = cmd.MarkFlagRequired("file")
	_ = cmd.MarkFlagRequired("api")
	return cmd
}
