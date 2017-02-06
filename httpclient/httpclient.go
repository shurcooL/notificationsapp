// Package httpclient contains notifications.Service implementation over HTTP.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/shurcooL/go/ctxhttp"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/notificationsapp/httproute"
	"github.com/shurcooL/users"
)

// Notifications implements notifications.Service remotely over HTTP.
type Notifications struct{}

func (Notifications) List(ctx context.Context, opt notifications.ListOptions) (notifications.Notifications, error) {
	v := url.Values{} // TODO: Automate this conversion process.
	if opt.Repo != nil {
		v.Set("RepoURI", opt.Repo.URI)
	}
	u := url.URL{
		Path:     httproute.List,
		RawQuery: v.Encode(),
	}
	resp, err := ctxhttp.Get(ctx, nil, u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var ns notifications.Notifications
	err = json.NewDecoder(resp.Body).Decode(&ns)
	return ns, err
}

func (Notifications) Count(ctx context.Context, opt interface{}) (uint64, error) {
	resp, err := ctxhttp.Get(ctx, nil, httproute.Count)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return 0, fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	var u uint64
	err = json.NewDecoder(resp.Body).Decode(&u)
	return u, err
}

func (Notifications) MarkRead(ctx context.Context, appID string, repo notifications.RepoSpec, threadID uint64) error {
	u := url.URL{
		Path: httproute.MarkRead,
		RawQuery: url.Values{
			"AppID":    {appID},
			"RepoURI":  {repo.URI},
			"ThreadID": {fmt.Sprint(threadID)},
		}.Encode(),
	}
	resp, err := ctxhttp.Post(ctx, nil, u.String(), "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	return nil
}

func (Notifications) MarkAllRead(ctx context.Context, repo notifications.RepoSpec) error {
	u := url.URL{
		Path: httproute.MarkAllRead,
		RawQuery: url.Values{
			"RepoURI": {repo.URI},
		}.Encode(),
	}
	resp, err := ctxhttp.Post(ctx, nil, u.String(), "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("did not get acceptable status code: %v body: %q", resp.Status, body)
	}
	return nil
}

func (Notifications) Subscribe(_ context.Context, appID string, repo notifications.RepoSpec, threadID uint64, subscribers []users.UserSpec) error {
	return fmt.Errorf("Subscribe: not implemented")
}

func (Notifications) Notify(_ context.Context, appID string, repo notifications.RepoSpec, threadID uint64, nr notifications.NotificationRequest) error {
	return fmt.Errorf("Notify: not implemented")
}
