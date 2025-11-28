package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/anchore/syft/syft"
	"github.com/anchore/syft/syft/format"
	"github.com/anchore/syft/syft/format/syftjson"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/encrypt"
	"github.com/securestor/securestor/internal/replicate"
)

type FileMetadata struct {
	ID             string            `json:"id"`
	Filename       string            `json:"filename"`
	BucketName     string            `json:"bucket_name"`
	UploadTime     string            `json:"upload_time"`
	ContentType    string            `json:"content_type"`
	Size           int64             `json:"size"`
	Hash           string            `json:"hash"`
	Owner          string            `json:"owner"`
	Tags           []string          `json:"tags"`
	Description    string            `json:"description"`
	Version        string            `json:"version"`
	Permissions    string            `json:"permissions"`
	Checksum       string            `json:"checksum"`
	LastAccessed   string            `json:"last_accessed"`
	Expiration     string            `json:"expiration"`
	CustomMetadata map[string]string `json:"custom_metadata"`
}

type CustomMetadataRequest struct {
	ID             string            `json:"id"`
	CustomMetadata map[string]string `json:"custom_metadata"`
	Action         string            `json:"action"` // "add", "update", "delete"
}

var metadataFile = filepath.Join(storageDir, "metadata.json")

var enableEncryption = false

func computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func generateUniqueID() string {
	return uuid.New().String()
}

func saveMetadata(metadata FileMetadata) error {
	var metadataList []FileMetadata

	// Ensure the storage directory exists
	if err := os.MkdirAll(filepath.Dir(metadataFile), os.ModePerm); err != nil {
		return err
	}

	// Read existing metadata
	file, err := os.Open(metadataFile)
	if err == nil {
		defer file.Close()
		if err := json.NewDecoder(file).Decode(&metadataList); err != nil {
			return err
		}
	}

	// Append new metadata
	metadataList = append(metadataList, metadata)

	// Write updated metadata
	file, err = os.Create(metadataFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(metadataList)
}

func UpdateCustomMetadataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req CustomMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var metadataList []FileMetadata

	// Read existing metadata
	file, err := os.Open(metadataFile)
	if err != nil {
		http.Error(w, "Failed to read metadata", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&metadataList); err != nil {
		http.Error(w, "Failed to decode metadata", http.StatusInternalServerError)
		return
	}

	// Find the specific file's metadata
	var updated bool
	for i, metadata := range metadataList {
		if metadata.ID == req.ID {
			switch req.Action {
			case "add", "update":
				for key, value := range req.CustomMetadata {
					metadata.CustomMetadata[key] = value
				}
			case "delete":
				for key := range req.CustomMetadata {
					delete(metadata.CustomMetadata, key)
				}
			default:
				http.Error(w, "Invalid action", http.StatusBadRequest)
				return
			}
			metadataList[i] = metadata
			updated = true
			break
		}
	}

	if !updated {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Write updated metadata back to the same file
	file, err = os.Create(metadataFile)
	if err != nil {
		http.Error(w, "Failed to save metadata", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(metadataList); err != nil {
		http.Error(w, "Failed to encode metadata", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Custom metadata updated successfully")
}

func generateSBOM(filePath string) (string, error) {

	// Get the source
	src, err := syft.GetSource(context.Background(), filePath, nil)
	if err != nil {
		panic(err)
	}

	// Generate the SBOM
	sbom, err := syft.CreateSBOM(context.Background(), src, nil)
	if err != nil {
		panic(err)
	}

	// Define the SBOM file path
	sbomFilePath := filePath + ".sbom.json"

	// Create the SBOM file
	sbomFile, err := os.Create(sbomFilePath)
	if err != nil {
		return "", err
	}
	defer sbomFile.Close()

	bytes, err := format.Encode(*sbom, syftjson.NewFormatEncoder())
	if err != nil {
		panic(err)
	}

	// Write the SBOM to the file
	if _, err := sbomFile.Write(bytes); err != nil {
		return "", err
	}

	return sbomFilePath, nil
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	bucketName := r.URL.Query().Get("bucket")
	if bucketName == "" {
		http.Error(w, "Bucket name is required", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file from request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Save the uploaded file to a temporary location
	tempFilePath := filepath.Join(storageDir, header.Filename)
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		http.Error(w, "Failed to create temporary file", http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(tempFile, file); err != nil {
		tempFile.Close()
		os.Remove(tempFilePath)
		http.Error(w, "Failed to save temporary file", http.StatusInternalServerError)
		return
	}

	tempFile.Close()
	defer os.Remove(tempFilePath) // Ensure the temporary file is removed

	// Compute file hash
	fileBytes, err := os.ReadFile(tempFilePath)
	if err != nil {
		http.Error(w, "Failed to read temporary file", http.StatusInternalServerError)
		return
	}
	fileHash := computeHash(fileBytes)

	// Generate SBOM for the uploaded file
	sbomFilePath, err := generateSBOM(tempFilePath)
	if err != nil {
		http.Error(w, "Failed to generate SBOM", http.StatusInternalServerError)
		return
	}

	var finalFilePath string
	if enableEncryption {
		// Encrypt the file content
		encryptedFilePath := tempFilePath + ".enc"
		if err := encrypt.EncryptFile(tempFilePath, encryptedFilePath, []byte(encrypt.EncryptionKey)); err != nil {
			fmt.Println("Error encrypting file:", err)
			http.Error(w, "Failed to encrypt file", http.StatusInternalServerError)
			return
		}
		defer os.Remove(encryptedFilePath) // Clean up the encrypted file
		finalFilePath = encryptedFilePath
	} else {
		finalFilePath = tempFilePath
	}

	// Save encrypted file to the main bucket
	bucketPath := filepath.Join(storageDir, bucketName)
	if _, err := os.Stat(bucketPath); os.IsNotExist(err) {
		http.Error(w, "Bucket does not exist", http.StatusBadRequest)
		return
	}

	dst, err := os.Create(filepath.Join(bucketPath, header.Filename))
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	finalFile, err := os.Open(finalFilePath)
	if err != nil {
		http.Error(w, "Failed to open encrypted file", http.StatusInternalServerError)
		return
	}
	defer finalFile.Close()

	if _, err := io.Copy(dst, finalFile); err != nil {
		http.Error(w, "Failed to save encrypted file", http.StatusInternalServerError)
		return
	}

	// Automatically generate the description
	description := fmt.Sprintf("File %s uploaded to bucket %s on %s", header.Filename, bucketName, time.Now().Format(time.RFC3339))

	// Save metadata
	metadata := FileMetadata{
		ID:           generateUniqueID(),
		Filename:     header.Filename,
		BucketName:   bucketName,
		UploadTime:   time.Now().Format(time.RFC3339),
		ContentType:  header.Header.Get("Content-Type"),
		Size:         header.Size,
		Hash:         fileHash,
		Owner:        "user123", // Replace with actual user information
		Tags:         []string{"example", "file"},
		Description:  description,
		Version:      "1.0",
		Permissions:  "rw-r--r--",
		Checksum:     computeHash(fileBytes), // Use the same hash function for checksum
		LastAccessed: time.Now().Format(time.RFC3339),
		Expiration:   "", // Set if applicable
		CustomMetadata: map[string]string{
			"customField1": "value1",
			"customField2": "value2",
			"sbomFilePath": sbomFilePath,
		},
	}

	if err := saveMetadata(metadata); err != nil {
		http.Error(w, "Failed to save metadata", http.StatusInternalServerError)
		return
	}

	// Synchronously replicate the file to replica nodes with timeout
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		rs := replicate.GetInstance()
		result, err := rs.ReplicateFile(ctx, bucketName, header.Filename, fileBytes)
		if err != nil {
			fmt.Printf("WARNING: Replication failed for %s: %v\n", header.Filename, err)
		} else if result != nil {
			fmt.Printf("Replication successful: file=%s, replicas=%d, checksum=%s\n",
				header.Filename, result.ActualReplicas, result.Checksum)
		}
	}()

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File uploaded successfully: %s\n", header.Filename)
}

func SearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	var metadataList []FileMetadata

	// Read metadata
	file, err := os.Open(metadataFile)
	if err != nil {
		http.Error(w, "Failed to read metadata", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&metadataList); err != nil {
		http.Error(w, "Failed to decode metadata", http.StatusInternalServerError)
		return
	}

	// Search for matching files
	var results []FileMetadata
	for _, metadata := range metadataList {
		if metadata.Filename == query || metadata.BucketName == query {
			results = append(results, metadata)
		}
	}

	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, "Failed to encode results", http.StatusInternalServerError)
		return
	}
}

func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	bucketName := r.URL.Query().Get("bucket")
	filename := r.URL.Query().Get("filename")
	if bucketName == "" || filename == "" {
		http.Error(w, "Bucket name and filename are required", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(storageDir, bucketName, filename)
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	var fileBytes []byte

	if enableEncryption {
		// Read encrypted file content
		encryptedBytes, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}

		// Decrypt file content
		fileBytes, err = encrypt.Decrypt(encryptedBytes, encrypt.EncryptionKey)
		if err != nil {
			http.Error(w, "Failed to decrypt file", http.StatusInternalServerError)
			return
		}
	} else {
		// Read file content
		fileBytes, err = io.ReadAll(file)
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := w.Write(fileBytes); err != nil {
		http.Error(w, "Failed to download file", http.StatusInternalServerError)
		return
	}
}

func ListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	bucketName := r.URL.Query().Get("bucket")
	if bucketName == "" {
		http.Error(w, "Bucket name is required", http.StatusBadRequest)
		return
	}

	bucketPath := filepath.Join(storageDir, bucketName)
	files, err := os.ReadDir(bucketPath)
	if err != nil {
		http.Error(w, "Failed to list files", http.StatusInternalServerError)
		return
	}

	for _, file := range files {
		fmt.Fprintln(w, file.Name())
	}
}

func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	bucketName := r.URL.Query().Get("bucket")
	filename := r.URL.Query().Get("filename")
	if bucketName == "" || filename == "" {
		http.Error(w, "Bucket name and filename are required", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(storageDir, bucketName, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if err := os.Remove(filePath); err != nil {
		http.Error(w, "Failed to delete file", http.StatusInternalServerError)
		return
	}

	// Delete the file from the replica buckets
	for _, replica := range replicationBuckets {
		replicaFilePath := filepath.Join(storageDir, replica, bucketName, filename)
		if _, err := os.Stat(replicaFilePath); !os.IsNotExist(err) {
			if err := os.Remove(replicaFilePath); err != nil {
				http.Error(w, "Failed to delete file from replica bucket", http.StatusInternalServerError)
				return
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File deleted successfully: %s\n", filename)
}
