package config

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Config struct {
	Address         string `env:"SERVER_ADDRESS"`
	HostedOn        string `env:"BASE_URL"`
	LogLevel        string `env:"LOG_LEVEL" envDefault:"INFO"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
}

func (cfg *Config) Sanitize() {
	if !strings.HasSuffix(cfg.HostedOn, "/") {
		cfg.HostedOn = cfg.HostedOn + "/"
	}
}

var Settings Config

func NewConfigFromArgs(argsConfig ArgsConfig) Config {
	return Config{
		Address:         argsConfig.Address.String(),
		HostedOn:        argsConfig.HostedOn.String(),
		FileStoragePath: argsConfig.FileStoragePath.String(),
	}
}

type ArgsConfig struct {
	Address         NetAddress
	HostedOn        HTTPAddress
	FileStoragePath FileStoragePath
}

var argsConfig ArgsConfig

type NetAddress struct {
	Host string
	Port int
}

func (n *NetAddress) String() string {
	return n.Host + ":" + strconv.Itoa(n.Port)
}

func (n *NetAddress) Set(flagValue string) error {
	host, port, err := net.SplitHostPort(flagValue)
	if err != nil {
		return err
	}
	if host == "" && port == "" {
		n.Host = "localhost"
		n.Port = 8080
		return nil
	}
	port = strings.TrimSuffix(port, "/")
	n.Host = host
	n.Port, err = strconv.Atoi(port)
	return err
}

type HTTPAddress struct {
	Scheme string
	Host   string
	Port   int
}

func (h *HTTPAddress) String() string {
	return fmt.Sprintf(`%s%s:%d/`, h.Scheme, h.Host, h.Port)
}

func (h *HTTPAddress) Set(flagValue string) error {
	addressParts := strings.Split(flagValue, "://")
	if addressParts == nil {
		h.Scheme = "http://"
		h.Host = "localhost"
		h.Port = 8080
		return nil
	}
	if len(addressParts) != 2 {
		fmt.Println("wrong base address format (must be schema://host:port)")
		return errors.New("wrong base address format (must be schema://host:port)")
	}
	netAddress := new(NetAddress)
	err := netAddress.Set(addressParts[1])
	if err != nil {
		fmt.Println("error setting host:port from base address:", err)
		return err
	}
	h.Scheme = addressParts[0] + "://"
	h.Host = netAddress.Host
	h.Port = netAddress.Port
	return err
}

type FileStoragePath struct {
	Path string
}

func (f *FileStoragePath) String() string {
	return f.Path
}

func (f *FileStoragePath) Set(s string) error {
	if s == "" {
		return errors.New("file storage path must not be empty")
	}
	f.Path = s
	return nil
}

func ParseFlags() {
	hostAddr := new(NetAddress)
	baseAddr := new(HTTPAddress)
	fileStoragePath := new(FileStoragePath)
	flag.Var(hostAddr, "a", "Address to host on host:port")
	flag.Var(baseAddr, "b", "base URL for resulting short URL (scheme://host:port)")
	flag.Var(fileStoragePath, "f", "path to file to store short URLs")
	flag.Parse()
	if hostAddr.Host == "" && hostAddr.Port == 0 {
		hostAddr.Host = "localhost"
		hostAddr.Port = 8080
	}
	if baseAddr.Host == "" && baseAddr.Port == 0 && baseAddr.Scheme == "" {
		baseAddr.Scheme = "http://"
		baseAddr.Host = "localhost"
		baseAddr.Port = 8080
	}
	if fileStoragePath.Path == "" {
		fileStoragePath.Path = "./internal/app/storage/storage.json"
	}
	argsConfig.Address = *hostAddr
	argsConfig.HostedOn = *baseAddr
	argsConfig.FileStoragePath = *fileStoragePath
	Settings = NewConfigFromArgs(argsConfig)
}

func init() {
	Settings.Address = "localhost:8080"
	Settings.HostedOn = "http://localhost:8080/"
	Settings.LogLevel = "INFO"
	Settings.FileStoragePath = "./storage.json"
}
