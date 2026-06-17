package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Credentials struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type credentialsFile map[string]*Credentials

func credentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "flux", "credentials.json"), nil
}

func Load(env string) (*Credentials, error) {
	path, err := credentialsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var file credentialsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}

	return file[env], nil
}

func Save(env string, creds *Credentials) error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	var file credentialsFile
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &file) //nolint:errcheck
	}
	if file == nil {
		file = make(credentialsFile)
	}

	file[env] = creds

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func Clear(env string) error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var file credentialsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}

	delete(file, env)

	if len(file) == 0 {
		return os.Remove(path)
	}

	out, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0600)
}
