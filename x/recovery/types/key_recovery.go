package types

import "cosmossdk.io/collections"

// RecoveryKey is the prefix to retrieve all Recovery
var RecoveryKey = collections.NewPrefix("recovery/value/")
