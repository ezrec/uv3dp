package phz

import (
	"testing"
)

func TestKeyring(t *testing.T) {
	kr := NewKeyring(0xdcbe9950, 20)

	if kr.Init != 0xe7012f20 {
		t.Fatalf("Init: expected %#v, got %#v", uint32(0xe7012f20), kr.Init)
	}

	if kr.Key != 0xafa1abc0 {
		t.Fatalf("Key: expected %#v, got %#v", uint32(0xafa1abc0), kr.Key)
	}
}
