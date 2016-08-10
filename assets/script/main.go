// +build js

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gopherjs/gopherjs/js"
	"github.com/shurcooL/go/gopherjs_http/jsutil"
	"github.com/shurcooL/notificationsapp/common"
	"honnef.co/go/js/dom"
)

var document = dom.GetWindow().Document().(dom.HTMLDocument)

var state common.State

func main() {
	js.Global.Set("MarkRead", jsutil.Wrap(MarkRead))
	js.Global.Set("MarkAllRead", jsutil.Wrap(MarkAllRead))

	stateJSON := js.Global.Get("State").String()
	err := json.Unmarshal([]byte(stateJSON), &state)
	if err != nil {
		panic(err)
	}

	document.AddEventListener("DOMContentLoaded", false, func(_ dom.Event) {
		setup()
	})
}

func setup() {}

func MarkRead(this dom.HTMLElement, appID string, repoURI string, threadID uint64) {
	if appID == "" && repoURI == "" && threadID == 0 {
		// Don't perform mark read operation ourselves, it's expected to be done by
		// externally by another service when the user views the notification. Just
		// make it appear as read, and return.
		getAncestorByClassName(this, "mark-as-read").(dom.HTMLElement).Class().Add("read")
		return
	}

	go func() {
		resp, err := postJSON(state.BaseURI+"/mark-read", common.MarkReadRequest{
			AppID:    appID,
			RepoURI:  repoURI,
			ThreadID: threadID,
		})
		if err != nil {
			log.Println(err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return
		}

		switch resp.StatusCode {
		case http.StatusOK:
			getAncestorByClassName(this, "mark-as-read").(dom.HTMLElement).Class().Add("read")
		default:
			log.Printf("did not get acceptable status code: %v\n%q\n", resp.Status, string(body))
		}
	}()
}

func MarkAllRead(this dom.HTMLElement, repoURI string) {
	go func() {
		resp, err := postJSON(state.BaseURI+"/mark-all-read", common.MarkAllReadRequest{
			RepoURI: repoURI,
		})
		if err != nil {
			log.Println(err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return
		}

		switch resp.StatusCode {
		case http.StatusOK:
			getAncestorByClassName(this, "mark-as-read").(dom.HTMLElement).Class().Add("read")
		default:
			log.Printf("did not get acceptable status code: %v\n%q\n", resp.Status, string(body))
		}
	}()
}

func postJSON(url string, v interface{}) (*http.Response, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

func getAncestorByClassName(el dom.Element, class string) dom.Element {
	for ; el != nil && !el.Class().Contains(class); el = el.ParentElement() {
	}
	return el
}

// getAncestorByTagName expects tag to be lowercase, like "span".
func getAncestorByTagName(el dom.Element, tag string) dom.Element {
	for ; el != nil && strings.ToLower(el.TagName()) != tag; el = el.ParentElement() {
	}
	return el
}
