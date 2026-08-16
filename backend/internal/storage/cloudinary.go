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
	_ = w.WriteField("api_key", c.APIKey)
	_ = w.WriteField("timestamp", timestamp)
	_ = w.WriteField("signature", signature)
	if c.Folder != "" {
		_ = w.WriteField("folder", c.Folder)
	}
	_ = w.WriteField("public_id", params["public_id"])
	fw, err := w.CreateFormFile("file", filepath.Base(key))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(fw, r); err != nil {
		return "", err
	}
	_ = w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/video/upload", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", err
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
		return "", err
	}
	if out.SecureURL != "" {
		return out.SecureURL, nil
	}
	return out.URL, nil
}

func (c *CloudinaryStorage) GetURL(key string) string {
	return c.BaseURL + "/video/upload" // management endpoint; actual URL returned by Put
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
