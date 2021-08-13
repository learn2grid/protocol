package web3

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/Oneledger/protocol/config"
	"github.com/Oneledger/protocol/log"
	rpctypes "github.com/Oneledger/protocol/web3/types"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

// Server holds an RPC server that is served over HTTP
type Server struct {
	rpc     *rpc.Server
	logger  *log.Logger
	cfg     *config.Server
	httpURL *url.URL
	wsURL   *url.URL
}

func NewServer(config *config.Server) *Server {
	return &Server{
		logger: log.NewLoggerWithPrefix(os.Stdout, "rpc"),
		cfg:    config,
	}
}

func RegisterApis(src *rpc.Server, apis map[string]rpctypes.Web3Service) error {
	for name, svc := range apis {
		err := src.RegisterName(name, svc)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) start(srv *rpc.Server, handler http.Handler, rpcInfo interface{}, apis map[string]rpctypes.Web3Service) error {
	var (
		err               error
		uri               string
		enabled           bool
		availableAPINames []string
		availableAPIs     = make(map[string]rpctypes.Web3Service, 0)
		name              string
	)
	channel := make(chan error)
	timeout := make(chan error)

	switch rpcCfg := rpcInfo.(type) {
	case *config.HTTPConfig:
		name = "HTTP"
		uri = fmt.Sprintf("%s:%d", rpcCfg.Addr, rpcCfg.Port)
		enabled = rpcCfg.Enabled
		availableAPINames = rpcCfg.API
	case *config.WSConfig:
		name = "WS"
		uri = fmt.Sprintf("%s:%d", rpcCfg.Addr, rpcCfg.Port)
		enabled = rpcCfg.Enabled
		availableAPINames = rpcCfg.API
	default:
		s.logger.Info("Config for Web3 RPC not properly configured, skipping")
		return nil
	}

	if !enabled {
		s.logger.Info("Web3 " + name + " RPC server not enabled, skipping")
		return nil
	}

	for _, apiName := range availableAPINames {
		apiService, ok := apis[apiName]
		if ok {
			availableAPIs[apiName] = apiService
		}
	}

	if err := RegisterApis(srv, availableAPIs); err != nil {
		return err
	}

	//Timeout Go routine
	go func() {
		time.Sleep(time.Duration(2) * time.Second)
		timeout <- nil
	}()

	go func(ch chan error) {
		defer srv.Stop()

		s.logger.Info("starting Web3 " + name + " RPC server on " + uri)
		err := http.ListenAndServe(uri, handler)
		if err != nil {
			s.logger.Fatalf("server: %s", err)
		}
		ch <- err
	}(channel)

	select {
	case err = <-channel:
	case err = <-timeout:
	}

	return err
}

func (s *Server) StartHTTP(apis map[string]rpctypes.Web3Service) error {
	srv := rpc.NewServer()
	handler := node.NewHTTPHandlerStack(srv, s.cfg.API.HTTPConfig.CORSDomain, s.cfg.API.HTTPConfig.VHosts)
	return s.start(srv, handler, s.cfg.API.HTTPConfig, apis)
}

func (s *Server) StartWS(apis map[string]rpctypes.Web3Service) error {
	srv := rpc.NewServer()
	handler := srv.WebsocketHandler(s.cfg.API.WSConfig.Origins)
	return s.start(srv, handler, s.cfg.API.WSConfig, apis)
}
