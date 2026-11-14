package webhook

import (
	"testing"
)

func TestDriftNoteMissingField(t *testing.T) {
	healthy := [][]string{
		{"order", "order.id", "order.rider.phone"},
		{"order", "order.id", "order.rider.phone"},
	}
	failed := []string{"order", "order.id"}
	note := DriftNote(failed, healthy)
	want := "Field 'order.rider.phone' is missing compared with recent healthy webhooks."
	if note != want {
		t.Fatalf("got %q want %q", note, want)
	}
}

func TestDriftNoteRename(t *testing.T) {
	healthy := [][]string{{"order", "order.id", "order.rider.phone"}}
	failed := []string{"order", "order.id", "order.rider.phoneNumber"}
	note := DriftNote(failed, healthy)
	want := "Field 'order.rider.phone' looks renamed to 'order.rider.phoneNumber'."
	if note != want {
		t.Fatalf("got %q want %q", note, want)
	}
}

func TestDriftNoteNewField(t *testing.T) {
	healthy := [][]string{{"order", "order.id"}}
	failed := []string{"order", "order.id", "order.extra"}
	note := DriftNote(failed, healthy)
	want := "New field 'order.extra' appeared that healthy webhooks did not have."
	if note != want {
		t.Fatalf("got %q want %q", note, want)
	}
}

func TestDriftNoteNoBaseline(t *testing.T) {
	if note := DriftNote([]string{"a"}, nil); note != "" {
		t.Fatalf("expected empty note, got %q", note)
	}
}

func TestExtractKeys(t *testing.T) {
	body := []byte(`{"order": {"id": 1, "rider": {"phone": "x"}}, "ok": true}`)
	keys := ExtractKeys(body)
	want := map[string]bool{
		"order": true, "order.id": true, "order.rider": true, "order.rider.phone": true, "ok": true,
	}
	if len(keys) != len(want) {
		t.Fatalf("got %v", keys)
	}
	for _, k := range keys {
		if !want[k] {
			t.Fatalf("unexpected key %s", k)
		}
	}
}

func TestExtractKeysNonJSON(t *testing.T) {
	if keys := ExtractKeys([]byte("not json")); len(keys) != 0 {
		t.Fatalf("expected no keys, got %v", keys)
	}
}
