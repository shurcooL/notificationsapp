package notificationsapp

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/go-goon"
	"github.com/shurcooL/go/gzip_file_server"
	"github.com/shurcooL/httpfs/html/vfstemplate"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/notificationsapp/common"
	"github.com/shurcooL/users"
	"golang.org/x/net/context"
)

type Options struct {
	Context  func(req *http.Request) context.Context
	RepoSpec func(req *http.Request) notifications.RepoSpec
	BaseURI  func(req *http.Request) string
	HeadPre  template.HTML

	// TODO.
	BaseState func(req *http.Request) BaseState
}

type handler struct {
	http.Handler

	Options
}

// TODO: Get rid of globals.
var (
	ns notifications.InternalService
	us users.Service
)

// New returns a notifications app http.Handler using given services and options.
func New(service notifications.InternalService, users users.Service, opt Options) http.Handler {
	err := loadTemplates()
	if err != nil {
		log.Fatalln("loadTemplates:", err)
	}

	// TODO: Move into handler?
	ns = service
	us = users

	h := http.NewServeMux()
	h.HandleFunc("/mock/", mockHandler)
	r := mux.NewRouter()
	// TODO: Make redirection work.
	//r.StrictSlash(true) // THINK: Can't use this due to redirect not taking baseURI into account.
	r.HandleFunc("/", notificationsHandler).Methods("GET")
	h.Handle("/", r)
	assetsFileServer := gzip_file_server.New(Assets)
	h.Handle("/assets/", assetsFileServer)

	globalHandler = &handler{
		Options: opt,
		Handler: h,
	}
	return globalHandler
}

// TODO: Refactor to avoid global.
var globalHandler *handler

var t *template.Template

func loadTemplates() error {
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
	})
	t, err = vfstemplate.ParseGlob(Assets, t, "/assets/*.tmpl")
	return err
}

type state struct {
	BaseState
}

type BaseState struct {
	ctx  context.Context
	req  *http.Request
	vars map[string]string

	HeadPre template.HTML

	common.State
}

func baseState(req *http.Request) (BaseState, error) {
	b := globalHandler.BaseState(req)
	b.ctx = globalHandler.Context(req)
	b.req = req
	b.vars = mux.Vars(req)
	b.HeadPre = globalHandler.HeadPre

	return b, nil
}

type repoNotifications struct {
	Repo          notifications.RepoSpec
	RepoURL       template.URL
	Notifications notifications.Notifications

	updatedAt time.Time // Most recent notification.
}

// byUpdatedAt implements sort.Interface.
type byUpdatedAt []repoNotifications

func (s byUpdatedAt) Len() int           { return len(s) }
func (s byUpdatedAt) Less(i, j int) bool { return !s[i].updatedAt.Before(s[j].updatedAt) }
func (s byUpdatedAt) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s state) RepoNotifications() ([]repoNotifications, error) {
	ns, err := ns.List(s.ctx, nil)
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

func notificationsHandler(w http.ResponseWriter, req *http.Request) {
	if err := loadTemplates(); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if u, err := us.GetAuthenticated(globalHandler.Context(req)); err != nil {
		log.Println("us.GetAuthenticated:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if u == nil {
		http.Error(w, "this page requires an authenticated user", http.StatusUnauthorized)
		return
	}

	baseState, err := baseState(req)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
