package config

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

// MigrateConfig описывает конфигурацию процесса миграции базы данных
//
// DatabaseURI - строка подключения к базе данных.
//
// Logging - настройки логирования мигратора.
type MigrateConfig struct {
	DatabaseURI string  `koanf:"database_uri"`
	Logging     Logging `koanf:"logging"`
}

// LoadMigrate читает конфигурацию мигратора с учетом приоритета "дефолтное значение < значение из конфигурационного
// файла < переменная окружения < флаг командной строки".
func LoadMigrate() (MigrateConfig, error) {
	flags, configPath, err := parseMigrateFlags(os.Args[1:])
	if err != nil {
		return MigrateConfig{}, err
	}

	return loadConfig[MigrateConfig](loadOptions[MigrateConfig]{
		Flags:      flags,
		ConfigPath: configPath,
		Defaults:   databaseAndLoggingDefaults(),
		Validate:   validateMigrateConfig,
	})
}

func parseMigrateFlags(args []string) (*pflag.FlagSet, string, error) {
	flags := pflag.NewFlagSet("goph-profile-migrate", pflag.ContinueOnError)
	flags.StringP("config", "c", "", "path to config file")
	addDatabaseAndLoggingFlags(flags)

	if err := flags.Parse(args); err != nil {
		return nil, "", fmt.Errorf("failed to parse CLI flags: %w", err)
	}
	configPath, err := configPathFromFlags(flags)
	if err != nil {
		return nil, "", err
	}
	return flags, configPath, nil
}

func validateMigrateConfig(cfg MigrateConfig) error {
	return validateDatabaseURI(cfg.DatabaseURI)
}
