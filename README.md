# cross-blogger  
Soon-to-be headless CMS for static site generators powered by Google's Blogger.

### Installation  
#### Compiled Binaries  
Compiled binaries can be downloaded from Github Releases.  
#### Compile locally  
To compile this program, run `go build` inside the project root after cloning the repository.  
#### `go install`  
Using `go install`, you can compile and add the program to the PATH.  
Either run `go install github.com/slashtechno/cross-blogger@latest`, follow the same process as compiling the program locally, but replace `go build` with `go install`.  

### Usage  
Sources and destinations should first be configured in the `config.toml` file.  
For Google OAuth, the `--client-id` and `--client-secret` flags are required but can be set as environment variables (`CROSS_BLOGGER_GOOGLE_CLIENT_ID`/`CROSS_BLOGGER_GOOGLE_CLIENT_SECRET`). However these can also be set in the `config.toml` file, passed as environment variables, or put in a `.env` file. When a refresh token is not provided, the program will commence the OAuth flow. This will write the refresh token, along with any other configuration, to the `config.toml` file. If you prefer to use other methods to pass the credentials, you can remove the lines and use the other methods.  
#### Help Output  
From `cross-blogger publish --help` (run `cross-blogger --help` for the root help output):  
```text
Publish to a destination from a source. 
        Specify the source with the first positional argument. 
        The second positional argument is the specifier, such as a Blogger post URL or a file path.
        All arguments after the first are treated as destinations.
        Destinations should be the name of the destinations specified in the config file

Usage:
  cross-blogger publish [flags]
  cross-blogger publish [command]

Available Commands:
  watch       Act as a headless CMS of sorts by watching a source for new content and publishing it to configured destinations.

Flags:
  -r, --dry-run                       Don't actually publish
      --google-client-id string       Google OAuth client ID
      --google-client-secret string   Google OAuth client secret
      --google-refresh-token string   Google OAuth refresh token
  -h, --help                          help for publish

Global Flags:
      --config string      config file path (default "config.toml")
      --log-level string   Set the log level

Use "cross-blogger publish [command] --help" for more information about a command.
```  
