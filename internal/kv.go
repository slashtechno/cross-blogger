// TODO: When Charm is enabled, call a method of Blogger in watch.go via a ticker that checks if any posts previously known to the program have been deleted. If they have, delete contentDir/<slug>.md, commit, and push. The method should take a pointer to a Markdown object
package internal

import (
	"encoding/json"
	"path/filepath"

	charmclient "github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/kv"
	"github.com/dgraph-io/badger/v3"
	"github.com/slashtechno/cross-blogger/pkg/utils"
)

var Kv *kv.KV

// Attempt to do kv.OpenWithDefaults but in a much more complex manner for the sole reason of being able to set a custom server
func InitializeKv(name string, charmClientConfig charmclient.Config) error {
	if name == "" {
		name = "cross-blogger"
	}

	// Defaults: https://github.com/charmbracelet/charm/blob/f0b0b9512820de64996aecc019933431584cc92d/client/client.go#L29-L36
	charmClientConfig.Host = utils.DefaultString(charmClientConfig.Host, "cloud.charm.sh")
	charmClientConfig.SSHPort = utils.DefaultInt(charmClientConfig.SSHPort, 35353)
	charmClientConfig.HTTPPort = utils.DefaultInt(charmClientConfig.HTTPPort, 35354)
	// Debug default is already false, no need to set
	charmClientConfig.KeyType = utils.DefaultString(charmClientConfig.KeyType, "ed25519")
	// Logfile, DataDir, and IdentityKey defaults are effectively no-operations (no-ops) because they have empty defaults
	cc, err := charmclient.NewClient(&charmClientConfig)
	if err != nil {
		return err
	}

	// --- Begin copied code from kv.OpenWithDefaults (minimal changes) ---
	dd, err := cc.DataPath()
	if err != nil {
		return err
	}
	pn := filepath.Join(dd, "/kv/", name)
	opts := badger.DefaultOptions(pn).WithLoggingLevel(badger.ERROR)

	// By default we have no logger as it will interfere with Bubble Tea
	// rendering. Use Open with custom options to specify one.
	opts.Logger = nil

	// We default to a 10MB vlog max size (which BadgerDB turns into 20MB vlog
	// files). The Badger default results in 2GB vlog files, which is quite
	// large. This will limit the values to 10MB maximum size. If you need more,
	// please use Open with custom options.
	opts = opts.WithValueLogFileSize(10000000)
	// --- End copied code from kv.OpenWithDefaults ---
	kv.Open(cc, name, opts)
	return nil
}

// Marshal an array into JSON and store it in the KV store.
func SetList(key string, value []string) error {
	// Marshal the array into JSON
	if value, err := json.Marshal(value); err != nil {
		return err
	} else {
		// Store the JSON in the KV store
		return Kv.Set([]byte(key), value)
	}
}

// Get an array from the KV store and unmarshal it from JSON.
func GetList(key string) ([]string, error) {
	// Sync
	if err := Kv.Sync(); err != nil {
		return nil, err
	}
	// Get the value from the KV store
	if value, err := Kv.Get([]byte(key)); err != nil {
		return nil, err
	} else {
		// Unmarshal the JSON into an array
		var list []string
		if err := json.Unmarshal(value, &list); err != nil {
			return nil, err
		}
		return list, nil
	}
}

// ViperMapToConfig converts a map[string]interface{} to a charmclient.Config object.
func ViperMapToConfig(viperMap map[string]interface{}) (charmclient.Config, error) {
	// Default configuration
	clientConfig := charmclient.Config{
		Host:        "cloud.charm.sh",
		SSHPort:     35353,
		HTTPPort:    35354,
		Debug:       false,
		Logfile:     "",
		KeyType:     "ed25519",
		DataDir:     "",
		IdentityKey: "",
	}

	// Check if enabled, return early if not
	if enabled, ok := viperMap["enabled"].(bool); !ok || !enabled {
		return charmclient.Config{}, nil
	}

	// Helper function to update config fields if they exist in viperMap
	updateConfig := func(key string, updateFunc func(val interface{})) {
		if val, exists := viperMap[key]; exists {
			updateFunc(val)
		}
	}

	// Update fields from viperMap
	updateConfig("host", func(val interface{}) {
		if v, ok := val.(string); ok {
			clientConfig.Host = v
		}
	})
	updateConfig("ssh_port", func(val interface{}) {
		if v, ok := val.(int); ok {
			clientConfig.SSHPort = v
		}
	})
	updateConfig("http_port", func(val interface{}) {
		if v, ok := val.(int); ok {
			clientConfig.HTTPPort = v
		}
	})
	updateConfig("debug", func(val interface{}) {
		if v, ok := val.(bool); ok {
			clientConfig.Debug = v
		}
	})
	updateConfig("logfile", func(val interface{}) {
		if v, ok := val.(string); ok {
			clientConfig.Logfile = v
		}
	})
	updateConfig("key_type", func(val interface{}) {
		if v, ok := val.(string); ok {
			clientConfig.KeyType = v
		}
	})
	updateConfig("data_dir", func(val interface{}) {
		if v, ok := val.(string); ok {
			clientConfig.DataDir = v
		}
	})
	updateConfig("identity_key", func(val interface{}) {
		if v, ok := val.(string); ok {
			clientConfig.IdentityKey = v
		}
	})

	return clientConfig, nil
}
