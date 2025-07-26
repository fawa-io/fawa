package config

import (
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"sync"

	"github.com/fawa-io/fawa/pkg/fwlog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	Addr      string `mapstructure:"addr"`
	UploadDir string `mapstructure:"uploadDir"`
	CertFile  string `mapstructure:"certFile"`
	KeyFile   string `mapstructure:"keyfile"`
}

var (
	once sync.Once

	mu sync.RWMutex

	config Config
)

func Initconfig() error {
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
	pflag.String("server.addr", "", "List of HTTP service address (e.g., '127.0.0.1:9090')")
	pflag.String("file.uploadDir", "", "Upload files dir")
	pflag.String("server.certFile", "", "Path to the TLS certificate file.")
	pflag.String("server.keyFile", "", "Path to the TLS private key file.")
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

	viper.OnConfigChange(func(e fsnotify.Event) {
		fwlog.Infof("the Profile HasChanged: %s。reloading...", e.Name)

		mu.Lock()
		defer mu.Unlock()

		if err := viper.Unmarshal(&config); err != nil {
			fwlog.Errorf("error Reloading Th eConfiguration: %v", err)
		} else {
			fwlog.Infof("the Configuration Has Been Successfully Reloaded。")
		}
	})
	viper.WatchConfig()

	return nil
}
