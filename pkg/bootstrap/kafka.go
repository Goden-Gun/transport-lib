package bootstrap

import "github.com/Goden-Gun/transport-lib/pkg/kafka"

// InitKafka initializes a shared Kafka manager.
func InitKafka(cfg kafka.Config) (*kafka.Manager, error) {
	return kafka.NewManager(cfg)
}
