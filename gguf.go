package main

// Ollama Model Downloader
// This program downloads models from the Ollama registry.

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

// UserAgent is the user agent string used for HTTP requests
const UserAgent = "GGUF-Downloader/1.0 (github.com/emreugur35/ggufDownloader)"

type Manifest struct {
	Layers []Layer `json:"layers"`
}

type Layer struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
}

// ModelInfo represents information about an available model
type ModelInfo struct {
	Name         string
	Description  string
	Parameters   []string
	Capabilities []string
	PullCount    string
	TagCount     string
	UpdatedAt    string
}

func fetchManifest(modelName, modelParameters string) (*Manifest, error) {
	url := fmt.Sprintf("https://registry.ollama.ai/v2/library/%s/manifests/%s", modelName, modelParameters)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch manifest: " + resp.Status)
	}

	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, errors.New("invalid JSON response")
	}

	return &manifest, nil
}

func downloadFile(url, filename string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to download file: " + resp.Status)
	}

	totalSize := resp.ContentLength
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	bar := progressbar.DefaultBytes(totalSize, "Downloading")
	_, err = io.Copy(io.MultiWriter(file, bar), resp.Body)
	return err
}

func fetchAvailableModels() ([]ModelInfo, error) {
	req, err := http.NewRequest("GET", "https://ollama.com/search?o=popular&c=all&q=", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch model list: " + resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var models []ModelInfo
	doc.Find("li[x-test-model]").Each(func(i int, li *goquery.Selection) {
		model := ModelInfo{}

		// Extract model name
		titleSpan := li.Find("span[x-test-search-response-title]")
		model.Name = strings.TrimSpace(titleSpan.Text())

		// Extract description
		descPara := li.Find("p.max-w-lg.break-words.text-neutral-800")
		model.Description = strings.TrimSpace(descPara.Text())

		// Extract parameter options (sizes)
		li.Find("span[x-test-size]").Each(func(_ int, param *goquery.Selection) {
			paramText := strings.TrimSpace(param.Text())
			if paramText != "" {
				model.Parameters = append(model.Parameters, paramText)
			}
		})

		// Extract capabilities
		li.Find("span[x-test-capability]").Each(func(_ int, cap *goquery.Selection) {
			capText := strings.TrimSpace(cap.Text())
			if capText != "" {
				model.Capabilities = append(model.Capabilities, capText)
			}
		})

		// Extract metadata
		pullCountSpan := li.Find("span[x-test-pull-count]")
		model.PullCount = strings.TrimSpace(pullCountSpan.Text())

		tagCountSpan := li.Find("span[x-test-tag-count]")
		model.TagCount = strings.TrimSpace(tagCountSpan.Text())

		updatedAtSpan := li.Find("span[x-test-updated]")
		model.UpdatedAt = strings.TrimSpace(updatedAtSpan.Text())

		if model.Name != "" {
			models = append(models, model)
		}
	})

	return models, nil
}

func displayUsageExamples() {
	fmt.Println(color.CyanString("\nCommand-line Usage Examples:"))
	fmt.Println(color.WhiteString("  # List all available models:"))
	fmt.Println("  ./ggufDownloader")
	fmt.Println("  ./ggufDownloader -list")

	fmt.Println(color.WhiteString("\n  # Download a specific model:"))
	fmt.Println("  ./ggufDownloader -model llama2 -params 7b")
	fmt.Println("  ./ggufDownloader -model phi -params latest")
	fmt.Println("  ./ggufDownloader -model mistral -params 7b-instruct")

	fmt.Println(color.WhiteString("\n  # The downloaded file will be saved as:"))
	fmt.Println("  # modelname:params.gguf (e.g., llama2:7b.gguf)")
}

func displaySimpleUsage() {
	fmt.Println(color.CyanString("\nSimple Usage:"))
	fmt.Println(color.WhiteString("  List models:  ./ggufDownloader -list"))
	fmt.Println(color.WhiteString("  Download:     ./ggufDownloader -model MODEL -params PARAMS"))
	fmt.Println(color.WhiteString("  Help:         ./ggufDownloader -help"))

	// Add some basic examples to the simple usage display
	fmt.Println(color.YellowString("\nQuick Examples:"))
	fmt.Println("  ./ggufDownloader -model llama2 -params 7b")
	fmt.Println("  ./ggufDownloader -model phi -params latest")
}

// printModelsTable prints the models in a table format
func printModelsTable(models []ModelInfo, showDetails bool) {
	// Define column headers and widths
	nameWidth := 20
	sizesWidth := 30
	capabilitiesWidth := 30
	infoWidth := 20

	// Find the max width needed for model names
	for _, model := range models {
		if len(model.Name) > nameWidth-3 {
			nameWidth = len(model.Name) + 3
		}
	}

	// Print table header
	fmt.Println()
	headerFmt := color.CyanString
	fmt.Printf(headerFmt("%-*s", nameWidth, "MODEL"))
	fmt.Printf(headerFmt("%-*s", sizesWidth, "AVAILABLE SIZES"))

	if showDetails {
		fmt.Printf(headerFmt("%-*s", capabilitiesWidth, "CAPABILITIES"))
		fmt.Printf(headerFmt("%-*s", infoWidth, "DOWNLOADS"))
		fmt.Printf(headerFmt("%s", "UPDATED"))
	}
	fmt.Println()

	// Print separator line
	separator := strings.Repeat("-", nameWidth+sizesWidth)
	if showDetails {
		separator += strings.Repeat("-", capabilitiesWidth+infoWidth+20)
	}
	fmt.Println(headerFmt(separator))

	// Print each model
	for _, model := range models {
		// Model name in green
		fmt.Printf(color.GreenString("%-*s", nameWidth, model.Name))

		// Sizes in yellow
		sizes := strings.Join(model.Parameters, ", ")
		if len(sizes) > sizesWidth-3 {
			sizes = sizes[:sizesWidth-6] + "..."
		}
		fmt.Printf(color.YellowString("%-*s", sizesWidth, sizes))

		// Additional details
		if showDetails {
			// Capabilities
			caps := strings.Join(model.Capabilities, ", ")
			if len(caps) > capabilitiesWidth-3 {
				caps = caps[:capabilitiesWidth-6] + "..."
			}
			fmt.Printf(color.CyanString("%-*s", capabilitiesWidth, caps))

			// Pull count
			fmt.Printf(color.WhiteString("%-*s", infoWidth, model.PullCount))

			// Updated date
			fmt.Printf(color.WhiteString("%s", model.UpdatedAt))
		}
		fmt.Println()
	}
}

func main() {
	modelName := flag.String("model", "", "The name of the model to download (e.g., phi3)")
	modelParameters := flag.String("params", "", "The model parameters to use (e.g., 3.8b)")
	listModels := flag.Bool("list", false, "List available models")
	flag.Parse()

	// If no flags provided, or only -list flag is used, show available models
	noArgsProvided := len(os.Args) == 1 // Just the program name, no args
	if noArgsProvided || *listModels {
		models, err := fetchAvailableModels()
		if err != nil {
			fmt.Println(color.RedString("[ERROR] %s", err))
			os.Exit(1)
		}

		// Show the header with a clear separator for better visibility
		fmt.Println(color.CyanString("\n=== Available models from Ollama ==="))

		// Limit the number of models shown in the simple view to avoid overwhelming
		maxModelsToShow := 10
		if noArgsProvided && len(models) > maxModelsToShow {
			modelsToShow := models
			if len(models) > maxModelsToShow {
				modelsToShow = models[:maxModelsToShow]
			}
			printModelsTable(modelsToShow, false)
			fmt.Printf(color.WhiteString("\n... and %d more (use -list to see all)\n"), len(models)-maxModelsToShow)
		} else {
			printModelsTable(models, *listModels) // Show full details when -list is explicitly used
		}

		// Always show usage information, with varying detail based on context
		if noArgsProvided {
			displaySimpleUsage()
		} else {
			displayUsageExamples()
		}
		return
	}

	// Only check for required parameters if we're trying to download a model
	if *modelName == "" || *modelParameters == "" {
		displayUsageExamples()
		fmt.Println(color.RedString("[ERROR] Model name and parameters are required."))
		fmt.Println(color.CyanString("\nRun without arguments to see available models."))
		os.Exit(1)
	}

	manifest, err := fetchManifest(*modelName, *modelParameters)
	if err != nil {
		fmt.Println(color.RedString("[ERROR] %s", err))
		os.Exit(1)
	}

	var modelDigest string
	for _, layer := range manifest.Layers {
		if layer.MediaType == "application/vnd.ollama.image.model" {
			modelDigest = layer.Digest
			break
		}
	}

	if modelDigest == "" {
		fmt.Println(color.RedString("[ERROR] Model digest not found in manifest."))
		os.Exit(1)
	}

	downloadURL := fmt.Sprintf("https://registry.ollama.ai/v2/library/%s/blobs/%s", *modelName, modelDigest)
	outputFilename := fmt.Sprintf("%s:%s.gguf", *modelName, *modelParameters)

	fmt.Println(color.CyanString("[INFO] Downloading %s...", outputFilename))
	if err := downloadFile(downloadURL, outputFilename); err != nil {
		fmt.Println(color.RedString("[ERROR] %s", err))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("[SUCCESS] Download completed: %s", outputFilename))
}
