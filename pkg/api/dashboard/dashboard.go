package dashboard

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/webui"
)

// Handler expose dashboard routes.
type Handler struct {
	assets fs.FS // optional assets, to override the webui.FS default
}

// Append adds dashboard routes on the given router, optionally using the given
// assets (or webui.FS otherwise).
func Append(router *mux.Router, prefix string, customAssets fs.FS) {
	assets := customAssets
	if assets == nil {
		assets = webui.FS
	}

	indexTemplate, err := template.ParseFS(assets, "index.html")
	if err != nil {
		log.WithoutContext().WithError(err).Error("unable to load index.html")
	}

	// Expose dashboard
	router.Methods(http.MethodGet).
		Path(fmt.Sprintf("%s/", prefix)).
		HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			http.Redirect(rw, req, safePrefix(req)+fmt.Sprintf("%s/dashboard/", prefix), http.StatusFound)
		})

	router.Methods(http.MethodGet).
		Path(fmt.Sprintf("%s/dashboard/robots.txt", prefix)).
		PathPrefix(fmt.Sprintf("%s/dashboard/assets/", prefix)).
		HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			http.StripPrefix(fmt.Sprintf("%s/dashboard/", prefix), http.FileServerFS(assets)).ServeHTTP(rw, req)
		})

	router.Methods(http.MethodGet).
		PathPrefix(fmt.Sprintf("%s/dashboard/", prefix)).
		HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			basePath := req.Header.Get("X-Forwarded-Prefix")

			// Ensure there's a trailing slash at the end of the base path.
			// Browsers removes everything after the last slash before building relative URLs.
			basePath = ensureTrailingSlash(basePath)

			if err = indexTemplate.Execute(rw, indexTemplateData{BasePath: prefix}); err != nil {
				log.WithoutContext().WithError(err).Error("Unable to serve APIPortal index.html page")
			}
		})
}

type indexTemplateData struct {
	BasePath string
}

func ensureTrailingSlash(path string) string {
	if path == "" || path[len(path)-1:] != "/" {
		return path + "/"
	}

	return path
}

func (g Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	assets := g.assets
	if assets == nil {
		assets = webui.FS
	}

	if req.RequestURI == "/" {
		indexTemplate, err := template.ParseFS(assets, "index.html")
		if err != nil {
			log.WithoutContext().WithError(err).Error("unable to load index.html")
		}

		basePath := req.Header.Get("X-Forwarded-Prefix")

		// Ensure there's a trailing slash at the end of the base path.
		// Browsers removes everything after the last slash before building relative URLs.
		basePath = ensureTrailingSlash(basePath)

		if err = indexTemplate.Execute(rw, indexTemplateData{BasePath: basePath}); err != nil {
			log.WithoutContext().WithError(err).Error("Unable to serve dashboard index.html page")
		}

		return
	}

	http.FileServerFS(assets).ServeHTTP(rw, req)
}

func safePrefix(req *http.Request) string {
	prefix := req.Header.Get("X-Forwarded-Prefix")
	if prefix == "" {
		return ""
	}

	parse, err := url.Parse(prefix)
	if err != nil {
		return ""
	}

	if parse.Host != "" {
		return ""
	}

	return parse.Path
}
