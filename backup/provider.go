package backup

import "fmt"

func providerFor(sourceType string) (Provider, error) {
	switch sourceType {
	case "mysql":
		return MySQLBackupProvider{}, nil
	case "postgres":
		return PostgreSQLBackupProvider{}, nil
	case "mongo":
		return MongoBackupProvider{}, nil
	case "redis":
		return RedisBackupProvider{}, nil
	case "file":
		return FileBackupProvider{}, nil
	case "config":
		return ConfigBackupProvider{}, nil
	default:
		return nil, fmt.Errorf("unsupported provider %q", sourceType)
	}
}
