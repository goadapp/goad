package client

import (
	"encoding/json"
	"net/url"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"golang.org/x/net/context"
)

// ConfigList returns the list of configs.
func (cli *Client) ConfigList(ctx context.Context, options types.ConfigListOptions) ([]swarm.Config, error) {
	query := url.Values{}

	if options.Filters.Len() > 0 {
		filterJSON, err := filters.ToParam(options.Filters)
		if err != nil {
			return nil, err
		}

		query.Set("filters", filterJSON)
	}

	resp, err := cli.get(ctx, "/configs", query, nil)
	if err != nil {
		return nil, err
	}

	var configs []swarm.Config
	err = json.NewDecoder(resp.body).Decode(&configs)
	ensureReaderClosed(resp)
	return configs, err
}
