# cross-blogger  
Cross-service (and cross-platform) blog posting utility  

### Installation  
#### Compiled Binaries  
Compiled binaries can be downloaded from Github Releases.  
#### Compile locally  
To compile this program, run `go build` inside the project root after cloning the repository.  
#### `go install`  
Using `go install`, you can compile and add the program to the PATH.  
Either run `go install github.com/slashtechno/cross-blogger@latest`, follow the same process as compiling the program locally, but replace `go build` with `go install`.  

### Usage  
To use this program, just run the executable in the terminal.  
#### Help Output  
From `cross-blogger --help`  
```text
Usage: cross-blogger.exe --client-id CLIENT-ID --client-secret CLIENT-SECRET [--refresh-token REFRESH-TOKEN] [--log-level LOG-LEVEL] [--log-color] <command> [<args>]
    
Options:
  --client-id CLIENT-ID
                         Google OAuth client ID [env: CLIENT_ID]
  --client-secret CLIENT-SECRET
                         Google OAuth client secret [env: CLIENT_SECRET]
  --refresh-token REFRESH-TOKEN
                         Google OAuth refresh token [env: REFRESH_TOKEN]
  --log-level LOG-LEVEL
                         "debug", "info", "warning", "error", or "fatal" [default: info, env: LOG_LEVEL]
  --log-color            Force colored logs [default: false, env: LOG_COLOR]
  --help, -h             display this help and exit

Commands:
  google-oauth           Store Google OAuth refresh token
  publish                Publish to a destination
```  
