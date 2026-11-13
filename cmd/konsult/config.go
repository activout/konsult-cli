package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/zalando/go-keyring"
)

const keyringService = "konsult"

type creds struct {
	User   string `json:"user"`
	Sender string `json:"sender"`
}

// loadCreds: environment variables win over the OS keychain. Returns the source for `auth status`.
func loadCreds(env string) (creds, string, error) {
	u, s := os.Getenv("KONSULT_USER_KEY"), os.Getenv("KONSULT_SENDER_KEY")
	if u != "" && s != "" {
		return creds{u, s}, "environment", nil
	}
	raw, err := keyring.Get(keyringService, env)
	if err == nil {
		var c creds
		if json.Unmarshal([]byte(raw), &c) == nil && c.User != "" && c.Sender != "" {
			return c, "os keychain", nil
		}
		return creds{}, "", fail(exitAuth, "keychain entry for %q is corrupt; run `konsult auth login --env %s`", env, env)
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		return creds{}, "", fail(exitAuth, "keychain unavailable (%v); set KONSULT_USER_KEY and KONSULT_SENDER_KEY instead", err)
	}
	return creds{}, "", fail(exitAuth, "no credentials for %q: run `konsult auth login --env %s` or set KONSULT_USER_KEY and KONSULT_SENDER_KEY", env, env)
}

func saveCreds(env string, c creds) error {
	b, _ := json.Marshal(c)
	return keyring.Set(keyringService, env, string(b))
}

func deleteCreds(env string) error {
	err := keyring.Delete(keyringService, env)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// resolveBase returns the API base URL. --base-url is a developer override and must be https
// (plain http only for loopback mock servers).
func resolveBase(env, override string) (string, error) {
	if override != "" {
		u, err := url.Parse(override)
		if err != nil || u.Host == "" {
			return "", fail(exitUsage, "invalid --base-url")
		}
		loop := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"
		if u.Scheme != "https" && !(u.Scheme == "http" && loop) {
			return "", fail(exitUsage, "--base-url must be https (http only for localhost)")
		}
		return override, nil
	}
	b, ok := bases[strings.ToLower(env)]
	if !ok {
		return "", fail(exitUsage, "unknown environment %q (use sandbox or production)", env)
	}
	return b, nil
}

func describeCreds(env string) string {
	_, src, err := loadCreds(env)
	if err != nil {
		return "not configured"
	}
	return fmt.Sprintf("configured (%s)", src)
}
