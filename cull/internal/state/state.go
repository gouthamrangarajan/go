package state

import (
	"encoding/json"
	"os"
)

type State struct {
	UIDValidity     uint32          `json:"uidvalidity"`
	LastUID         uint32          `json:"last_uid"`
	UnsubscribeSent map[uint32]bool `json:"unsubscribed"`
	Junked          map[uint32]bool `json:"junked"`
}

func Load(path string) (State, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return State{UnsubscribeSent: make(map[uint32]bool), Junked: make(map[uint32]bool)}, nil
	}
	if err != nil {
		return State{UnsubscribeSent: make(map[uint32]bool), Junked: make(map[uint32]bool)}, err
	}
	var state State
	err = json.Unmarshal(b, &state)
	if err != nil {
		return State{UnsubscribeSent: make(map[uint32]bool), Junked: make(map[uint32]bool)}, err
	}
	return state, nil
}

func Save(path string, state State) error {
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(path, b, 0644)
	if err != nil {
		return err
	}
	return nil
}
