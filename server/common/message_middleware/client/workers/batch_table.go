package message_middleware

const INITIAL_SIZE = 1000

type BatchTable struct {
	table         map[string][][]byte
	size          uint
	sizeLimit     uint
	flushCallback func(string, [][]byte)
}

func createBatchTable(callback func(string, [][]byte), sizeLimit uint) BatchTable {
	return BatchTable{
		table:         make(map[string][][]byte, INITIAL_SIZE),
		size:          0,
		sizeLimit:     sizeLimit,
		flushCallback: callback,
	}
}

func (self *BatchTable) addEntry(key string, value []byte) {
	_, exists := self.table[key]
	if exists {
		self.table[key] = append(self.table[key], value)
	} else {
		self.table[key] = [][]byte{value}
	}

	self.size += uint(len(value))

	if self.size >= self.sizeLimit {
		self.flush()
	}
}

func (self *BatchTable) isEmpty() bool {
	return self.size == 0
}

func (self *BatchTable) flush() {
	if self.isEmpty() {
		return
	}

	for k, v := range self.table {
		self.flushCallback(k, v)
	}

	self.table = make(map[string][][]byte, INITIAL_SIZE)
	self.size = 0
}
