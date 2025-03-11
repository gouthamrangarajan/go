package main

import "fmt"

func main() {
	performBinarySearch()
	performHeapOperations()
	performTrieOperations()
}

func performTrieOperations() {
	trie := GetNewTrie()
	trie.Insert("Cattle")
	trie.Insert("Cat")
	trie.Insert("Pot")
	trie.Insert("Luck")
	trie.Insert("Pottery")
	// trie.Print()
	// fmt.Printf("Trie Print completed\n")
	srchResults := trie.Search("Cat")
	for _, txt := range srchResults {
		fmt.Printf("%v\n", txt)
	}
}

func performHeapOperations() {
	heap := GetNewHeap[int]()
	heap.Insert(4)
	heap.Insert(2)
	heap.Insert(3)
	heap.Insert(1)
	heap.Print()
	el, err := heap.Remove()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("%v\n", el)
	}
	heap.Print()
}

func performBinarySearch() {
	arr := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21}
	numToFind := 2
	res := BinarySearch(arr, numToFind)
	fmt.Printf("number %d found at index %d\n", numToFind, res)
	if res > -1 {
		fmt.Printf("number at index %d is %d\n", res, arr[res])
	}
}
