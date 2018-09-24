package roxy

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
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
				sendError(w, s.config.ErrorLogWriter, fmt.Errorf("failed reading meta file %s: %s", meta, err))
			} else {
				// addr:= path.Joi fmt.Sprintf("%s://%s", r.URL.Scheme, r.Host)


				fmt.Fprintf(w, "%s://%s%s%s", r.URL.Scheme, r.Host, patternCache, meta)
			}

			return
		}

		if err := os.MkdirAll(path, 0755); err != nil {
			sendError(w, s.config.ErrorLogWriter, fmt.Errorf("failed creating directory %s: %s", path, err))
			return
		}

		res, err := s.client.Get(url)

		if err != nil {
			sendError(w, s.config.ErrorLogWriter, fmt.Errorf("failed downloading file %s: %s", url, err))
			return
		}

		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			http.Error(w, http.StatusText(res.StatusCode), res.StatusCode)
			return
		}

		data := make([]byte, 512)

		if _, err := io.ReadFull(res.Body, data); err != nil && err != io.ErrUnexpectedEOF {
			sendError(w, s.config.ErrorLogWriter, fmt.Errorf("failed reading downloaded file header: %s", err))
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
			sendError(w, s.config.ErrorLogWriter, fmt.Errorf("failed creating temp file: %s", err))
			return
		}

		if _, err := file.Write(data); err != nil {
			file.Close()

			sendError(w, s.config.ErrorLogWriter, fmt.Errorf("failed writing to temp file: %s", err))
			return
		}

		if _, err := io.Copy(file, res.Body); err != nil {
			file.Close()

			sendError(w, s.config.ErrorLogWriter, fmt.Errorf("faied writing to temp file: %s", err))
			return
		}

		file.Close()

		dest := filepath.Join(path, name)

		if err := os.Rename(file.Name(), dest); err != nil {
			os.Remove(file.Name())

			sendError(w, s.config.ErrorLogWriter, fmt.Errorf("failed renaming temp file %s to %s: %s", file.Name(), dest, err))
			return
		}

		furl := []byte(fmt.Sprintf("http://%s/file/%s/%s/%s", r.Host, hash[0:2], hash[2:], name))

		if err := ioutil.WriteFile(meta, furl, 0644); err != nil {
			sendError(w, s.config.ErrorLogWriter, fmt.Errorf("failed writing meta file %s: %s", meta, err))
			return
		}

		w.Write(furl)
	})
}

func sendError(w http.ResponseWriter, out io.Writer, err error) {
	fmt.Fprintf(out, "%s [error] %s\n",
		time.Now().Format(timeFormat), // :datetime
		err) // :error

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
