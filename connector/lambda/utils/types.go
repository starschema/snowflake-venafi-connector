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

const MachineIDTypeTLS = "TLS"

type SnowFlakeType struct {
	Data [][]interface{} `json:"data,omitempty"`
}

const LIST_MID_TYPE = "list"
const REQUEST_MID_TYPE = "request"
const GET_MID_TYPE = "get"
const GET_STATUS_MID_TYPE = "status"
const RENEW_MID_TYPE = "renew"
const REVOKE_MID_TYPE = "revoke"
