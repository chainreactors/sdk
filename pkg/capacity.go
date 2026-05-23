package sdk

import "github.com/chainreactors/sdk/pkg/types"

type Capacity = types.Capacity

// NewCapacity creates a Capacity bucket with the given total units.
func NewCapacity(total int) *Capacity {
	return types.NewCapacity(total)
}
