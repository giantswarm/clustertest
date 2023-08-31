package application

import "encoding/json"

// ClusterValues holds common values for cluster-<provider> charts. These are
// the provider independent values and are present for all the charts
//
// The `NodePools` property supports both the []Nodepool and map[string]NodePool
// types in the yaml values files and will handle both correctly as a map.
type ClusterValues struct {
	BaseDomain   string       `yaml:"baseDomain"`
	ControlPlane ControlPlane `yaml:"controlPlane"`
	NodePools    NodePools    `yaml:"nodePools"`
}

// NodePools is a special type containing a custom unmarshaller that can handle
// both []Nodepool and map[string]NodePool types in the yaml values.
type NodePools map[string]NodePool

// UnmarshalJSON is a custom unmarshaller than handles both types of NodePools that our
// apps use: []Nodepool and map[string]NodePool. Both will be unmarshalled into a map[string]NodePool
func (np *NodePools) UnmarshalJSON(b []byte) error {
	if b[0] != '[' {
		// We're not dealing with an array so we can assume its the map we expect
		return json.Unmarshal(b, (*map[string]NodePool)(np))
	}
	// We need to unmarshal as an array and then convert to the map we need
	var nps []NodePool
	if err := json.Unmarshal(b, &nps); err != nil {
		return err
	}
	npMap := map[string]NodePool{}
	for _, n := range nps {
		npMap[*n.Name] = n
	}
	*np = NodePools(npMap)
	return nil
}

type ControlPlane struct {
	Replicas int `yaml:"replicas"`
}

type NodePool struct {
	Replicas int     `yaml:"replicas"`
	MaxSize  int     `yaml:"maxSize"`
	MinSize  int     `yaml:"minSize"`
	Name     *string `yaml:"name"`
}

// DefaultAppsValues holds common values for default-apps-<provider> charts. These are
// the provider independent values and are present for all the charts
type DefaultAppsValues struct {
	BaseDomain string `yaml:"baseDomain"`
}
