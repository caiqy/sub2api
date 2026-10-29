package domain

// GroupExclusiveRateModels limits user-specific rates to exact client model IDs.
// It does not restrict access or interpret wildcard patterns.
type GroupExclusiveRateModels struct {
	Enabled bool     `json:"enabled"`
	Models  []string `json:"models"`
}

func (c GroupExclusiveRateModels) AppliesTo(model string) bool {
	if !c.Enabled {
		return true
	}
	if model == "" {
		return false
	}
	for _, selected := range c.Models {
		if model == selected {
			return true
		}
	}
	return false
}
