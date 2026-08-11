package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nick-zettelgarden/zg/internal/api"
	"github.com/nick-zettelgarden/zg/internal/output"
	"github.com/spf13/cobra"
)

// File mirrors the backend File model (go-backend/models/file.go).
type File struct {
	ID        int      `json:"id"`
	UserID    int      `json:"user_id"`
	Name      string   `json:"name"`
	Filetype  string   `json:"filetype"`
	Path      string   `json:"path"`
	Filename  string   `json:"filename"`
	Size      int      `json:"size"`
	CreatedBy int      `json:"created_by"`
	UpdatedBy int      `json:"updated_by"`
	CardPK    *int     `json:"card_pk,omitempty"`
	IsDeleted bool     `json:"is_deleted"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
	Tags      []string `json:"tags,omitempty"`
	Card      any      `json:"card"`
}

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Manage files",
	Long:  `Manage files: list, upload, download, tag, and import epubs.`,
}

var fileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List files",
	RunE:  runFileList,
}

var fileGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get file metadata",
	Args:  cobra.ExactArgs(1),
	RunE:  runFileGet,
}

var fileDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a file",
	Args:  cobra.ExactArgs(1),
	RunE:  runFileDelete,
}

var fileEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit file metadata (name, description, linked card)",
	Args:  cobra.ExactArgs(1),
	RunE:  runFileEdit,
}

var fileTagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List your file tags (with file counts)",
	RunE:  runFileTags,
}

var fileTagCmd = &cobra.Command{
	Use:   "tag <id> <tag...>",
	Short: "Tag a file",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runFileTag,
}

var fileUntagCmd = &cobra.Command{
	Use:   "untag <id> <tag>",
	Short: "Remove a tag from a file",
	Args:  cobra.ExactArgs(2),
	RunE:  runFileUntag,
}

var fileUploadCmd = &cobra.Command{
	Use:   "upload <path>",
	Short: "Upload a file",
	Args:  cobra.ExactArgs(1),
	RunE:  runFileUpload,
}

var fileDownloadCmd = &cobra.Command{
	Use:   "download <id>",
	Short: "Download a file to disk",
	Args:  cobra.ExactArgs(1),
	RunE:  runFileDownload,
}

var fileImportEpubCmd = &cobra.Command{
	Use:   "import-epub <id>",
	Short: "Import an epub file as cards",
	Args:  cobra.ExactArgs(1),
	RunE:  runFileImportEpub,
}

var (
	fileListLimit  int
	fileListOffset int
	fileSearch     string
	fileEditName   string
	fileEditDesc   string
	fileEditCardPK int
	fileUploadCard int
	fileDownloadTo string
	fileEpubCardID string
)

func init() {
	fileListCmd.Flags().IntVarP(&fileListLimit, "limit", "l", 20, "Limit results")
	fileListCmd.Flags().IntVarP(&fileListOffset, "offset", "o", 0, "Offset results")
	fileListCmd.Flags().StringVar(&fileSearch, "search", "", "Search files by name/description/content")
	fileEditCmd.Flags().StringVar(&fileEditName, "name", "", "New file name")
	fileEditCmd.Flags().StringVar(&fileEditDesc, "description", "", "New description (empty clears it)")
	fileEditCmd.Flags().IntVar(&fileEditCardPK, "card-pk", 0, "Link/unlink to a card (0 = unlink)")
	fileUploadCmd.Flags().IntVar(&fileUploadCard, "card-id", 0, "Attach the file to a card id")
	fileDownloadCmd.Flags().StringVarP(&fileDownloadTo, "output", "o", "", "Save to this path (default: the file's original name)")
	fileImportEpubCmd.Flags().StringVar(&fileEpubCardID, "card-id", "", "Card ID for the book (e.g. '3a')")

	fileCmd.AddCommand(fileListCmd)
	fileCmd.AddCommand(fileGetCmd)
	fileCmd.AddCommand(fileDeleteCmd)
	fileCmd.AddCommand(fileEditCmd)
	fileCmd.AddCommand(fileTagsCmd)
	fileCmd.AddCommand(fileTagCmd)
	fileCmd.AddCommand(fileUntagCmd)
	fileCmd.AddCommand(fileUploadCmd)
	fileCmd.AddCommand(fileDownloadCmd)
	fileCmd.AddCommand(fileImportEpubCmd)
}

// GetFileCmd returns the file command for registration in main.
func GetFileCmd() *cobra.Command {
	return fileCmd
}

func runFileList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	limit := fileListLimit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := fileListOffset
	if offset < 0 {
		offset = 0
	}

	path := fmt.Sprintf("/api/files?page=%d&per_page=%d", offset/limit+1, limit)
	if fileSearch != "" {
		path += "&search=" + strings.ReplaceAll(fileSearch, " ", "%20")
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(path)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}
	if resp.StatusCode != 200 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	var result struct {
		Files       []File `json:"files"`
		Page        int    `json:"page"`
		PerPage     int    `json:"per_page"`
		Total       int    `json:"total"`
		TotalPages  int    `json:"total_pages"`
		Search      string `json:"search"`
		StorageUsed int64  `json:"storage_used"`
		MaxStorage  int64  `json:"max_storage"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}
	if result.Files == nil {
		result.Files = []File{}
	}

	return output.WriteSuccess(os.Stdout, result.Files)
}

func runFileGet(cmd *cobra.Command, args []string) error {
	fileID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid file ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(fmt.Sprintf("/api/files/%d", fileID))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}
	if resp.StatusCode != 200 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	var file File
	if err := json.Unmarshal(body, &file); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, file)
}

func runFileDelete(cmd *cobra.Command, args []string) error {
	fileID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid file ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Delete(fmt.Sprintf("/api/files/%d", fileID))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	return output.WriteMessage(os.Stdout, fmt.Sprintf("File %d deleted", fileID))
}

func runFileEdit(cmd *cobra.Command, args []string) error {
	fileID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid file ID", "ID must be a number")
	}
	if fileEditName == "" && fileEditDesc == "" && !cmd.Flags().Changed("card-pk") {
		return output.WriteError(os.Stdout, "No updates", "Specify --name, --description, and/or --card-pk")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody := map[string]any{"name": fileEditName}
	if fileEditDesc != "" || cmd.Flags().Changed("description") {
		requestBody["description"] = fileEditDesc
	}
	if cmd.Flags().Changed("card-pk") {
		if fileEditCardPK == 0 {
			requestBody["card_pk"] = nil
		} else {
			requestBody["card_pk"] = fileEditCardPK
		}
	}
	bodyBytes, _ := json.Marshal(requestBody)

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Patch(fmt.Sprintf("/api/files/%d", fileID), bodyBytes)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}
	if resp.StatusCode != 200 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	var file File
	if err := json.Unmarshal(body, &file); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, file)
}

func runFileTags(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get("/api/files/tags")
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}
	if resp.StatusCode != 200 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	var tags []map[string]any
	if err := json.Unmarshal(body, &tags); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}
	if tags == nil {
		tags = []map[string]any{}
	}

	return output.WriteSuccess(os.Stdout, tags)
}

func runFileTag(cmd *cobra.Command, args []string) error {
	fileID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid file ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody, _ := json.Marshal(map[string]any{"tag_names": args[1:]})

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Post(fmt.Sprintf("/api/files/%d/tags", fileID), requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	return output.WriteMessage(os.Stdout, fmt.Sprintf("File %d tagged %s", fileID, strings.Join(args[1:], ", ")))
}

func runFileUntag(cmd *cobra.Command, args []string) error {
	fileID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid file ID", "ID must be a number")
	}
	tagName := strings.TrimPrefix(args[1], "#")

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Delete(fmt.Sprintf("/api/files/%d/tags/%s", fileID, tagName))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	return output.WriteMessage(os.Stdout, fmt.Sprintf("Tag #%s removed from file %d", tagName, fileID))
}

func runFileUpload(cmd *cobra.Command, args []string) error {
	path := args[0]
	data, err := os.ReadFile(path)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading file failed", err.Error())
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	// Build the multipart body: "file" part + "card_pk" form value.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return output.WriteError(os.Stdout, "Multipart error", err.Error())
	}
	if _, err := part.Write(data); err != nil {
		return output.WriteError(os.Stdout, "Multipart error", err.Error())
	}
	if err := mw.WriteField("card_pk", strconv.Itoa(fileUploadCard)); err != nil {
		return output.WriteError(os.Stdout, "Multipart error", err.Error())
	}
	if err := mw.Close(); err != nil {
		return output.WriteError(os.Stdout, "Multipart error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.PostWithContentType("/api/files/upload", mw.FormDataContentType(), buf.Bytes())
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	var result struct {
		Message string `json:"message"`
		File    File   `json:"file"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return output.WriteError(os.Stdout, "Parse error", err.Error())
	}

	return output.WriteSuccess(os.Stdout, result.File)
}

func runFileDownload(cmd *cobra.Command, args []string) error {
	fileID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid file ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Get(fmt.Sprintf("/api/files/download/%d", fileID))
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	outPath := fileDownloadTo
	if outPath == "" {
		outPath = contentDispositionFilename(resp.Header.Get("Content-Disposition"))
	}
	if outPath == "" {
		outPath = fmt.Sprintf("file-%d", fileID)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return output.WriteError(os.Stdout, "Creating output file failed", err.Error())
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return output.WriteError(os.Stdout, "Download failed", err.Error())
	}

	return output.WriteMessage(os.Stdout, fmt.Sprintf("Downloaded file %d to %s", fileID, outPath))
}

// contentDispositionFilename extracts the filename from a
// `attachment; filename="..."` header value.
func contentDispositionFilename(header string) string {
	for _, part := range strings.Split(header, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "filename=") {
			return strings.Trim(strings.TrimPrefix(part, "filename="), `"`)
		}
	}
	return ""
}

func runFileImportEpub(cmd *cobra.Command, args []string) error {
	fileID, err := strconv.Atoi(args[0])
	if err != nil {
		return output.WriteError(os.Stdout, "Invalid file ID", "ID must be a number")
	}

	cfg, err := loadConfig()
	if err != nil {
		return output.WriteError(os.Stdout, "Config error", err.Error())
	}

	requestBody, _ := json.Marshal(map[string]string{"card_id": fileEpubCardID})

	client := api.NewClient(cfg.APIURL, cfg.Token, cfg.TimeoutSeconds)
	resp, err := client.Post(fmt.Sprintf("/api/files/%d/import-epub", fileID), requestBody)
	if err != nil {
		return output.WriteError(os.Stdout, "API request failed", err.Error())
	}

	body, err := api.GetBodyBytes(resp)
	if err != nil {
		return output.WriteError(os.Stdout, "Reading response failed", err.Error())
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return output.WriteError(os.Stdout, fmt.Sprintf("API error: %d", resp.StatusCode), string(body))
	}

	var result []map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		// Some responses are a single object rather than an array.
		var single map[string]any
		if err2 := json.Unmarshal(body, &single); err2 != nil {
			return output.WriteError(os.Stdout, "Parse error", err.Error())
		}
		return output.WriteSuccess(os.Stdout, single)
	}

	return output.WriteSuccess(os.Stdout, result)
}
