package config

import (
	"errors"
	"fmt"
	"maps"

	"github.com/spf13/pflag"
)

func addCommonFlags(flags *pflag.FlagSet) {
	addDatabaseAndLoggingFlags(flags)
	addFileStorageFlags(flags)
	addQueueFlags(flags)
}

func addDatabaseAndLoggingFlags(flags *pflag.FlagSet) {
	flags.String("database-uri", "", "database connection URI")
	flags.String("logging.format", "", "log format: text or json")
	flags.String("logging.level", "", "log level: debug, info, warn or error")
	flags.Bool("logging.add-source", false, "add source location to logs")
}

func addFileStorageFlags(flags *pflag.FlagSet) {
	flags.String("file-storage.endpoint", "", "S3-compatible file storage address")
	flags.String("file-storage.access-key", "", "S3-compatible file storage access key")
	flags.String("file-storage.secret-key", "", "S3-compatible file storage secret key")
	flags.String("file-storage.bucket", "", "S3-compatible file storage bucket")
	flags.Bool("file-storage.use-ssl", false, "use SSL for S3-compatible file storage")
}

func addQueueFlags(flags *pflag.FlagSet) {
	flags.String("queue.url", "", "message queue connection URL")
	flags.String("queue.exchange", "", "message queue exchange name")
	flags.String("queue.avatar-processing-queue", "", "avatar processing queue")
	flags.String("queue.avatar-deletion-queue", "", "avatar deletion queue")
	flags.String("queue.avatar-processing-routing-key", "", "avatar processing routing key")
	flags.String("queue.avatar-deletion-routing-key", "", "avatar deletion routing key")
}

func configPathFromFlags(flags *pflag.FlagSet) (string, error) {
	configPath, err := flags.GetString("config")
	if err != nil {
		return "", fmt.Errorf("failed to read config flag: %w", err)
	}
	return configPath, nil
}

func commonDefaults() map[string]any {
	defaults := databaseAndLoggingDefaults()
	maps.Copy(defaults, fileStorageDefaults())
	maps.Copy(defaults, queueDefaults())
	return defaults
}

func databaseAndLoggingDefaults() map[string]any {
	return map[string]any{
		configKey("logging", "format"):     defaultLoggingFormat,
		configKey("logging", "level"):      defaultLoggingLevel,
		configKey("logging", "add_source"): defaultLoggingAddSource,
	}
}

func fileStorageDefaults() map[string]any {
	return map[string]any{
		configKey("file_storage", "use_ssl"): defaultFileStorageUseSSL,
	}
}

func queueDefaults() map[string]any {
	return map[string]any{
		configKey("queue", "exchange"):                      defaultQueueExchange,
		configKey("queue", "avatar_processing_queue"):       defaultQueueProcessQueue,
		configKey("queue", "avatar_deletion_queue"):         defaultQueueDeleteQueue,
		configKey("queue", "avatar_processing_routing_key"): defaultQueueProcessKey,
		configKey("queue", "avatar_deletion_routing_key"):   defaultQueueDeleteKey,
	}
}

func validateCommonConfig(databaseURI string, fileStorage FileStorage, queue Queue) error {
	if err := validateDatabaseURI(databaseURI); err != nil {
		return err
	}
	if err := validateFileStorage(fileStorage); err != nil {
		return err
	}
	return validateQueue(queue)
}

func validateDatabaseURI(databaseURI string) error {
	if databaseURI == "" {
		return errors.New("database URI is required")
	}
	return nil
}

func validateFileStorage(fileStorage FileStorage) error {
	if fileStorage.Endpoint == "" {
		return errors.New("file storage endpoint is required")
	}
	if fileStorage.AccessKey == "" {
		return errors.New("file storage access key is required")
	}
	if fileStorage.SecretKey == "" {
		return errors.New("file storage secret key is required")
	}
	if fileStorage.Bucket == "" {
		return errors.New("file storage bucket is required")
	}
	return nil
}

func validateQueue(queue Queue) error {
	if queue.URL == "" {
		return errors.New("queue URL is required")
	}
	if queue.Exchange == "" {
		return errors.New("queue exchange is required")
	}
	if queue.AvatarProcessingQueue == "" {
		return errors.New("queue avatar processing queue is required")
	}
	if queue.AvatarDeletionQueue == "" {
		return errors.New("queue avatar deletion queue is required")
	}
	if queue.AvatarProcessingRoutingKey == "" {
		return errors.New("queue avatar processing routing key is required")
	}
	if queue.AvatarDeletionRoutingKey == "" {
		return errors.New("queue avatar deletion routing key is required")
	}
	return nil
}
