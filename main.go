package notificationsapp

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/notificationsapp/assets"
	"github.com/shurcooL/notificationsapp/component"
)

// New returns a notifications app http.Handler using given services and options.
//
// In order to serve HTTP requests, the returned http.Handler expects each incoming
// request to have a parameter provided to it via BaseURIContextKey context key.
// For example:
//
// 	notificationsApp := notificationsapp.New(...)
//
// 	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
// 		req = req.WithContext(context.WithValue(req.Context(), notificationsapp.BaseURIContextKey, string(...)))
// 		notificationsApp.ServeHTTP(w, req)
// 	})
//
// An HTTP API must be available:
//
// 	// Register HTTP API endpoints.
// 	apiHandler := httphandler.Notifications{Notifications: service}
// 	http.Handle(httproute.MarkRead, errorHandler(apiHandler.MarkRead))
// 	http.Handle(httproute.MarkAllRead, errorHandler(apiHandler.MarkAllRead))
func New(service notifications.Service, opt Options) http.Handler {
	return &handler{
		ns:               service,
		assetsFileServer: httpgzip.FileServer(assets.Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed}),
		opt:              opt,
	}
}

// handler handles all requests to notificationsapp. It acts
// like a request multiplexer, choosing from various endpoints.
type handler struct {
	ns notifications.Service

	assetsFileServer http.Handler

	opt Options
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Handle "/assets/...".
	if strings.HasPrefix(req.URL.Path, "/assets/") {
		req = stripPrefix(req, len("/assets"))
		h.assetsFileServer.ServeHTTP(w, req)
		return
	}

	// Handle all other non-"/".
	if req.URL.Path != "/" {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}

	// Handle "/".
	h.NotificationsHandler(w, req)
}

// Options for configuring notifications app.
type Options struct {
	HeadPre template.HTML
	BodyPre template.HTML

	// BodyTop provides components to include on top of <body> of page rendered for req. It can be nil.
	BodyTop func(req *http.Request) ([]htmlg.Component, error)
}

// BaseURIContextKey is a context key for the request's base URI.
// That value specifies the base URI prefix to use for all absolute URLs.
// The associated value will be of type string.
var BaseURIContextKey = &contextKey{"BaseURI"}

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation.
type contextKey struct {
	name string
}

func (k *contextKey) String() string {
	return "github.com/shurcooL/notificationsapp context value " + k.name
}

var notificationsHTML = template.Must(template.New("").Parse(`<html>
	<head>
		{{.HeadPre}}
		<link href="{{.BaseURI}}/assets/style.css" rel="stylesheet" type="text/css" />
		<script src="{{.BaseURI}}/assets/script.js" type="text/javascript"></script>
	</head>
	<body>
		{{.BodyPre}}`))

func (h *handler) NotificationsHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		httperror.HandleMethod(w, httperror.Method{Allowed: []string{"GET"}})
		return
	}

	// TODO: Caller still does a lot of work outside to calculate req.URL.Path by
	//       subtracting BaseURI from full original req.URL.Path. We should be able
	//       to compute it here internally by using req.RequestURI and BaseURI.
	baseURI, ok := req.Context().Value(BaseURIContextKey).(string)
	if !ok {
		err := fmt.Errorf("request to %v doesn't have notificationsapp.BaseURIContextKey context key set", req.URL.Path)
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	all, _ := strconv.ParseBool(req.URL.Query().Get("all"))
	ns, err := h.ns.List(req.Context(), notifications.ListOptions{
		All: all,
	})
	if os.IsPermission(err) {
		// HACK: os.IsPermission(err) could be 401 or 403, we don't know,
		//       so just going with 403 for now. This should be cleaned up.
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	} else if err != nil {
		log.Println("h.ns.List:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	state := struct {
		BaseURI string
		HeadPre template.HTML
		BodyPre template.HTML // E.g., <div style="max-width: 800px; margin: 0 auto 100px auto;">.
	}{
		baseURI,
		h.opt.HeadPre,
		h.opt.BodyPre,
	}
	err = notificationsHTML.Execute(w, &state)
	if err != nil {
		log.Println("notificationsHTML.ExecuteTemplate:", err)
		return
	}

	// E.g., a header component.
	if h.opt.BodyTop != nil {
		c, err := h.opt.BodyTop(req)
		if err != nil {
			log.Println("h.opt.BodyTop:", err)
			return
		}
		err = htmlg.RenderComponents(w, c...)
		if err != nil {
			log.Println("htmlg.RenderComponents:", err)
			return
		}
	}

	// Render the notifications contents.
	err = htmlg.RenderComponents(w, component.NotificationsByRepo{Notifications: ns})
	if err != nil {
		log.Println("htmlg.RenderComponents:", err)
		return
	}

	_, err = io.WriteString(w, `</body></html>`)
	if err != nil {
		log.Println("io.WriteString:", err)
		return
	}
}

// stripPrefix returns request r with prefix of length prefixLen stripped from r.URL.Path.
// prefixLen must not be longer than len(r.URL.Path), otherwise stripPrefix panics.
// If r.URL.Path is empty after the prefix is stripped, the path is changed to "/".
func stripPrefix(r *http.Request, prefixLen int) *http.Request {
	r2 := new(http.Request)
	*r2 = *r
	r2.URL = new(url.URL)
	*r2.URL = *r.URL
	r2.URL.Path = r.URL.Path[prefixLen:]
	if r2.URL.Path == "" {
		r2.URL.Path = "/"
	}
	return r2
}
