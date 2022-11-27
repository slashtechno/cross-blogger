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
In order to use this program, a configuration file must be supplied. Depending on your use case, the configuration varies. When running the program, it will create a basic `config.json` file which you can add the required information to.   
#### CLI Flags  
The default behaviour of the program is to bring up a simple interactive menu. However, command line flags can be used as well. The following are examples, for full usage, run `cross-blogger --help`  
* `cross-blogger --source markdown --source-specifier file.md --title "Post from cross-blogger" --post-to-devto --post-to-html output.html --skip-destination-prompt` - Post to an HTML file and [dev.to](https://dev.to/) with a markdown file as the source without using the interactive menu  
* `cross-blogger --source dev.to --source-specifier https://dev.to/example/example-post-0000 --title "Another post from cross-blogger" --post-to-blogger` - Post to blogger from a dev.to article. This does not prevent the interactive menu from being outputted  