package tftp

import (
	"log"

	"goosed/services/pxe-stack/internal/config"
)

type Server struct {
	cfg    config.TFTPConfig
	logger *log.Logger
}
