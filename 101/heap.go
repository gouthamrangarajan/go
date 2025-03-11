package main

import (
	"errors"
	"fmt"
)

type heapDataType interface {
	int | int16 | int32 | int64 | int8 | float32 | float64 | string
}

type Heap[T heapDataType] struct {
	data   []T
	length int
}

func GetNewHeap[T heapDataType]() Heap[T] {
	return Heap[T]{[]T{}, 0}
}

func (heap *Heap[T]) Print() {
	for idx := 0; idx < heap.length; idx++ {
		fmt.Printf("%v ", heap.data[idx])
	}
	fmt.Println()
}
func (heap *Heap[T]) Insert(item T) {
	heap.data = append(heap.data, item)
	heapifyUp(&heap.data, heap.length)
	heap.length++
}
func heapifyUp[T heapDataType](data *[]T, index int) {
	arr := *data
	if len(arr) > 0 {
		pIndex := getParentIndex(index)
		pEl := arr[pIndex]
		val := arr[index]
		if pEl > val {
			arr[index] = pEl
			arr[pIndex] = val
			if pIndex != 0 {
				heapifyUp(data, pIndex)
			}
		}
	}
}
func (heap *Heap[T]) Remove() (T, error) {
	var dataToRet T
	if heap.length == 0 {
		return dataToRet, errors.New("Heap is empty")
	}
	dataToRet = heap.data[0]
	heap.length--
	if heap.length == 0 {
		return dataToRet, nil
	}
	heap.data[0] = heap.data[heap.length]
	heapifyDown(&heap.data, 0, heap.length)
	return dataToRet, nil
}
func heapifyDown[T heapDataType](data *[]T, index int, length int) {
	arr := *data
	val := arr[index]
	lChildIndex := getLeftChildIndex(index)
	rChildIndex := getRightChildIndex(index)
	if lChildIndex+1 > length {
		return
	}
	lChild := arr[lChildIndex]
	var rChild T
	swapLeftChild := false
	swapRightChild := false
	if rChildIndex+1 > length {
		rChild = arr[rChildIndex]
		if val > lChild && val > rChild {
			if lChild > rChild {
				swapRightChild = true
			} else {
				swapLeftChild = true
			}

		} else if val > lChild {
			swapLeftChild = true
		} else if val > rChild {
			swapRightChild = true
		}
	} else if val > lChild {
		swapLeftChild = true
	}
	if swapLeftChild {
		arr[index] = lChild
		arr[lChildIndex] = val
		heapifyDown(&arr, lChildIndex, length)
	} else if swapRightChild {
		arr[index] = rChild
		arr[rChildIndex] = val
		heapifyDown(&arr, rChildIndex, length)
	}
}
func getParentIndex(index int) int {
	if index < 1 {
		return 0
	}
	return (index - 1) / 2
}

func getLeftChildIndex(index int) int {
	return (index * 2) + 1
}

func getRightChildIndex(index int) int {
	return (index * 2) + 2
}
