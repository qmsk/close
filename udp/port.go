package udp

import (
    "math/rand"
)

type RandPort struct {
    rand    *rand.Rand
    base    uint16
    mask    uint16
}

func (self *RandPort) init(seed int64) {
    self.rand = rand.New(rand.NewSource(seed))
}

func (self *RandPort) SetPort(port uint) {
    if port == 0 {
        self.base = uint16(self.rand.Uint32())
    } else {
        self.base = uint16(port)
    }
}

func (self *RandPort) SetRandom(bits uint) {
    if bits > 0 {
        self.mask = uint16((1<<bits) - 1)
        self.base = self.base & ^self.mask
    } else {
        self.mask = 0
    }
}

func (self *RandPort) Port() uint16 {
    if self.mask == 0 {
        return self.base
    } else {
        return self.base | (self.mask & uint16(self.rand.Uint32()))
    }
}
