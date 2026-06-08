// Package storage provides object storage for user uploads + generated media on an S3-compatible
// backend (DigitalOcean Spaces). It signs short-lived presigned PUT URLs so the browser uploads file
// bytes DIRECTLY to the bucket — the platform never proxies the bytes; it only signs the URL and
// records the public location. A zero/unconfigured Store is disabled (callers 501), so the platform
// runs fine without storage configured.
package storage

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config holds S3-compatible storage settings (DigitalOcean Spaces).
type Config struct {
	Endpoint   string // host only, e.g. "nyc3.digitaloceanspaces.com"
	Region     string // e.g. "nyc3"
	Bucket     string
	AccessKey  string
	SecretKey  string
	PublicBase string // public URL base for objects; default https://<bucket>.<endpoint>
}

// Store signs presigned upload URLs and resolves public object URLs.
type Store struct {
	client     *minio.Client
	bucket     string
	publicBase string
}

// New builds a Store from config. Returns a disabled Store (Enabled()==false, no error) when the
// endpoint/keys/bucket aren't all set, so the service starts without storage.
func New(c Config) (*Store, error) {
	if c.Endpoint == "" || c.AccessKey == "" || c.SecretKey == "" || c.Bucket == "" {
		return &Store{}, nil
	}
	cl, err := minio.New(c.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.AccessKey, c.SecretKey, ""),
		Secure: true,
		Region: c.Region,
	})
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(c.PublicBase, "/")
	if base == "" {
		base = fmt.Sprintf("https://%s.%s", c.Bucket, c.Endpoint)
	}
	return &Store{client: cl, bucket: c.Bucket, publicBase: base}, nil
}

// Enabled reports whether object storage is configured.
func (s *Store) Enabled() bool { return s != nil && s.client != nil }

// nonWord matches characters that don't belong in a clean object key segment.
var nonWord = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// objectKey builds a tenant-scoped, collision-resistant key (uploads/<tenant>/<uuid>-<safe name>).
func objectKey(tenantID, filename string) string {
	clean := nonWord.ReplaceAllString(path.Base(filename), "-")
	clean = strings.Trim(clean, "-.")
	if clean == "" {
		clean = "file"
	}
	return fmt.Sprintf("uploads/%s/%s-%s", tenantID, uuid.NewString(), clean)
}

// Presigned is the result of signing an upload: where the browser PUTs the bytes, and where the object
// will be publicly readable afterward.
type Presigned struct {
	Key       string `json:"key"`
	UploadURL string `json:"upload_url"`
	FileURL   string `json:"file_url"`
}

// PresignUpload signs a short-lived PUT URL for a tenant's upload and returns it plus the eventual
// public URL. The browser PUTs the file to UploadURL; nothing flows through this service.
func (s *Store) PresignUpload(ctx context.Context, tenantID, filename string, ttl time.Duration) (Presigned, error) {
	if !s.Enabled() {
		return Presigned{}, fmt.Errorf("storage not configured")
	}
	key := objectKey(tenantID, filename)
	u, err := s.client.PresignedPutObject(ctx, s.bucket, key, ttl)
	if err != nil {
		return Presigned{}, err
	}
	return Presigned{Key: key, UploadURL: u.String(), FileURL: s.publicBase + "/" + key}, nil
}
