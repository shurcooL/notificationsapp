// +build js

// script is the frontend script for notificationsapp.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gopherjs/gopherjs/js"
	"github.com/shurcooL/go/gopherjs_http/jsutil"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/notificationsapp/httpclient"
	"golang.org/x/oauth2"
	"honnef.co/go/js/dom"
)

var document = dom.GetWindow().Document().(dom.HTMLDocument)

func main() {
	httpClient := httpClient()

	notificationsService = httpclient.NewNotifications(httpClient, "", "")

	js.Global.Set("MarkRead", jsutil.Wrap(MarkRead))
	js.Global.Set("MarkAllRead", jsutil.Wrap(MarkAllRead))
}

var notificationsService notifications.Service

func MarkRead(el dom.HTMLElement, appID string, repoURI string, threadID uint64) {
	if appID == "" && repoURI == "" && threadID == 0 {
		// When user clicks on the notification link, don't perform mark read operation
		// ourselves, it's expected to be done externally by the service that displays
		// the notification to the user views. Just make it appear as read, and return.
		markRead(el)
		return
	}

	go func() {
		err := notificationsService.MarkRead(context.Background(), appID, notifications.RepoSpec{URI: repoURI}, threadID)
		if err != nil {
			log.Println("MarkRead:", err)
			return
		}
		markRead(el)
	}()
}

func MarkAllRead(el dom.HTMLElement, repoURI string) {
	go func() {
		err := notificationsService.MarkAllRead(context.Background(), notifications.RepoSpec{URI: repoURI})
		if err != nil {
			log.Println("MarkAllRead:", err)
			return
		}
		markRead(el)
	}()
}

// markRead marks the notification containing element el as read.
func markRead(el dom.HTMLElement) {
	// TODO: Re-render entire component.RepoNotifications so that mark-all-read button
	//       disappears when last notification for repo is marked as read.
	//       Get rid of "mark-as-read", replace with something for getting root of component.RepoNotifications
	//       and find a good way to provide it with notifications data (needed to render component.RepoNotifications).
	getAncestorByClassName(el, "mark-as-read").(dom.HTMLElement).Class().Add("read")
}

func getAncestorByClassName(el dom.Element, class string) dom.Element {
	for ; el != nil && !el.Class().Contains(class); el = el.ParentElement() {
	}
	return el
}

// httpClient gives an *http.Client for making API requests.
func httpClient() *http.Client {
	document := &http.Request{Header: http.Header{"Cookie": {document.Cookie()}}}
	if accessToken, err := document.Cookie("accessToken"); err == nil {
		// Authenticated client.
		src := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: accessToken.Value},
		)
		return oauth2.NewClient(context.Background(), src)
	}
	// Not authenticated client.
	return http.DefaultClient
}
