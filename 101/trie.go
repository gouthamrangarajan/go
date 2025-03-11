package main

import (
	"fmt"
	"strings"
)

type trieNode struct {
	children    map[rune]*trieNode
	isEndOfWord bool
}

type Trie struct {
	root *trieNode
}

func GetNewTrie() Trie {
	return Trie{}
}

func getNewTrieNode() *trieNode {
	return &trieNode{children: getNewMap()}
}
func getNewMap() map[rune]*trieNode {
	return map[rune]*trieNode{}
}

func (trie *Trie) Search(srchTxt string) []string {
	srchTxt = strings.Trim(srchTxt, "")
	retValues := []string{}
	if trie.root == nil {
		return retValues
	}
	for key, value := range trie.root.children {
		recursiveSearch(value, key, srchTxt, "", &retValues)
	}
	return retValues
}

func recursiveSearch(value *trieNode, key rune, srchTxt string,
	prefix string, retValues *[]string) {
	srchTxt = strings.Trim(srchTxt, "")
	if value.isEndOfWord {
		if srchTxt == "" {
			*retValues = append(*retValues, prefix+string(key))
		} else if len(srchTxt) == 1 && string(key) == srchTxt {
			*retValues = append(*retValues, prefix+string(key))
		}
	}
	srchTxtChars := []rune(srchTxt)
	if srchTxt == "" || (len(srchTxtChars) > 0 && srchTxtChars[0] == key) {
		_, newSrchTxt, _ := strings.Cut(srchTxt, string(key))
		for newKey, newValue := range value.children {
			recursiveSearch(newValue, newKey, newSrchTxt, prefix+string(key), retValues)
		}
	}
}

func (trie *Trie) Print() {
	if trie.root == nil {
		fmt.Printf("Trie is empty\n")
	} else {
		recursivePrint([]rune("∫")[0], trie.root, trie.root.children)
	}
}

func recursivePrint(parent rune, node *trieNode, children map[rune]*trieNode) {
	if node.isEndOfWord {
		fmt.Printf("\n")
	}
	for key, value := range children {
		fmt.Printf("%c => %c |", parent, key)
		recursivePrint(key, value, value.children)
	}
}

func (trie *Trie) Insert(val string) {
	if trie.root == nil {
		trie.root = getNewTrieNode()
	}
	insertChildRecursive(trie.root, strings.Trim(val, ""))
}

func insertChildRecursive(trieNode *trieNode, substr string) {
	if strings.Trim(substr, "") == "" {
		trieNode.isEndOfWord = true
		return
	}
	char := []rune(substr)[0]
	_, subStrFirstCharRemoved, _ := strings.Cut(substr, string(char))
	child, exists := trieNode.children[char]
	if exists {
		insertChildRecursive(child, subStrFirstCharRemoved)
	} else {
		newTrieNode := getNewTrieNode()
		trieNode.children[char] = newTrieNode
		insertChildRecursive(newTrieNode, subStrFirstCharRemoved)
	}
}
