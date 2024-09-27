package gophmarkthttpserver

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	storage "github.com/zvfkjytytw/gophmarkt/internal/server/storage"
)

const (
	authTTL       = 60
	updateAuthTTL = time.Minute
)

type authUser struct {
	ttl   int32
	login string
}

type Config struct {
	Host         string `yaml:"host"`
	Port         int32  `yaml:"port"`
	ReadTimeout  int32  `yaml:"read_timeout"`
	WriteTimeout int32  `yaml:"write_timeout"`
	IdleTimeout  int32  `yaml:"idle_timeout"`
}

type HTTPServer struct {
	sync.RWMutex
	server    *http.Server
	logger    *zap.Logger
	storage   *storage.PGStorage
	authUsers map[string]*authUser
}

func NewHTTPServer(
	runAddress string,
	comlog *zap.Logger,
	storage *storage.PGStorage,
) (*HTTPServer, error) {
	addr := strings.Split(runAddress, ":")
	if len(addr) < 2 {
		return nil, fmt.Errorf("address %s has not enough information", runAddress)
	}

	port, err := strconv.Atoi(addr[1])
	if err != nil {
		return nil, fmt.Errorf("port value %s ii invalid: %v", addr[1], err)
	}

	config := &Config{
		Host:         addr[0],
		Port:         int32(port),
		ReadTimeout:  5,
		WriteTimeout: 5,
		IdleTimeout:  10,
	}

	return NewHTTPServerFromConfig(config, comlog, storage)
}

func NewHTTPServerFromConfig(
	config *Config,
	comlog *zap.Logger,
	storage *storage.PGStorage,
) (*HTTPServer, error) {
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		ReadTimeout:  time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(config.IdleTimeout) * time.Second,
	}

	logger, err := initLogger(comlog)
	if err != nil {
		comlog.Sugar().Errorf("failed init http logger: %v", err)
		logger = comlog
	}

	return &HTTPServer{
		server:    server,
		logger:    logger,
		storage:   storage,
		authUsers: make(map[string]*authUser),
	}, nil
}

func (h *HTTPServer) Start(ctx context.Context) error {
	router := h.newRouter()
	h.server.Handler = router

	stopCheckAuth := make(chan struct{})
	defer close(stopCheckAuth)
	go h.CheckAuth(stopCheckAuth)

	err := h.server.ListenAndServe()
	if err != nil {
		h.logger.Sugar().Errorf("failed start http server: %w", err)
		return err
	}

	return nil
}

func (h *HTTPServer) Stop(ctx context.Context) error {
	defer h.logger.Sync()
	err := h.server.Shutdown(ctx)
	if err != nil {
		h.logger.Sugar().Errorf("failed stop http server: %w", err)
		return err
	}

	err = h.storage.Close()
	if err != nil {
		h.logger.Sugar().Errorf("failed close storage: %w", err)
		return err
	}

	return nil
}

func initLogger(comlog *zap.Logger) (*zap.Logger, error) {
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(comlog.Level()),
		Development:      true,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout", "httpacc.log"},
		ErrorOutputPaths: []string{"stderr", "httperr.log"},
	}
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

func (h *HTTPServer) CheckAuth(stop chan struct{}) {
	ticker := time.NewTicker(updateAuthTTL)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			func() {
				h.Lock()
				defer h.Unlock()
				for token, user := range h.authUsers {
					if user.ttl <= 0 {
						delete(h.authUsers, token)
					} else {
						user.ttl--
					}
				}
			}()
		}
	}
}
