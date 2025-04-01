# GGUF Downloader

A Go utility for downloading GGUF model files from the Ollama registry.

## Overview

GGUF Downloader is a simple command-line tool that allows you to:
- List available models from the Ollama registry in a tabular format
- Download specific model versions in GGUF format

## Installation

### Prerequisites
- Go 1.16 or later

### Building from source
```bash
git clone https://github.com/emreugur35/ggufDownloader
cd ggufDownloader
go build
```

## Usage

### List all available models
```bash
./ggufDownloader
```

This will display a table of the most popular models with their available sizes:

```
=== Available models from Ollama ===

MODEL                AVAILABLE SIZES                  
--------------------------------------------------
llama2               7b, 13b, 70b                     
phi                  3b, mini                         
mistral              7b, 7b-instruct, 7b-instruct-v0.2
dolphin-phi          2.7b                             
neural-chat          7b                               
vicuna               7b, 13b, 33b                     
llama3               8b, 70b                          
codellama            7b, 13b, 34b                     
orca-mini            3b, 7b, 13b                      
gemma                2b, 7b                           

... and more (use -list to see all)
```

### List all models with details
```bash
./ggufDownloader -list
```

This will display an extended table with capabilities, download counts, and update dates:

```
=== Available models from Ollama ===

MODEL                AVAILABLE SIZES                  CAPABILITIES                      DOWNLOADS            UPDATED
-----------------------------------------------------------------------------------------------------------------
llama2               7b, 13b, 70b                     chat, vision                      1.2M                 3 weeks ago
phi                  3b, mini                         code, math                        856K                 1 month ago
mistral              7b, 7b-instruct, 7b-instruct-v0.2 chat, reasoning                   742K                 2 weeks ago
...
```

### Download a specific model
```bash
./ggufDownloader -model llama2 -params 7b
```

This will download the specified model and save it as `llama2:7b.gguf` in the current directory.

## Command-line Options

| Option    | Description                                          | Example                         |
|-----------|------------------------------------------------------|---------------------------------|
| `-model`  | The name of the model to download                    | `-model llama2`                 |
| `-params` | The parameters/size of the model to download         | `-params 7b`                    |
| `-list`   | Show detailed list of all available models           | `-list`                         |
| `-help`   | Display help information                             | `-help`                         |

## Examples

### Quick model download
```bash
./ggufDownloader -model phi -params latest
```

### Download a specific model version
```bash
./ggufDownloader -model mistral -params 7b-instruct
```

## License

GPL v3
