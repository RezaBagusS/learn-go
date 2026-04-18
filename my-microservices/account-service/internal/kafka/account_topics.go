package kafka

import (
	"log"
	"net"
	"strconv"

	"github.com/segmentio/kafka-go"
)

const (
	TopicAccountCreated = "account.created"
	TopicAccountFailed  = "account.failed"
)

// EnsureTopics memastikan topic yang dibutuhkan sudah ada di Kafka
func EnsureTopics(brokers []string) {
	if len(brokers) == 0 {
		return
	}

	// Koneksi ke broker pertama
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		log.Printf("⚠️ Gagal menghubungi Kafka untuk cek topic: %v", err)
		return
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		log.Printf("⚠️ Gagal mendapatkan controller Kafka: %v", err)
		return
	}

	controllerConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		log.Printf("⚠️ Gagal menghubungi controller Kafka: %v", err)
		return
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{
		{Topic: TopicAccountCreated, NumPartitions: 1, ReplicationFactor: 1},
		{Topic: TopicAccountFailed, NumPartitions: 1, ReplicationFactor: 1},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		// Kita abaikan jika errornya "Topic Already Exists"
		log.Printf("ℹ️ Info Kafka: %v (mungkin sudah ada)", err)
	} else {
		log.Println("✅ Kafka Topics berhasil diverifikasi/dibuat.")
	}
}
