package ctb

type Keyring struct {
	Init  uint32
	Key   uint32
	index int
}

// Key encoding provided by:
// https://github.com/cbiffle/catibo/blob/master/doc/cbddlp-ctb.adoc
func NewKeyring(seed uint32, slice uint32) (kr *Keyring) {
	init := uint64(seed)*0x2d83cdac + 0xd8a83423
	key := (uint32(uint64(slice)*0x1e1530cd) + uint32(0xec3d47cd)) * uint32(init)

	kr = &Keyring{
		Init: uint32(init),
		Key:  uint32(key),
	}

	return
}

func (kr *Keyring) Next() (k byte) {
	k = byte(kr.Key >> (8 * kr.index))
	kr.index += 1
	if kr.index&3 == 0 {
		kr.Key += kr.Init
		kr.index = 0
	}

	return
}

func (kr *Keyring) Read(buff []byte) (size int, err error) {
	for n := range buff {
		buff[n] = kr.Next()
	}

	return
}
