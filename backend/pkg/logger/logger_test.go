package logger

import "testing"

func TestNew(t *testing.T) {
	t.Parallel()
	if New() == nil {
		t.Fatal("nil logger")
	}
}
