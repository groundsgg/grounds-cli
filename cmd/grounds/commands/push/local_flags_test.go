package push

import "testing"

func TestPushDefinesLocalOverrideFlags(t *testing.T) {
	cmd := newPush()

	if flag := cmd.Flag("local"); flag == nil {
		t.Fatal("expected --local flag")
	}
	if flag := cmd.Flag("with-local"); flag == nil {
		t.Fatal("expected --with-local flag")
	}
}
