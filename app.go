// +build ignore

// An app that serves mock notifications for development and testing.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/shurcooL/home/httputil"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/notificationsapp"
	"github.com/shurcooL/notificationsapp/httphandler"
	"github.com/shurcooL/notificationsapp/httproute"
	"github.com/shurcooL/users"
)

var (
	httpFlag = flag.String("http", ":8080", "Listen for HTTP connections on this address.")
)

func main() {
	flag.Parse()

	err := run()
	if err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	users := mockUsers{}
	service := mockNotifications{}

	opt := notificationsapp.Options{
		HeadPre: `<title>Notifications</title>
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
	notificationsApp := notificationsapp.New(service, opt)

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		req = req.WithContext(context.WithValue(req.Context(), notificationsapp.BaseURIContextKey, ".")) // TODO: Confirm "." vs "/" vs "".
		notificationsApp.ServeHTTP(w, req)
	})

	// Register HTTP API endpoints.
	apiHandler := httphandler.Notifications{Notifications: service}
	http.Handle(httproute.List, httputil.ErrorHandler(users, apiHandler.List))
	http.Handle(httproute.Count, httputil.ErrorHandler(users, apiHandler.Count))
	http.Handle(httproute.MarkRead, httputil.ErrorHandler(users, apiHandler.MarkRead))
	http.Handle(httproute.MarkAllRead, httputil.ErrorHandler(users, apiHandler.MarkAllRead))

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

type mockNotifications struct {
	notifications.ExternalService
}

func (mockNotifications) List(ctx context.Context, opt notifications.ListOptions) (notifications.Notifications, error) {
	return ns, nil
}

func (mockNotifications) Count(ctx context.Context, opt interface{}) (uint64, error) {
	return uint64(len(ns)), nil
}

func (mockNotifications) MarkRead(ctx context.Context, appID string, repo notifications.RepoSpec, threadID uint64) error {
	// TODO: Perhaps have it modify what List returns, etc.
	fmt.Println("MarkRead:", repo.URI, appID, threadID)
	return nil
}

func (mockNotifications) MarkAllRead(ctx context.Context, repo notifications.RepoSpec) error {
	// TODO: Perhaps have it modify what List returns, etc.
	fmt.Println("MarkAllRead:", repo.URI)
	return nil
}

// ns is a list of mock notifications.
var ns = func() notifications.Notifications {
	passed := time.Since(time.Date(1, 1, 1, 0, 0, 63621777703, 945428426, time.UTC))
	return (notifications.Notifications)(notifications.Notifications{
		(notifications.Notification)(notifications.Notification{
			AppID: (string)("PullRequest"),
			RepoSpec: (notifications.RepoSpec)(notifications.RepoSpec{
				URI: (string)("github.com/bradleyfalzon/gopherci"),
			}),
			ThreadID: (uint64)(60),
			RepoURL:  (string)("https://github.com/bradleyfalzon/gopherci"),
			Title:    (string)("Support GitHub PushEvent"),
			Icon:     (notifications.OcticonID)("git-pull-request"),
			Color: (notifications.RGB)(notifications.RGB{
				R: (uint8)(108),
				G: (uint8)(198),
				B: (uint8)(68),
			}),
			Actor: (users.User)(users.User{
				UserSpec: (users.UserSpec)(users.UserSpec{
					ID:     (uint64)(2354108),
					Domain: (string)("github.com"),
				}),
				Elsewhere: ([]users.UserSpec)(nil),
				Login:     (string)("coveralls"),
				Name:      (string)(""),
				Email:     (string)(""),
				AvatarURL: (string)("https://avatars.githubusercontent.com/u/2354108?s=36&v=3"),
				HTMLURL:   (string)("https://github.com/coveralls"),
				CreatedAt: (time.Time)(time.Time{}),
				UpdatedAt: (time.Time)(time.Time{}),
				SiteAdmin: (bool)(false),
			}),
			UpdatedAt: (time.Time)(time.Date(1, 1, 1, 0, 0, 63621776801, 0, time.UTC).Add(passed)),
			HTMLURL:   (string)("https://github.com/bradleyfalzon/gopherci/pull/60#comment-277416148"),
		}),
		(notifications.Notification)(notifications.Notification{
			AppID: (string)("PullRequest"),
			RepoSpec: (notifications.RepoSpec)(notifications.RepoSpec{
				URI: (string)("github.com/ryanuber/go-glob"),
			}),
			ThreadID: (uint64)(5),
			RepoURL:  (string)("https://github.com/ryanuber/go-glob"),
			Title:    (string)("Add GlobI for case-insensitive globbing"),
			Icon:     (notifications.OcticonID)("git-pull-request"),
			Color: (notifications.RGB)(notifications.RGB{
				R: (uint8)(108),
				G: (uint8)(198),
				B: (uint8)(68),
			}),
			Actor: (users.User)(users.User{
				UserSpec: (users.UserSpec)(users.UserSpec{
					ID:     (uint64)(3022496),
					Domain: (string)("github.com"),
				}),
				Elsewhere: ([]users.UserSpec)(nil),
				Login:     (string)("blockloop"),
				Name:      (string)(""),
				Email:     (string)(""),
				AvatarURL: (string)("https://avatars.githubusercontent.com/u/3022496?s=36&v=3"),
				HTMLURL:   (string)("https://github.com/blockloop"),
				CreatedAt: (time.Time)(time.Time{}),
				UpdatedAt: (time.Time)(time.Time{}),
				SiteAdmin: (bool)(false),
			}),
			UpdatedAt: (time.Time)(time.Date(1, 1, 1, 0, 0, 63621776446, 0, time.UTC).Add(passed)),
			HTMLURL:   (string)("https://github.com/ryanuber/go-glob/pull/5#comment-277415841"),
		}),
		(notifications.Notification)(notifications.Notification{
			AppID: (string)("Issue"),
			RepoSpec: (notifications.RepoSpec)(notifications.RepoSpec{
				URI: (string)("github.com/nsf/gocode"),
			}),
			ThreadID: (uint64)(419),
			RepoURL:  (string)("https://github.com/nsf/gocode"),
			Title:    (string)("panic: unknown export format version 4"),
			Icon:     (notifications.OcticonID)("issue-closed"),
			Color: (notifications.RGB)(notifications.RGB{
				R: (uint8)(189),
				G: (uint8)(44),
				B: (uint8)(0),
			}),
			Actor: (users.User)(users.User{
				UserSpec: (users.UserSpec)(users.UserSpec{
					ID:     (uint64)(45629),
					Domain: (string)("github.com"),
				}),
				Elsewhere: ([]users.UserSpec)(nil),
				Login:     (string)("davidlazar"),
				Name:      (string)(""),
				Email:     (string)(""),
				AvatarURL: (string)("https://avatars.githubusercontent.com/u/45629?s=36&v=3"),
				HTMLURL:   (string)("https://github.com/davidlazar"),
				CreatedAt: (time.Time)(time.Time{}),
				UpdatedAt: (time.Time)(time.Time{}),
				SiteAdmin: (bool)(false),
			}),
			UpdatedAt: (time.Time)(time.Date(1, 1, 1, 0, 0, 63621775009, 0, time.UTC).Add(passed)),
			HTMLURL:   (string)("https://github.com/nsf/gocode/issues/419#comment-277414645"),
		}),
		(notifications.Notification)(notifications.Notification{
			AppID: (string)("Issue"),
			RepoSpec: (notifications.RepoSpec)(notifications.RepoSpec{
				URI: (string)("github.com/robpike/ivy"),
			}),
			ThreadID: (uint64)(31),
			RepoURL:  (string)("https://github.com/robpike/ivy"),
			Title:    (string)("loop termination condition seems wrong"),
			Icon:     (notifications.OcticonID)("issue-opened"),
			Color: (notifications.RGB)(notifications.RGB{
				R: (uint8)(108),
				G: (uint8)(198),
				B: (uint8)(68),
			}),
			Actor: (users.User)(users.User{
				UserSpec: (users.UserSpec)(users.UserSpec{
					ID:     (uint64)(4324516),
					Domain: (string)("github.com"),
				}),
				Elsewhere: ([]users.UserSpec)(nil),
				Login:     (string)("robpike"),
				Name:      (string)(""),
				Email:     (string)(""),
				AvatarURL: (string)("https://avatars.githubusercontent.com/u/4324516?s=36&v=3"),
				HTMLURL:   (string)("https://github.com/robpike"),
				CreatedAt: (time.Time)(time.Time{}),
				UpdatedAt: (time.Time)(time.Time{}),
				SiteAdmin: (bool)(false),
			}),
			UpdatedAt: (time.Time)(time.Date(1, 1, 1, 0, 0, 63621763429, 0, time.UTC).Add(passed)),
			HTMLURL:   (string)("https://github.com/robpike/ivy/issues/31#comment-277396571"),
		}),
		(notifications.Notification)(notifications.Notification{
			AppID: (string)("PullRequest"),
			RepoSpec: (notifications.RepoSpec)(notifications.RepoSpec{
				URI: (string)("github.com/nsf/gocode"),
			}),
			ThreadID: (uint64)(417),
			RepoURL:  (string)("https://github.com/nsf/gocode"),
			Title:    (string)("[WIP] package_bin: support type alias"),
			Icon:     (notifications.OcticonID)("git-pull-request"),
			Color: (notifications.RGB)(notifications.RGB{
				R: (uint8)(108),
				G: (uint8)(198),
				B: (uint8)(68),
			}),
			Actor: (users.User)(users.User{
				UserSpec: (users.UserSpec)(users.UserSpec{
					ID:     (uint64)(12567),
					Domain: (string)("github.com"),
				}),
				Elsewhere: ([]users.UserSpec)(nil),
				Login:     (string)("nsf"),
				Name:      (string)(""),
				Email:     (string)(""),
				AvatarURL: (string)("https://avatars.githubusercontent.com/u/12567?s=36&v=3"),
				HTMLURL:   (string)("https://github.com/nsf"),
				CreatedAt: (time.Time)(time.Time{}),
				UpdatedAt: (time.Time)(time.Time{}),
				SiteAdmin: (bool)(false),
			}),
			UpdatedAt: (time.Time)(time.Date(1, 1, 1, 0, 0, 63621764131, 0, time.UTC).Add(passed)),
			HTMLURL:   (string)("https://github.com/nsf/gocode/pull/417#comment-277398182"),
		}),
		(notifications.Notification)(notifications.Notification{
			AppID: (string)("PullRequest"),
			RepoSpec: (notifications.RepoSpec)(notifications.RepoSpec{
				URI: (string)("github.com/google/go-github"),
			}),
			ThreadID: (uint64)(538),
			RepoURL:  (string)("https://github.com/google/go-github"),
			Title:    (string)("Added listing outside collaborators for an organization"),
			Icon:     (notifications.OcticonID)("git-pull-request"),
			Color: (notifications.RGB)(notifications.RGB{
				R: (uint8)(108),
				G: (uint8)(198),
				B: (uint8)(68),
			}),
			Actor: (users.User)(users.User{
				UserSpec: (users.UserSpec)(users.UserSpec{
					ID:     (uint64)(6598971),
					Domain: (string)("github.com"),
				}),
				Elsewhere: ([]users.UserSpec)(nil),
				Login:     (string)("gmlewis"),
				Name:      (string)(""),
				Email:     (string)(""),
				AvatarURL: (string)("https://avatars.githubusercontent.com/u/6598971?s=36&v=3"),
				HTMLURL:   (string)("https://github.com/gmlewis"),
				CreatedAt: (time.Time)(time.Time{}),
				UpdatedAt: (time.Time)(time.Time{}),
				SiteAdmin: (bool)(false),
			}),
			UpdatedAt: (time.Time)(time.Date(1, 1, 1, 0, 0, 63621757401, 0, time.UTC).Add(passed)),
			HTMLURL:   (string)("https://github.com/google/go-github/pull/538#comment-277378904"),
		}),
		(notifications.Notification)(notifications.Notification{
			AppID: (string)("Issue"),
			RepoSpec: (notifications.RepoSpec)(notifications.RepoSpec{
				URI: (string)("github.com/nsf/gocode"),
			}),
			ThreadID: (uint64)(396),
			RepoURL:  (string)("https://github.com/nsf/gocode"),
			Title:    (string)("PANIC!!! "),
			Icon:     (notifications.OcticonID)("issue-opened"),
			Color: (notifications.RGB)(notifications.RGB{
				R: (uint8)(108),
				G: (uint8)(198),
				B: (uint8)(68),
			}),
			Actor: (users.User)(users.User{
				UserSpec: (users.UserSpec)(users.UserSpec{
					ID:     (uint64)(8503),
					Domain: (string)("github.com"),
				}),
				Elsewhere: ([]users.UserSpec)(nil),
				Login:     (string)("samuel"),
				Name:      (string)(""),
				Email:     (string)(""),
				AvatarURL: (string)("https://avatars.githubusercontent.com/u/8503?s=36&v=3"),
				HTMLURL:   (string)("https://github.com/samuel"),
				CreatedAt: (time.Time)(time.Time{}),
				UpdatedAt: (time.Time)(time.Time{}),
				SiteAdmin: (bool)(false),
			}),
			UpdatedAt: (time.Time)(time.Date(1, 1, 1, 0, 0, 63621747822, 0, time.UTC).Add(passed)),
			HTMLURL:   (string)("https://github.com/nsf/gocode/issues/396#comment-277343192"),
		}),
		(notifications.Notification)(notifications.Notification{
			AppID: (string)("Issue"),
			RepoSpec: (notifications.RepoSpec)(notifications.RepoSpec{
				URI: (string)("github.com/primer/octicons"),
			}),
			ThreadID: (uint64)(154),
			RepoURL:  (string)("https://github.com/primer/octicons"),
			Title:    (string)("Please add more variants for refresh icon."),
			Icon:     (notifications.OcticonID)("issue-closed"),
			Color: (notifications.RGB)(notifications.RGB{
				R: (uint8)(189),
				G: (uint8)(44),
				B: (uint8)(0),
			}),
			Actor: (users.User)(users.User{
				UserSpec: (users.UserSpec)(users.UserSpec{
					ID:     (uint64)(11073943),
					Domain: (string)("github.com"),
				}),
				Elsewhere: ([]users.UserSpec)(nil),
				Login:     (string)("souravbadami"),
				Name:      (string)(""),
				Email:     (string)(""),
				AvatarURL: (string)("https://avatars.githubusercontent.com/u/11073943?s=36&v=3"),
				HTMLURL:   (string)("https://github.com/souravbadami"),
				CreatedAt: (time.Time)(time.Time{}),
				UpdatedAt: (time.Time)(time.Time{}),
				SiteAdmin: (bool)(false),
			}),
			UpdatedAt: (time.Time)(time.Date(1, 1, 1, 0, 0, 63621746110, 0, time.UTC).Add(passed)),
			HTMLURL:   (string)("https://github.com/primer/octicons/issues/154"),
		}),
		(notifications.Notification)(notifications.Notification{
			AppID: (string)("Issue"),
			RepoSpec: (notifications.RepoSpec)(notifications.RepoSpec{
				URI: (string)("github.com/primer/octicons"),
			}),
			ThreadID: (uint64)(78),
			RepoURL:  (string)("https://github.com/primer/octicons"),
			Title:    (string)("Add pause icon"),
			Icon:     (notifications.OcticonID)("issue-closed"),
			Color: (notifications.RGB)(notifications.RGB{
				R: (uint8)(189),
				G: (uint8)(44),
				B: (uint8)(0),
			}),
			Actor: (users.User)(users.User{
				UserSpec: (users.UserSpec)(users.UserSpec{
					ID:     (uint64)(6053067),
					Domain: (string)("github.com"),
				}),
				Elsewhere: ([]users.UserSpec)(nil),
				Login:     (string)("Odonno"),
				Name:      (string)(""),
				Email:     (string)(""),
				AvatarURL: (string)("https://avatars.githubusercontent.com/u/6053067?s=36&v=3"),
				HTMLURL:   (string)("https://github.com/Odonno"),
				CreatedAt: (time.Time)(time.Time{}),
				UpdatedAt: (time.Time)(time.Time{}),
				SiteAdmin: (bool)(false),
			}),
			UpdatedAt: (time.Time)(time.Date(1, 1, 1, 0, 0, 63621746061, 0, time.UTC).Add(passed)),
			HTMLURL:   (string)("https://github.com/primer/octicons/issues/78"),
		}),
		(notifications.Notification)(notifications.Notification{
			AppID: (string)("Issue"),
			RepoSpec: (notifications.RepoSpec)(notifications.RepoSpec{
				URI: (string)("github.com/neelance/graphql-go"),
			}),
			ThreadID: (uint64)(53),
			RepoURL:  (string)("https://github.com/neelance/graphql-go"),
			Title:    (string)("Opentracing not tracing graphql traces"),
			Icon:     (notifications.OcticonID)("issue-opened"),
			Color: (notifications.RGB)(notifications.RGB{
				R: (uint8)(108),
				G: (uint8)(198),
				B: (uint8)(68),
			}),
			Actor: (users.User)(users.User{
				UserSpec: (users.UserSpec)(users.UserSpec{
					ID:     (uint64)(1966521),
					Domain: (string)("github.com"),
				}),
				Elsewhere: ([]users.UserSpec)(nil),
				Login:     (string)("bsr203"),
				Name:      (string)(""),
				Email:     (string)(""),
				AvatarURL: (string)("https://avatars.githubusercontent.com/u/1966521?s=36&v=3"),
				HTMLURL:   (string)("https://github.com/bsr203"),
				CreatedAt: (time.Time)(time.Time{}),
				UpdatedAt: (time.Time)(time.Time{}),
				SiteAdmin: (bool)(false),
			}),
			UpdatedAt: (time.Time)(time.Date(1, 1, 1, 0, 0, 63621743050, 0, time.UTC).Add(passed)),
			HTMLURL:   (string)("https://github.com/neelance/graphql-go/issues/53#comment-277322972"),
		}),
	})
}()
