// +build ignore

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/shurcooL/notifications/fs"
	"github.com/shurcooL/notificationsapp"
	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

var (
	httpFlag = flag.String("http", ":8080", "Listen for HTTP connections on this address.")
)

func main() {
	err := run()
	if err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	flag.Parse()

	users := mockUsers{}
	service := fs.NewService(webdav.Dir("/Users/Dmitri/Dropbox/Work/2013/GoLand/src/github.com/shurcooL/home/http/testdata/notifications"), users)

	opt := notificationsapp.Options{
		// TODO: Update and unify octicons.css.
		//       But be mindful of https://github.com/shurcooL/notifications/blob/c38c34c46358723f7f329fa80f9a4ae105b60985/notifications.go#L39.
		HeadPre: `<title>Notifications</title>
<link href="//cdnjs.cloudflare.com/ajax/libs/octicons/3.1.0/octicons.css" media="all" rel="stylesheet" type="text/css" />
<style type="text/css">
	body {
		margin: 20px;
	}
	body, table {
		font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
		font-size: 14px;
		line-height: initial;
		color: #373a3c;
	}
</style>`,
	}
	opt.BodyPre = `<div style="max-width: 800px; margin: 0 auto 100px auto;">`
	notificationsApp := notificationsapp.New(service, users, opt)

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		req = req.WithContext(context.WithValue(req.Context(), notificationsapp.BaseURIContextKey, ".")) // TODO: Confirm "." vs "/" vs "".
		notificationsApp.ServeHTTP(w, req)
	})

	log.Println("Started.")

	err := http.ListenAndServe(*httpFlag, nil)
	return err
}

type mockUsers struct {
	users.Service
}

func (mockUsers) Get(_ context.Context, user users.UserSpec) (users.User, error) {
	switch {
	case user == users.UserSpec{ID: 1, Domain: "example.org"}:
		return users.User{
			UserSpec: user,
			Login:    "gopher",
			Name:     "Sample Gopher",
			Email:    "gopher@example.org",
		}, nil
	default:
		return users.User{}, fmt.Errorf("user %v not found", user)
	}
}

func (mockUsers) GetAuthenticatedSpec(_ context.Context) (users.UserSpec, error) {
	return users.UserSpec{ID: 1, Domain: "example.org"}, nil
}

func (m mockUsers) GetAuthenticated(ctx context.Context) (users.User, error) {
	userSpec, err := m.GetAuthenticatedSpec(ctx)
	if err != nil {
		return users.User{}, err
	}
	if userSpec.ID == 0 {
		return users.User{}, nil
	}
	return m.Get(ctx, userSpec)
}
