package application

import (
	"testing"

	"sigs.k8s.io/yaml"
)

func TestClusterValuesNodePool(t *testing.T) {
	type errorTestCases struct {
		description string
		input       string
		expectError bool
		expected    ClusterValues
	}

	for _, scenario := range []errorTestCases{
		{
			description: "empty values",
			input:       ``,
			expectError: false,
			expected:    ClusterValues{},
		},
		{
			description: "single map",
			input: `nodePools:
  pool1:
    minSize: 1
    maxSize: 1
    replicas: 1`,
			expectError: false,
			expected: ClusterValues{
				NodePools: NodePools{
					"pool1": NodePool{
						Replicas: 1,
						MinSize:  1,
						MaxSize:  1,
					},
				},
			},
		},
		{
			description: "multiple map",
			input: `nodePools:
  pool1:
    minSize: 1
    maxSize: 1
    replicas: 1
  pool2:
    minSize: 2
    maxSize: 2
    replicas: 2`,
			expectError: false,
			expected: ClusterValues{
				NodePools: NodePools{
					"pool1": NodePool{
						Replicas: 1,
						MinSize:  1,
						MaxSize:  1,
					},
					"pool2": NodePool{
						Replicas: 2,
						MinSize:  2,
						MaxSize:  2,
					},
				},
			},
		},
		{
			description: "single array",
			input: `nodePools:
- name: pool1
  minSize: 1
  maxSize: 1
  replicas: 1`,
			expectError: false,
			expected: ClusterValues{
				NodePools: NodePools{
					"pool1": NodePool{
						Replicas: 1,
						MinSize:  1,
						MaxSize:  1,
					},
				},
			},
		},
		{
			description: "multiple array",
			input: `nodePools:
- name: pool1
  minSize: 1
  maxSize: 1
  replicas: 1
- name: pool2
  minSize: 2
  maxSize: 2
  replicas: 2`,
			expectError: false,
			expected: ClusterValues{
				NodePools: NodePools{
					"pool1": NodePool{
						Replicas: 1,
						MinSize:  1,
						MaxSize:  1,
					},
					"pool2": NodePool{
						Replicas: 2,
						MinSize:  2,
						MaxSize:  2,
					},
				},
			},
		},
		{
			description: "not a node pool type",
			input:       `nodePools: 123`,
			expectError: true,
			expected:    ClusterValues{},
		},
	} {
		t.Run(scenario.description, func(t *testing.T) {
			actual := &ClusterValues{}
			err := yaml.Unmarshal([]byte(scenario.input), actual)
			if err != nil && !scenario.expectError {
				t.Fatalf("Didn't expect an error but there was one - %s", err)
			} else if err == nil && scenario.expectError {
				t.Fatalf("Expected an error but there wasn't one")
			}
			if len(actual.NodePools) != len(scenario.expected.NodePools) {
				t.Errorf("Result didn't have expected number of node pools. Expected %d, Actual %d", len(actual.NodePools), len(scenario.expected.NodePools))
			}
		})
	}
}

func TestClusterValues(t *testing.T) {
	type errorTestCases struct {
		description string
		input       string
		expectError bool
		expected    ClusterValues
	}

	for _, scenario := range []errorTestCases{
		{
			description: "empty values",
			input:       ``,
			expectError: false,
			expected:    ClusterValues{},
		},
		{
			description: "old schema",
			input: `baseDomain: "foo.com"
controlPlane:
  replicas: 3`,
			expectError: false,
			expected: ClusterValues{
				BaseDomain: "foo.com",
				ControlPlane: ControlPlane{
					Replicas: 3,
				},
			},
		},
		{
			description: "old schema with nodepools map",
			input: `baseDomain: "foo.com"
controlPlane:
  replicas: 3
nodePools:
  pool1:
    minSize: 1
    maxSize: 1
    replicas: 1`,
			expectError: false,
			expected: ClusterValues{
				BaseDomain: "foo.com",
				ControlPlane: ControlPlane{
					Replicas: 3,
				},
				NodePools: NodePools{
					"pool1": NodePool{
						MinSize:  1,
						MaxSize:  1,
						Replicas: 1,
					},
				},
			},
		},
		{
			description: "old schema with nodepools array",
			input: `baseDomain: "foo.com"
controlPlane:
  replicas: 3
nodePools:
  - name: pool1
    minSize: 1
    maxSize: 1
    replicas: 1`,
			expectError: false,
			expected: ClusterValues{
				BaseDomain: "foo.com",
				ControlPlane: ControlPlane{
					Replicas: 3,
				},
				NodePools: NodePools{
					"pool1": NodePool{
						MinSize:  1,
						MaxSize:  1,
						Replicas: 1,
					},
				},
			},
		},
		{
			description: "new schema",
			input: `global:
  connectivity:
    baseDomain: foo.com
  controlPlane:
    replicas: 3`,
			expectError: false,
			expected: ClusterValues{
				BaseDomain: "foo.com",
				ControlPlane: ControlPlane{
					Replicas: 3,
				},
			},
		},
		{
			description: "new schema with nodepool",
			input: `global:
  connectivity:
    baseDomain: foo.com
  controlPlane:
    replicas: 3
  nodePools:
    pool1:
      minSize: 1
      maxSize: 1
      replicas: 1`,
			expectError: false,
			expected: ClusterValues{
				BaseDomain: "foo.com",
				ControlPlane: ControlPlane{
					Replicas: 3,
				},
				NodePools: NodePools{
					"pool1": NodePool{
						MinSize:  1,
						MaxSize:  1,
						Replicas: 1,
					},
				},
			},
		},
		{
			description: "old schema but has a 'global' property",
			input: `global:
  metadata: {}
baseDomain: "foo.com"
controlPlane:
  replicas: 3`,
			expectError: false,
			expected: ClusterValues{
				BaseDomain: "foo.com",
				ControlPlane: ControlPlane{
					Replicas: 3,
				},
			},
		},
	} {
		t.Run(scenario.description, func(t *testing.T) {
			actual := &ClusterValues{}
			err := yaml.Unmarshal([]byte(scenario.input), actual)
			if err != nil && !scenario.expectError {
				t.Fatalf("Didn't expect an error but there was one - %s", err)
			} else if err == nil && scenario.expectError {
				t.Fatalf("Expected an error but there wasn't one")
			}
			if actual.BaseDomain != scenario.expected.BaseDomain {
				t.Errorf("Result didn't have expected base domain. Expected %d, Actual %d", len(actual.BaseDomain), len(scenario.expected.BaseDomain))
			}
			if actual.ControlPlane.Replicas != scenario.expected.ControlPlane.Replicas {
				t.Errorf("Result didn't have expected control plane replicas. Expected %d, Actual %d", actual.ControlPlane.Replicas, scenario.expected.ControlPlane.Replicas)
			}
			if len(actual.NodePools) != len(scenario.expected.NodePools) {
				t.Errorf("Result didn't have expected number of node pools. Expected %d, Actual %d", len(actual.NodePools), len(scenario.expected.NodePools))
			}
			for key, val := range scenario.expected.NodePools {
				np, ok := actual.NodePools[key]
				if !ok {
					t.Errorf("Result didn't have expected node pool name. Expected %s", key)
				} else {
					if val.MaxSize != np.MaxSize {
						t.Errorf("Result NodePool didn't have expected MaxSize. Expected %d, Actual %d", val.MaxSize, np.MaxSize)
					}
					if val.MinSize != np.MinSize {
						t.Errorf("Result NodePool didn't have expected MinSize. Expected %d, Actual %d", val.MinSize, np.MinSize)
					}
					if val.Replicas != np.Replicas {
						t.Errorf("Result NodePool didn't have expected Replicas. Expected %d, Actual %d", val.Replicas, np.Replicas)
					}
				}
			}
		})
	}
}
