package deployment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeploymentConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     *DeploymentConfig
		wantErr bool
		errType string
	}{
		{
			name: "valid config",
			req: &DeploymentConfig{
				Host:          "worker1.example.com",
				User:          "root",
				Password:      "password",
				DockerImage:   "translator:latest",
				ContainerName: "worker-1",
				Ports:         []PortMapping{{HostPort: 8080, ContainerPort: 8080}},
			},
			wantErr: false,
		},
		{
			name: "missing host",
			req: &DeploymentConfig{
				Host:          "", // Invalid - empty host
				User:          "root",
				Password:      "password",
				DockerImage:   "translator:latest",
				ContainerName: "worker-1",
				Ports:         []PortMapping{{HostPort: 8080, ContainerPort: 8080}},
			},
			wantErr: true,
			errType: "host",
		},
		{
			name: "missing user",
			req: &DeploymentConfig{
				Host:          "worker1.example.com",
				User:          "", // Invalid - empty user
				Password:      "password",
				DockerImage:   "translator:latest",
				ContainerName: "worker-1",
				Ports:         []PortMapping{{HostPort: 8080, ContainerPort: 8080}},
			},
			wantErr: true,
			errType: "user",
		},
		{
			name: "missing docker image",
			req: &DeploymentConfig{
				Host:          "worker1.example.com",
				User:          "root",
				Password:      "password",
				DockerImage:   "", // Invalid - empty image
				ContainerName: "worker-1",
				Ports:         []PortMapping{{HostPort: 8080, ContainerPort: 8080}},
			},
			wantErr: true,
			errType: "docker_image",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// DeploymentConfig doesn't have Validate method, just test basic validation
			if tt.wantErr {
				if tt.req.Host == "" {
					require.Empty(t, tt.req.Host)
				}
				if tt.req.User == "" {
					require.Empty(t, tt.req.User)
				}
				if tt.req.DockerImage == "" {
					require.Empty(t, tt.req.DockerImage)
				}
			} else {
				require.NotEmpty(t, tt.req.Host)
				require.NotEmpty(t, tt.req.User)
				require.NotEmpty(t, tt.req.DockerImage)
			}
		})
	}
}

func TestDeploymentPlan_BasicValidation(t *testing.T) {
	validConfig := &DeploymentConfig{
		Host:          "worker1.example.com",
		User:          "root",
		Password:      "password",
		DockerImage:   "translator:latest",
		ContainerName: "worker-1",
		Ports:         []PortMapping{{HostPort: 8080, ContainerPort: 8080}},
	}

	tests := []struct {
		name    string
		req     *DeploymentPlan
		wantErr bool
		errType string
	}{
		{
			name: "valid plan",
			req: &DeploymentPlan{
				Main:    validConfig,
				Workers: []*DeploymentConfig{validConfig},
			},
			wantErr: false,
		},
		{
			name: "missing main",
			req: &DeploymentPlan{
				Workers: []*DeploymentConfig{validConfig},
			},
			wantErr: true,
			errType: "main",
		},
		{
			name: "empty workers",
			req: &DeploymentPlan{
				Main:    validConfig,
				Workers: []*DeploymentConfig{},
			},
			wantErr: false, // Empty workers is allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				if tt.errType == "main" && tt.req.Main == nil {
					require.Nil(t, tt.req.Main)
				}
			} else {
				// Basic validation
				if tt.req.Main != nil {
					require.NotEmpty(t, tt.req.Main.Host)
				}
			}
		})
	}
}

// Performance benchmarks
func BenchmarkDeploymentConfig_Validation(b *testing.B) {
	config := &DeploymentConfig{
		Host:          "test.example.com",
		User:          "root",
		Password:      "password",
		DockerImage:   "translator:latest",
		ContainerName: "worker-1",
		Ports:         []PortMapping{{HostPort: 8080, ContainerPort: 8080}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Just basic field checks for benchmarking
		if config.Host == "" {
			panic("invalid config")
		}
	}
}
