package main

import "testing"

func TestRepeatedStringFlagValuesDeduplicateAndDropEmptyItems(t *testing.T) {
	var values repeatedStringFlag
	for _, value := range []string{" auth ", "", "billing", "auth"} {
		if err := values.Set(value); err != nil {
			t.Fatalf("set repeated flag: %v", err)
		}
	}

	got := values.Values()
	want := []string{"auth", "billing"}
	if len(got) != len(want) {
		t.Fatalf("Values() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Values() = %#v, want %#v", got, want)
		}
	}
}

func TestRepeatedStringFlagNilValuesAreSafe(t *testing.T) {
	var values *repeatedStringFlag
	if got := values.String(); got != "" {
		t.Fatalf("nil String() = %q, want empty", got)
	}
	if got := values.Values(); got != nil {
		t.Fatalf("nil Values() = %#v, want nil", got)
	}
}

func TestSplitCSVTrimsAndDeduplicates(t *testing.T) {
	got := splitCSV("auth, billing,auth,, platform ")
	want := []string{"auth", "billing", "platform"}
	if len(got) != len(want) {
		t.Fatalf("splitCSV() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitCSV() = %#v, want %#v", got, want)
		}
	}
}
