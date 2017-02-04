package notificationsapp

import (
	"bytes"
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
	"github.com/shurcooL/users"
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

type Options struct {
	HeadPre template.HTML
	BodyPre template.HTML

	// BodyTop provides components to include on top of <body> of page rendered for req. It can be nil.
	BodyTop func(req *http.Request) ([]htmlg.ComponentContext, error)
}

type handler struct {
	http.Handler

	ns notifications.Service
	us users.Service

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
// 	http.Handle(httproute.MarkRead, errorHandler{apiHandler.MarkRead})
// 	http.Handle(httproute.MarkAllRead, errorHandler{apiHandler.MarkAllRead})
//
func New(service notifications.Service, users users.Service, opt Options) http.Handler {
	handler := &handler{
		ns:  service,
		us:  users,
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
		{{.BodyPre}}
		{{.BodyTop}}`))

func (h *handler) NotificationsHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		httperror.HandleMethod(w, httperror.Method{Allowed: []string{"GET"}})
		return
	}

	// THINK: Try to let service take care of authorization check. Let's see if it's a good idea...
	//        Nope, seems like bad idea, at least with the current err = t.ExecuteTemplate() error handling,
	//        maybe need to fix that up.
	if user, err := h.us.GetAuthenticatedSpec(req.Context()); err != nil {
		log.Println("us.GetAuthenticatedSpec:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if user.ID == 0 {
		http.Error(w, "this page requires an authenticated user", http.StatusUnauthorized)
		return
	}

	ns, err := h.ns.List(req.Context(), notifications.ListOptions{})
	if os.IsPermission(err) {
		http.Error(w, "403 Forbidden", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println("h.ns.List:", err)
		return
	}

	type state struct {
		BaseURI string
		HeadPre template.HTML
		BodyPre template.HTML
		BodyTop template.HTML
	}

	s, err := func() (state, error) {
		baseURI, ok := req.Context().Value(BaseURIContextKey).(string)
		if !ok {
			return state{}, fmt.Errorf("request to %v doesn't have notificationsapp.BaseURIContextKey context key set", req.URL.Path)
		}

		// TODO: Caller still does a lot of work outside to calculate req.URL.Path by
		//       subtracting BaseURI from full original req.URL.Path. We should be able
		//       to compute it here internally by using req.RequestURI and BaseURI.
		b := state{
			BaseURI: baseURI,
			HeadPre: h.opt.HeadPre,
			BodyPre: h.opt.BodyPre,
		}
		if h.opt.BodyTop != nil {
			c, err := h.opt.BodyTop(req)
			if err != nil {
				return state{}, err
			}
			var buf bytes.Buffer
			err = htmlg.RenderComponentsContext(req.Context(), &buf, c...)
			if err != nil {
				return state{}, err
			}
			b.BodyTop = template.HTML(buf.String())
		}

		return b, nil
	}()
	if err != nil {
		log.Println("state:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = notificationsHTML.Execute(w, &s)
	if err != nil {
		log.Println("notificationsHTML.ExecuteTemplate:", err)
		return
	}

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
