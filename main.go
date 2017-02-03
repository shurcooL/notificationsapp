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
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/httperror"
	"github.com/shurcooL/httpfs/html/vfstemplate"
	"github.com/shurcooL/httpgzip"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/notificationsapp/assets"
	"github.com/shurcooL/notificationsapp/component"
	"github.com/shurcooL/octiconssvg"
	"github.com/shurcooL/users"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
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

	err := handler.loadTemplates()
	if err != nil {
		log.Fatalln("loadTemplates:", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.NotificationsHandler)
	assetsFileServer := httpgzip.FileServer(assets.Assets, httpgzip.FileServerOptions{ServeError: httpgzip.Detailed})
	mux.Handle("/assets/", assetsFileServer)

	handler.Handler = mux
	return handler
}

var t *template.Template

func (h *handler) loadTemplates() error {
	var err error
	t = template.New("").Funcs(template.FuncMap{
		"json": func(v interface{}) (string, error) {
			b, err := json.Marshal(v)
			return string(b), err
		},
		"reltime": humanize.Time,
		"base":    path.Base,

		"render": func(c htmlg.Component) template.HTML { return htmlg.Render(c.Render()...) },
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
	b := state{
		BaseURI: baseURI,
		req:     req,
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

	b.ns = h.ns

	return b, nil
}

type state struct {
	BaseURI string

	req *http.Request

	HeadPre template.HTML
	BodyPre template.HTML
	BodyTop template.HTML

	ns notifications.Service
}

// notification for display purposes.
type notification struct {
	notifications.Notification
}

func (n notification) Render() []*html.Node {
	// TODO: Make this much nicer.
	/*
		<div class="list-entry-body multilist-entry mark-as-read">
			<span class="content">
				<table style="width: 100%;">
				<tr>
				<td class="notification" style="width: 70%;">
					<span class="fade-when-read" style="color: {{.Color.HexString}}; margin-right: 6px; vertical-align: top;"><octiconssvg.Icon(.Icon)></span>
					<a class="black gray-when-read" onclick="MarkRead(this, {{`` | json}}, {{`` | json}}, 0);" href="{{.HTMLURL}}">{{.Title}}</a>
				</td>
				<td>
					{{if .Actor.AvatarURL}}<img class="avatar fade-when-read" title="@{{.Actor.Login}}" src="{{.Actor.AvatarURL}}">{{end -}}
					<span class="tiny gray-when-read">Time{.UpdatedAt}</span>
				</td>
				</tr>
				</table>
			</span>
			<span class="right-icon hide-when-read"><a href="javascript:" onclick="MarkRead(this, {{.AppID | json}}, {{.RepoSpec.URI | json}}, {{.ThreadID}});" title="Mark as read"><octiconssvg.Check()>"</a></span>
		</div>
	*/
	icon := htmlg.SpanClass("fade-when-read", octiconssvg.Icon(string(n.Icon)))
	icon.Attr = append(icon.Attr, html.Attribute{
		Key: atom.Style.String(), Val: fmt.Sprintf("color: %s; margin-right: 6px; vertical-align: top;", n.Color.HexString()),
	})
	td1 := htmlg.TD(
		icon,
		&html.Node{
			Type: html.ElementNode, Data: atom.A.String(),
			Attr: []html.Attribute{
				{Key: atom.Class.String(), Val: "black gray-when-read"},
				{Key: atom.Onclick.String(), Val: `MarkRead(this, "", "", 0);`},
				{Key: atom.Href.String(), Val: string(n.HTMLURL)},
			},
			FirstChild: htmlg.Text(n.Title),
		},
	)
	td1.Attr = append(td1.Attr, html.Attribute{Key: atom.Style.String(), Val: "width: 70%;"})
	td2 := htmlg.TD(
		htmlg.SpanClass("tiny gray-when-read", component.Time{n.UpdatedAt}.Render()...),
	)
	tr := htmlg.TR(td1, td2)
	table := &html.Node{
		Type: html.ElementNode, Data: atom.Table.String(),
		Attr: []html.Attribute{
			{Key: atom.Style.String(), Val: "width: 100%;"},
		},
		FirstChild: tr,
	}
	span1 := htmlg.SpanClass("content", table)
	span2 := htmlg.SpanClass("right-icon hide-when-read",
		&html.Node{
			Type: html.ElementNode, Data: atom.A.String(),
			Attr: []html.Attribute{
				{Key: atom.Href.String(), Val: "javascript:"},
				{Key: atom.Onclick.String(), Val: fmt.Sprintf("MarkRead(this, %q, %q, %v);", n.AppID, n.RepoSpec.URI, n.ThreadID)},
				{Key: atom.Title.String(), Val: "Mark as read"},
			},
			FirstChild: octiconssvg.Check(),
		},
	)
	div := htmlg.DivClass("list-entry-body multilist-entry mark-as-read", span1, span2)
	return []*html.Node{div}
}

// notificationsByUpdatedAt implements sort.Interface.
type notificationsByUpdatedAt []notification

func (s notificationsByUpdatedAt) Len() int           { return len(s) }
func (s notificationsByUpdatedAt) Less(i, j int) bool { return !s[i].UpdatedAt.Before(s[j].UpdatedAt) }
func (s notificationsByUpdatedAt) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type repoNotifications struct {
	Repo          notifications.RepoSpec
	RepoURL       template.URL
	Notifications notificationsByUpdatedAt

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
				Notifications: notificationsByUpdatedAt{notification{n}},
				updatedAt:     n.UpdatedAt,
			}
			rnm[r] = &rn
		default:
			if rnp.updatedAt.Before(n.UpdatedAt) {
				rnp.updatedAt = n.UpdatedAt
			}
			rnp.Notifications = append(rnp.Notifications, notification{n})
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

func (h *handler) NotificationsHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		httperror.HandleMethod(w, httperror.Method{Allowed: []string{"GET"}})
		return
	}

	// TODO: Get rid of this in production mode.
	if err := h.loadTemplates(); err != nil {
		log.Println("loadTemplates:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
