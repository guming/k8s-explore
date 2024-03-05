package models

type FrpcClientConfig map[string]interface{}

type Service struct {
	Type       string `toml:"type,omitempty"`
	RemotePort string `toml:"remote_port,omitempty"`
	LocalIP    string `toml:"local_ip,omitempty"`
	LocalPort  string `toml:"local_port,omitempty"`
}

type Common struct {
	ServerAddress string `toml:"server_addr,omitempty"`
	ServerPort    string `toml:"server_port,omitempty"`
}
