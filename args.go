package main

type BloggerCmd struct {
	BlogAddress string `arg:"positional, required" help:"Blog address to get content from"`
	PostAddress string `arg:"positional, required" help:"Post slug to get content from"`
}

type FileCmd struct {
	Filepath string `arg:"positional, required" help:"Filepath to get content from"`
}

type PublishCmd struct {
	File    *FileCmd    `arg:"subcommand:file" help:"Publish from a file"`
	Blogger *BloggerCmd `arg:"subcommand:blogger" help:"Publish from Blogger"`
	// Perhaps instead of needing both a key and a value for destinations, parse a single value
	// For example, check if it's a file, and if so, check the file ending to determine the type
	// Maybe check if it contains blogger.com
	// Of course, an override would be nice
	Destinations map[string]string `arg:"--destinations, required" help:"Destination(s) to publish to\nAvailable destinations: blogger, markdown, html\nMake sure to specify with <platform>=<Filepath, blog address, etc>"`
	Title        string            `arg:"-t,--title" help:"Specify custom title instead of using the default"`
	DryRun       bool              `arg:"-d,--dry-run" help:"Don't actually publish"`
}

type GoogleOauthCmd struct {
}

var Args struct {
	// Subcommands
	GoogleOauth *GoogleOauthCmd `arg:"subcommand:google-oauth" help:"Store Google OAuth refresh token"`
	Publish     *PublishCmd     `arg:"subcommand:publish" help:"Publish to a destination"`

	// Google OAuth flags
	ClientId     string `arg:"--client-id, env:CLIENT_ID" help:"Google OAuth client ID"`
	ClientSecret string `arg:"--I client-secret, env:CLIENT_SECRET" help:"Google OAuth client secret"`
	RefreshToken string `arg:"--refresh-token, env:REFRESH_TOKEN" help:"Google OAuth refresh token" default:""`

	// Misc flags
	LogLevel string `arg:"--log-level, env:LOG_LEVEL" help:"\"debug\", \"info\", \"warning\", \"error\", or \"fatal\"" default:"info"`
	LogColor bool   `arg:"--log-color, env:LOG_COLOR" help:"Force colored logs" default:"false"`
}
