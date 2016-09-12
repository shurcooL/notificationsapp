package notificationsapp

import (
	"bytes"
	"encoding/json"
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

type Options struct {
	BaseURI func(req *http.Request) string
	HeadPre template.HTML

	// BodyTop provides components to include on top of <body> of page rendered for req. It can be nil.
	BodyTop func(req *http.Request) ([]htmlg.ComponentContext, error)

	// TODO.
	BaseState func(req *http.Request) BaseState
}

type handler struct {
	http.Handler

	ns notifications.Service
	us users.Service

	Options
}

// New returns a notifications app http.Handler using given services and options.
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

type BaseState struct {
	req  *http.Request
	vars map[string]string

	HeadPre template.HTML
	BodyTop template.HTML

	ns notifications.Service

	common.State
}

func (h *handler) baseState(req *http.Request) (BaseState, error) {
	b := h.BaseState(req)
	b.req = req
	b.vars = mux.Vars(req)
	b.HeadPre = h.HeadPre
	if h.BodyTop != nil {
		c, err := h.BodyTop(req)
		if err != nil {
			return BaseState{}, err
		}
		var buf bytes.Buffer
		err = htmlg.RenderComponentsContext(req.Context(), &buf, c...)
		if err != nil {
			return BaseState{}, err
		}
		b.BodyTop = template.HTML(buf.String())
	}

	b.ns = h.ns

	return b, nil
}

type state struct {
	BaseState
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

	baseState, err := h.baseState(req)
	if err != nil {
		log.Println("baseState:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	state := state{
		BaseState: baseState,
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
