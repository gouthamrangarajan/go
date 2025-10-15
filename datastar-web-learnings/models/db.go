package models

import "time"

type FirebaseConfig struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
	UniverseDomain          string `json:"universe_domain"`
}

type GetVideosRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type VideoResponse struct {
	Rank     int       `json:"rank"`
	Title    string    `json:"title"`
	VideoId  string    `json:"videoId"`
	Subtitle string    `json:"subtitle,omitempty"`
	Tags     []string  `json:"tags"`
	Created  time.Time `json:"createdAt"`
}
