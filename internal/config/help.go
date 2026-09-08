package config

import "github.com/spf13/pflag"

func (c *Config) PrintHelp() {
	pflag.PrintDefaults()
}

func (c *Config) ShowHelp() bool {
	return pflag.Lookup("help") != nil && pflag.Lookup("help").Changed
}
