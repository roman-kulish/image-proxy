package roxy

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var dldURL, reqScheme string
		var isSSL bool

		if dldURL = r.URL.Path; dldURL == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if v := r.URL.Query().Get("ssl"); v != "" {
			if f, err := strconv.ParseBool(v); err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			} else {
				isSSL = f
			}
		}

		fileName := filepath.Base(dldURL)

		if isSSL {
			dldURL = "https://" + dldURL
		} else {
			dldURL = "http://" + dldURL
		}

		if r.TLS != nil {
			reqScheme = "https://"
		} else {
			reqScheme = "http://"
		}

		fileNameHash := fmt.Sprintf("%x", md5.Sum([]byte(dldURL)))
		fileDir := filepath.Join(s.config.CacheDir, fileNameHash[0:2], fileNameHash[2:])
		fileMeta := filepath.Join(fileDir, meta)

		// Check if fileMeta tempFile exists and short circuit if so
		if _, err := os.Stat(fileMeta); err == nil {
			if meta, err := ioutil.ReadFile(fileMeta); err != nil {
				s.sendError(w, fmt.Errorf("failed reading meta file %s: %s", meta, err))
			} else {
				fmt.Fprint(w, reqScheme, path.Join(r.Host, patternFile, string(meta)))
			}

			return
		}

		if err := os.MkdirAll(fileDir, 0755); err != nil {
			s.sendError(w, fmt.Errorf("failed creating directory %s: %s", fileDir, err))
			return
		}

		res, err := s.client.Get(dldURL)

		if err != nil {
			s.sendError(w, fmt.Errorf("failed downloading file %s: %s", dldURL, err))
			return
		}

		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			http.Error(w, http.StatusText(res.StatusCode), res.StatusCode)
			return
		}

		data := make([]byte, 512)

		if _, err := io.ReadFull(res.Body, data); err != nil && err != io.ErrUnexpectedEOF {
			s.sendError(w, fmt.Errorf("failed reading header: %s", err))
			return
		}

		ct := http.DetectContentType(data)
		ext, ok := mapping[ct]

		if !ok {
			http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
			return
		}

		fileName += ext // force extension even if it exists

		tempFile, err := ioutil.TempFile(fileDir, "download")

		if err != nil {
			s.sendError(w, fmt.Errorf("failed creating temp file: %s", err))
			return
		}

		if _, err := tempFile.Write(data); err != nil {
			tempFile.Close()

			s.sendError(w, fmt.Errorf("failed writing to temp file: %s", err))
			return
		}

		if _, err := io.Copy(tempFile, res.Body); err != nil {
			tempFile.Close()

			s.sendError(w, fmt.Errorf("faied writing to temp file: %s", err))
			return
		}

		tempFile.Close()

		destFile := filepath.Join(fileDir, fileName)

		if err := os.Rename(tempFile.Name(), destFile); err != nil {
			os.Remove(tempFile.Name())

			s.sendError(w, fmt.Errorf("failed renaming temp file %s to %s: %s", tempFile.Name(), destFile, err))
			return
		}

		fileBasePath := fmt.Sprintf("%s/%s/%s", fileNameHash[0:2], fileNameHash[2:], fileName)

		if err := ioutil.WriteFile(fileMeta, []byte(fileBasePath), 0644); err != nil {
			s.sendError(w, fmt.Errorf("failed writing meta file %s: %s", fileMeta, err))
			return
		}

		fmt.Fprint(w, reqScheme, path.Join(r.Host, patternFile, fileBasePath))
	})
}
