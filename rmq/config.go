package rmq

type ConfigRmq struct {
	Endpoints                []string `json:"endpoints"`
	ManagementExtension      string   `json:"management-extension"`
	ManagementPort           int      `json:"management-port"`
	QueueExtension           string   `json:"queue-extension"`
	QueuePort                int      `json:"queue-port"`
	TlsCertificateCommonName string   `json:"tls-certificate-common-name"`
	TlsEnabled               bool     `json:"tls-enabled"`
	Username                 string   `json:"username"`
	Password                 string   `json:"password"`
	VirtualHost              string   `json:"virtual-host"`
}

type ConfigRmqQueues struct {
	Alerts           string `json:"alerts"`
	ExportCanbus     string `json:"export-canbus"`
	ExportMessages   string `json:"export-messages"`
	ExportPositions  string `json:"export-positions"`
	ExportTacho      string `json:"export-tacho"`
	Geocoding        string `json:"geocoding"`
	Pois             string `json:"pois"`
	Reports          string `json:"reports"`
	CallcenterAlarms string `json:"callcenter-alarms"`
}
