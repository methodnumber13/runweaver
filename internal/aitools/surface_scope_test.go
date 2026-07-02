package aitools

import "testing"

func TestInferSurfaceScopeTreatsClientBoundaryAsClientSurface(t *testing.T) {
	scope := inferSurfaceScope(
		"client-contract-reviewer",
		"Reviews API client request/response contracts and client compatibility",
		[]string{"frontend/src/client/launch_client.go"},
	)

	if !scope.Frontend {
		t.Fatalf("scope = %#v, want frontend/client surface", scope)
	}
	if scope.Backend {
		t.Fatalf("scope = %#v, want client boundary not to be classified as backend", scope)
	}
}
