package teleport

import (
	"context"
	"fmt"
	"strings"

	"github.com/gravitational/teleport/api/client"
	"github.com/gravitational/teleport/api/types"
)

// Client extends the teleport api client
type Client struct {
	client.Client
}

// Note: this is based on the teleport client creation in cluster-test-suites
func New(ctx context.Context, identityFilePath string) (*Client, error) {
	proxyAddr := "teleport.giantswarm.io:443"
	client, err := client.New(ctx, client.Config{
		Addrs: []string{
			proxyAddr,
		},
		Credentials: []client.Credentials{
			client.LoadIdentityFile(identityFilePath),
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = client.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{Client: *client}, nil
}

func (c *Client) GetKubeConfig(ctx context.Context, clusterName string) ([]byte, error) {
	// get the clusters kubernetes server
	server, err := c.GetKubernetesServer(ctx, clusterName)
	if err != nil {
		return nil, err
	}
	cluster := server.GetCluster()
	if !cluster.IsKubeconfig() {
		return nil, fmt.Errorf("cluster %s does not have a kubeconfig", cluster.GetName())
	}

	// get the kubeconfig
	return cluster.GetKubeconfig(), nil
}

func (c *Client) GetKubernetesServer(ctx context.Context, clusterName string) (types.KubeServer, error) {
	servers, err := c.GetKubernetesServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get servers from teleport: %w", err)
	}
	for _, server := range servers {
		if strings.Contains(server.GetName(), clusterName) {
			return server, nil
		}
	}

	// TODO: check if the server is reachable

	return nil, fmt.Errorf("server %s not found in teleport", clusterName)
}
