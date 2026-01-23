package config

import (
	"os"

	"github.com/jllopis/kairos/pkg/config"
)

// Load lee la configuración de archivos y flags de la CLI
func Load(args []string) (*config.Config, error) {
	return config.LoadWithCLI(args)
}

// MustLoad es un helper para main que hace panic si hay un error
func MustLoad() *config.Config {
	cfg, err := Load(os.Args[1:])
	if err != nil {
		panic(err)
	}
	return cfg
}
