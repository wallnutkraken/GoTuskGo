package serial_test

import (
	"testing"

	"github.com/wallnutkraken/gotuskgo/tuskbrain/dbwrap"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/serial"
)

func TestSerialize_Deserializes(t *testing.T) {
	testData := "Hit or miss, I guess they never miss, huh?"
	serialized, err := serial.Marshal([]dbwrap.Message{dbwrap.Message{Content: testData}})
	if err != nil {
		t.Logf("Error marshalling: %s", err.Error())
		t.FailNow()
	}

	unserialized, err := serial.Unmarshal(serialized)
	if err != nil {
		t.Logf("Error unmarshalling: %s", err.Error())
		t.FailNow()
	}

	if len(unserialized) != 1 {
		t.Logf("Deserialized len mismatch, expected 1, got %d", len(unserialized))
		t.FailNow()
	}
	if unserialized[0] != testData {
		t.Logf("Deserialized data mismatch, expected \n[%s]\n, got \n[%s]", testData, unserialized[0])
	}
}
