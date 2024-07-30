package gophmarktapp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	server "github.com/zvfkjytytw/gophmarkt/internal/server/http"
	storage "github.com/zvfkjytytw/gophmarkt/internal/server/storage"
)

type Service interface {
	Start(context.Context) error
	Stop(context.Context) error
}

type AppConfig struct {
	HTTPConfig    *server.Config  `yaml:"http_config" json:"http_config"`
	StorageConfig *storage.Config `yaml:"storage_config" json:"storage_config"`
	MigrationDir  string          `yaml:"migration_dir" json:"migration_dir"`
}

type App struct {
	services []Service
	logger   *zap.Logger
}

func NewApp(
	// migrationDir,
	runAddress,
	databaseURI,
	accrualSystem string,
) (*App, error) {
	logger, err := InitLogger()
	if err != nil {
		return nil, fmt.Errorf("failed init logger: %v", err)
	}

	services := make([]Service, 0, 1)

	pgStorage, err := storage.NewPGStorage(databaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed init storage: %v", err)
	}

	// if err := storage.ApplyMigrations(databaseURI, migrationDir); err != nil {
	// 	logger.Sugar().Errorf("failed init DB: %v", err)
	// 	return nil, err
	// }

	if err := pgStorage.ApplyMigrations(); err != nil {
		logger.Sugar().Errorf("failed init DB: %v", err)
		// return nil, err
	}

	httpServer, err := server.NewHTTPServer(runAddress, logger, pgStorage)
	if err != nil {
		logger.Sugar().Errorf("failed init HTTP server: %v", err)
		return nil, err
	}

	services = append(services, httpServer)

	return &App{
		services: services,
		logger:   logger,
	}, nil
}

func NewAppFromConfig(config *AppConfig) (*App, error) {
	logger, err := InitLogger()
	if err != nil {
		return nil, fmt.Errorf("failed init logger: %v", err)
	}

	services := make([]Service, 0, 1)

	pgDSN, err := storage.GetDSNFromConfig(config.StorageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed format dsn: %v", err)
	}

	pgStorage, err := storage.NewPGStorageFromConfig(config.StorageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed init storage: %v", err)
	}

	if err := storage.ApplyMigrations(pgDSN, config.MigrationDir); err != nil {
		logger.Sugar().Errorf("failed init DB: %v", err)
		return nil, err
	}

	httpServer, err := server.NewHTTPServerFromConfig(
		config.HTTPConfig,
		logger,
		pgStorage,
	)
	if err != nil {
		return nil, fmt.Errorf("failed init HTTP server: %v", err)
	}

	services = append(services, httpServer)

	return &App{
		services: services,
		logger:   logger,
	}, nil
}

func NewAppFromFile(configFile string) (*App, error) {
	config := &AppConfig{}
	configData, err := ReadConfigFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed read config file: %v", err)
	}

	err = yaml.Unmarshal(configData, config)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshalling config file: %v", err)
	}

	return NewAppFromConfig(config)
}

func (a *App) Run(ctx context.Context) {
	defer a.logger.Sync()
	sigChanel := make(chan os.Signal, 1)
	signal.Notify(sigChanel,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	for _, service := range a.services {
		go func(service Service) {
			err := service.Start(ctx)
			if err != nil {
				a.logger.Sugar().Errorf("service not started: %w", err)
				a.StopAll(ctx)
				return
			}
		}(service)
	}

	stopSignal := <-sigChanel
	a.logger.Sugar().Debugf("Stop by %v", stopSignal)
	a.StopAll(ctx)
}

func (a *App) StopAll(ctx context.Context) {
	for _, service := range a.services {
		err := service.Stop(ctx)
		if err != nil {
			a.logger.Error("stop failed")
		}
	}
}
