package main

import (
	"io/ioutil"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// config represents a fully parsed configuration file
type config struct {
	SortBy   string
	Format   string
	Timezone *time.Location
	Sources  []sourceConfig
}

// sourceConfig holds configuration parameters of a single source
type sourceConfig struct {
	API          string
	Host         string
	Token        string
	Repositories regexp.Regexp
}

// readConfig reads the configuration file at the given path
func readConfig(path string) (*config, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var unmarshallTarget struct {
		SortBy   string `yaml:"sort_by"`
		Format   string `yaml:"format"`
		Timezone string
		Sources  []struct {
			API          string   `yaml:"api"`
			Host         string   `yaml:"host"`
			Token        string   `yaml:"token"`
			Repositories []string `yaml:"repositories"`
		} `yaml:"sources"`
	}

	err = yaml.Unmarshal(bytes, &unmarshallTarget)
	if err != nil {
		return nil, err
	}

	timezone := time.Local
	if unmarshallTarget.Timezone != "" {
		timezone, err = time.LoadLocation(unmarshallTarget.Timezone)
		if err != nil {
			return nil, err
		}
	}

	c := config{
		SortBy:   strings.ToLower(unmarshallTarget.SortBy),
		Format:   strings.ToLower(unmarshallTarget.Format),
		Timezone: timezone,
		Sources:  make([]sourceConfig, len(unmarshallTarget.Sources)),
	}

	for i, s := range unmarshallTarget.Sources {
		re, err := regexp.Compile(strings.Join(s.Repositories, "|"))
		if err != nil {
			return nil, err
		}

		c.Sources[i] = sourceConfig{
			API:          strings.ToLower(s.API),
			Host:         s.Host,
			Token:        s.Token,
			Repositories: *re,
		}
	}

	return &c, nil
}
