package storage

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CloudinaryStorage uploads to Cloudinary via the raw REST API (no SDK needed).
// Configure with CLOUDINARY_URL=cloudinary://<key>:<secret>@<cloud_name>[/<folder>].
type CloudinaryStorage struct {
	CloudName string
	APIKey    string
	APISecret string
	Folder    string
	BaseURL   string
	Client    *http.Client
}

// NewCloudinary parses a CLOUDINARY_URL and returns a ready storage backend.
func NewCloudinary(cloudinaryURL string) (*CloudinaryStorage, error) {
	u, err := url.Parse(cloudinaryURL)
	if err != nil {
		return nil, fmt.Errorf("invalid CLOUDINARY_URL: %w", err)
	}
	if u.Scheme != "cloudinary" || u.Host == "" {
		return nil, fmt.Errorf("CLOUDINARY_URL must be cloudinary://<key>:<secret>@<cloud_name>")
	}
	secret, _ := u.User.Password()
	cs := &CloudinaryStorage{
		CloudName: u.Host,
		APIKey:    u.User.Username(),
		APISecret: secret,
		BaseURL:   "https://api.cloudinary.com/v1_1/" + u.Host,
		Client:    &http.Client{Timeout: 60 * time.Second},
	}
	folder := strings.Trim(u.Path, "/")
	if folder != "" {
		cs.Folder = folder
	}
	if cs.APIKey == "" || cs.APISecret == "" {
		return nil, fmt.Errorf("CLOUDINARY_URL must include API key and secret")
	}
	return cs, nil
}

func (c *CloudinaryStorage) Put(ctx context.Context, key string, r io.Reader) (string, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	params := map[string]string{
		"timestamp": timestamp,
		"public_id": strings.TrimSuffix(key, filepath.Ext(key)),
	}
	if c.Folder != "" {
		params["folder"] = c.Folder
	}
	signature := c.sign(params)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	writeField := func(name, val string) error { return w.WriteField(name, val) }
	if err := writeField("api_key", c.APIKey); err != nil {
		return "", fmt.Errorf("write api_key field: %w", err)
	}
	if err := writeField("timestamp", timestamp); err != nil {
		return "", fmt.Errorf("write timestamp field: %w", err)
	}
	if err := writeField("signature", signature); err != nil {
		return "", fmt.Errorf("write signature field: %w", err)
	}
	if c.Folder != "" {
		if err := writeField("folder", c.Folder); err != nil {
			return "", fmt.Errorf("write folder field: %w", err)
		}
	}
	if err := writeField("public_id", params["public_id"]); err != nil {
		return "", fmt.Errorf("write public_id field: %w", err)
	}
	fw, err := w.CreateFormFile("file", filepath.Base(key))
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(fw, r); err != nil {
		return "", fmt.Errorf("copy file data: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/video/upload", &body)
	if err != nil {
		return "", fmt.Errorf("create upload request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("cloudinary upload failed (%d): %s", resp.StatusCode, string(msg))
	}
	var out struct {
		SecureURL string `json:"secure_url"`
		URL       string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode upload response: %w", err)
	}
	if out.SecureURL != "" {
		return out.SecureURL, nil
	}
	return out.URL, nil
}

func (c *CloudinaryStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	fileURL := c.buildFileURL(key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create get request: %w", err)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("cloudinary get failed (%d) for key: %s", resp.StatusCode, key)
	}
	return resp.Body, nil
}

func (c *CloudinaryStorage) List(ctx context.Context, prefix string) ([]string, error) {
	return nil, fmt.Errorf("cloudinary list not supported: use local storage for recorded video playback")
}

func (c *CloudinaryStorage) buildFileURL(key string) string {
	publicID := strings.TrimSuffix(key, filepath.Ext(key))
	if c.Folder != "" {
		publicID = c.Folder + "/" + publicID
	}
	return fmt.Sprintf("https://res.cloudinary.com/%s/video/upload/%s", c.CloudName, publicID)
}

// sign builds the Cloudinary message signature: sha1 of sorted "k=v&..." + secret.
func (c *CloudinaryStorage) sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(params[k])
	}
	sb.WriteString(c.APISecret)
	sum := sha1.Sum([]byte(sb.String()))
	return fmt.Sprintf("%x", sum)
}
