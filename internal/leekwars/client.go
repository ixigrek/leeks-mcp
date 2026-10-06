package leekwars

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DefaultBaseURL est la racine de l'API LeekWars.
const DefaultBaseURL = "https://leekwars.com"

// Client appelle l'API LeekWars en respectant la limite de 5 requêtes par seconde.
type Client struct {
	base  string
	token string
	http  *http.Client

	mu   sync.Mutex
	next time.Time
}

const minInterval = 200 * time.Millisecond

// NewClient construit un client ; token vide = appels publics seulement.
func NewClient(base, token string) *Client {
	return &Client{
		base:  strings.TrimRight(base, "/"),
		token: token,
		http:  &http.Client{Timeout: 15 * time.Second},
	}
}

// HasToken indique si le client est authentifié.
func (c *Client) HasToken() bool { return c.token != "" }

// Get appelle GET /api/<path> et renvoie le corps brut. Une réponse HTTP non 200
// ou un corps {"success": false} devient une erreur qui ne cite jamais le token.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.do(ctx, http.MethodGet, path, nil, "")
}

// Post appelle POST /api/<path> avec payload sérialisé en JSON et renvoie le corps brut.
func (c *Client) Post(ctx context.Context, path string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("POST %s : encodage : %w", path, err)
	}
	return c.do(ctx, http.MethodPost, path, bytes.NewReader(data), "application/json; charset=UTF-8")
}

// do envoie la requête après passage par le limiteur, ajoute le Bearer et
// traduit les réponses en échec (HTTP ≠ 200 ou success:false) en erreur sans le token.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, error) {
	if err := c.wait(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+"/api/"+path, body)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s : %w", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s %s : lecture du corps : %w", method, path, err)
	}
	if msg, failed := apiError(data); failed || resp.StatusCode != http.StatusOK {
		if msg == "" {
			msg = strings.TrimSpace(string(data))
			if len(msg) > 200 {
				msg = msg[:200]
			}
		}
		return nil, fmt.Errorf("%s %s : HTTP %d : %s", method, path, resp.StatusCode, msg)
	}
	return data, nil
}

func apiError(body []byte) (string, bool) {
	var probe struct {
		Success *bool           `json:"success"`
		Error   json.RawMessage `json:"error"`
	}
	if json.Unmarshal(body, &probe) != nil || probe.Success == nil || *probe.Success {
		return "", false
	}
	return strings.Trim(string(probe.Error), `"`), true
}

// wait bloque jusqu'au prochain créneau du limiteur (une requête toutes les 200 ms).
func (c *Client) wait(ctx context.Context) error {
	c.mu.Lock()
	now := time.Now()
	if c.next.After(now) {
		slot := c.next
		c.next = slot.Add(minInterval)
		c.mu.Unlock()
		select {
		case <-time.After(slot.Sub(now)):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	c.next = now.Add(minInterval)
	c.mu.Unlock()
	return nil
}
