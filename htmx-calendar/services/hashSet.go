package services

type ElementType interface {
	int | string | int32 | int64 | uint | uint32 | uint64 | bool | float64 | float32
}

type HashSet[T ElementType] map[T]bool

func NewHashSet[T ElementType]() HashSet[T] {
	return make(HashSet[T])
}

func (hs HashSet[T]) Add(element T) {
	hs[element] = true
}

func (hs HashSet[T]) Contains(element T) bool {
	_, ok := hs[element]
	return ok
}

func (hs HashSet[T]) Remove(element T) {
	delete(hs, element)
}

func (hs HashSet[T]) Size() int {
	return len(hs)
}
