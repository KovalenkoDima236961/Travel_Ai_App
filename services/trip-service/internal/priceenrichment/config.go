package priceenrichment

import "strings"

type Config struct {
	Enabled              bool
	FailOpen             bool
	OverwriteAICosts     bool
	OverwriteManualCosts bool
	MinMatchConfidence   float64
	MaxItems             int
	DefaultCurrency      string
}

func (c Config) normalized() Config {
	if c.MinMatchConfidence <= 0 {
		c.MinMatchConfidence = 0.55
	}
	if c.MinMatchConfidence > 1 {
		c.MinMatchConfidence = 1
	}
	if c.MaxItems <= 0 {
		c.MaxItems = 30
	}
	c.DefaultCurrency = strings.ToUpper(strings.TrimSpace(c.DefaultCurrency))
	if c.DefaultCurrency == "" {
		c.DefaultCurrency = "EUR"
	}
	return c
}
