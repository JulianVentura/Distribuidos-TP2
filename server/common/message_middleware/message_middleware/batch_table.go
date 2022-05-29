package message_middleware

const INITIAL_SIZE = 1000

type BatchTable struct {
	table          map[string][]string
	size           uint
	size_limit     uint
	flush_callback func(string, []string)
}

func CreateBatchTable(callback func(string, []string), size_limit uint) BatchTable {
	return BatchTable{
		table:          make(map[string][]string, INITIAL_SIZE),
		size:           0,
		size_limit:     size_limit,
		flush_callback: callback,
	}
}

func (self *BatchTable) Add_entry(key string, value string) {
	_, exists := self.table[key]
	if exists {
		self.table[key] = append(self.table[key], value)
	} else {
		self.table[key] = []string{value}
	}

	self.size += uint(len(value))

	if self.size >= self.size_limit {
		self.Flush()
	}
}

func (self *BatchTable) Is_empty() bool {
	return self.size == 0
}

func (self *BatchTable) Flush() {
	if self.Is_empty() {
		return
	}

	for k, v := range self.table {
		self.flush_callback(k, v)
	}

	self.table = make(map[string][]string, INITIAL_SIZE)
	self.size = 0
}
