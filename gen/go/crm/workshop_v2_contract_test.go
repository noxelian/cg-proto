package crmv1

import (
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestStageSystemCodeRoundTrip(t *testing.T) {
	original := &Stage{Id: "stage-1", Name: "Auto in repair", SystemCode: "auto_in_repair"}
	payload, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded Stage
	if err := proto.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := decoded.GetSystemCode(); got != "auto_in_repair" {
		t.Fatalf("system_code = %q, want auto_in_repair", got)
	}
}
