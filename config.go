package main

import "github.com/ilyakaznacheev/cleanenv"

type ConfigVars struct {
	GrpcNodeURL string `yaml:"GrpcNodeUrl" env:"GRPCNODEURL" env-default:"n06.testnet.vega.xyz:3002"`
	Debug       bool   `yaml:"Debug" env:"DEBUG" env-default:"false"`
	DBHost      string `yaml:"DBHost" env:"DBHOST" env-default:"localhost"`
	DBPort      int32  `yaml:"DBPort" env:"DBPORT" env-default:"5432"`
	DBUser      string `yaml:"DBUser" env:"DBUSER" env-default:"postgres"`
	DBPassword  string `yaml:"DBPassword" env:"DBPASSWORD" env-default:"postgres"`
	DBName      string `yaml:"DBName" env:"DBName" env-default:"vega-fees"`
	DBDebug     bool   `yaml:"DBDebug" env:"DBDEBUG" env-default:"true"`
}

// ReadConfig import config struct from yaml file
func ReadConfig(path string) (ConfigVars, error) {
	var cfg ConfigVars
	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}
