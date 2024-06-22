# cross-blogger  
Headless CMS for static site generators powered by Google's Blogger.
It can be used, at the time of writing, to publish **between** the following destinations. However, the intention is to output to static site generators, especially Hugo.  
- Blogger  
- Markdown (with frontmatter)  

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
By default, `credentials.yaml` is used to store the Google OAuth credentials and `config.toml` is used to store the configuration. These will be generated with placeholders/defaults if they do not exist. You can specify both the path to the credentials file and the path to the config file using the `--credentials-file` and `--config` flags. The file extension will dictate the format of the file. Command-line flags can also be used. Environment variables can be used for credentials and the log level although they should be prefixed with `CROSS_BLOGGER_`. If credentials are not provided through the credentials file **and the refresh token is not passed**, the credentials will be written to the credentials file as a byproduct of the refresh token being stored. It's always possible to just pass the refresh token, once obtained, some other way to prevent the credentials from being written.  
Docker can be used by placing configuration files in `config/` and running `docker compose up -d` (`-d` runs it in the background). For additional configuration, the `docker-compose.yml` file can be edited.
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
      --dry-run                       Dry run - don't actually push the data
      --google-client-id string       Google OAuth client ID
      --google-client-secret string   Google OAuth client secret
      --google-refresh-token string   Google OAuth refresh token
  -h, --help                          help for publish

Global Flags:
      --config string             config file path (default "config.toml")
      --credentials-file string   credentials file path (default "credentials.yaml")
      --log-level string          Set the log level

Use "cross-blogger publish [command] --help" for more information about a command.
```  
