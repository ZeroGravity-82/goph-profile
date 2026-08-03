package config

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

const defaultWorkerHealthServerAddr = "localhost:3203"

// WorkerConfig описывает конфигурацию фонового обработчика аватарок.
//
// HealthServerAddr - адрес HTTP-сервера проверок жизнеспособности и готовности в формате host:port.
//
// DatabaseURI - строка подключения к базе данных.
//
// FileStorage - настройки S3-совместимого хранилища файлов.
//
// Queue - настройки очереди сообщений.
//
// Logging - настройки логирования сервиса.
type WorkerConfig struct {
	HealthServerAddr string      `koanf:"health_address"`
	DatabaseURI      string      `koanf:"database_uri"`
	FileStorage      FileStorage `koanf:"file_storage"`
	Queue            Queue       `koanf:"queue"`
	Logging          Logging     `koanf:"logging"`
}

// LoadWorker читает конфигурацию воркера с учетом приоритета "дефолтное значение < значение из конфигурационного
// файла < переменная окружения < флаг командной строки".
func LoadWorker() (WorkerConfig, error) {
	flags, configPath, err := parseWorkerFlags(os.Args[1:])
	if err != nil {
		return WorkerConfig{}, err
	}

	return loadConfig[WorkerConfig](loadOptions[WorkerConfig]{
		Flags:      flags,
		ConfigPath: configPath,
		Defaults:   workerDefaults(),
		Validate:   validateWorkerConfig,
	})
}

func parseWorkerFlags(args []string) (*pflag.FlagSet, string, error) {
	flags := pflag.NewFlagSet("goph-profile-worker", pflag.ContinueOnError)
	flags.StringP("config", "c", "", "path to config file")
	flags.String("health-address", "", `health server address (default "`+defaultWorkerHealthServerAddr+`")`)
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

func workerDefaults() map[string]any {
	defaults := commonDefaults()
	defaults[configKey("health_address")] = defaultWorkerHealthServerAddr
	return defaults
}

func validateWorkerConfig(cfg WorkerConfig) error {
	if err := validateServerAddr(cfg.HealthServerAddr); err != nil {
		return err
	}
	return validateCommonConfig(cfg.DatabaseURI, cfg.FileStorage, cfg.Queue)
}
