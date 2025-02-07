package api

import "fmt"

// HTTP || https://pkg.go.dev/github.com/michaelklishin/rabbit-hole/v2#section-readme

func (re *RabbitMqClient) createHttpUri(index int) string {
	var protocol string
	if re.Config.Tls {
		protocol = "https://"
	} else {
		//goland:noinspection HttpUrlsUsage
		protocol = "http://"
	}
	return fmt.Sprintf("%s%s:%d", protocol, re.Config.Urls[index], re.Config.Port)
}
