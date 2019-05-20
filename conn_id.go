package win

import "sync"

const (
	InvalidId  = 0
	maxConnNum = 1000000
	maxConnId  = 3000000
)

type connId struct {
	usedConnNum   int
	curIndex      int
	idPool        []uint32
	idRecyclePool []uint32
	idRecycleBits []uint64
	mu            sync.Mutex
}

func NewConnId() *connId {

	c := &connId{
		idPool:        make([]uint32, maxConnId),
		idRecyclePool: make([]uint32, 0, maxConnId),
		idRecycleBits: make([]uint64, maxConnId/64+1),
		curIndex:      0,
	}

	var id uint32 = 1
	for i := 0; i < maxConnId; i++ {
		c.idPool[i] = id
		id++
	}
	return c
}

func (c *connId) Get() uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.usedConnNum >= maxConnNum {
		return InvalidId
	}

	if c.curIndex == len(c.idPool) {
		c.idPool, c.idRecyclePool = c.idRecyclePool, c.idPool[:0]
		c.curIndex = 0
		for i := 0; i < len(c.idRecycleBits); i++ {
			c.idRecycleBits[i] = 0
		}
	}

	id := c.idPool[c.curIndex]
	c.curIndex++
	c.usedConnNum++
	return id
}

func (c *connId) Release(id uint32) bool {
	if id == InvalidId || id > maxConnId {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.idRecycleBits[id>>6]&(1<<(id&(64-1))) != 0 {
		return false
	}

	c.idRecyclePool = append(c.idRecyclePool, id)
	c.idRecycleBits[id>>6] |= 1 << (id & (64 - 1))

	if c.usedConnNum > 0 {
		c.usedConnNum--
	}

	return true
}
