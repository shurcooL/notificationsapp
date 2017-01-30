// +build js

package main

import (
	"context"
	"log"

	"github.com/gopherjs/gopherjs/js"
	"github.com/shurcooL/go/gopherjs_http/jsutil"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/notificationsapp/httpclient"
	"honnef.co/go/js/dom"
)

func main() {
	js.Global.Set("MarkRead", jsutil.Wrap(MarkRead))
	js.Global.Set("MarkAllRead", jsutil.Wrap(MarkAllRead))
}

func MarkRead(el dom.HTMLElement, appID string, repoURI string, threadID uint64) {
	if appID == "" && repoURI == "" && threadID == 0 {
		// When user clicks on the notification link, don't perform mark read operation
		// ourselves, it's expected to be done externally by the service that displays
		// the notification to the user views. Just make it appear as read, and return.
		markRead(el)
		return
	}

	go func() {
		err := httpclient.Notifications{}.MarkRead(context.Background(), appID, notifications.RepoSpec{URI: repoURI}, threadID)
		if err != nil {
			log.Println("MarkRead:", err)
			return
		}
		markRead(el)
	}()
}

func MarkAllRead(el dom.HTMLElement, repoURI string) {
	go func() {
		err := httpclient.Notifications{}.MarkAllRead(context.Background(), notifications.RepoSpec{URI: repoURI})
		if err != nil {
			log.Println("MarkAllRead:", err)
			return
		}
		markRead(el)
	}()
}

// markRead marks the notification containing element el as read.
func markRead(el dom.HTMLElement) {
	getAncestorByClassName(el, "mark-as-read").(dom.HTMLElement).Class().Add("read")
}

func getAncestorByClassName(el dom.Element, class string) dom.Element {
	for ; el != nil && !el.Class().Contains(class); el = el.ParentElement() {
	}
	return el
}
