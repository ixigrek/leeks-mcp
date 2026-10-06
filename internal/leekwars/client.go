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
	return c.send(ctx, http.MethodPost, path, payload)
}

// do envoie la requête après passage par le limiteur, ajoute le Bearer et
// traduit les réponses en échec (HTTP ≠ 200 ou success:false) en erreur sans le
// token. Un 429 (rate_limit, retry_after en secondes) est rejoué une fois après
// le délai demandé : le serveur compte plus strictement que le limiteur local.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, error) {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = io.ReadAll(body); err != nil {
			return nil, err
		}
	}
	data, status, err := c.once(ctx, method, path, payload, contentType)
	if status == http.StatusTooManyRequests {
		select {
		case <-time.After(retryAfter(data)):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		data, status, err = c.once(ctx, method, path, payload, contentType)
	}
	if err != nil {
		return nil, err
	}
	if msg, failed := apiError(data); failed || status != http.StatusOK {
		if msg == "" {
			msg = strings.TrimSpace(string(data))
			// Certains échecs (garden/start-*) renvoient une simple chaîne JSON.
			var bare string
			if json.Unmarshal(data, &bare) == nil {
				msg = bare
			}
			if len(msg) > 200 {
				msg = msg[:200]
			}
		}
		return nil, fmt.Errorf("%s %s : HTTP %d : %s", method, path, status, msg)
	}
	return data, nil
}

// retryAfter lit le délai d'un 429, 1 s par défaut.
func retryAfter(body []byte) time.Duration {
	var probe struct {
		RetryAfter float64 `json:"retry_after"`
	}
	if json.Unmarshal(body, &probe) == nil && probe.RetryAfter > 0 {
		return time.Duration(probe.RetryAfter * float64(time.Second))
	}
	return time.Second
}

// once envoie une requête et renvoie le corps et le code HTTP.
func (c *Client) once(ctx context.Context, method, path string, payload []byte, contentType string) ([]byte, int, error) {
	if err := c.wait(ctx); err != nil {
		return nil, 0, err
	}
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+"/api/"+path, body)
	if err != nil {
		return nil, 0, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%s %s : %w", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("%s %s : lecture du corps : %w", method, path, err)
	}
	return data, resp.StatusCode, nil
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

// Put appelle PUT /api/<path> avec payload sérialisé en JSON (loadout/update).
func (c *Client) Put(ctx context.Context, path string, payload any) ([]byte, error) {
	return c.send(ctx, http.MethodPut, path, payload)
}

// Delete appelle DELETE /api/<path> avec payload sérialisé en JSON, comme le
// client officiel (loadout/delete).
func (c *Client) Delete(ctx context.Context, path string, payload any) ([]byte, error) {
	return c.send(ctx, http.MethodDelete, path, payload)
}

func (c *Client) send(ctx context.Context, method, path string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%s %s : encodage : %w", method, path, err)
	}
	return c.do(ctx, method, path, bytes.NewReader(data), "application/json; charset=UTF-8")
}
