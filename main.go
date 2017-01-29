package notificationsapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"sort"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/go-goon"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httpfs/html/vfstemplate"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/notificationsapp/assets"
	"github.com/shurcooL/notificationsapp/common"
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

	Options
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
func New(service notifications.Service, users users.Service, opt Options) http.Handler {
	handler := &handler{
		ns:      service,
		us:      users,
		Options: opt,
	}

	err := handler.loadTemplates()
	if err != nil {
		log.Fatalln("loadTemplates:", err)
	}

	h := http.NewServeMux()
	h.HandleFunc("/mock/", handler.mockHandler)
	r := mux.NewRouter()
	// TODO: Make redirection work.
	//r.StrictSlash(true) // THINK: Can't use this due to redirect not taking baseURI into account.
	r.HandleFunc("/", handler.notificationsHandler).Methods("GET")
	r.HandleFunc("/mark-read", handler.postMarkReadHandler).Methods("POST")
	r.HandleFunc("/mark-all-read", handler.postMarkAllReadHandler).Methods("POST")
	h.Handle("/", r)
	assetsFileServer := httpgzip.FileServer(assets.Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed})
	h.Handle("/assets/", assetsFileServer)

	handler.Handler = h
	return handler
}

var t *template.Template

func (h *handler) loadTemplates() error {
	var err error
	t = template.New("").Funcs(template.FuncMap{
		"dump": func(v interface{}) string { return goon.Sdump(v) },
		"json": func(v interface{}) (string, error) {
			b, err := json.Marshal(v)
			return string(b), err
		},
		"jsonfmt": func(v interface{}) (string, error) {
			b, err := json.MarshalIndent(v, "", "\t")
			return string(b), err
		},
		"reltime": humanize.Time,
		"gfm":     func(s string) template.HTML { return template.HTML(github_flavored_markdown.Markdown([]byte(s))) },
		"string":  func(s *string) string { return *s },
		"base":    path.Base,
	})
	t, err = vfstemplate.ParseGlob(assets.Assets, t, "/assets/*.tmpl")
	return err
}

func (h *handler) state(req *http.Request) (state, error) {
	baseURI, ok := req.Context().Value(BaseURIContextKey).(string)
	if !ok {
		return state{}, fmt.Errorf("request to %v doesn't have notificationsapp.BaseURIContextKey context key set", req.URL.Path)
	}

	// TODO: Caller still does a lot of work outside to calculate req.URL.Path by
	//       subtracting BaseURI from full original req.URL.Path. We should be able
	//       to compute it here internally by using req.RequestURI and BaseURI.
	reqPath := req.URL.Path
	if reqPath == "/" {
		reqPath = "" // This is needed so that absolute URL for root view, i.e., /notifications, is "/notifications" and not "/notifications/" because of "/notifications" + "/".
	}
	b := state{
		State: common.State{
			BaseURI: baseURI,
			ReqPath: reqPath,
		},
	}
	b.req = req
	b.vars = mux.Vars(req)
	b.HeadPre = h.HeadPre
	b.BodyPre = h.BodyPre
	if h.BodyTop != nil {
		c, err := h.BodyTop(req)
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

	b.ns = h.ns

	return b, nil
}

type state struct {
	req  *http.Request
	vars map[string]string

	HeadPre template.HTML
	BodyPre template.HTML
	BodyTop template.HTML

	ns notifications.Service

	common.State
}

type repoNotifications struct {
	Repo          notifications.RepoSpec
	RepoURL       template.URL
	Notifications notifications.Notifications

	updatedAt time.Time // Most recent notification.
}

func (s state) RepoNotifications() ([]repoNotifications, error) {
	ns, err := s.ns.List(s.req.Context(), notifications.ListOptions{})
	if err != nil {
		return nil, err
	}

	rnm := make(map[notifications.RepoSpec]*repoNotifications)
	for _, n := range ns {
		var r notifications.RepoSpec = n.RepoSpec
		switch rnp := rnm[r]; rnp {
		case nil:
			rn := repoNotifications{
				Repo:          r,
				RepoURL:       n.RepoURL,
				Notifications: notifications.Notifications{n},
				updatedAt:     n.UpdatedAt,
			}
			rnm[r] = &rn
		default:
			if rnp.updatedAt.Before(n.UpdatedAt) {
				rnp.updatedAt = n.UpdatedAt
			}
			rnp.Notifications = append(rnp.Notifications, n)
		}
	}

	var rns []repoNotifications
	for _, rnp := range rnm {
		sort.Sort(rnp.Notifications)
		rns = append(rns, *rnp)
	}
	sort.Sort(byUpdatedAt(rns))

	return rns, nil
}

// byUpdatedAt implements sort.Interface.
type byUpdatedAt []repoNotifications

func (s byUpdatedAt) Len() int           { return len(s) }
func (s byUpdatedAt) Less(i, j int) bool { return !s[i].updatedAt.Before(s[j].updatedAt) }
func (s byUpdatedAt) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (h *handler) notificationsHandler(w http.ResponseWriter, req *http.Request) {
	if err := h.loadTemplates(); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// THINK: Try to let service take care of authorization check. Let's see if it's a good idea...
	//        Nope, seems like bad idea, at least with the current err = t.ExecuteTemplate() error handling,
	//        maybe need to fix that up.
	if user, err := h.us.GetAuthenticated(req.Context()); err != nil {
		log.Println("us.GetAuthenticated:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if user.ID == 0 {
		http.Error(w, "this page requires an authenticated user", http.StatusUnauthorized)
		return
	}

	state, err := h.state(req)
	if err != nil {
		log.Println("state:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = t.ExecuteTemplate(w, "notifications.html.tmpl", &state)
	if err != nil {
		log.Println("t.ExecuteTemplate:", err)
		template.HTMLEscape(w, []byte(err.Error()))
		return
	}
}

func (h *handler) postMarkReadHandler(w http.ResponseWriter, req *http.Request) {
	var mr common.MarkReadRequest
	err := json.NewDecoder(req.Body).Decode(&mr)
	if err != nil {
		log.Println("json.Decode:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.ns.MarkRead(req.Context(), mr.AppID, notifications.RepoSpec{URI: mr.RepoURI}, mr.ThreadID)
	if err != nil {
		log.Println("ns.MarkRead:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *handler) postMarkAllReadHandler(w http.ResponseWriter, req *http.Request) {
	var mar common.MarkAllReadRequest
	err := json.NewDecoder(req.Body).Decode(&mar)
	if err != nil {
		log.Println("json.Decode:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.ns.MarkAllRead(req.Context(), notifications.RepoSpec{URI: mar.RepoURI})
	if err != nil {
		log.Println("ns.MarkAllRead:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
