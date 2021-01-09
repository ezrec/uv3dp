package fdg

import (
	"testing"
)

func TestKeyring(t *testing.T) {
	kr := NewKeyring(0xdcbe9950, 20)

	if kr.Init != 0x9b8d06bc {
		t.Fatalf("Init: expected %#v, got %#v", uint32(0x9b8d06bc), kr.Init)
	}

	if kr.Key != 0x4749bbec {
		t.Fatalf("Key: expected %#v, got %#v", uint32(0x4749bbec), kr.Key)
	}
}
