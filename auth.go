package main

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	sessionCookie   = "session"
	sessionTTL      = 7 * 24 * time.Hour
	pbkdf2Iter      = 600_000
	maxLoginFails   = 5
	loginLockout    = 5 * time.Minute
	minPasswordLen  = 8
	maxAuthBodySize = 4 << 10
)

type credentials struct {
	Username   string `json:"username"`
	Salt       []byte `json:"salt"`
	Hash       []byte `json:"hash"`
	Iterations int    `json:"iterations"`
}

type failState struct {
	count int
	until time.Time
}

type authStore struct {
	mu       sync.Mutex
	path     string
	creds    *credentials         // nil until first-run setup
	sessions map[string]time.Time // token -> expiry
	fails    map[string]*failState
}

func newAuthStore(dataDir string) (*authStore, error) {
	a := &authStore{
		path:     filepath.Join(dataDir, "auth.json"),
		sessions: map[string]time.Time{},
		fails:    map[string]*failState{},
	}
	data, err := os.ReadFile(a.path)
	if errors.Is(err, os.ErrNotExist) {
		log.Printf("No user configured yet: open the web UI to create one")
		return a, nil
	}
	if err != nil {
		return nil, err
	}
	var c credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	a.creds = &c
	return a, nil
}

func hashPassword(password string, salt []byte, iter int) []byte {
	h, err := pbkdf2.Key(sha256.New, password, salt, iter, 32)
	if err != nil {
		panic(err)
	}
	return h
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (a *authStore) validSession(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	exp, ok := a.sessions[c.Value]
	if ok && time.Now().After(exp) {
		delete(a.sessions, c.Value)
		return false
	}
	return ok
}

func (a *authStore) startSession(w http.ResponseWriter, r *http.Request) {
	token := hex.EncodeToString(randomBytes(32))
	a.mu.Lock()
	a.sessions[token] = time.Now().Add(sessionTTL)
	a.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteStrictMode,
	})
}

type authReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func decodeAuthReq(w http.ResponseWriter, r *http.Request) (authReq, bool) {
	var req authReq
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodySize)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid JSON", http.StatusBadRequest)
		return req, false
	}
	req.Username = strings.TrimSpace(req.Username)
	return req, true
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (a *authStore) handleStatus(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	needsSetup := a.creds == nil
	a.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"needsSetup": needsSetup})
}

func (a *authStore) handleSetup(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeAuthReq(w, r)
	if !ok {
		return
	}
	if req.Username == "" || len(req.Password) < minPasswordLen {
		jsonError(w, "Username is required and password must have at least 8 characters", http.StatusBadRequest)
		return
	}

	salt := randomBytes(16)
	c := &credentials{
		Username:   req.Username,
		Salt:       salt,
		Hash:       hashPassword(req.Password, salt, pbkdf2Iter),
		Iterations: pbkdf2Iter,
	}
	data, _ := json.Marshal(c)

	a.mu.Lock()
	if a.creds != nil {
		a.mu.Unlock()
		jsonError(w, "Already configured", http.StatusForbidden)
		return
	}
	err := os.MkdirAll(filepath.Dir(a.path), 0700)
	if err == nil {
		err = os.WriteFile(a.path, data, 0600)
	}
	if err == nil {
		a.creds = c
	}
	a.mu.Unlock()

	if err != nil {
		log.Printf("[AUTH] Failed to save credentials: %v", err)
		jsonError(w, "Failed to save credentials", http.StatusInternalServerError)
		return
	}
	log.Printf("[AUTH] User %q created", c.Username)
	a.startSession(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (a *authStore) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)

	a.mu.Lock()
	c := a.creds
	f := a.fails[ip]
	locked := f != nil && time.Now().Before(f.until)
	a.mu.Unlock()

	if c == nil {
		jsonError(w, "Setup required", http.StatusConflict)
		return
	}
	if locked {
		jsonError(w, "Too many failed attempts, try again later", http.StatusTooManyRequests)
		return
	}

	req, ok := decodeAuthReq(w, r)
	if !ok {
		return
	}

	// Always hash so a wrong username takes as long as a wrong password.
	hash := hashPassword(req.Password, c.Salt, c.Iterations)
	userOK := subtle.ConstantTimeCompare([]byte(req.Username), []byte(c.Username)) == 1
	passOK := subtle.ConstantTimeCompare(hash, c.Hash) == 1

	if !userOK || !passOK {
		a.mu.Lock()
		f := a.fails[ip]
		if f == nil || time.Now().After(f.until) && f.count >= maxLoginFails {
			f = &failState{}
			a.fails[ip] = f
		}
		f.count++
		if f.count >= maxLoginFails {
			f.until = time.Now().Add(loginLockout)
		}
		a.mu.Unlock()
		log.Printf("[AUTH] Failed login from %s", ip)
		jsonError(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	a.mu.Lock()
	delete(a.fails, ip)
	a.mu.Unlock()
	a.startSession(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (a *authStore) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		a.mu.Lock()
		delete(a.sessions, c.Value)
		a.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	w.WriteHeader(http.StatusNoContent)
}

func (a *authStore) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if a.validSession(r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	data, _ := staticFS.ReadFile("static/login.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

var publicPaths = map[string]bool{
	"/login":            true,
	"/api/auth/status":  true,
	"/api/auth/login":   true,
	"/api/auth/setup":   true,
	"/static/style.css": true,
	"/static/login.js":  true,
}

// middleware adds basic security headers and requires a valid session for everything but the login flow.
func (a *authStore) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")

		if publicPaths[r.URL.Path] || a.validSession(r) {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			jsonError(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	})
}
