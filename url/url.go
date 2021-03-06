package url

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	"encore.dev/storage/sqldb"
)

type ListUrlResponse struct {
	URLs []*URL `json:"urls"`
}

type URL struct {
	ID  string // short-form URL id
	URL string // complete URL, in long form
}

type ShortenParams struct {
	URL string // the URL to shorten
}

// insert inserts a URL into the database.
func insert(ctx context.Context, id, url string) error {
	_, err := sqldb.Exec(ctx, `
        INSERT INTO url (id, original_url)
        VALUES ($1, $2)
    `, id, url)
	return err
}

// Shorten shortens a URL.
//encore:api public method=POST path=/url
func Shorten(ctx context.Context, p *ShortenParams) (*URL, error) {
	id, err := generateID()
	if err != nil {
		return nil, err
	} else if err := insert(ctx, id, p.URL); err != nil {
		return nil, err
	}
	return &URL{ID: id, URL: p.URL}, nil
}

// generateID generates a random short ID.
func generateID() (string, error) {
	var data [6]byte // 6 bytes of entropy
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data[:]), nil
}

// Get retrieves the original URL for the id.
//encore:api public method=GET path=/url/:id
func Get(ctx context.Context, id string) (*URL, error) {
	u := &URL{ID: id}
	err := sqldb.QueryRow(ctx, `
        SELECT original_url FROM url
        WHERE id = $1
    `, id).Scan(&u.URL)
	return u, err
}

// List lists all URLs.
//encore:api public method=GET path=/url
func List(ctx context.Context) (*ListUrlResponse, error) {
	rows, err := sqldb.Query(ctx, `
        SELECT * FROM url ORDER BY id DESC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var urls []*URL

	for rows.Next() {
		var u URL
		if err := rows.Scan(&u.ID, &u.URL); err != nil {
			return nil, err
		}
		urls = append(urls, &u)
	}
	return &ListUrlResponse{URLs: urls}, rows.Err()
}

// Redirects to Orginal URL based on Id
//encore:api public raw method=GET path=/redirect/:id
func Redirect(w http.ResponseWriter, req *http.Request) {
	id := strings.TrimPrefix(req.URL.Path, "/redirect/")
	u := &URL{ID: id}
	err := sqldb.QueryRow(context.Background(), `
		SELECT original_url FROM url
		WHERE id = $1
	`, id).Scan(&u.URL)
	// redirect to original URL
	if err == nil {
		http.Redirect(w, req, u.URL, http.StatusMovedPermanently)
		return
	}
	http.Redirect(w, req, "https://golangflow.io/", http.StatusMovedPermanently)
}
