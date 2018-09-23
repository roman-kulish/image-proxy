package roxy

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

const meta = ".meta"

var mapping = map[string]string{
	"image/gif":  ".gif",
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/bmp":  ".bmp",
	"image/webp": ".webp",
}

func (s *Server) cacheFile() http.Handler {
	logger := log.New(os.Stderr, "", log.LstdFlags)

	sendError := func(w http.ResponseWriter, err error) {
		logger.Println(err)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var url string
		var ssl bool

		if url = r.URL.Path; url == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if v := r.URL.Query().Get("ssl"); v != "" {
			if f, err := strconv.ParseBool(v); err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			} else {
				ssl = f
			}
		}

		name := filepath.Base(url)

		if ssl {
			url = "https://" + url
		} else {
			url = "http://" + url
		}

		hash := fmt.Sprintf("%x", md5.Sum([]byte(url)))
		path := filepath.Join(s.config.CacheDir, hash[0:2], hash[2:])
		meta := filepath.Join(path, meta)

		// Check if meta file exists and short circuit if so
		if _, err := os.Stat(meta); err == nil {
			if meta, err := ioutil.ReadFile(meta); err != nil {
				sendError(w, fmt.Errorf("error reading meta file %s: %s", meta, err))
			} else {
				w.Write(meta)
			}

			return
		}

		if err := os.MkdirAll(path, 0755); err != nil {
			sendError(w, fmt.Errorf("error creating directory %s: %s", path, err))
			return
		}

		res, err := s.client.Get(url)

		if err != nil {
			sendError(w, fmt.Errorf("error downloading file %s: %s", url, err))
			return
		}

		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			http.Error(w, http.StatusText(res.StatusCode), res.StatusCode)
			return
		}

		data := make([]byte, 512)

		if _, err := io.ReadFull(res.Body, data); err != nil && err != io.ErrUnexpectedEOF {
			sendError(w, fmt.Errorf("error reading response body: %s", err))
			return
		}

		ct := http.DetectContentType(data)
		ext, ok := mapping[ct]

		if !ok {
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return
		}

		if filepath.Ext(name) == "" {
			name += ext
		}

		file, err := ioutil.TempFile(path, "download")

		if err != nil {
			sendError(w, fmt.Errorf("error creating temp file: %s", err))
			return
		}

		if _, err := file.Write(data); err != nil {
			file.Close()

			sendError(w, fmt.Errorf("error writing to temp file: %s", err))
			return
		}

		if _, err := io.Copy(file, res.Body); err != nil {
			file.Close()

			sendError(w, fmt.Errorf("error writing to temp file: %s", err))
			return
		}

		file.Close()

		dest := filepath.Join(path, name)

		if err := os.Rename(file.Name(), dest); err != nil {
			os.Remove(file.Name())

			sendError(w, fmt.Errorf("error renaming temp file %s to %s: %s", file.Name(), dest, err))
			return
		}

		furl := []byte(fmt.Sprintf("http://%s/file/%s/%s/%s", r.Host, hash[0:2], hash[2:], name))

		if err := ioutil.WriteFile(meta, furl, 0644); err != nil {
			sendError(w, fmt.Errorf("error writing meta file %s: %s", meta, err))
			return
		}

		w.Write(furl)
	})
}
