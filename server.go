package roxy

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type Server struct {
	config *Config
	router *http.ServeMux
	client *http.Client
}

// NewServer returns a configured Server instance.
func NewServer(config *Config) *Server {
	if config == nil {
		config = NewConfig()
	}

	client := &http.Client{
		Timeout: config.ClientTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.SSLNoVerify,
			},
		},
	}

	return &Server{
		config: config,
		router: http.NewServeMux(),
		client: client,
	}
}

// Start starts the server on the configured address.
func (s *Server) Start() error {
	if err := checkStorageDir(s.config.CacheDir); err != nil {
		return fmt.Errorf("invalid storage dirictory %s: %s", s.config.CacheDir, err)
	}

	w := os.Stdout
	fs := http.FileServer(http.Dir(s.config.CacheDir))

	s.router.Handle("/cache/", http.StripPrefix("/cache/", chain(s.cacheFile(), logger(w), method(http.MethodGet))))
	s.router.Handle("/file/", http.StripPrefix("/file", chain(fs, logger(w), method(http.MethodGet))))

	srv := &http.Server{
		Addr:         s.config.Addr,
		Handler:      s.router,
		ReadTimeout:  s.config.ServerReadTimeout,
		WriteTimeout: s.config.ServerWriteTimeout,
	}

	return srv.ListenAndServe()
}

func checkStorageDir(dir string) error {
	finfo, err := os.Stat(dir)

	if err != nil {
		return err
	} else if !finfo.IsDir() {
		return errors.New("not a directory")
	}

	file, err := ioutil.TempFile(dir, "test")

	if err != nil {
		return err
	}

	file.Close()
	os.Remove(file.Name())

	return nil
}
