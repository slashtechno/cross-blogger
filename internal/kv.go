package internal

import (
	"encoding/json"
	"path/filepath"

	charmclient "github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/kv"
	"github.com/dgraph-io/badger/v3"
)

var Kv *kv.KV

// Attempt to do kv.OpenWithDefaults but in a much more complex manner for the sole reason of being able to set a custom server
func InitializeKv(name, host string, sshPort, httpPort int, debug bool, logfile, keyType, dataDir, identityKey string) error {
	if name == "" {
		name = "cross-blogger"
	}

	// TODO: Use defaults (https://github.com/charmbracelet/charm/blob/f0b0b9512820de64996aecc019933431584cc92d/client/client.go#L29-L36)

	cc, err := charmclient.NewClient(
		&charmclient.Config{
			Host:        host,
			SSHPort:     sshPort,
			HTTPPort:    httpPort,
			Debug:       debug,
			Logfile:     logfile,
			KeyType:     keyType,
			DataDir:     dataDir,
			IdentityKey: identityKey,
		},
	)
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
