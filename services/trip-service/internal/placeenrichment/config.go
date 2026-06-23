package placeenrichment

// Config controls automatic place enrichment behavior.
type Config struct {
	MinConfidence     float64
	MaxItems          int
	OverwriteExisting bool
	FailOpen          bool
}

func (c Config) normalized() Config {
	if c.MinConfidence <= 0 {
		c.MinConfidence = 0.75
	}
	if c.MinConfidence > 1 {
		c.MinConfidence = 1
	}
	if c.MaxItems <= 0 {
		c.MaxItems = 20
	}
	return c
}
