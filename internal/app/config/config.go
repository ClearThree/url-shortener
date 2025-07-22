// Package config contains project configuration utils.
package config

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is a structure that contains all the configurations for the application.
type Config struct {
	Address                            string `env:"SERVER_ADDRESS" json:"server_address"`
	HostedOn                           string `env:"BASE_URL" json:"base_url"`
	LogLevel                           string `env:"LOG_LEVEL" envDefault:"INFO"`
	FileStoragePath                    string `env:"FILE_STORAGE_PATH" json:"file_storage_path"`
	DatabaseDSN                        string `env:"DATABASE_DSN" json:"database_dsn"`
	SecretKey                          string `env:"SECRET_KEY" envDefault:"DontUseThatInProduction"`
	KeyPath                            string `env:"KEY_PATH" envDefault:"./cert.pem"`
	CertPath                           string `env:"CERT_PATH" envDefault:"./key.pem"`
	ConfigFile                         string `env:"CONFIG"`
	TrustedSubnet                      string `env:"TRUSTED_SUBNET" json:"trusted_subnet"`
	DatabaseMaxConnections             int    `env:"DATABASE_MAX_CONNECTIONS"  envDefault:"99"`
	JWTExpireHours                     int64  `env:"JWT_EXPIRE_HOURS" envDefault:"96"`
	DefaultChannelsBufferSize          int64  `env:"DEFAULT_CHANNELS_BUFFER_SIZE" envDefault:"1024"`
	DeletionBufferFlushIntervalSeconds int64  `env:"DELETION_BUFFER_FLUSH_INTERVAL_SECONDS" envDefault:"10"`
	TLSEnabled                         bool   `env:"ENABLE_HTTPS" envDefault:"false" json:"enable_https"`
	UseHeaderForSourceAddress          bool   `env:"USE_HEADER_FOR_SOURCE_ADDRESS" envDefault:"true" json:"use_header_for_source_address"`
}

// Sanitize fixes HostedOn variable with trailing slash.
func (cfg *Config) Sanitize() {
	if !strings.HasSuffix(cfg.HostedOn, "/") {
		cfg.HostedOn = cfg.HostedOn + "/"
	}

	if Settings.TLSEnabled {
		_, _, err := GetOrCreateCertAndKey()
		if err != nil {
			os.Exit(1)
		}
	}
}

// Settings is the global instance of Config type with all initialized settings.
var Settings Config

// NewConfigFromArgs returns the new Config instance with several settings redeclared from the command arguments.
func NewConfigFromArgs(argsConfig ArgsConfig) Config {
	return Config{
		Address:         argsConfig.Address.String(),
		HostedOn:        argsConfig.HostedOn.String(),
		FileStoragePath: argsConfig.FileStoragePath.String(),
		DatabaseDSN:     argsConfig.DatabaseDSN.String(),
		TLSEnabled:      argsConfig.TLSEnabled.TLSEnabled,
		ConfigFile:      argsConfig.ConfigFile.String(),
		TrustedSubnet:   argsConfig.TrustedSubnet.String(),
	}
}

// ArgsConfig is a structure that generalizes all the command line arguments.
type ArgsConfig struct {
	FileStoragePath FileStoragePath
	DatabaseDSN     DatabaseDSN
	ConfigFile      FileConfig
	TrustedSubnet   TrustedSubnet
	HostedOn        HTTPAddress
	Address         NetAddress
	TLSEnabled      TLSEnabled
}

var argsConfig ArgsConfig

// NetAddress is a structure that combines two variables for IP-address representation.
// Implements the Value interface.
type NetAddress struct {
	Host string
	Port int
}

// String returns the string representation of the address.
func (n *NetAddress) String() string {
	return n.Host + ":" + strconv.Itoa(n.Port)
}

// Set sets the structure fields from the string representation of an address.
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

// HTTPAddress is a structure that combines three variables for HTTP-address representation.
// Implements the Value interface.
type HTTPAddress struct {
	Scheme string
	Host   string
	Port   int
}

// String returns the string representation of the address.
func (h *HTTPAddress) String() string {
	return fmt.Sprintf(`%s%s:%d/`, h.Scheme, h.Host, h.Port)
}

// Set sets the structure fields from the string representation of an address.
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

// FileStoragePath is a structure that represents the path of file.
// Implements the Value interface.
type FileStoragePath struct {
	Path string
}

// String returns the string representation of the path.
func (f *FileStoragePath) String() string {
	return f.Path
}

// Set sets the path from its string representation.
func (f *FileStoragePath) Set(flagValue string) error {
	if flagValue == "" {
		return errors.New("file storage path must not be empty")
	}
	f.Path = flagValue
	return nil
}

// DatabaseDSN is a structure that represents the DSN for DB connection.
// Implements the Value interface.
type DatabaseDSN struct {
	DSN string
}

// String returns the string representation of the DSN.
func (d *DatabaseDSN) String() string {
	return d.DSN
}

// Set sets the DSN from its string representation.
func (d *DatabaseDSN) Set(flagValue string) error {
	if flagValue == "" {
		return errors.New("database DSN must not be empty")
	}
	d.DSN = flagValue
	return nil
}

// TLSEnabled is a structure that represents the flag of TLS mode for the project.
// Implements the Value interface.
type TLSEnabled struct {
	TLSEnabled bool
}

// String returns the string representation of the flag.
func (e *TLSEnabled) String() string {
	return strconv.FormatBool(e.TLSEnabled)
}

// Set sets the flag from its string representation.
func (e *TLSEnabled) Set(_ string) error {
	e.TLSEnabled = true
	return nil
}

// FileConfig is a structure that represents the path of config file for the project.
// Implements the Value interface.
type FileConfig struct {
	Path string
}

// String returns the string representation of the path.
func (f *FileConfig) String() string {
	return f.Path
}

// Set sets the path of config file from its string representation.
func (f *FileConfig) Set(flagValue string) error {
	if flagValue == "" {
		return errors.New("file config must not be empty")
	}
	f.Path = flagValue
	return nil
}

// TrustedSubnet is a structure that represents the string representation of CIDR to use for access check in internal routers.
// Implements the Value interface.
type TrustedSubnet struct {
	CIDR string
}

// String returns the string representation of the CIDR.
func (t *TrustedSubnet) String() string {
	return t.CIDR
}

// Set sets the string representation of CIDR to the structure.
func (t *TrustedSubnet) Set(flagValue string) error {
	if flagValue == "" {
		return errors.New("trusted subnet must not be empty")
	}
	t.CIDR = flagValue
	return nil
}

// ParseFlags is the function that parses all the command arguments and stores them in the corresponding structures.
func ParseFlags() {
	hostAddr := new(NetAddress)
	baseAddr := new(HTTPAddress)
	fileStoragePath := new(FileStoragePath)
	databaseDSN := new(DatabaseDSN)
	isTLSEnabled := new(TLSEnabled)
	fileConfig := new(FileConfig)
	trustedSubnet := new(TrustedSubnet)

	flag.Var(hostAddr, "a", "Address to host on host:port")
	flag.Var(baseAddr, "b", "base URL for resulting short URL (scheme://host:port)")
	flag.Var(fileStoragePath, "f", "path to file to store short URLs")
	flag.Var(databaseDSN, "d", "DSN to connect to the database")
	flag.Var(isTLSEnabled, "s", "TLS is enabled (default: false)")
	flag.Var(fileConfig, "c", "path to config file")
	flag.Var(trustedSubnet, "t", "trusted subnet to use for access check in internal routers")
	flag.Parse()
	jsonConfig := &Config{}
	var filePath string
	if os.Getenv("CONFIG") != "" {
		filePath = os.Getenv("CONFIG")
	} else if fileConfig.Path != "" {
		filePath = fileConfig.Path
	}
	err := readJSONConfig(jsonConfig, filePath)
	if err != nil {
		os.Exit(1)
	}
	if hostAddr.Host == "" && hostAddr.Port == 0 && jsonConfig.Address == "" {
		hostAddr.Host = "localhost"
		hostAddr.Port = 8080
	} else if jsonConfig.Address != "" {
		setErr := hostAddr.Set(jsonConfig.Address)
		if setErr != nil {
			return
		}
	}
	if baseAddr.Host == "" && baseAddr.Port == 0 && baseAddr.Scheme == "" && jsonConfig.HostedOn == "" {
		baseAddr.Scheme = "http://"
		baseAddr.Host = "localhost"
		baseAddr.Port = 8080
	} else if jsonConfig.Address != "" {
		setErr := baseAddr.Set(jsonConfig.HostedOn)
		if setErr != nil {
			os.Exit(1)
		}
	}
	if fileStoragePath.Path == "" && jsonConfig.FileStoragePath == "" {
		fileStoragePath.Path = "./internal/app/storage/storage.json"
	} else if jsonConfig.FileStoragePath != "" {
		setErr := fileStoragePath.Set(jsonConfig.FileStoragePath)
		if setErr != nil {
			os.Exit(1)
		}
	}
	if databaseDSN.DSN == "" && jsonConfig.DatabaseDSN != "" {
		setErr := databaseDSN.Set(jsonConfig.DatabaseDSN)
		if setErr != nil {
			os.Exit(1)
		}
	}
	if jsonConfig.TLSEnabled {
		setErr := argsConfig.TLSEnabled.Set("true")
		if setErr != nil {
			os.Exit(1)
		}
	}
	argsConfig.Address = *hostAddr
	argsConfig.HostedOn = *baseAddr
	argsConfig.FileStoragePath = *fileStoragePath
	argsConfig.DatabaseDSN = *databaseDSN
	argsConfig.TLSEnabled = *isTLSEnabled
	argsConfig.ConfigFile = *fileConfig
	argsConfig.TrustedSubnet = *trustedSubnet
	Settings = NewConfigFromArgs(argsConfig)
}

func closeWrapper(file *os.File) {
	closeErr := file.Close()
	if closeErr != nil {
		return
	}
}

// GetOrCreateCertAndKey is a function to read existing or generate a pair of pem certificate + private key.
func GetOrCreateCertAndKey() ([]byte, []byte, error) {
	certFile, err := os.OpenFile(Settings.CertPath, os.O_RDWR|os.O_CREATE, 0644)
	defer closeWrapper(certFile)
	if err != nil {
		return nil, nil, err
	}
	keyFile, err := os.OpenFile(Settings.KeyPath, os.O_RDWR|os.O_CREATE, 0644)
	defer closeWrapper(keyFile)
	if err != nil {
		return nil, nil, err
	}
	var certBytes, keyBytes []byte
	certBytes, err = io.ReadAll(certFile)
	if err != nil {
		return nil, nil, err
	}
	keyBytes, err = io.ReadAll(keyFile)
	if err != nil {
		return nil, nil, err
	}
	if len(certBytes) == 0 || len(keyBytes) == 0 {
		certBytes, keyBytes, err = generateCertAndKey()
		if err != nil {
			return nil, nil, err
		}
		_, writeErr := certFile.Write(certBytes)
		if writeErr != nil {
			return nil, nil, err
		}
		_, writeErr = keyFile.Write(keyBytes)
		if writeErr != nil {
			return nil, nil, err
		}
	}
	return certBytes, keyBytes, nil
}

func generateCertAndKey() ([]byte, []byte, error) {
	cert := &x509.Certificate{
		// указываем уникальный номер сертификата
		SerialNumber: big.NewInt(1337),
		// заполняем базовую информацию о владельце сертификата
		Subject: pkix.Name{
			Organization: []string{"Yandex.Praktikum"},
			Country:      []string{"RU"},
		},
		// разрешаем использование сертификата для 127.0.0.1 и ::1
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		// сертификат верен, начиная со времени создания
		NotBefore: time.Now(),
		// время жизни сертификата — 10 лет
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		// устанавливаем использование ключа для цифровой подписи,
		// а также клиентской и серверной авторизации
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	// создаём новый приватный RSA-ключ длиной 4096 бит
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	// создаём сертификат x.509
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	var certPEM bytes.Buffer
	err = pem.Encode(&certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return nil, nil, err
	}

	var privateKeyPEM bytes.Buffer
	err = pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		return nil, nil, err
	}
	return certPEM.Bytes(), privateKeyPEM.Bytes(), nil
}

func readJSONConfig(config *Config, filePath string) error {
	if filePath == "" {
		return nil
	}
	file, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(file) == 0 {
		return nil
	}
	if err = json.Unmarshal(file, &config); err != nil {
		return err
	}
	return nil
}

func init() {
	Settings.Address = "localhost:8080"
	Settings.HostedOn = "http://localhost:8080/"
	Settings.LogLevel = "INFO"
	Settings.FileStoragePath = "./storage.json"
	Settings.DatabaseDSN = ""
	Settings.SecretKey = "DontUseThatInProduction" // Ожидается, что настоящий ключ будет передан через env
	Settings.DeletionBufferFlushIntervalSeconds = 1
	Settings.KeyPath = "./key.pem"
	Settings.CertPath = "./cert.pem"
}
