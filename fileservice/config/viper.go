// Copyright 2025 The fawa Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"errors"
	"fmt"
	"sync"

	"github.com/fawa-io/fwpkg/fwlog"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	Addr     string `mapstructure:"addr"`
	CertFile string `mapstructure:"certFile"`
	KeyFile  string `mapstructure:"keyFile"`
	LogLevel string `mapstructure:"logLevel"`
}

var (
	once sync.Once

	mu sync.RWMutex

	config Config
)

func InitConfig() error {
	var initErr error
	once.Do(func() {
		initErr = LoadAndWatch()
	})
	return initErr
}

func Get() Config {
	mu.RLock()
	defer mu.RUnlock()
	return config
}

func LoadAndWatch() error {
	pflag.String("addr", "", "List of HTTP service address (e.g., '127.0.0.1:9090')")
	pflag.String("certFile", "", "Path to the TLS certificate file.")
	pflag.String("keyFile", "", "Path to the TLS private key file.")
	pflag.Parse()

	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return fmt.Errorf("failed to bind pflags: %w", err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/fawa/")

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			fwlog.Infof("Config file not found.")
		} else {
			return fmt.Errorf("fatal error config file: %w", err)
		}
	}

	mu.Lock()
	if err := viper.Unmarshal(&config); err != nil {
		mu.Unlock()
		return fmt.Errorf("the initial configuration cannot be decoded into the struct: %w", err)
	}
	mu.Unlock()

	viper.SetDefault("addr", "127.0.0.1:8080")
	viper.SetDefault("uploadDir", "./upload")
	viper.SetDefault("certFile", "")
	viper.SetDefault("keyFile", "")
	viper.SetDefault("logLevel", "info")

	viper.OnConfigChange(func(e fsnotify.Event) {
		fwlog.Infof("the Profile HasChanged: %s。reloading...", e.Name)

		mu.Lock()
		defer mu.Unlock()

		if err := viper.Unmarshal(&config); err != nil {
			fwlog.Errorf("Error while reloading config: %v", err)
		} else {
			newLogLevel, err := fwlog.ParseLevel(config.LogLevel)
			if err != nil {
				fwlog.Warnf("New log level in config is invalid: %v. Keeping previous level.", err)
			} else {
				fwlog.SetLevel(newLogLevel)
				fwlog.Infof("Log level reloaded successfully to: %s", config.LogLevel)
			}
		}
	})
	viper.WatchConfig()

	return nil
}
