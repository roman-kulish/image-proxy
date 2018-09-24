package roxy

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const (
	timeFormat   = "2006/01/02 15:04:05"
	patternCache = "/cache/"
	patternFile  = "/file/"
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

	if s.config.AccessWriter == nil {
		s.config.AccessWriter = os.Stdout
	}

	if s.config.ErrorWriter == nil {
		s.config.ErrorWriter = os.Stderr
	}

	fs := http.FileServer(http.Dir(s.config.CacheDir))
	logger := logger(s.config.AccessWriter)
	method := method(http.MethodGet)

	s.router.Handle(patternCache, http.StripPrefix(patternCache, chain(s.cacheFile(), logger, method)))
	s.router.Handle(patternFile, http.StripPrefix("/file", chain(fs, logger, method)))

	srv := &http.Server{
		Addr:         s.config.Addr,
		Handler:      s.router,
		ReadTimeout:  s.config.ServerReadTimeout,
		WriteTimeout: s.config.ServerWriteTimeout,
	}

	return srv.ListenAndServe()
}

func (s *Server) sendError(w http.ResponseWriter, err error) {
	fmt.Fprintf(s.config.ErrorWriter, "%s [error] %s\n",
		time.Now().Format(timeFormat), // :datetime
		err) // :error

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
