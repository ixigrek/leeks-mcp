package leekwars

import (
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
	if err := c.wait(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/"+path, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s : %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("GET %s : lecture du corps : %w", path, err)
	}
	if msg, failed := apiError(body); failed || resp.StatusCode != http.StatusOK {
		if msg == "" {
			msg = strings.TrimSpace(string(body))
			if len(msg) > 200 {
				msg = msg[:200]
			}
		}
		return nil, fmt.Errorf("GET %s : HTTP %d : %s", path, resp.StatusCode, msg)
	}
	return body, nil
}

// apiError détecte un corps {"success": false, "error": "..."}.
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
