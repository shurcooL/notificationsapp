// +build js

package main

import (
	"encoding/json"
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

	stateJSON := js.Global.Get("State").String()
	err := json.Unmarshal([]byte(stateJSON), &state)
	if err != nil {
		panic(err)
	}

	document.AddEventListener("DOMContentLoaded", false, func(_ dom.Event) {
		setup()
	})
}

func setup() {
	// TODO.
}

func MarkRead(this dom.HTMLElement) {
	getAncestorByClassName(this, "mark-as-read").(dom.HTMLElement).Class().Add("read")
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
