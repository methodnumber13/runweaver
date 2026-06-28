package aitools

import "testing"

func TestEvaluateAdoptionRunsDoctorAndStartSmoke(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.com/tool\n")
	writeTestFile(t, root, "cmd/tool/main.go", "package main\nfunc main() {}\n")
	if _, err := InitSmartWithOptions(root, InitOptions{
		Force:          true,
		Runtime:        RuntimeAll,
		Classification: ClassifyOptions{Mode: ClassificationDeterministic},
	}); err != nil {
		t.Fatal(err)
	}

	result, err := EvaluateAdoption(root, AdoptionEvalOptions{
		Runtime: RuntimeOpenCode,
		Task:    "Add a small CLI smoke feature",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready || result.Status != "ok" {
		t.Fatalf("eval = %#v, want ready ok", result)
	}
	if result.Start.Action != "created" {
		t.Fatalf("start action = %q, want created", result.Start.Action)
	}
	if result.Doctor.Status != "ok" {
		t.Fatalf("doctor status = %q, want ok", result.Doctor.Status)
	}
}
