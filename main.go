package notificationsapp

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/notificationsapp/assets"
	"github.com/shurcooL/notificationsapp/component"
)

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation.
type contextKey struct {
	name string
}

func (k *contextKey) String() string {
	return "github.com/shurcooL/notificationsapp context value " + k.name
}

// BaseURIContextKey is a context key for the request's base URI.
// That value specifies the base URI prefix to use for all absolute URLs.
// The associated value will be of type string.
var BaseURIContextKey = &contextKey{"BaseURI"}

// Options for configuring notifications app.
type Options struct {
	HeadPre template.HTML
	BodyPre template.HTML

	// BodyTop provides components to include on top of <body> of page rendered for req. It can be nil.
	BodyTop func(req *http.Request) ([]htmlg.Component, error)
}

type handler struct {
	http.Handler

	ns notifications.Service

	opt Options
}

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
	handler := &handler{
		ns:  service,
		opt: opt,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.NotificationsHandler)
	assetsFileServer := httpgzip.FileServer(assets.Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed})
	mux.Handle("/assets/", assetsFileServer)

	handler.Handler = mux
	return handler
}

var notificationsHTML = template.Must(template.New("").Parse(`<html>
	<head>
		{{.HeadPre}}
		<link href="{{.BaseURI}}/assets/style.css" rel="stylesheet" type="text/css" />
		<script src="{{.BaseURI}}/assets/script/script.js" type="text/javascript"></script>
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

	ns, err := h.ns.List(req.Context(), notifications.ListOptions{})
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
