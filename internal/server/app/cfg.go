package gophmarktapp

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func ReadConfigFile(configFile string) ([]byte, error) {
	filename, err := filepath.Abs(configFile)
	if err != nil {
		return nil, fmt.Errorf("file %s not found: %w", configFile, err)
	}

	yamlConfig, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("fail read file %s: %w", configFile, err)
	}

	return yamlConfig, nil
}

func InitLogger() (*zap.Logger, error) {
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapcore.InfoLevel),
		Development:      true,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout", "gophmarkt.log"},
		ErrorOutputPaths: []string{"stderr", "gophmarkt.err"},
	}
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}
