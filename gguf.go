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
	Size      int64  `json:"size"`
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
	registryPath := modelName
	if !strings.Contains(registryPath, "/") {
		registryPath = "library/" + registryPath
	}

	url := fmt.Sprintf("https://registry.ollama.ai/v2/%s/manifests/%s", registryPath, modelParameters)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.ollama.image.manifest.v1+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("model or tag not found: %s:%s (HTTP 404)", modelName, modelParameters)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch manifest: %s", resp.Status)
	}

	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, errors.New("invalid manifest JSON response")
	}

	return &manifest, nil
}

func downloadFile(url, filename string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	totalSize := resp.ContentLength
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	bar := progressbar.DefaultBytes(totalSize, "Downloading")
	_, err = io.Copy(io.MultiWriter(file, bar), resp.Body)
	if err != nil {
		_ = os.Remove(filename)
		return err
	}
	return nil
}

func isParameterSize(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return false
	}
	if strings.HasSuffix(s, "b") || strings.HasSuffix(s, "m") {
		numPart := strings.TrimSuffix(strings.TrimSuffix(s, "b"), "m")
		numPart = strings.TrimPrefix(numPart, "e")
		var f float64
		if _, err := fmt.Sscanf(numPart, "%f", &f); err == nil {
			return true
		}
	}
	sizes := map[string]bool{
		"mini": true, "small": true, "medium": true, "large": true, "full": true, "tiny": true,
	}
	return sizes[s]
}

func fetchAvailableModels(searchQuery string) ([]ModelInfo, error) {
	url := "https://ollama.com/search?o=popular&c=all&q=" + strings.TrimSpace(searchQuery)
	req, err := http.NewRequest("GET", url, nil)
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
	seen := make(map[string]bool)

	doc.Find("a[href^=\"/library/\"]").Each(func(i int, a *goquery.Selection) {
		href, exists := a.Attr("href")
		if !exists {
			return
		}

		rawName := strings.TrimPrefix(href, "/library/")
		nameParts := strings.Split(rawName, ":")
		modelName := strings.TrimSpace(nameParts[0])

		if modelName == "" || seen[modelName] {
			return
		}
		seen[modelName] = true

		model := ModelInfo{
			Name: modelName,
		}

		// Extract description
		if descP := a.Find("p.max-w-lg").First(); descP.Length() > 0 {
			model.Description = strings.TrimSpace(descP.Text())
		} else if descP := a.Find("p").First(); descP.Length() > 0 {
			model.Description = strings.TrimSpace(descP.Text())
		}

		// Extract parameter sizes and capabilities from badges
		a.Find("span.inline-flex").Each(func(_ int, badge *goquery.Selection) {
			text := strings.TrimSpace(badge.Text())
			if text == "" {
				return
			}
			if isParameterSize(text) {
				model.Parameters = append(model.Parameters, text)
			} else {
				model.Capabilities = append(model.Capabilities, text)
			}
		})

		// Extract metadata
		var metaTexts []string
		a.Find("span.flex.items-center").Each(func(_ int, meta *goquery.Selection) {
			t := strings.Join(strings.Fields(strings.TrimSpace(meta.Text())), " ")
			if t != "" {
				metaTexts = append(metaTexts, t)
			}
		})

		if len(metaTexts) > 0 {
			model.PullCount = metaTexts[0]
		}
		if len(metaTexts) > 1 {
			model.TagCount = metaTexts[1]
		}
		if len(metaTexts) > 2 {
			model.UpdatedAt = metaTexts[2]
		}

		models = append(models, model)
	})

	return models, nil
}

func sanitizeFilename(name string) string {
	r := strings.NewReplacer(
		":", "-",
		"/", "-",
		"\\", "-",
		" ", "_",
	)
	return r.Replace(name)
}

func displayUsageExamples() {
	fmt.Println(color.CyanString("\nCommand-line Usage Examples:"))
	fmt.Println(color.WhiteString("  # List available models:"))
	fmt.Println("  ./ggufDownloader")
	fmt.Println("  ./ggufDownloader -list")
	fmt.Println("  ./ggufDownloader -search llama")

	fmt.Println(color.WhiteString("\n  # Download a specific model:"))
	fmt.Println("  ./ggufDownloader -model llama2 -params 7b")
	fmt.Println("  ./ggufDownloader -model llama3:8b")
	fmt.Println("  ./ggufDownloader -model phi3")
	fmt.Println("  ./ggufDownloader -model mistral -params 7b-instruct -out my_mistral.gguf")

	fmt.Println(color.WhiteString("\n  # Output format:"))
	fmt.Println("  # Files are saved as: modelname-params.gguf (e.g., llama2-7b.gguf)")
}

func displaySimpleUsage() {
	fmt.Println(color.CyanString("\nSimple Usage:"))
	fmt.Println(color.WhiteString("  List models:  ./ggufDownloader -list"))
	fmt.Println(color.WhiteString("  Search:       ./ggufDownloader -search QUERY"))
	fmt.Println(color.WhiteString("  Download:     ./ggufDownloader -model MODEL [-params PARAMS] [-out FILENAME]"))
	fmt.Println(color.WhiteString("  Help:         ./ggufDownloader -help"))

	fmt.Println(color.YellowString("\nQuick Examples:"))
	fmt.Println("  ./ggufDownloader -model llama3:8b")
	fmt.Println("  ./ggufDownloader -model phi3 -params latest")
}

// printModelsTable prints the models in a table format
func printModelsTable(models []ModelInfo, showDetails bool) {
	nameWidth := 20
	sizesWidth := 30
	capabilitiesWidth := 30
	infoWidth := 20

	for _, model := range models {
		if len(model.Name) > nameWidth-3 {
			nameWidth = len(model.Name) + 3
		}
	}

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

	separator := strings.Repeat("-", nameWidth+sizesWidth)
	if showDetails {
		separator += strings.Repeat("-", capabilitiesWidth+infoWidth+20)
	}
	fmt.Println(headerFmt(separator))

	for _, model := range models {
		fmt.Printf(color.GreenString("%-*s", nameWidth, model.Name))

		sizes := strings.Join(model.Parameters, ", ")
		if sizes == "" {
			sizes = "latest"
		}
		if len(sizes) > sizesWidth-3 {
			sizes = sizes[:sizesWidth-6] + "..."
		}
		fmt.Printf(color.YellowString("%-*s", sizesWidth, sizes))

		if showDetails {
			caps := strings.Join(model.Capabilities, ", ")
			if caps == "" {
				caps = "-"
			}
			if len(caps) > capabilitiesWidth-3 {
				caps = caps[:capabilitiesWidth-6] + "..."
			}
			fmt.Printf(color.CyanString("%-*s", capabilitiesWidth, caps))

			pullCount := model.PullCount
			if pullCount == "" {
				pullCount = "-"
			}
			fmt.Printf(color.WhiteString("%-*s", infoWidth, pullCount))

			updated := model.UpdatedAt
			if updated == "" {
				updated = "-"
			}
			fmt.Printf(color.WhiteString("%s", updated))
		}
		fmt.Println()
	}
}

func main() {
	modelName := flag.String("model", "", "The name of the model to download (e.g., phi3 or llama3:8b)")
	modelParameters := flag.String("params", "", "The model parameters to use (e.g., 8b or latest)")
	outputFile := flag.String("out", "", "Output filename for the GGUF file (optional)")
	searchQuery := flag.String("search", "", "Search query for filtering models")
	listModels := flag.Bool("list", false, "List available models")
	flag.Parse()

	noArgsProvided := len(os.Args) == 1
	if noArgsProvided || *listModels || *searchQuery != "" {
		models, err := fetchAvailableModels(*searchQuery)
		if err != nil {
			fmt.Println(color.RedString("[ERROR] %s", err))
			os.Exit(1)
		}

		if len(models) == 0 {
			fmt.Println(color.YellowString("\nNo models found matching your query."))
			return
		}

		fmt.Println(color.CyanString("\n=== Available models from Ollama ==="))

		maxModelsToShow := 10
		if noArgsProvided && len(models) > maxModelsToShow {
			printModelsTable(models[:maxModelsToShow], false)
			fmt.Printf(color.WhiteString("\n... and %d more (use -list to see all)\n"), len(models)-maxModelsToShow)
		} else {
			printModelsTable(models, *listModels || *searchQuery != "")
		}

		if noArgsProvided {
			displaySimpleUsage()
		} else {
			displayUsageExamples()
		}
		return
	}

	rawModel := strings.TrimSpace(*modelName)
	params := strings.TrimSpace(*modelParameters)

	if rawModel == "" {
		displayUsageExamples()
		fmt.Println(color.RedString("[ERROR] Model name is required (-model)."))
		fmt.Println(color.CyanString("\nRun without arguments to see available models."))
		os.Exit(1)
	}

	if strings.Contains(rawModel, ":") {
		parts := strings.SplitN(rawModel, ":", 2)
		rawModel = parts[0]
		if params == "" {
			params = parts[1]
		}
	}

	if params == "" {
		params = "latest"
	}

	manifest, err := fetchManifest(rawModel, params)
	if err != nil {
		fmt.Println(color.RedString("[ERROR] %s", err))
		os.Exit(1)
	}

	var modelDigest string
	var maxLayerSize int64
	for _, layer := range manifest.Layers {
		if layer.MediaType == "application/vnd.ollama.image.model" {
			modelDigest = layer.Digest
			break
		}
		if layer.Size > maxLayerSize {
			maxLayerSize = layer.Size
			modelDigest = layer.Digest
		}
	}

	if modelDigest == "" && len(manifest.Layers) > 0 {
		modelDigest = manifest.Layers[0].Digest
	}

	if modelDigest == "" {
		fmt.Println(color.RedString("[ERROR] Model digest not found in manifest."))
		os.Exit(1)
	}

	registryPath := rawModel
	if !strings.Contains(registryPath, "/") {
		registryPath = "library/" + registryPath
	}

	downloadURL := fmt.Sprintf("https://registry.ollama.ai/v2/%s/blobs/%s", registryPath, modelDigest)

	outputFilename := strings.TrimSpace(*outputFile)
	if outputFilename == "" {
		outputFilename = fmt.Sprintf("%s-%s.gguf", sanitizeFilename(rawModel), sanitizeFilename(params))
	}

	fmt.Println(color.CyanString("[INFO] Downloading %s (model: %s, tag: %s)...", outputFilename, rawModel, params))
	if err := downloadFile(downloadURL, outputFilename); err != nil {
		fmt.Println(color.RedString("[ERROR] %s", err))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("[SUCCESS] Download completed: %s", outputFilename))
}
