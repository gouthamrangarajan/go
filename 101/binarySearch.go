package main

type binarySearchDataType interface {
	int | int16 | int32 | int64 | int8 | float32 | float64 | string
}

func BinarySearch[T binarySearchDataType](arr []T, item T) int {
	return binaySearchRecursion(arr, item, 0)
}
func binaySearchRecursion[T binarySearchDataType](arr []T, numberToSearch T, offset int) int {
	if len(arr) == 1 && arr[0] == numberToSearch {
		return offset
	} else if len(arr) > 1 {
		mid := len((arr)) / 2
		if arr[mid] == numberToSearch {
			return mid + offset
		} else if numberToSearch < arr[mid] {
			return binaySearchRecursion(arr[0:mid], numberToSearch, offset)
		} else {
			return binaySearchRecursion(arr[mid:], numberToSearch, offset+mid)
		}
	}
	return -1
}
