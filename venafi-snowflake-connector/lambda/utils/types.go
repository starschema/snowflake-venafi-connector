package utils

type VenafiConnectorConfig struct {
	AccessToken string `json:"token,omitempty"`
	TppURL      string `json:"tppUrl,omitempty"`
	Zone        string `json:"zone,omitempty"`
	UPN         string `json:"upn,omitempty"`
	DNSName     string `json:"dnsName,omitempty"`
	CommonName  string `json:"commonName,omitempty"`
	RequestID   string `json:"requestID,omitempty"`
}

type SnowFlakeType struct {
	Data [][]interface{} `json:"data,omitempty"`
}
