package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	APIURL           string `mapstructure:"apiUrl" yaml:"apiUrl"`
	DefaultTarget    string `mapstructure:"defaultTarget" yaml:"defaultTarget"`
	DefaultProjectID string `mapstructure:"defaultProjectId" yaml:"defaultProjectId,omitempty"`
	Output           string `mapstructure:"output" yaml:"output"`
	Color            string `mapstructure:"color" yaml:"color"`
	Dir              string `mapstructure:"-" yaml:"-"`
}

func Load(dir string) (*Config, error) {
	return load(dir, true)
}

func LoadFile(dir string) (*Config, error) {
	return load(dir, false)
}

func load(dir string, includeEnv bool) (*Config, error) {
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
	v.SetDefault("defaultProjectId", "")
	v.SetDefault("output", DefaultOutput)
	v.SetDefault("color", DefaultColor)
	if includeEnv {
		v.AutomaticEnv()
		v.BindEnv("apiUrl", "GROUNDS_API_URL")
	}

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

func Save(dir string, cfg *Config) error {
	if dir == "" {
		var err error
		dir, err = ResolveDir()
		if err != nil {
			return err
		}
	}
	if cfg == nil {
		cfg = &Config{}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
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
