# GGUF Downloader

A Go utility for downloading GGUF model files directly from the Ollama registry.

## Overview

GGUF Downloader is a lightweight command-line tool that allows you to:
- Search and list available models from the Ollama registry in a tabular format
- Download GGUF model files directly from the official Ollama registry
- Support `model:tag` shorthand syntax (e.g., `-model llama3:8b`)
- Save downloaded GGUF files with clean, cross-platform filenames

## Installation

### Prerequisites
- Go 1.18 or later

### Building from source
```bash
git clone https://github.com/emreugur35/ggufDownloader
cd ggufDownloader
go build
```

## Usage

### List popular models
```bash
./ggufDownloader
```

This will display a table of the top popular models with their available sizes and parameter options:

```
=== Available models from Ollama ===

MODEL                AVAILABLE SIZES                  
--------------------------------------------------
glm-5.2              latest                           
kimi-k3              latest                           
laguna-s-2.1          latest                           
gemma4               e2b, e4b, 12b, 26b, 31b          
qwen3.5              0.8b, 2b, 4b, 9b, 27b, 35b, 122b 
...
```

### Search models
```bash
./ggufDownloader -search llama
```

### Detailed listing
```bash
./ggufDownloader -list
```

### Download a specific model

You can specify the model and tag/params in several ways:

```bash
# Using model:tag syntax:
./ggufDownloader -model llama3:8b

# Using separate flags:
./ggufDownloader -model llama2 -params 7b

# Defaulting to latest tag:
./ggufDownloader -model phi3

# Custom output filename:
./ggufDownloader -model mistral -params 7b-instruct -out custom_mistral.gguf
```

## Command-line Options

| Option    | Description                                          | Example                         |
|-----------|------------------------------------------------------|---------------------------------|
| `-model`  | The name of the model to download (or `model:tag`)   | `-model llama3:8b`              |
| `-params` | The parameters/tag of the model to download          | `-params 7b`                    |
| `-out`    | Custom output filename for the downloaded GGUF file  | `-out my_model.gguf`            |
| `-search` | Search query for model names                         | `-search llama`                 |
| `-list`   | Show detailed list of available models               | `-list`                         |
| `-help`   | Display help information                             | `-help`                         |

## License

GPL v3
