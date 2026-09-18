package types

import "fmt"

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:         DefaultParams(),
		GuardiansetMap: []Guardianset{}, RecoveryMap: []Recovery{}, SafedestinationsMap: []Safedestinations{}}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	guardiansetIndexMap := make(map[string]struct{})

	for _, elem := range gs.GuardiansetMap {
		index := fmt.Sprint(elem.Owner)
		if _, ok := guardiansetIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for guardianset")
		}
		guardiansetIndexMap[index] = struct{}{}
	}
	recoveryIndexMap := make(map[string]struct{})

	for _, elem := range gs.RecoveryMap {
		index := fmt.Sprint(elem.Account)
		if _, ok := recoveryIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for recovery")
		}
		recoveryIndexMap[index] = struct{}{}
	}
	safedestinationsIndexMap := make(map[string]struct{})

	for _, elem := range gs.SafedestinationsMap {
		index := fmt.Sprint(elem.Owner)
		if _, ok := safedestinationsIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for safedestinations")
		}
		safedestinationsIndexMap[index] = struct{}{}
	}

	return gs.Params.Validate()
}
