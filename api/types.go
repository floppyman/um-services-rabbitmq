package api

import (
	"time"

	rabbithole "github.com/michaelklishin/rabbit-hole/v2"
)

type RabbitMqClient struct {
	Config RabbitMqClientConfig
	Cli    *rabbithole.Client
}

type RabbitMqClientConfig struct {
	Urls                     []string `json:"Urls"`
	Port                     int      `json:"Port" default:"15672"`
	Tls                      bool     `json:"Tls" default:"true"`
	TlsCertificateCommonName string   `json:"TlsCertificateCommonName"`
	VirtualHost              string   `json:"VirtualHost"`
	Username                 string   `json:"Username"`
	Password                 string   `json:"Password"`
}

type RabbitMqClusterNodeStatus struct {
	Name      string
	IsRunning bool
}

type RabbitMqQueueItem struct {
	QueueId           int
	QueueName         string
	OnlineTimestamp   time.Time
	OnlineQueueLength int
}
