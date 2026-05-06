package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

func TestParseInput_BV(t *testing.T) {
	client := &mockClient{
		getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
			return "", errors.New("redirect not found")
		},
	}

	result, err := ParseInput(context.Background(), client, "BV1xx411c7mD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "2" {
		t.Fatalf("expected '2', got %s", result)
	}
}

func TestParseInput_AV(t *testing.T) {
	client := &mockClient{
		getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
			return "", errors.New("redirect not found")
		},
	}

	result, err := ParseInput(context.Background(), client, "av12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "12345" {
		t.Fatalf("expected '12345', got %s", result)
	}
}

func TestParseInput_EP(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "ep123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ep:123" {
		t.Fatalf("expected 'ep:123', got %s", result)
	}
}

func TestParseInput_PassThrough(t *testing.T) {
	client := &mockClient{}
	cases := []string{
		"listBizId:123",
		"seriesBizId:123",
		"favId:123:456",
		"mid:123",
		"cheese:123",
		"ep:123",
	}

	for _, input := range cases {
		result, err := ParseInput(context.Background(), client, input)
		if err != nil {
			t.Fatalf("input %q: unexpected error: %v", input, err)
		}
		if result != input {
			t.Fatalf("input %q: expected %q, got %q", input, input, result)
		}
	}
}

func TestParseInput_B23TvRedirect(t *testing.T) {
	client := &mockClient{
		getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
			if strings.Contains(url, "b23.tv") {
				return "https://www.bilibili.com/video/BV1xx411c7mD", nil
			}
			return "", errors.New("redirect not found")
		},
	}

	result, err := ParseInput(context.Background(), client, "https://b23.tv/abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "2" {
		t.Fatalf("expected '2', got %s", result)
	}
}

func TestParseInput_B23TvInfiniteRedirect(t *testing.T) {
	client := &mockClient{
		getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
			return url, nil
		},
	}

	_, err := ParseInput(context.Background(), client, "https://b23.tv/abc123")
	if err == nil {
		t.Fatal("expected error for infinite redirect, got nil")
	}
	if !strings.Contains(err.Error(), "infinite redirect") {
		t.Fatalf("expected infinite redirect error, got: %v", err)
	}
}

func TestParseInput_URLVideoAV(t *testing.T) {
	client := &mockClient{
		getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
			return "", errors.New("redirect not found")
		},
	}

	result, err := ParseInput(context.Background(), client, "https://www.bilibili.com/video/av12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "12345" {
		t.Fatalf("expected '12345', got %s", result)
	}
}

func TestParseInput_URLVideoBV(t *testing.T) {
	client := &mockClient{
		getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
			return "", errors.New("redirect not found")
		},
	}

	result, err := ParseInput(context.Background(), client, "https://www.bilibili.com/video/BV1xx411c7mD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "2" {
		t.Fatalf("expected '2', got %s", result)
	}
}

func TestParseInput_URLCheeseEP(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://www.bilibili.com/cheese/ep123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "cheese:123" {
		t.Fatalf("expected 'cheese:123', got %s", result)
	}
}

func TestParseInput_URLCheeseSS(t *testing.T) {
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			body := `{"data":{"episodes":[{"id":456}]}}`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	result, err := ParseInput(context.Background(), client, "https://www.bilibili.com/cheese/ss789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "cheese:456" {
		t.Fatalf("expected 'cheese:456', got %s", result)
	}
}

func TestParseInput_URLBangumiEP(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://www.bilibili.com/bangumi/play/ep123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ep:123" {
		t.Fatalf("expected 'ep:123', got %s", result)
	}
}

func TestParseInput_URLBangumiSS(t *testing.T) {
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			body := `{"result":{"episodes":[{"id":456}]}}`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	result, err := ParseInput(context.Background(), client, "https://www.bilibili.com/bangumi/play/ss789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ep:456" {
		t.Fatalf("expected 'ep:456', got %s", result)
	}
}

func TestParseInput_URLSpace(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://space.bilibili.com/12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "mid:12345" {
		t.Fatalf("expected 'mid:12345', got %s", result)
	}
}

func TestParseInput_URLFavlist(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://space.bilibili.com/12345/favlist?fid=67890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "favId:67890:12345" {
		t.Fatalf("expected 'favId:67890:12345', got %s", result)
	}
}

func TestParseInput_URLMedialistSpaceCollection(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://www.bilibili.com/medialist/detail?business_id=123&business=space_collection")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "listBizId:123" {
		t.Fatalf("expected 'listBizId:123', got %s", result)
	}
}

func TestParseInput_URLMedialistSpaceSeries(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://www.bilibili.com/medialist/detail?business_id=123&business=space_series")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "seriesBizId:123" {
		t.Fatalf("expected 'seriesBizId:123', got %s", result)
	}
}

func TestParseInput_URLChannelCollection(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://space.bilibili.com/123/channel/collectiondetail?sid=456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "listBizId:456" {
		t.Fatalf("expected 'listBizId:456', got %s", result)
	}
}

func TestParseInput_URLChannelSeries(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://space.bilibili.com/123/channel/seriesdetail?sid=456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "seriesBizId:456" {
		t.Fatalf("expected 'seriesBizId:456', got %s", result)
	}
}

func TestParseInput_URLSpaceListsSeries(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://space.bilibili.com/123/lists/456?type=series")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "seriesBizId:456" {
		t.Fatalf("expected 'seriesBizId:456', got %s", result)
	}
}

func TestParseInput_URLSpaceListsPlaylist(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://space.bilibili.com/123/lists/456?type=playlist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "listBizId:456" {
		t.Fatalf("expected 'listBizId:456', got %s", result)
	}
}

func TestParseInput_URLEpIDQuery(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://www.bilibili.com/bangumi/play?ep_id=123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ep:123" {
		t.Fatalf("expected 'ep:123', got %s", result)
	}
}

func TestParseInput_URLGlobalEp(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://www.bilibili.tv/en/play/123/456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ep:456" {
		t.Fatalf("expected 'ep:456', got %s", result)
	}
}

func TestParseInput_URLBangumiMD(t *testing.T) {
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			body := `{"result":{"media":{"new_ep":{"id":789}}}}`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	result, err := ParseInput(context.Background(), client, "https://www.bilibili.com/bangumi/media/md123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ep:789" {
		t.Fatalf("expected 'ep:789', got %s", result)
	}
}

func TestParseInput_URLDefaultPageState(t *testing.T) {
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			body := `<html><script>window.__INITIAL_STATE__={"epList":[{"id":999}]};(function(){})</script></html>`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	result, err := ParseInput(context.Background(), client, "https://www.bilibili.com/some/random/page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ep:999" {
		t.Fatalf("expected 'ep:999', got %s", result)
	}
}

func TestParseInput_URLDefaultPageStateNoEp(t *testing.T) {
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			body := `<html><script>window.__INITIAL_STATE__={"epList":[]};(function(){})</script></html>`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	_, err := ParseInput(context.Background(), client, "https://www.bilibili.com/some/random/page")
	if err == nil {
		t.Fatal("expected error for no ep found, got nil")
	}
	if !strings.Contains(err.Error(), "no ep found") {
		t.Fatalf("expected 'no ep found' error, got: %v", err)
	}
}

func TestParseInput_URLDefaultUnrecognized(t *testing.T) {
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			body := `<html><head><title>Test</title></head></html>`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	_, err := ParseInput(context.Background(), client, "https://www.bilibili.com/some/random/page")
	if err == nil {
		t.Fatal("expected error for unrecognized URL, got nil")
	}
	if !strings.Contains(err.Error(), "unrecognized URL format") {
		t.Fatalf("expected 'unrecognized URL format' error, got: %v", err)
	}
}

func TestParseInput_CheesePathEP(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "cheese/ep123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "cheese:123" {
		t.Fatalf("expected 'cheese:123', got %s", result)
	}
}

func TestParseInput_CheesePathSS(t *testing.T) {
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			body := `{"data":{"episodes":[{"id":456}]}}`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	result, err := ParseInput(context.Background(), client, "cheese/ss789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "cheese:456" {
		t.Fatalf("expected 'cheese:456', got %s", result)
	}
}

func TestParseInput_SS(t *testing.T) {
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			body := `{"result":{"episodes":[{"id":456}]}}`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	result, err := ParseInput(context.Background(), client, "ss789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ep:456" {
		t.Fatalf("expected 'ep:456', got %s", result)
	}
}

func TestParseInput_MD(t *testing.T) {
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			body := `{"result":{"media":{"new_ep":{"id":789}}}}`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	result, err := ParseInput(context.Background(), client, "md123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ep:789" {
		t.Fatalf("expected 'ep:789', got %s", result)
	}
}

func TestParseInput_InvalidInput(t *testing.T) {
	client := &mockClient{}

	_, err := ParseInput(context.Background(), client, "notavalidinput")
	if err == nil {
		t.Fatal("expected error for invalid input, got nil")
	}
	if !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected 'invalid input' error, got: %v", err)
	}
}

func TestParseInput_FixAvidRedirectToEP(t *testing.T) {
	client := &mockClient{
		getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
			if strings.Contains(url, "video/av2") {
				return "https://www.bilibili.com/bangumi/play/ep999", nil
			}
			return "", errors.New("redirect not found")
		},
	}

	result, err := ParseInput(context.Background(), client, "BV1xx411c7mD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ep:999" {
		t.Fatalf("expected 'ep:999', got %s", result)
	}
}

func TestParseInput_FetchBodyError(t *testing.T) {
	expectedErr := errors.New("connection refused")
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			return nil, expectedErr
		},
	}

	_, err := ParseInput(context.Background(), client, "ss789")
	if err == nil {
		t.Fatal("expected error for fetch body failure, got nil")
	}
	if !strings.Contains(err.Error(), "http get failed") {
		t.Fatalf("expected 'http get failed' error, got: %v", err)
	}
}

func TestParseInput_CheeseSSNoEpisodes(t *testing.T) {
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			body := `{"data":{"episodes":[]}}`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	_, err := ParseInput(context.Background(), client, "cheese/ss789")
	if err == nil {
		t.Fatal("expected error for no cheese episodes, got nil")
	}
	if !strings.Contains(err.Error(), "no episodes found") {
		t.Fatalf("expected 'no episodes found' error, got: %v", err)
	}
}

func TestParseInput_BangumiSSNoEpisodes(t *testing.T) {
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			body := `{"result":{"episodes":[]}}`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	_, err := ParseInput(context.Background(), client, "ss789")
	if err == nil {
		t.Fatal("expected error for no bangumi episodes, got nil")
	}
	if !strings.Contains(err.Error(), "no episodes found") {
		t.Fatalf("expected 'no episodes found' error, got: %v", err)
	}
}

func TestParseInput_MDInvalidResponse(t *testing.T) {
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			body := `not json`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	_, err := ParseInput(context.Background(), client, "md123")
	if err == nil {
		t.Fatal("expected error for invalid md response, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse md review") {
		t.Fatalf("expected 'failed to parse md review' error, got: %v", err)
	}
}

func TestParseInput_B23TvRedirectError(t *testing.T) {
	expectedErr := errors.New("network error")
	client := &mockClient{
		getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
			return "", expectedErr
		},
	}

	_, err := ParseInput(context.Background(), client, "https://b23.tv/abc123")
	if err == nil {
		t.Fatal("expected error for b23.tv redirect failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to resolve b23.tv link") {
		t.Fatalf("expected 'failed to resolve b23.tv link' error, got: %v", err)
	}
}

func TestParseInput_FavlistNoUID(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://space.bilibili.com/favlist?fid=67890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "favId:67890:" {
		t.Fatalf("expected 'favId:67890:', got %s", result)
	}
}

func TestParseInput_LowercaseBV(t *testing.T) {
	client := &mockClient{
		getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
			return "", errors.New("redirect not found")
		},
	}

	result, err := ParseInput(context.Background(), client, "bv1xx411c7mD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "2" {
		t.Fatalf("expected '2', got %s", result)
	}
}

func TestParseInput_LowercaseAV(t *testing.T) {
	client := &mockClient{
		getRedirectLocationFunc: func(ctx context.Context, url string) (string, error) {
			return "", errors.New("redirect not found")
		},
	}

	result, err := ParseInput(context.Background(), client, "AV12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "12345" {
		t.Fatalf("expected '12345', got %s", result)
	}
}

func TestParseInput_URLFavlistNoFid(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://space.bilibili.com/12345/favlist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "favId::12345" {
		t.Fatalf("expected 'favId::12345', got %s", result)
	}
}

func TestParseInput_CheeseEmptyEP(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://www.bilibili.com/cheese/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "cheese:" {
		t.Fatalf("expected 'cheese:', got %s", result)
	}
}

func TestParseInput_URLDefaultFetchError(t *testing.T) {
	expectedErr := errors.New("connection refused")
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			return nil, expectedErr
		},
	}

	_, err := ParseInput(context.Background(), client, "https://www.bilibili.com/some/random/page")
	if err == nil {
		t.Fatal("expected error for fetch failure, got nil")
	}
	if !strings.Contains(err.Error(), "fetch web source") {
		t.Fatalf("expected 'fetch web source' error, got: %v", err)
	}
}

func TestParseInput_PageStateInvalidJSON(t *testing.T) {
	client := &mockClient{
		getFunc: func(ctx context.Context, url string, opts ...httpclient.RequestOption) (*http.Response, error) {
			body := `<html><script>window.__INITIAL_STATE__=not valid json;(function(){})</script></html>`
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(body)),
				StatusCode: 200,
			}, nil
		},
	}

	_, err := ParseInput(context.Background(), client, "https://www.bilibili.com/some/random/page")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse page state") {
		t.Fatalf("expected 'failed to parse page state' error, got: %v", err)
	}
}

func TestParseInput_BVDecodeError(t *testing.T) {
	client := &mockClient{}

	_, err := ParseInput(context.Background(), client, "BV1invalid")
	if err == nil {
		t.Fatal("expected error for invalid BV, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode BV") {
		t.Fatalf("expected 'failed to decode BV' error, got: %v", err)
	}
}

func TestParseInput_URLBVDecodeError(t *testing.T) {
	client := &mockClient{}

	_, err := ParseInput(context.Background(), client, "https://www.bilibili.com/video/BV1invalid")
	if err == nil {
		t.Fatal("expected error for invalid BV URL, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode BV") {
		t.Fatalf("expected 'failed to decode BV' error, got: %v", err)
	}
}

func TestParseInput_URLSpaceListsNoType(t *testing.T) {
	client := &mockClient{}

	result, err := ParseInput(context.Background(), client, "https://space.bilibili.com/123/lists/456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "listBizId:456" {
		t.Fatalf("expected 'listBizId:456', got %s", result)
	}
}
