package fdg

type Keyring struct {
	Init  uint32
	Key   uint32
	index int
}

// Key encoding was very similar to, but had different constants than:
// https://github.com/cbiffle/catibo/blob/master/doc/cbddlp-ctb.adoc
func NewKeyring(seed uint32, slice uint32) (kr *Keyring) {
	init := (seed - 0x1dcb76c3) ^ 0x257e2431
	key := init * 0x82391efd * (slice ^ 0x110bdacd)

	kr = &Keyring{
		Init: init,
		Key:  key,
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
