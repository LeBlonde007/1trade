// Package storage provides object storage for user uploads + generated media on an S3-compatible
// backend (DigitalOcean Spaces). It signs short-lived presigned PUT URLs so the browser uploads file
// bytes DIRECTLY to the bucket — the platform never proxies the bytes; it only signs the URL and
// records the public location. A zero/unconfigured Store is disabled (callers 501), so the platform
// runs fine without storage configured.
package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
	PublicBase string // explicit public URL base for objects; overrides the derived origin/CDN base
	UseCDN     bool   // serve objects via the DigitalOcean Spaces CDN edge (must be enabled on the Space)
	PublicRead bool   // sign uploads public-read so the public/CDN FileURL actually loads (default on)
	Prefix     string // optional key prefix — a "directory" within the bucket (e.g. "sandbox") to namespace uploads
}

// Store signs presigned upload URLs and resolves public object URLs.
type Store struct {
	client     *minio.Client
	bucket     string
	publicBase string
	publicRead bool
	prefix     string
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
		base = "https://" + publicHost(c.Bucket, c.Endpoint, c.UseCDN)
	}
	return &Store{
		client: cl, bucket: c.Bucket, publicBase: base,
		publicRead: c.PublicRead, prefix: strings.Trim(c.Prefix, "/"),
	}, nil
}

// publicHost builds the object host: the bucket subdomain of the endpoint, swapped to the CDN edge
// (…cdn.digitaloceanspaces.com) when useCDN is set — the DO Spaces CDN serves cached, edge-delivered
// objects (must be enabled on the Space). For a non-DO endpoint the CDN swap is a harmless no-op.
func publicHost(bucket, endpoint string, useCDN bool) string {
	host := fmt.Sprintf("%s.%s", bucket, endpoint)
	if useCDN {
		host = strings.Replace(host, ".digitaloceanspaces.com", ".cdn.digitaloceanspaces.com", 1)
	}
	return host
}

// Enabled reports whether object storage is configured.
func (s *Store) Enabled() bool { return s != nil && s.client != nil }

// nonWord matches characters that don't belong in a clean object key segment.
var nonWord = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// objectKey builds a tenant-scoped, collision-resistant key
// (<prefix>/uploads/<tenant>/<uuid>-<safe name>); prefix is the optional bucket "directory".
func objectKey(prefix, tenantID, filename string) string {
	clean := nonWord.ReplaceAllString(path.Base(filename), "-")
	clean = strings.Trim(clean, "-.")
	if clean == "" {
		clean = "file"
	}
	key := fmt.Sprintf("uploads/%s/%s-%s", tenantID, uuid.NewString(), clean)
	if prefix != "" {
		key = prefix + "/" + key
	}
	return key
}

// Presigned is the result of signing an upload: where the browser PUTs the bytes, the headers it MUST
// echo on that PUT (the signature covers them), and where the object is publicly readable afterward.
type Presigned struct {
	Key       string            `json:"key"`
	UploadURL string            `json:"upload_url"`
	FileURL   string            `json:"file_url"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// PresignUpload signs a short-lived PUT URL for a tenant's upload and returns it plus the eventual
// public URL. The browser PUTs the file to UploadURL; nothing flows through this service. When the Store
// serves public/CDN URLs it signs an x-amz-acl: public-read header into the URL so the uploaded object
// is world-readable — the browser must echo every header in Headers or the signature check fails.
func (s *Store) PresignUpload(ctx context.Context, tenantID, filename string, ttl time.Duration) (Presigned, error) {
	if !s.Enabled() {
		return Presigned{}, fmt.Errorf("storage not configured")
	}
	key := objectKey(s.prefix, tenantID, filename)
	if !s.publicRead {
		u, err := s.client.PresignedPutObject(ctx, s.bucket, key, ttl)
		if err != nil {
			return Presigned{}, err
		}
		return Presigned{Key: key, UploadURL: u.String(), FileURL: s.publicBase + "/" + key}, nil
	}
	h := http.Header{}
	h.Set("x-amz-acl", "public-read")
	u, err := s.client.PresignHeader(ctx, http.MethodPut, s.bucket, key, ttl, url.Values{}, h)
	if err != nil {
		return Presigned{}, err
	}
	return Presigned{
		Key: key, UploadURL: u.String(), FileURL: s.publicBase + "/" + key,
		Headers: map[string]string{"x-amz-acl": "public-read"},
	}, nil
}
