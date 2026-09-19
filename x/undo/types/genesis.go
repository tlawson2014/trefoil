package types

import "fmt"

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:      DefaultParams(),
		PendingList: []Pending{}, RecordMap: []Record{}}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	pendingIdMap := make(map[uint64]bool)
	pendingCount := gs.GetPendingCount()
	for _, elem := range gs.PendingList {
		if _, ok := pendingIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for pending")
		}
		if elem.Id >= pendingCount {
			return fmt.Errorf("pending id should be lower or equal than the last id")
		}
		pendingIdMap[elem.Id] = true
	}
	recordIndexMap := make(map[string]struct{})

	for _, elem := range gs.RecordMap {
		index := fmt.Sprint(elem.Index)
		if _, ok := recordIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for record")
		}
		recordIndexMap[index] = struct{}{}
	}

	return gs.Params.Validate()
}
