// +build notificationsappdev

package assets

import (
	"go/build"
	"log"
	"net/http"
	"path/filepath"

	"github.com/shurcooL/go/gopherjs_http"
	"github.com/shurcooL/httpfs/union"
	"github.com/shurcooL/httpfs/vfsutil"
)

// Assets contains assets for notificationsapp.
var Assets = union.New(map[string]http.FileSystem{
	"/script.js": gopherjs_http.Package("github.com/shurcooL/notificationsapp/frontend"),
	"/style.css": vfsutil.File(filepath.Join(importPathToDir("github.com/shurcooL/notificationsapp/_data"), "style.css")),
})

func importPathToDir(importPath string) string {
	p, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		log.Fatalln(err)
	}
	return p.Dir
}
