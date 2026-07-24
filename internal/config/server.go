package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

const (
	defaultLoggingLevel      = "info"
	defaultLoggingFormat     = "json"
	defaultLoggingAddSource  = false
	defaultHTTPServerAddr    = "localhost:3201"
	defaultTLSCertPath       = "certs/server.crt"
	defaultTLSKeyPath        = "certs/server.key"
	defaultFileStorageUseSSL = false
	defaultQueueExchange     = "goph-profile.avatar"
	defaultQueueProcessQueue = "goph-profile.avatar-processing"
	defaultQueueDeleteQueue  = "goph-profile.avatar-deletion"
	defaultQueueProcessKey   = "avatar.process"
	defaultQueueDeleteKey    = "avatar.delete"
)

// Logging описывает настройки логирования сервиса.
//
// Format - формат логов: text или json.
//
// Level - минимальный уровень логирования: debug, info, warn или error.
//
// AddSource - признак необходимости добавлять в лог место вызова.
type Logging struct {
	Format    string `koanf:"format"`
	Level     string `koanf:"level"`
	AddSource bool   `koanf:"add_source"`
}

// FileStorage описывает настройки S3-совместимого хранилища файлов.
//
// Endpoint - адрес хранилища файлов в формате host:port.
//
// AccessKey - идентификатор ключа доступа к хранилищу файлов.
//
// SecretKey - секретная часть ключа доступа к хранилищу файлов.
//
// Bucket - имя bucket для хранения файлов аватарок.
//
// UseSSL - флаг необходимости использовать HTTPS при подключении к хранилищу файлов.
type FileStorage struct {
	Endpoint  string `koanf:"endpoint"`
	AccessKey string `koanf:"access_key"`
	SecretKey string `koanf:"secret_key"`
	Bucket    string `koanf:"bucket"`
	UseSSL    bool   `koanf:"use_ssl"`
}

// Queue описывает настройки очереди сообщений.
//
// URL - строка подключения к RabbitMQ в формате amqp://user:password@host:port/vhost.
//
// Exchange - имя обменника для задач обработки и удаления аватарок.
//
// AvatarProcessingQueue - очередь задач обработки исходных файлов аватарок.
//
// AvatarDeletionQueue - очередь задач удаления файлов аватарок.
//
// AvatarProcessingRoutingKey - ключ маршрутизации задач обработки исходных файлов.
//
// AvatarDeletionRoutingKey - ключ маршрутизации задач удаления файлов.
type Queue struct {
	URL                        string `koanf:"url"`
	Exchange                   string `koanf:"exchange"`
	AvatarProcessingQueue      string `koanf:"avatar_processing_queue"`
	AvatarDeletionQueue        string `koanf:"avatar_deletion_queue"`
	AvatarProcessingRoutingKey string `koanf:"avatar_processing_routing_key"`
	AvatarDeletionRoutingKey   string `koanf:"avatar_deletion_routing_key"`
}

// ServerConfig описывает конфигурацию HTTP-сервиса.
//
// HTTPServerAddr - адрес HTTP-сервера в формате host:port.
//
// TLSCertPath - путь к TLS-сертификату HTTP-сервера.
//
// TLSKeyPath - путь к приватному TLS-ключу HTTP-сервера.
//
// DatabaseURI - строка подключения к базе данных.
//
// FileStorage - настройки S3-совместимого хранилища файлов.
//
// Queue - настройки очереди сообщений.
//
// Logging - настройки логирования сервиса.
type ServerConfig struct {
	HTTPServerAddr string      `koanf:"http_address"`
	TLSCertPath    string      `koanf:"tls_cert"`
	TLSKeyPath     string      `koanf:"tls_key"`
	DatabaseURI    string      `koanf:"database_uri"`
	FileStorage    FileStorage `koanf:"file_storage"`
	Queue          Queue       `koanf:"queue"`
	Logging        Logging     `koanf:"logging"`
}

// LoadServer читает конфигурацию сервера с учетом приоритета "дефолтное значение < значение из конфигурационного
// файла < переменная окружения < флаг командной строки".
func LoadServer() (ServerConfig, error) {
	flags, configPath, err := parseServerFlags(os.Args[1:])
	if err != nil {
		return ServerConfig{}, err
	}

	return loadConfig[ServerConfig](loadOptions[ServerConfig]{
		Flags:      flags,
		ConfigPath: configPath,
		Defaults:   serverDefaults(),
		Validate:   validateServerConfig,
	})
}

func parseServerFlags(args []string) (*pflag.FlagSet, string, error) {
	flags := pflag.NewFlagSet("goph-profile-server", pflag.ContinueOnError)
	flags.StringP("config", "c", "", "path to config file")
	flags.String("http-address", "", `HTTP server address (default "`+defaultHTTPServerAddr+`")`)
	flags.String("tls-cert", "", "TLS certificate path for HTTP server")
	flags.String("tls-key", "", "TLS private key path for HTTP server")
	addCommonFlags(flags)

	if err := flags.Parse(args); err != nil {
		return nil, "", fmt.Errorf("failed to parse CLI flags: %w", err)
	}
	configPath, err := configPathFromFlags(flags)
	if err != nil {
		return nil, "", err
	}
	return flags, configPath, nil
}

func serverDefaults() map[string]any {
	defaults := commonDefaults()
	defaults[configKey("http_address")] = defaultHTTPServerAddr
	defaults[configKey("tls_cert")] = defaultTLSCertPath
	defaults[configKey("tls_key")] = defaultTLSKeyPath
	return defaults
}

func validateServerConfig(cfg ServerConfig) error {
	if err := validateServerAddr(cfg.HTTPServerAddr); err != nil {
		return err
	}
	if cfg.TLSCertPath == "" {
		return errors.New("TLS certificate path is required")
	}
	if cfg.TLSKeyPath == "" {
		return errors.New("TLS private key path is required")
	}
	return validateCommonConfig(cfg.DatabaseURI, cfg.FileStorage, cfg.Queue)
}
