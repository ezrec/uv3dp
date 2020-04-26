package ctb

import (
	"testing"
)

func TestKeyring(t *testing.T) {
	kr := NewKeyring(0xdcbe9950, 20)

	if kr.Init != 0x4d6c45e3 {
		t.Fatalf("Init: expected %#v, got %#v", uint32(0x4d6c45e3), kr.Init)
	}

	if kr.Key != 0xa2bb7353 {
		t.Fatalf("Key: expected %#v, got %#v", uint32(0xa2bb7353), kr.Key)
	}
}
