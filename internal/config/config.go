package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	APIURL        string `mapstructure:"apiUrl"`
	DefaultTarget string `mapstructure:"defaultTarget"`
	Output        string `mapstructure:"output"`
	Color         string `mapstructure:"color"`
	Dir           string `mapstructure:"-"`
}

func Load(dir string) (*Config, error) {
	if dir == "" {
		var err error
		dir, err = ResolveDir()
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(dir)
	v.SetEnvPrefix("GROUNDS")
	v.SetDefault("apiUrl", DefaultAPIURL)
	v.SetDefault("defaultTarget", DefaultTarget)
	v.SetDefault("output", DefaultOutput)
	v.SetDefault("color", DefaultColor)
	v.AutomaticEnv()
	v.BindEnv("apiUrl", "GROUNDS_API_URL")

	if err := v.ReadInConfig(); err != nil {
		if _, missing := err.(viper.ConfigFileNotFoundError); !missing {
			return nil, err
		}
	}

	cfg := &Config{Dir: dir}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ResolveDir picks the OS-appropriate config directory.
//
//	Linux:   $XDG_CONFIG_HOME/grounds  (default ~/.config/grounds)
//	macOS:   ~/Library/Application Support/grounds
//	Windows: %APPDATA%\grounds
func ResolveDir() (string, error) {
	if v := os.Getenv("GROUNDS_CONFIG_DIR"); v != "" {
		return v, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "grounds"), nil
}
