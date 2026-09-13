package elasticsearch

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/goravel/framework/contracts/config"
)

// NewClient builds an ES client from config (connection name empty = elasticsearch.default).
// Pings once after create.
func NewClient(cfg config.Config, connectionName string) (*elasticsearch.Client, error) {
	if connectionName == "" {
		connectionName = cfg.GetString("elasticsearch.default", "default")
	}

	base := fmt.Sprintf("elasticsearch.connections.%s", connectionName)
	urlsStr := cfg.GetString(base+".urls", "")
	cloudID := cfg.GetString(base+".cloud_id", "")
	if cloudID == "" && strings.TrimSpace(urlsStr) == "" {
		return nil, fmt.Errorf("elasticsearch [%s]: configure urls or cloud_id", connectionName)
	}

	esCfg := elasticsearch.Config{
		Addresses: splitURLs(urlsStr),
		CloudID:   cloudID,
		Username:  cfg.GetString(base+".username", ""),
		Password:  cfg.GetString(base+".password", ""),
		APIKey:    cfg.GetString(base+".api_key", ""),
	}

	if cfg.GetBool(base+".insecure_skip_verify", false) {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
		esCfg.Transport = t
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch [%s] new client: %w", connectionName, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res, err := client.Ping(client.Ping.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("elasticsearch [%s] ping: %w", connectionName, err)
	}
	if res != nil {
		_ = res.Body.Close()
	}
	if res != nil && res.IsError() {
		return nil, fmt.Errorf("elasticsearch [%s] ping status: %s", connectionName, res.Status())
	}
	return client, nil
}

func splitURLs(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		u := strings.TrimSpace(p)
		if u != "" {
			out = append(out, u)
		}
	}
	return out
}

// FullIndexName applies elasticsearch.index_prefix.
func FullIndexName(cfg config.Config, name string) string {
	return cfg.GetString("elasticsearch.index_prefix", "") + name
}
