package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Concourse Concourse `yaml:"concourse"`
	Server    Server    `yaml:"server"`
}

type Concourse struct {
	URL                string        `yaml:"url"`
	InsecureSkipVerify bool          `yaml:"insecure_skip_verify"`
	Auth               ConcourseAuth `yaml:"auth"`
}

type ConcourseAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Server struct {
	TLS  TLS    `yaml:"tls"`
	Port uint16 `yaml:"port"`
}

type TLS struct {
	Enabled     bool   `yaml:"enabled"`
	Certificate string `yaml:"certificate"`
	PrivateKey  string `yaml:"private_key"`
}

func Load(filepath string) (*Config, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	yDec := yaml.NewDecoder(f)
	yDec.SetStrict(true)
	ret := &Config{
		Server: Server{Port: 4580},
	}
	err = yDec.Decode(ret)
	return ret, err
}
