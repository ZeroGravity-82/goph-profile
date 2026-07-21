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
	flags, configPath, err := parseFlags(os.Args[1:])
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

func parseFlags(args []string) (*pflag.FlagSet, string, error) {
	flags := pflag.NewFlagSet("goph-profile-server", pflag.ContinueOnError)
	flags.StringP("config", "c", "", "path to config file")
	flags.String("http-address", "", `HTTP server address (default "`+defaultHTTPServerAddr+`")`)
	flags.String("tls-cert", "", "TLS certificate path for HTTP server")
	flags.String("tls-key", "", "TLS private key path for HTTP server")
	flags.String("database-uri", "", "database connection URI")
	flags.String("file-storage.endpoint", "", "S3-compatible file storage address")
	flags.String("file-storage.access-key", "", "S3-compatible file storage access key")
	flags.String("file-storage.secret-key", "", "S3-compatible file storage secret key")
	flags.String("file-storage.bucket", "", "S3-compatible file storage bucket")
	flags.Bool("file-storage.use-ssl", false, "use SSL for S3-compatible file storage")
	flags.String("queue.url", "", "message queue connection URL")
	flags.String("queue.exchange", "", "message queue exchange name")
	flags.String("queue.avatar-processing-queue", "", "avatar processing queue")
	flags.String("queue.avatar-deletion-queue", "", "avatar deletion queue")
	flags.String("queue.avatar-processing-routing-key", "", "avatar processing routing key")
	flags.String("queue.avatar-deletion-routing-key", "", "avatar deletion routing key")
	flags.String("logging.format", "", "log format: text or json")
	flags.String("logging.level", "", "log level: debug, info, warn or error")
	flags.Bool("logging.add-source", false, "add source location to logs")

	if err := flags.Parse(args); err != nil {
		return nil, "", fmt.Errorf("failed to parse CLI flags: %w", err)
	}
	configPath, err := flags.GetString("config")
	if err != nil {
		return nil, "", fmt.Errorf("failed to read config flag: %w", err)
	}
	return flags, configPath, nil
}

func serverDefaults() map[string]any {
	return map[string]any{
		configKey("logging", "format"):                      defaultLoggingFormat,
		configKey("logging", "level"):                       defaultLoggingLevel,
		configKey("logging", "add_source"):                  defaultLoggingAddSource,
		configKey("http_address"):                           defaultHTTPServerAddr,
		configKey("tls_cert"):                               defaultTLSCertPath,
		configKey("tls_key"):                                defaultTLSKeyPath,
		configKey("file_storage", "use_ssl"):                defaultFileStorageUseSSL,
		configKey("queue", "exchange"):                      defaultQueueExchange,
		configKey("queue", "avatar_processing_queue"):       defaultQueueProcessQueue,
		configKey("queue", "avatar_deletion_queue"):         defaultQueueDeleteQueue,
		configKey("queue", "avatar_processing_routing_key"): defaultQueueProcessKey,
		configKey("queue", "avatar_deletion_routing_key"):   defaultQueueDeleteKey,
	}
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
	if cfg.DatabaseURI == "" {
		return errors.New("database URI is required")
	}
	if cfg.FileStorage.Endpoint == "" {
		return errors.New("file storage endpoint is required")
	}
	if cfg.FileStorage.AccessKey == "" {
		return errors.New("file storage access key is required")
	}
	if cfg.FileStorage.SecretKey == "" {
		return errors.New("file storage secret key is required")
	}
	if cfg.FileStorage.Bucket == "" {
		return errors.New("file storage bucket is required")
	}
	if cfg.Queue.URL == "" {
		return errors.New("queue URL is required")
	}
	if cfg.Queue.Exchange == "" {
		return errors.New("queue exchange is required")
	}
	if cfg.Queue.AvatarProcessingQueue == "" {
		return errors.New("queue avatar processing queue is required")
	}
	if cfg.Queue.AvatarDeletionQueue == "" {
		return errors.New("queue avatar deletion queue is required")
	}
	if cfg.Queue.AvatarProcessingRoutingKey == "" {
		return errors.New("queue avatar processing routing key is required")
	}
	if cfg.Queue.AvatarDeletionRoutingKey == "" {
		return errors.New("queue avatar deletion routing key is required")
	}
	return nil
}
