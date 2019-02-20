package config

import (
	"io/ioutil"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// TargetSet ...
type TargetSet struct {
	URLs        []string `yaml:"urls"`
	Description string   `yaml:"description"`
	Type        string   `yaml:"type"`
	Regexp      string   `yaml:"regexp"`
	MultiLine   bool     `yaml:"multiLine"`
	TimeFormat  string   `yaml:"timeFormat"`
	TimeZone    string   `yaml:"timeZone"`
	Tags        []string `yaml:"tags"`
}

// Target ...
type Target struct {
	URL              string
	Description      string
	Type             string
	Regexp           string
	MultiLine        bool
	TimeFormat       string
	TimeZone         string
	Tags             []string
	Scheme           string
	Host             string
	User             string
	Port             int
	Path             string
	SSHKeyPassphrase []byte
}

// Config ...
type Config struct {
	Targets    []Target
	TargetSets []TargetSet `yaml:"targetSets"`
}

// NewConfig ...
func NewConfig() (*Config, error) {
	return &Config{
		Targets:    []Target{},
		TargetSets: []TargetSet{},
	}, nil
}

// LoadConfigFile ...
func (c *Config) LoadConfigFile(path string) error {
	if path == "" {
		return errors.New("failed to load config file")
	}
	fullPath, err := filepath.Abs(path)
	if err != nil {
		return errors.Wrap(errors.WithStack(err), "failed to load config file")
	}
	buf, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return errors.Wrap(errors.WithStack(err), "failed to load config file")
	}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return errors.Wrap(errors.WithStack(err), "failed to load config file")
	}
	for _, t := range c.TargetSets {
		for _, URL := range t.URLs {
			target := Target{}
			target.URL = URL
			target.Description = t.Description
			target.Type = t.Type
			target.Regexp = t.Regexp
			target.MultiLine = t.MultiLine
			target.TimeFormat = t.TimeFormat
			target.TimeZone = t.TimeZone
			target.Tags = t.Tags

			u, err := url.Parse(URL)
			if err != nil {
				return err
			}
			target.Scheme = u.Scheme
			target.Path = u.Path
			target.User = u.User.Username()
			if strings.Contains(u.Host, ":") {
				splited := strings.Split(u.Host, ":")
				target.Host = splited[0]
				target.Port, _ = strconv.Atoi(splited[1])
			} else {
				target.Host = u.Host
				target.Port = 0
			}
			c.Targets = append(c.Targets, target)
		}
	}
	return nil
}
