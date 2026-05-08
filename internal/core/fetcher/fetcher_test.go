package fetcher

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/sailist/BBDown-go/internal/config"
	"github.com/sailist/BBDown-go/pkg/httpclient"
)

type mockClient struct{}

func (m *mockClient) Get(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, nil
}

func (m *mockClient) Post(ctx context.Context, url string, body []byte, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, nil
}

func (m *mockClient) Head(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
	return nil, nil
}

func (m *mockClient) GetRedirectLocation(ctx context.Context, url string) (string, error) {
	return "", nil
}

func (m *mockClient) GetContentLength(ctx context.Context, url string) (int64, error) {
	return 0, nil
}

func newTestFactory() *Factory {
	client := &mockClient{}
	cfg := config.NewConfig()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return NewFactory(client, cfg, logger)
}

func TestFactory_ReturnsFetcherForImplementedPrefixes(t *testing.T) {
	factory := newTestFactory()
	cases := []struct {
		id         string
		useIntl    bool
		expectType string
	}{
		{"mid:123", false, "*fetcher.SpaceVideoFetcher"},
		{"listBizId:123", false, "*fetcher.MediaListFetcher"},
		{"seriesBizId:123", false, "*fetcher.SeriesListFetcher"},
		{"favId:123:456", false, "*fetcher.FavListFetcher"},
	}

	for _, tc := range cases {
		f, err := factory.Create(tc.id, tc.useIntl)
		if err != nil {
			t.Errorf("Create(%q, %v) expected no error, got %v", tc.id, tc.useIntl, err)
			continue
		}
		if f == nil {
			t.Errorf("Create(%q, %v) returned nil fetcher", tc.id, tc.useIntl)
		}
	}
}

func TestFactory_ReturnsErrorForEmptyOrInvalid(t *testing.T) {
	factory := newTestFactory()
	cases := []struct {
		id      string
		useIntl bool
	}{
		{"", false},
		{"unknown", false},
		{"random_prefix", false},
		{"mid123", false},
		{"listBizId123", false},
		{"seriesBizId123", false},
		{"favId123", false},
		{"BV123", false},
	}

	for _, tc := range cases {
		f, err := factory.Create(tc.id, tc.useIntl)
		if err == nil {
			t.Errorf("Create(%q, %v) expected error, got fetcher %v", tc.id, tc.useIntl, f)
		}
	}
}

func TestFactory_ReturnsNormalFetcherForNumericID(t *testing.T) {
	factory := newTestFactory()
	cases := []string{"123", "0", "999999999"}

	for _, id := range cases {
		f, err := factory.Create(id, false)
		if err != nil {
			t.Errorf("Create(%q, false) expected no error, got %v", id, err)
			continue
		}
		if f == nil {
			t.Errorf("Create(%q, false) returned nil fetcher", id)
			continue
		}
		_, ok := f.(*NormalFetcher)
		if !ok {
			t.Errorf("Create(%q, false) expected *NormalFetcher, got %T", id, f)
		}
	}
}

func TestFactory_ReturnsBangumiFetcherForEpPrefix(t *testing.T) {
	factory := newTestFactory()
	f, err := factory.Create("ep:123", false)
	if err != nil {
		t.Fatalf("Create(%q, false) expected no error, got %v", "ep:123", err)
	}
	if f == nil {
		t.Fatal("expected non-nil fetcher")
	}
	_, ok := f.(*BangumiFetcher)
	if !ok {
		t.Errorf("expected *BangumiFetcher, got %T", f)
	}
}

func TestFactory_ReturnsIntlBangumiFetcherForEpWithIntl(t *testing.T) {
	factory := newTestFactory()
	f, err := factory.Create("ep:123", true)
	if err != nil {
		t.Fatalf("Create(%q, true) expected no error, got %v", "ep:123", err)
	}
	if f == nil {
		t.Fatal("expected non-nil fetcher")
	}
	_, ok := f.(*IntlBangumiFetcher)
	if !ok {
		t.Errorf("expected *IntlBangumiFetcher, got %T", f)
	}
}

func TestFactory_ReturnsCheeseFetcherForCheesePrefix(t *testing.T) {
	factory := newTestFactory()
	f, err := factory.Create("cheese:123", false)
	if err != nil {
		t.Fatalf("Create(%q, false) expected no error, got %v", "cheese:123", err)
	}
	if f == nil {
		t.Fatal("expected non-nil fetcher")
	}
	_, ok := f.(*CheeseFetcher)
	if !ok {
		t.Errorf("expected *CheeseFetcher, got %T", f)
	}
}
