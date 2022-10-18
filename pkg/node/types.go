package node

type NodeSchema struct {
	Name           string `json:"name" validate:"required,alphanum"`
	IPv4           string `json:"ipv4" validate:"required,ip"`
	Connections    int    `json:"connections"`
	MaxConnections int    `json:"maxConnections" validate:"required,number"`
	Port           int    `json:"port" validate:"required"`
	Scheme         string `json:"scheme" validate:"required"`
}
