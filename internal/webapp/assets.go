package webapp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const maxAssetUploadBytes = 32 << 20

type AssetStorage struct {
	Endpoint  string
	PublicURL string
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	LocalDir  string
}

type AssetUploadResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	MIMEType   string `json:"mimeType"`
	Size       int64  `json:"size"`
	Kind       string `json:"kind"`
	URL        string `json:"url"`
	StorageKey string `json:"storageKey"`
	Source     string `json:"source"`
}

func AssetStorageFromEnv() AssetStorage {
	endpoint := strings.TrimRight(envOrDefault("MUTESOLO_MINIO_ENDPOINT", "http://127.0.0.1:9000"), "/")
	return AssetStorage{
		Endpoint:  endpoint,
		PublicURL: strings.TrimRight(envOrDefault("MUTESOLO_MINIO_PUBLIC_URL", endpoint), "/"),
		Bucket:    envOrDefault("MUTESOLO_MINIO_BUCKET", "Mutesolo-assets"),
		Region:    envOrDefault("MUTESOLO_MINIO_REGION", "us-east-1"),
		AccessKey: envOrDefault("MUTESOLO_MINIO_ACCESS_KEY", "Mutesolo"),
		SecretKey: envOrDefault("MUTESOLO_MINIO_SECRET_KEY", "Mutesolo123"),
		LocalDir:  envOrDefault("MUTESOLO_ASSET_FALLBACK_DIR", ".openclaw/assets"),
	}
}

func (s AssetStorage) Upload(ctx context.Context, name string, contentType string, body []byte) (AssetUploadResponse, error) {
	if len(body) == 0 {
		return AssetUploadResponse{}, fmt.Errorf("asset is empty")
	}
	if len(body) > maxAssetUploadBytes {
		return AssetUploadResponse{}, fmt.Errorf("asset exceeds %d bytes", maxAssetUploadBytes)
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = http.DetectContentType(body)
	}
	objectID := randomHex(16)
	key := assetObjectKey(objectID, name, contentType)

	// Always persist a local copy so the asset is reachable from the browser
	// via the backend's /assets/... route, regardless of where MinIO lives.
	localResult, localErr := s.writeLocalAsset(name, contentType, key, body)
	if localErr != nil {
		return AssetUploadResponse{}, localErr
	}

	// Best-effort upload to MinIO. The MinIO URL is useful for backend/LLM
	// consumers that can reach the object store directly, but it must not be
	// what the browser tries to render — many deployments expose MinIO on a
	// host/port the user's browser cannot reach (e.g. 127.0.0.1 inside a
	// Docker network). Failures here do not affect the returned URL.
	if putErr := s.putObject(ctx, key, contentType, body); putErr == nil {
		localResult.Source = "minio"
	}
	return localResult, nil
}

func (s AssetStorage) writeLocalAsset(name string, contentType string, key string, body []byte) (AssetUploadResponse, error) {
	localDir := strings.TrimSpace(s.LocalDir)
	if localDir == "" {
		return AssetUploadResponse{}, fmt.Errorf("local asset dir is not configured")
	}
	fullPath := filepath.Join(localDir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return AssetUploadResponse{}, fmt.Errorf("create local asset dir: %w", err)
	}
	if err := os.WriteFile(fullPath, body, 0o644); err != nil {
		return AssetUploadResponse{}, fmt.Errorf("write local asset: %w", err)
	}
	return AssetUploadResponse{
		ID:         strings.TrimSuffix(path.Base(key), path.Ext(key)),
		Name:       name,
		MIMEType:   contentType,
		Size:       int64(len(body)),
		Kind:       assetKind(contentType),
		URL:        "/assets/" + key,
		StorageKey: key,
		Source:     "local_static_fallback",
	}, nil
}

func (s AssetStorage) uploadLocalFallback(name string, contentType string, key string, body []byte, uploadErr error) (AssetUploadResponse, error) {
	localDir := strings.TrimSpace(s.LocalDir)
	if localDir == "" {
		return AssetUploadResponse{}, uploadErr
	}
	if err := cleanupLocalAssets(localDir, 7*24*time.Hour); err != nil {
		return AssetUploadResponse{}, fmt.Errorf("MinIO unavailable (%v); cleanup local fallback assets: %w", uploadErr, err)
	}
	return s.writeLocalAsset(name, contentType, key, body)
}

func cleanupLocalAssets(root string, maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(root, func(current string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.ModTime().Before(cutoff) {
			return os.Remove(current)
		}
		return nil
	})
}

func (s AssetStorage) putObject(ctx context.Context, key string, contentType string, body []byte) error {
	endpoint, err := url.Parse(s.Endpoint)
	if err != nil {
		return fmt.Errorf("parse MinIO endpoint: %w", err)
	}
	endpoint.Path = "/" + path.Join(strings.TrimPrefix(endpoint.Path, "/"), s.Bucket, key)
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	hash := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(hash[:])
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("X-Amz-Content-Sha256", payloadHash)
	signS3Request(request, s, payloadHash)

	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("upload asset to MinIO: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("upload asset to MinIO failed: status %d %s", response.StatusCode, strings.TrimSpace(string(data)))
	}
	return nil
}

func signS3Request(request *http.Request, storage AssetStorage, payloadHash string) {
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	shortDate := now.Format("20060102")
	region := storage.Region
	if strings.TrimSpace(region) == "" {
		region = "us-east-1"
	}
	request.Header.Set("X-Amz-Date", amzDate)

	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"
	canonicalHeaders := strings.Join([]string{
		"content-type:" + request.Header.Get("Content-Type"),
		"host:" + request.URL.Host,
		"x-amz-content-sha256:" + payloadHash,
		"x-amz-date:" + amzDate,
		"",
	}, "\n")
	canonicalRequest := strings.Join([]string{
		request.Method,
		request.URL.EscapedPath(),
		"",
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")
	canonicalHash := sha256.Sum256([]byte(canonicalRequest))
	scope := shortDate + "/" + region + "/s3/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		hex.EncodeToString(canonicalHash[:]),
	}, "\n")
	signingKey := s3SigningKey(storage.SecretKey, shortDate, region)
	signature := hmacSHA256Hex(signingKey, stringToSign)
	request.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		storage.AccessKey,
		scope,
		signedHeaders,
		signature,
	))
}

func s3SigningKey(secretKey string, shortDate string, region string) []byte {
	dateKey := hmacSHA256([]byte("AWS4"+secretKey), shortDate)
	regionKey := hmacSHA256(dateKey, region)
	serviceKey := hmacSHA256(regionKey, "s3")
	return hmacSHA256(serviceKey, "aws4_request")
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func hmacSHA256Hex(key []byte, value string) string {
	return hex.EncodeToString(hmacSHA256(key, value))
}

func (s AssetStorage) publicObjectURL(key string) string {
	return strings.TrimRight(s.PublicURL, "/") + "/" + path.Join(s.Bucket, key)
}

func assetObjectKey(id string, name string, contentType string) string {
	ext := strings.ToLower(path.Ext(name))
	if ext == "" {
		if exts, err := mime.ExtensionsByType(contentType); err == nil && len(exts) > 0 {
			ext = exts[0]
		}
	}
	if ext == "" {
		ext = ".bin"
	}
	return time.Now().UTC().Format("2006/01/02") + "/" + id + ext
}

func assetKind(contentType string) string {
	if strings.HasPrefix(contentType, "image/") {
		return "image"
	}
	return "file"
}

func randomHex(bytesLen int) string {
	data := make([]byte, bytesLen)
	if _, err := rand.Read(data); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(data)
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func writeAssetUploadError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
