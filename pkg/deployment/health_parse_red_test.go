package deployment

import "testing"

// TestParseHealthFromComposePS_UnhealthyIsNotHealthy is a reproduce-first RED test
// for a PASS-bluff in DockerOrchestrator.checkServicesHealth / checkServiceHealth
// (docker_orchestrator.go:324, :522). Those methods judge a service "healthy" with
//
//	strings.Contains(outputStr, "healthy") || strings.Contains(outputStr, "running")
//
// The literal "healthy" is a SUBSTRING of "unhealthy", so a service that
// docker-compose reports as State/Health == "unhealthy" is mis-parsed as healthy.
// waitForServicesHealthy / waitForServiceHealthy then return success and
// DeployWithCompose / UpdateService / RestartService report the deployment as
// completed successfully while a service is actually unhealthy — exactly the
// "tests/flows pass but the feature is broken for the user" PASS-bluff §11.4 forbids.
//
// docker compose ps --format json emits one JSON object per service, e.g.
//
//	{"Name":"main","State":"running","Health":"unhealthy"}
//	{"Name":"main","State":"exited","Health":""}
//
// The parser must report healthy ONLY when no service is unhealthy/down.
func TestParseHealthFromComposePS_UnhealthyIsNotHealthy(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name:   "unhealthy service must NOT be healthy",
			output: `{"Name":"main","State":"running","Health":"unhealthy"}`,
			want:   false,
		},
		{
			name:   "genuinely healthy service is healthy",
			output: `{"Name":"main","State":"running","Health":"healthy"}`,
			want:   true,
		},
		{
			name:   "running with no healthcheck is healthy",
			output: `{"Name":"main","State":"running","Health":""}`,
			want:   true,
		},
		{
			name:   "exited service is not healthy",
			output: `{"Name":"main","State":"exited","Health":""}`,
			want:   false,
		},
		{
			name: "one unhealthy among several is not healthy",
			output: `{"Name":"main","State":"running","Health":"healthy"}
{"Name":"worker-1","State":"running","Health":"unhealthy"}`,
			want: false,
		},
		{
			name:   "empty output (no services) is not healthy",
			output: ``,
			want:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseHealthFromComposePS(tc.output)
			if got != tc.want {
				t.Fatalf("parseHealthFromComposePS(%q) = %v, want %v", tc.output, got, tc.want)
			}
		})
	}
}
