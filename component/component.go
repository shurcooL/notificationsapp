package component

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/shurcooL/htmlg"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/octiconssvg"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// NotificationsByRepo component displays notifications grouped by repos.
type NotificationsByRepo struct {
	Notifications notifications.Notifications
}

func (a NotificationsByRepo) Render() []*html.Node {
	// TODO: Make this much nicer.
	/*
		{{if .}}{{range .}}
			{{render .}}
		{{end}}{{else}}
			<div style="text-align: center; margin-top: 80px; margin-bottom: 80px;">No new notifications.</div>
		{{end}}
	*/
	if len(a.Notifications) == 0 {
		// TODO: Maybe use blankslate Primer CSS component?
		div := &html.Node{
			Type: html.ElementNode, Data: atom.Div.String(),
			Attr: []html.Attribute{
				{Key: atom.Style.String(), Val: "text-align: center; margin-top: 80px; margin-bottom: 80px;"},
			},
			FirstChild: htmlg.Text("No new notifications."),
		}
		return []*html.Node{div}
	}

	var ns []*html.Node
	for _, repoNotifications := range a.groupAndSort() {
		ns = append(ns, repoNotifications.Render()...)
	}
	return ns
}

func (a NotificationsByRepo) groupAndSort() []RepoNotifications {
	// Group by RepoSpec into RepoNotifications collections.
	rnm := make(map[notifications.RepoSpec]*RepoNotifications)
	for _, n := range a.Notifications {
		r := n.RepoSpec
		switch rnp := rnm[r]; rnp {
		case nil: // First notification for this RepoSpec.
			rn := RepoNotifications{
				Repo:          r,
				RepoURL:       string(n.RepoURL),
				Notifications: []Notification{{n}},
				updatedAt:     n.UpdatedAt,
			}
			rnm[r] = &rn
		default: // Add notification to existing RepoNotifications.
			if rnp.updatedAt.Before(n.UpdatedAt) {
				rnp.updatedAt = n.UpdatedAt
			}
			rnp.Notifications = append(rnp.Notifications, Notification{n})
		}
	}

	// Sort by UpdatedAt time.
	var rns []RepoNotifications
	for _, rnp := range rnm {
		sort.Sort(nByUpdatedAt(rnp.Notifications))
		rns = append(rns, *rnp)
	}
	sort.Sort(rnByUpdatedAt(rns))

	return rns
}

// rnByUpdatedAt implements sort.Interface.
type rnByUpdatedAt []RepoNotifications

func (s rnByUpdatedAt) Len() int           { return len(s) }
func (s rnByUpdatedAt) Less(i, j int) bool { return !s[i].updatedAt.Before(s[j].updatedAt) }
func (s rnByUpdatedAt) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// nByUpdatedAt implements sort.Interface.
type nByUpdatedAt []Notification

func (s nByUpdatedAt) Len() int           { return len(s) }
func (s nByUpdatedAt) Less(i, j int) bool { return !s[i].UpdatedAt.Before(s[j].UpdatedAt) }
func (s nByUpdatedAt) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// RepoNotifications component is a collection of notifications for the same repo.
type RepoNotifications struct {
	Repo          notifications.RepoSpec
	RepoURL       string
	Notifications []Notification

	updatedAt time.Time // Most recent notification.
}

func (r RepoNotifications) Render() []*html.Node {
	// TODO: Make this much nicer.
	/*
		<div class="list-entry list-entry-border mark-as-read">
			<div class="list-entry-header">
				<span class="content"><a class="black" href="{{.RepoURL}}"><strong>{{.Repo.URI}}</strong></a></span>
				<span class="right-icon hide-when-read"><a href="javascript:" onclick="MarkAllRead(this, {{.Repo.URI | json}});" title="Mark all {{base .Repo.URI}} notifications as read"><span class="octicon octicon-check"></span></a></span>
			</div>
			{{range .Notifications}}
				{{render .}}
			{{end}}
		</div>
	*/
	var ns []*html.Node
	ns = append(ns, htmlg.DivClass("list-entry-header",
		htmlg.SpanClass("content",
			&html.Node{
				Type: html.ElementNode, Data: atom.A.String(),
				Attr: []html.Attribute{
					{Key: atom.Class.String(), Val: "black"},
					{Key: atom.Href.String(), Val: r.RepoURL},
				},
				FirstChild: htmlg.Strong(r.Repo.URI),
			},
		),
		htmlg.SpanClass("right-icon hide-when-read",
			&html.Node{
				Type: html.ElementNode, Data: atom.A.String(),
				Attr: []html.Attribute{
					{Key: atom.Href.String(), Val: "javascript:"},
					{Key: atom.Onclick.String(), Val: fmt.Sprintf("MarkAllRead(this, %q);", strconv.Quote(r.Repo.URI))},
					{Key: atom.Title.String(), Val: fmt.Sprintf("Mark all %s notifications as read", path.Base(r.Repo.URI))},
				},
				FirstChild: octiconssvg.Check(),
			},
		),
	))
	for _, notification := range r.Notifications {
		ns = append(ns, notification.Render()...)
	}
	div := htmlg.DivClass("list-entry list-entry-border mark-as-read", ns...)
	return []*html.Node{div}
}

// Notification component for display purposes.
type Notification struct {
	notifications.Notification
}

func (n Notification) Render() []*html.Node {
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
				{Key: atom.Onclick.String(), Val: `MarkRead(this, '""', '""', 0);`},
				{Key: atom.Href.String(), Val: string(n.HTMLURL)},
			},
			FirstChild: htmlg.Text(n.Title),
		},
	)
	td1.Attr = append(td1.Attr, html.Attribute{Key: atom.Style.String(), Val: "width: 70%;"})
	td2 := htmlg.TD()
	if n.Actor.ID != 0 && n.Actor.AvatarURL != "" {
		td2.AppendChild(&html.Node{
			Type: html.ElementNode, Data: atom.Img.String(),
			Attr: []html.Attribute{
				{Key: atom.Class.String(), Val: "avatar fade-when-read"},
				{Key: atom.Title.String(), Val: "@" + n.Actor.Login},
				{Key: atom.Src.String(), Val: string(n.Actor.AvatarURL)},
			},
		})
	}
	td2.AppendChild(htmlg.SpanClass("tiny gray-when-read", Time{n.UpdatedAt}.Render()...))
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
				{Key: atom.Onclick.String(), Val: fmt.Sprintf("MarkRead(this, %q, %q, %v);", strconv.Quote(n.AppID), strconv.Quote(n.RepoSpec.URI), n.ThreadID)},
				{Key: atom.Title.String(), Val: "Mark as read"},
			},
			FirstChild: octiconssvg.Check(),
		},
	)
	div := htmlg.DivClass("list-entry-body multilist-entry mark-as-read", span1, span2)
	return []*html.Node{div}
}

// Time component that displays human friendly relative time (e.g., "2 hours ago", "yesterday"),
// but also contains a tooltip with the full absolute time (e.g., "Jan 2, 2006, 3:04 PM MST").
type Time struct {
	Time time.Time
}

func (t Time) Render() []*html.Node {
	// TODO: Make this much nicer.
	// <abbr title="{{.Format "Jan _2, 2006, 3:04 PM MST"}}">{{reltime .}}</abbr>
	abbr := &html.Node{
		Type: html.ElementNode, Data: atom.Abbr.String(),
		Attr:       []html.Attribute{{Key: atom.Title.String(), Val: t.Time.Format("Jan _2, 2006, 3:04 PM MST")}},
		FirstChild: htmlg.Text(humanize.Time(t.Time)),
	}
	return []*html.Node{abbr}
}
