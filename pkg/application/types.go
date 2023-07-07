package application

// ClusterValues holds common values for cluster-<provider> charts. These are
// the provider independent values and are present for all the charts
type ClusterValues struct {
	BaseDomain   string              `yaml:"baseDomain"`
	ControlPlane ControlPlane        `yaml:"controlPlane"`
	NodePools    map[string]NodePool `yaml:"nodePools"`
}

type ControlPlane struct {
	Replicas int `yaml:"replicas"`
}

type NodePool struct {
	Replicas int `yaml:"replicas"`
	MaxSize  int `yaml:"maxSize"`
	MinSize  int `yaml:"minSize"`
}

// DefaultAppsValues holds common values for default-apps-<provider> charts. These are
// the provider independent values and are present for all the charts
type DefaultAppsValues struct {
	BaseDomain string `yaml:"baseDomain"`
}
