package internal

import "github.com/spf13/viper"

// CredentialViper is a Viper instance for flags and environment variables
var CredentialViper = viper.New()

// ConfigViper is a Viper instance for configuration
var ConfigViper = viper.New()
