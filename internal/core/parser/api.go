package parser

import (
	"net/url"

	"github.com/sailist/BBDown-go/internal/config"
	"github.com/sailist/BBDown-go/internal/core/util"
)

// BuildWebPlayurlAPI builds the web playurl API URL.
// For non-bangumi content, WbiSign is applied to the query string.
func BuildWebPlayurlAPI(aid, cid, epid, qn string, cfg *config.Config, wbi string, bangumi bool) string {
	v := url.Values{}
	v.Set("support_multi_audio", "true")
	v.Set("from_client", "BROWSER")
	v.Set("avid", aid)
	v.Set("cid", cid)
	v.Set("fnval", "4048")
	v.Set("fnver", "0")
	v.Set("fourk", "1")
	if cfg.Area != "" {
		v.Set("access_key", cfg.Token)
		v.Set("area", cfg.Area)
	}
	v.Set("otype", "json")
	v.Set("qn", qn)
	if bangumi {
		v.Set("module", "bangumi")
		v.Set("ep_id", epid)
		v.Set("session", "")
	}
	if cfg.Cookie == "" {
		v.Set("try_look", "1")
	}
	v.Set("wts", util.GetTimestamp(true))

	query := v.Encode()

	var prefix string
	if bangumi {
		prefix = "https://" + cfg.Host + "/pgc/player/web/v2/playurl?"
	} else {
		prefix = "https://api.bilibili.com/x/player/wbi/playurl?"
	}

	if bangumi {
		return prefix + query
	}
	return prefix + util.WbiSign(query, wbi)
}

// BuildTVPlayurlAPI builds the TV playurl API URL with appkey and sign.
func BuildTVPlayurlAPI(aid, cid, epid, qn string, cfg *config.Config, bangumi bool) string {
	v := url.Values{}
	if cfg.Token != "" {
		v.Set("access_key", cfg.Token)
	}
	v.Set("appkey", "4409e2ce8ffd12b8")
	v.Set("build", "106500")
	v.Set("cid", cid)
	v.Set("device", "android")
	if bangumi {
		v.Set("ep_id", epid)
		v.Set("expire", "0")
	}
	v.Set("fnval", "4048")
	v.Set("fnver", "0")
	v.Set("fourk", "1")
	v.Set("mid", "0")
	v.Set("mobi_app", "android_tv_yst")
	v.Set("object_id", aid)
	v.Set("platform", "android")
	v.Set("playurl_type", "1")
	v.Set("qn", qn)
	v.Set("ts", util.GetTimestamp(true))

	query := v.Encode()
	sign := util.GetSign(query, false)

	var prefix string
	if bangumi {
		prefix = "https://" + cfg.TvHost + "/pgc/player/api/playurltv?"
	} else {
		prefix = "https://" + cfg.TvHost + "/x/tv/playurl?"
	}

	return prefix + query + "&sign=" + sign
}

// BuildIntlPlayurlAPI builds the international playurl API URL.
// When the host is not api.bilibili.com (biliplus mode), appkey and sign are included.
func BuildIntlPlayurlAPI(aid, cid, epid, qn, code string, cfg *config.Config) string {
	isBiliPlus := cfg.Host != "api.bilibili.com"

	host := "api.biliintl.com"
	if isBiliPlus {
		host = cfg.Host
	}
	prefix := "https://" + host + "/intl/gateway/v2/ogv/playurl?"

	v := url.Values{}
	if cfg.Token != "" {
		v.Set("access_key", cfg.Token)
	}
	v.Set("aid", aid)
	if isBiliPlus {
		v.Set("appkey", "7d089525d3611b1c")
		area := cfg.Area
		if area == "" {
			area = "th"
		}
		v.Set("area", area)
	}
	v.Set("cid", cid)
	v.Set("ep_id", epid)
	v.Set("platform", "android")
	v.Set("prefer_code_type", code)
	v.Set("qn", qn)
	if isBiliPlus {
		v.Set("ts", util.GetTimestamp(true))
	}
	v.Set("s_locale", "zh_SG")

	query := v.Encode()
	if isBiliPlus {
		sign := util.GetSign(query, true)
		return prefix + query + "&sign=" + sign
	}
	return prefix + query
}

// BuildAppPlayurlAPI builds the gRPC app playurl API URL.
// Not implemented yet — gRPC helper will be added in commit 19.
func BuildAppPlayurlAPI(aid, cid, epid, qn string, cfg *config.Config, bangumi bool) string {
	return ""
}
