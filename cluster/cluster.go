package cluster

import "github.com/ntnn/tensile/engines"

type Config struct {
	// Port to listen on for cluster communication.
	DefaultPort int

	// Members are the hostnames or IP addresses of other members.
	// If no port is given DefaultPort will be appended.
	Members []string

	// The pre shared key is used for initial authentication before TLS
	// certificates are exchanged.
	PreSharedKey string
}

type Cluster struct {
	Config *Config
	Engine engines.Engine
}
