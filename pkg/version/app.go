package version

// AppVersion is the single authoritative application version reported by every
// binary in this module. It MUST be kept equal to the repository-root VERSION
// file (CLAUDE.md: VERSION is authoritative). The binding is enforced
// mechanically by TestAppVersionMatchesVERSIONFile in app_test.go, which reads
// the VERSION file and fails the build if this constant diverges.
//
// Do NOT hardcode per-binary version literals. Every cmd/*/main.go MUST report
// version.AppVersion so all binaries agree with VERSION.
const AppVersion = "2.3.1"
