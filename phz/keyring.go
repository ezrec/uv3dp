package phz

type Keyring struct {
	Init  uint32
	Key   uint32
	index int
}

// Key encoding provided by:
// https://github.com/cbiffle/catibo/blob/master/doc/phz.adoc
func NewKeyring(seed uint32, slice uint32) (kr *Keyring) {
	seed %= 0x4324

	init := seed * 0x34a32231
	key := (slice ^ 0x3fad2212) * (seed * 0x4910913d)

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
