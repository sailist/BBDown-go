package parser

import (
	"strings"
	"testing"

	"github.com/nilaonai/bbdown-go/internal/config"
)

func TestBuildWebPlayurlAPI(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Token = "testtoken"
	cfg.Area = "hk"
	cfg.Cookie = ""

	// Non-bangumi
	url := BuildWebPlayurlAPI("123", "456", "789", "80", cfg, "testwbi", false)
	if !strings.HasPrefix(url, "https://api.bilibili.com/x/player/wbi/playurl?") {
		t.Errorf("unexpected prefix: %s", url)
	}
	if !strings.Contains(url, "avid=123") {
		t.Errorf("missing avid: %s", url)
	}
	if !strings.Contains(url, "cid=456") {
		t.Errorf("missing cid: %s", url)
	}
	if !strings.Contains(url, "qn=80") {
		t.Errorf("missing qn: %s", url)
	}
	if !strings.Contains(url, "support_multi_audio=true") {
		t.Errorf("missing support_multi_audio: %s", url)
	}
	if !strings.Contains(url, "from_client=BROWSER") {
		t.Errorf("missing from_client: %s", url)
	}
	if !strings.Contains(url, "access_key=testtoken") {
		t.Errorf("missing access_key: %s", url)
	}
	if !strings.Contains(url, "area=hk") {
		t.Errorf("missing area: %s", url)
	}
	if !strings.Contains(url, "try_look=1") {
		t.Errorf("missing try_look: %s", url)
	}
	if !strings.Contains(url, "w_rid=") {
		t.Errorf("missing w_rid (WbiSign): %s", url)
	}
	// Verify w_rid is a 32-char hex string
	wridIdx := strings.Index(url, "w_rid=")
	if wridIdx >= 0 {
		wridVal := url[wridIdx+len("w_rid="):]
		if len(wridVal) != 32 {
			t.Errorf("w_rid length = %d, want 32", len(wridVal))
		}
		for _, c := range wridVal {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("w_rid contains non-hex char: %c", c)
				break
			}
		}
	}

	// Bangumi
	urlBangumi := BuildWebPlayurlAPI("123", "456", "789", "80", cfg, "testwbi", true)
	if !strings.HasPrefix(urlBangumi, "https://api.bilibili.com/pgc/player/web/v2/playurl?") {
		t.Errorf("unexpected bangumi prefix: %s", urlBangumi)
	}
	if !strings.Contains(urlBangumi, "module=bangumi") {
		t.Errorf("missing module=bangumi: %s", urlBangumi)
	}
	if !strings.Contains(urlBangumi, "ep_id=789") {
		t.Errorf("missing ep_id: %s", urlBangumi)
	}
	if !strings.Contains(urlBangumi, "session=") {
		t.Errorf("missing session: %s", urlBangumi)
	}
	// Bangumi should NOT have w_rid because WbiSign is not applied
	if strings.Contains(urlBangumi, "w_rid=") {
		t.Errorf("bangumi should not have w_rid: %s", urlBangumi)
	}

	// With cookie — no try_look
	cfg.Cookie = "somecookie"
	urlWithCookie := BuildWebPlayurlAPI("123", "456", "789", "80", cfg, "testwbi", false)
	if strings.Contains(urlWithCookie, "try_look=1") {
		t.Errorf("should not have try_look when cookie is set: %s", urlWithCookie)
	}
}

func TestBuildTVPlayurlAPI(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Token = "testtoken"

	// Non-bangumi
	url := BuildTVPlayurlAPI("123", "456", "789", "80", cfg, false)
	if !strings.HasPrefix(url, "https://api.snm0516.aisee.tv/x/tv/playurl?") {
		t.Errorf("unexpected prefix: %s", url)
	}
	if !strings.Contains(url, "appkey=4409e2ce8ffd12b8") {
		t.Errorf("missing appkey: %s", url)
	}
	if !strings.Contains(url, "build=106500") {
		t.Errorf("missing build: %s", url)
	}
	if !strings.Contains(url, "cid=456") {
		t.Errorf("missing cid: %s", url)
	}
	if !strings.Contains(url, "device=android") {
		t.Errorf("missing device: %s", url)
	}
	if !strings.Contains(url, "fnval=4048") {
		t.Errorf("missing fnval: %s", url)
	}
	if !strings.Contains(url, "mid=0") {
		t.Errorf("missing mid: %s", url)
	}
	if !strings.Contains(url, "mobi_app=android_tv_yst") {
		t.Errorf("missing mobi_app: %s", url)
	}
	if !strings.Contains(url, "object_id=123") {
		t.Errorf("missing object_id: %s", url)
	}
	if !strings.Contains(url, "platform=android") {
		t.Errorf("missing platform: %s", url)
	}
	if !strings.Contains(url, "playurl_type=1") {
		t.Errorf("missing playurl_type: %s", url)
	}
	if !strings.Contains(url, "qn=80") {
		t.Errorf("missing qn: %s", url)
	}
	if !strings.Contains(url, "access_key=testtoken") {
		t.Errorf("missing access_key: %s", url)
	}
	if !strings.Contains(url, "sign=") {
		t.Errorf("missing sign: %s", url)
	}
	// Verify sign is a 32-char hex string
	signIdx := strings.Index(url, "sign=")
	if signIdx >= 0 {
		signVal := url[signIdx+len("sign="):]
		if len(signVal) != 32 {
			t.Errorf("sign length = %d, want 32", len(signVal))
		}
		for _, c := range signVal {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("sign contains non-hex char: %c", c)
				break
			}
		}
	}

	// Bangumi
	urlBangumi := BuildTVPlayurlAPI("123", "456", "789", "80", cfg, true)
	if !strings.HasPrefix(urlBangumi, "https://api.snm0516.aisee.tv/pgc/player/api/playurltv?") {
		t.Errorf("unexpected bangumi prefix: %s", urlBangumi)
	}
	if !strings.Contains(urlBangumi, "ep_id=789") {
		t.Errorf("missing ep_id: %s", urlBangumi)
	}
	if !strings.Contains(urlBangumi, "expire=0") {
		t.Errorf("missing expire: %s", urlBangumi)
	}
}

func TestBuildIntlPlayurlAPI(t *testing.T) {
	// Default host (not biliplus)
	cfg := config.NewConfig()
	cfg.Host = "api.bilibili.com"

	url := BuildIntlPlayurlAPI("123", "456", "789", "80", "0", cfg)
	if !strings.HasPrefix(url, "https://api.biliintl.com/intl/gateway/v2/ogv/playurl?") {
		t.Errorf("unexpected prefix: %s", url)
	}
	if !strings.Contains(url, "aid=123") {
		t.Errorf("missing aid: %s", url)
	}
	if !strings.Contains(url, "cid=456") {
		t.Errorf("missing cid: %s", url)
	}
	if !strings.Contains(url, "ep_id=789") {
		t.Errorf("missing ep_id: %s", url)
	}
	if !strings.Contains(url, "platform=android") {
		t.Errorf("missing platform: %s", url)
	}
	if !strings.Contains(url, "prefer_code_type=0") {
		t.Errorf("missing prefer_code_type: %s", url)
	}
	if !strings.Contains(url, "qn=80") {
		t.Errorf("missing qn: %s", url)
	}
	if !strings.Contains(url, "s_locale=zh_SG") {
		t.Errorf("missing s_locale: %s", url)
	}
	// Should NOT have appkey or sign when not biliplus
	if strings.Contains(url, "appkey=") {
		t.Errorf("should not have appkey when not biliplus: %s", url)
	}
	if strings.Contains(url, "sign=") {
		t.Errorf("should not have sign when not biliplus: %s", url)
	}
	if strings.Contains(url, "ts=") {
		t.Errorf("should not have ts when not biliplus: %s", url)
	}

	// Biliplus host
	cfg.Host = "example.com"
	cfg.Token = "testtoken"
	cfg.Area = ""

	urlBiliPlus := BuildIntlPlayurlAPI("123", "456", "789", "80", "0", cfg)
	if !strings.HasPrefix(urlBiliPlus, "https://example.com/intl/gateway/v2/ogv/playurl?") {
		t.Errorf("unexpected biliplus prefix: %s", urlBiliPlus)
	}
	if !strings.Contains(urlBiliPlus, "appkey=7d089525d3611b1c") {
		t.Errorf("missing appkey: %s", urlBiliPlus)
	}
	if !strings.Contains(urlBiliPlus, "area=th") {
		t.Errorf("missing default area=th: %s", urlBiliPlus)
	}
	if !strings.Contains(urlBiliPlus, "sign=") {
		t.Errorf("missing sign: %s", urlBiliPlus)
	}
	if !strings.Contains(urlBiliPlus, "access_key=testtoken") {
		t.Errorf("missing access_key: %s", urlBiliPlus)
	}
	if !strings.Contains(urlBiliPlus, "ts=") {
		t.Errorf("missing ts: %s", urlBiliPlus)
	}
	// Verify sign is a 32-char hex string
	signIdx := strings.Index(urlBiliPlus, "sign=")
	if signIdx >= 0 {
		signVal := urlBiliPlus[signIdx+len("sign="):]
		if len(signVal) != 32 {
			t.Errorf("sign length = %d, want 32", len(signVal))
		}
		for _, c := range signVal {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("sign contains non-hex char: %c", c)
				break
			}
		}
	}

	// Biliplus with custom area
	cfg.Area = "sg"
	urlBiliPlusArea := BuildIntlPlayurlAPI("123", "456", "789", "80", "0", cfg)
	if !strings.Contains(urlBiliPlusArea, "area=sg") {
		t.Errorf("missing custom area: %s", urlBiliPlusArea)
	}
}

func TestBuildAppPlayurlAPI(t *testing.T) {
	cfg := config.NewConfig()
	url := BuildAppPlayurlAPI("123", "456", "789", "80", cfg, false)
	if url != "" {
		t.Errorf("expected empty string, got %s", url)
	}
}
