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
	defaultFileStorageUseSSL = false
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

// ServerConfig описывает конфигурацию HTTP-сервиса.
//
// HTTPServerAddr - адрес HTTP-сервера в формате host:port.
//
// DatabaseURI - строка подключения к базе данных.
//
// FileStorage - настройки S3-совместимого хранилища файлов.
//
// Logging - настройки логирования сервиса.
type ServerConfig struct {
	HTTPServerAddr string      `koanf:"http_address"`
	DatabaseURI    string      `koanf:"database_uri"`
	FileStorage    FileStorage `koanf:"file_storage"`
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
	flags.String("database-uri", "", "database connection URI")
	flags.String("file-storage.endpoint", "", "S3-compatible file storage address")
	flags.String("file-storage.access-key", "", "S3-compatible file storage access key")
	flags.String("file-storage.secret-key", "", "S3-compatible file storage secret key")
	flags.String("file-storage.bucket", "", "S3-compatible file storage bucket")
	flags.Bool("file-storage.use-ssl", false, "use SSL for S3-compatible file storage")
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
		configKey("logging", "format"):       defaultLoggingFormat,
		configKey("logging", "level"):        defaultLoggingLevel,
		configKey("logging", "add_source"):   defaultLoggingAddSource,
		configKey("http_address"):            defaultHTTPServerAddr,
		configKey("file_storage", "use_ssl"): defaultFileStorageUseSSL,
	}
}

func validateServerConfig(cfg ServerConfig) error {
	if err := validateServerAddr(cfg.HTTPServerAddr); err != nil {
		return err
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
	return nil
}
