package api

import (
	"crypto/tls"
	"math/rand"
	"net/http"
	"time"

	rabbithole "github.com/michaelklishin/rabbit-hole/v2"
)

//goland:noinspection GoUnusedExportedFunction
func NewRabbitMqHttpClient(conn RabbitMqClientConfig) (*RabbitMqClient, error) {
	randEp := rand.Intn(len(conn.Urls)-1) + 1
	client, err := NewRabbitMqHttpClientForIndex(conn, randEp)
	if err != nil {
		return nil, err
	}
	return client, nil
}

//goland:noinspection GoUnusedExportedFunction
func NewRabbitMqHttpClientForIndex(conn RabbitMqClientConfig, index int) (*RabbitMqClient, error) {
	client := &RabbitMqClient{Config: conn}

	tlsConfig := http.DefaultTransport.(*http.Transport).TLSClientConfig
	if conn.Tls {
		tlsConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: false,
			ClientAuth:         tls.NoClientCert,
		}
		if conn.TlsCertificateCommonName != "" {
			tlsConfig.ServerName = conn.TlsCertificateCommonName
		}
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}

	var err error
	client.Cli, err = rabbithole.NewTLSClient(client.createHttpUri(index), client.Config.Username, client.Config.Password, transport)
	if err != nil {
		return nil, err
	}

	return client, nil
}

//goland:noinspection GoUnusedExportedFunction
func (re *RabbitMqClient) QueueStats() ([]RabbitMqQueueItem, error) {
	queues, err := re.Cli.ListQueues()
	if err != nil {
		return make([]RabbitMqQueueItem, 0), err
	}

	res := make([]RabbitMqQueueItem, 0)

	for _, queue := range queues {
		res = append(res, RabbitMqQueueItem{
			QueueName:         queue.Name,
			OnlineQueueLength: queue.Messages,
			OnlineTimestamp:   time.Now(),
		})
	}

	return res, nil
}

//goland:noinspection GoUnusedExportedFunction
func (re *RabbitMqClient) NodeStatus() ([]RabbitMqClusterNodeStatus, error) {
	infos, err := re.Cli.ListNodes()
	if err != nil {
		return make([]RabbitMqClusterNodeStatus, 0), err
	}

	res := make([]RabbitMqClusterNodeStatus, 0)
	for _, i := range infos {
		res = append(res, RabbitMqClusterNodeStatus{
			Name:      i.Name,
			IsRunning: i.IsRunning,
		})
	}

	return res, err
}

//goland:noinspection GoUnusedExportedFunction
func (re *RabbitMqClient) HealthCheckResources() (rabbithole.ResourceAlarmCheckStatus, error) {
	status, err := re.Cli.HealthCheckAlarms()
	if err != nil {
		return rabbithole.ResourceAlarmCheckStatus{}, err
	}

	return status, nil
}

//goland:noinspection GoUnusedExportedFunction
func (re *RabbitMqClient) HealthCheckCluster() (rabbithole.HealthCheckStatus, rabbithole.HealthCheckStatus, error) {
	mirrorStatus, err := re.Cli.HealthCheckNodeIsMirrorSyncCritical()
	if err != nil {
		return rabbithole.HealthCheckStatus{}, rabbithole.HealthCheckStatus{}, err
	}

	quorumStatus, err := re.Cli.HealthCheckNodeIsQuorumCritical()
	if err != nil {
		return rabbithole.HealthCheckStatus{}, rabbithole.HealthCheckStatus{}, err
	}

	return mirrorStatus, quorumStatus, nil
}

//goland:noinspection GoUnusedExportedFunction
func (re *RabbitMqClient) HealthCheckCertificate(within uint, unit rabbithole.TimeUnit) (rabbithole.HealthCheckStatus, error) {
	status, err := re.Cli.HealthCheckCertificateExpiration(within, unit)
	if err != nil {
		return rabbithole.HealthCheckStatus{}, err
	}
	return status, nil
}

func (re *RabbitMqClient) ListConnections() ([]rabbithole.ConnectionInfo, error) {
	connections, err := re.Cli.ListConnections()
	if err != nil {
		return make([]rabbithole.ConnectionInfo, 0), err
	}
	return connections, nil
}
