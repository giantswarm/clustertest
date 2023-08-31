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
