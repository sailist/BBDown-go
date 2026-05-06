package util

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/nilaonai/bbdown-go/internal/config"
	"github.com/nilaonai/bbdown-go/internal/core/entity"
	"github.com/nilaonai/bbdown-go/pkg/httpclient"
)

// GetSubtitles fetches subtitles from Bilibili APIs.
// It chooses the API endpoints based on isIntl and cfg.Cookie.
func GetSubtitles(ctx context.Context, client httpclient.Client, cfg *config.Config, aid, cid, epid string, index int, isIntl bool) ([]entity.Subtitle, error) {
	var subtitles []entity.Subtitle
	var err error

	if isIntl {
		subtitles, err = getIntlSubtitlesFromAPI1(ctx, client, cfg, aid, cid, epid, index)
		if err != nil || len(subtitles) == 0 {
			subtitles, err = getIntlSubtitlesFromAPI2(ctx, client, cfg, aid, cid, epid, index)
		}
	} else {
		if cfg.Cookie == "" {
			subtitles, err = getSubtitlesFromAPI3(ctx, client, cfg, aid, cid, epid, index)
		} else {
			subtitles, err = getSubtitlesFromAPI2(ctx, client, cfg, aid, cid, epid, index)
			if err != nil || len(subtitles) == 0 {
				subtitles, err = getSubtitlesFromAPI1(ctx, client, cfg, aid, cid, epid, index)
			}
			if err != nil || len(subtitles) == 0 {
				subtitles, err = getSubtitlesFromAPI3(ctx, client, cfg, aid, cid, epid, index)
			}
		}
	}

	if subtitles == nil {
		return []entity.Subtitle{}, nil
	}

	for i := range subtitles {
		if strings.HasPrefix(subtitles[i].Url, "//") {
			subtitles[i].Url = "https:" + subtitles[i].Url
		}
	}

	return subtitles, nil
}

func getSubtitlesFromAPI1(ctx context.Context, client httpclient.Client, cfg *config.Config, aid, cid, epid string, index int) ([]entity.Subtitle, error) {
	url := fmt.Sprintf("https://api.bilibili.com/x/web-interface/view?aid=%s&cid=%s", aid, cid)
	data, err := doGet(ctx, client, url, cfg)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data struct {
			Subtitle struct {
				List []struct {
					Lan         string `json:"lan"`
					SubtitleUrl string `json:"subtitle_url"`
				} `json:"list"`
			} `json:"subtitle"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	var subtitles []entity.Subtitle
	for _, item := range resp.Data.Subtitle.List {
		subtitles = append(subtitles, entity.Subtitle{
			Lan:  item.Lan,
			Url:  item.SubtitleUrl,
			Path: fmt.Sprintf("%s/%s.%s.%s.srt", aid, aid, cid, item.Lan),
		})
	}

	if hasEmptyURL(subtitles) {
		return nil, fmt.Errorf("bad url")
	}

	return subtitles, nil
}

func getSubtitlesFromAPI2(ctx context.Context, client httpclient.Client, cfg *config.Config, aid, cid, epid string, index int) ([]entity.Subtitle, error) {
	url := fmt.Sprintf("https://api.bilibili.com/x/player/wbi/v2?cid=%s&aid=%s", cid, aid)
	data, err := doGet(ctx, client, url, cfg)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data struct {
			Subtitle struct {
				Subtitles []struct {
					Lan         string `json:"lan"`
					SubtitleUrl string `json:"subtitle_url"`
				} `json:"subtitles"`
			} `json:"subtitle"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	var subtitles []entity.Subtitle
	for _, item := range resp.Data.Subtitle.Subtitles {
		subtitles = append(subtitles, entity.Subtitle{
			Lan:  item.Lan,
			Url:  item.SubtitleUrl,
			Path: fmt.Sprintf("%s/%s.%s.%s.srt", aid, aid, cid, item.Lan),
		})
	}

	if hasEmptyURL(subtitles) {
		return nil, fmt.Errorf("bad url")
	}

	return subtitles, nil
}

func getSubtitlesFromAPI3(ctx context.Context, client httpclient.Client, cfg *config.Config, aid, cid, epid string, index int) ([]entity.Subtitle, error) {
	// APP gRPC subtitle API - not implemented yet.
	return nil, fmt.Errorf("not implemented")
}

func getIntlSubtitlesFromAPI1(ctx context.Context, client httpclient.Client, cfg *config.Config, aid, cid, epid string, index int) ([]entity.Subtitle, error) {
	host := cfg.EpHost
	if host == "api.bilibili.com" {
		host = "api.biliintl.com"
	}
	url := fmt.Sprintf("https://%s/intl/gateway/web/v2/subtitle?episode_id=%s", host, epid)
	data, err := doGet(ctx, client, url, cfg)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data struct {
			Subtitles []struct {
				LangKey string `json:"lang_key"`
				Url     string `json:"url"`
			} `json:"subtitles"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	var subtitles []entity.Subtitle
	for _, item := range resp.Data.Subtitles {
		ext := ".srt"
		if strings.Contains(item.Url, ".json") {
			ext = ".srt"
		} else {
			ext = ".ass"
		}
		subtitles = append(subtitles, entity.Subtitle{
			Lan:  item.LangKey,
			Url:  item.Url,
			Path: fmt.Sprintf("%s/%s.%s.%s%s", aid, aid, cid, item.LangKey, ext),
		})
	}

	if hasEmptyURL(subtitles) {
		return nil, fmt.Errorf("bad url")
	}

	return subtitles, nil
}

func getIntlSubtitlesFromAPI2(ctx context.Context, client httpclient.Client, cfg *config.Config, aid, cid, epid string, index int) ([]entity.Subtitle, error) {
	host := cfg.Host
	if host == "api.bilibili.com" {
		host = "api.bilibili.tv"
	}
	url := fmt.Sprintf("https://%s/intl/gateway/v2/ogv/view/app/season?ep_id=%s&platform=android&s_locale=zh_SG", host, epid)
	if cfg.Token != "" {
		url += "&access_key=" + cfg.Token
	}
	data, err := doGet(ctx, client, url, cfg)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Result struct {
			Modules []struct {
				Data struct {
					Episodes []struct {
						Subtitles []struct {
							Key string `json:"key"`
							Url string `json:"url"`
						} `json:"subtitles"`
					} `json:"episodes"`
				} `json:"data"`
			} `json:"modules"`
		} `json:"result"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	if len(resp.Result.Modules) == 0 {
		return nil, fmt.Errorf("no modules")
	}
	eps := resp.Result.Modules[0].Data.Episodes
	if index < 1 || index > len(eps) {
		return nil, fmt.Errorf("index out of range")
	}

	var subtitles []entity.Subtitle
	for _, item := range eps[index-1].Subtitles {
		url := strings.ReplaceAll(item.Url, "\\/", "/")
		ext := ".srt"
		if strings.Contains(url, ".json") {
			ext = ".srt"
		} else {
			ext = ".ass"
		}
		subtitles = append(subtitles, entity.Subtitle{
			Lan:  item.Key,
			Url:  url,
			Path: fmt.Sprintf("%s/%s.%s.%s%s", aid, aid, cid, item.Key, ext),
		})
	}

	if hasEmptyURL(subtitles) {
		return nil, fmt.Errorf("bad url")
	}

	return subtitles, nil
}

func doGet(ctx context.Context, client httpclient.Client, url string, cfg *config.Config) ([]byte, error) {
	var opts []httpclient.RequestOption
	if cfg.Cookie != "" {
		opts = append(opts, httpclient.WithCookie(cfg.Cookie))
	}
	resp, err := client.Get(ctx, url, opts...)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func hasEmptyURL(subs []entity.Subtitle) bool {
	for _, s := range subs {
		if s.Url == "" {
			return true
		}
	}
	return false
}

// ConvertSubFromJSON converts Bilibili subtitle JSON to SRT format.
func ConvertSubFromJSON(jsonStr string) (string, error) {
	var sub struct {
		Body []struct {
			From    float64 `json:"from"`
			To      float64 `json:"to"`
			Content string  `json:"content"`
		} `json:"body"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &sub); err != nil {
		return "", err
	}

	var b strings.Builder
	for i, line := range sub.Body {
		b.WriteString(fmt.Sprintf("%d\n", i+1))
		from := 0.0
		if line.From > 0 {
			from = line.From
		}
		b.WriteString(fmt.Sprintf("%s --> %s\n", formatTime(from), formatTime(line.To)))
		if line.Content != "" {
			b.WriteString(line.Content)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String(), nil
}

func formatTime(sec float64) string {
	h := int(sec) / 3600
	m := (int(sec) % 3600) / 60
	s := int(sec) % 60
	ms := int((sec - float64(int(sec))) * 1000 + 0.5)
	if ms >= 1000 {
		ms = 999
	}
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}

// GetSubtitleCode maps a Bilibili subtitle language key to ISO-639-2/B code
// and a human-readable language name.
func GetSubtitleCode(key string) (string, string) {
	// Handle cases like "zh-hans" -> "zh-Hans"
	if idx := strings.Index(key, "-"); idx >= 0 && idx+1 < len(key) {
		runes := []rune(key)
		if unicode.IsLower(runes[idx+1]) {
			runes[idx+1] = unicode.ToUpper(runes[idx+1])
			key = string(runes)
		}
	}

	switch key {
	case "ai-Zh":
		return "chi", "Chinese (Simplified, AI)"
	case "ai-En":
		return "eng", "English (generated by ai)"
	case "zh-CN":
		return "chi", "Chinese (Simplified)"
	case "zh-HK":
		return "chi", "Chinese (Hong Kong Traditional)"
	case "zh-Hans":
		return "chi", "Chinese (Simplified)"
	case "zh-TW":
		return "chi", "Chinese (Taiwan Traditional)"
	case "zh-Hant":
		return "chi", "Chinese (Traditional)"
	case "en-US":
		return "eng", "English (USA)"
	case "ja":
		return "jpn", "Japanese"
	case "ko":
		return "kor", "Korean"
	case "zh-SG":
		return "chi", "Chinese (Singapore)"
	case "ab":
		return "abk", "Abkhaz"
	case "aa":
		return "aar", "Afar"
	case "af":
		return "afr", "Afrikaans"
	case "sq":
		return "alb", "Albanian"
	case "ase":
		return "ase", "American Sign Language"
	case "am":
		return "amh", "Amharic"
	case "arc":
		return "arc", "Aramaic"
	case "hy":
		return "arm", "Armenian"
	case "as":
		return "asm", "Assamese"
	case "ay":
		return "aym", "Aymara"
	case "az":
		return "aze", "Azerbaijani"
	case "bn":
		return "ben", "Bengali"
	case "ba":
		return "bak", "Bashkir"
	case "eu":
		return "baq", "Basque"
	case "be":
		return "bel", "Belarusian"
	case "bh":
		return "bih", "Bihari"
	case "bi":
		return "bis", "Bislama"
	case "bs":
		return "bos", "Bosnian"
	case "br":
		return "bre", "Breton"
	case "bg":
		return "bul", "Bulgarian"
	case "yue":
		return "chi", "Cantonese"
	case "yue-HK":
		return "chi", "Cantonese (Hong Kong)"
	case "ca":
		return "cat", "Catalan"
	case "chr":
		return "chr", "Cherokee"
	case "cho":
		return "cho", "Choctaw"
	case "co":
		return "cos", "Corsican"
	case "hr":
		return "hrv", "Croatian"
	case "cs":
		return "cze", "Czech"
	case "da":
		return "dan", "Danish"
	case "nl":
		return "dut", "Dutch"
	case "nl-BE":
		return "dut", "Dutch (Belgian)"
	case "nl-NL":
		return "dut", "Dutch (Netherlands)"
	case "dz":
		return "dzo", "Dzongkha"
	case "en":
		return "eng", "English"
	case "en-CA":
		return "eng", "English (Canada)"
	case "en-IE":
		return "eng", "English (Ireland)"
	case "en-GB":
		return "eng", "English (UK)"
	case "eo":
		return "epo", "Esperanto"
	case "et":
		return "est", "Estonian"
	case "fo":
		return "fao", "Faroese"
	case "fj":
		return "fij", "Fijian"
	case "fil":
		return "phi", "Filipino"
	case "fi":
		return "fin", "Finnish"
	case "fr":
		return "fre", "French"
	case "fr-BE":
		return "fre", "French (Belgium)"
	case "fr-CA":
		return "fre", "French (Canada)"
	case "fr-FR":
		return "fre", "French (France)"
	case "fr-CH":
		return "fre", "French (Switzerland)"
	case "ff":
		return "ful", "Fulah"
	case "gl":
		return "glg", "Galician"
	case "ka":
		return "geo", "Georgian"
	case "de":
		return "ger", "German"
	case "de-AT":
		return "ger", "German (Austria)"
	case "de-DE":
		return "ger", "German (Germany)"
	case "de-CH":
		return "ger", "German (Switzerland)"
	case "el":
		return "gre", "Greek"
	case "kl":
		return "kal", "Greenlandic"
	case "gn":
		return "grn", "Guarani"
	case "gu":
		return "guj", "Gujarati"
	case "hak":
		return "hak", "Hakka"
	case "hak-TW":
		return "hak", "Hakka (Taiwan)"
	case "ha":
		return "hau", "Hausa"
	case "iw":
		return "heb", "Hebrew"
	case "hi":
		return "hin", "Hindi"
	case "hi-Latn":
		return "hin", "Hindi (Phonetic)"
	case "hu":
		return "hun", "Hungarian"
	case "is":
		return "ice", "Icelandic"
	case "ig":
		return "ibo", "Igbo"
	case "id":
		return "ind", "Indonesian"
	case "ia":
		return "ina", "Interlingua"
	case "iu":
		return "iku", "Inuktitut"
	case "ik":
		return "ipk", "Inupiaq"
	case "ga":
		return "gle", "Irish"
	case "it":
		return "ita", "Italian"
	case "jv":
		return "jav", "Javanese"
	case "kn":
		return "kan", "Kannada"
	case "ks":
		return "kas", "Kashmiri"
	case "kk":
		return "kaz", "Kazakh"
	case "km":
		return "khm", "Khmer"
	case "rw":
		return "kin", "Kinyarwanda"
	case "tlh":
		return "tlh", "Klingon"
	case "ku":
		return "kur", "Kurdish"
	case "ky":
		return "kir", "Kyrgyz"
	case "lo":
		return "lao", "Lao"
	case "la":
		return "lat", "Latin"
	case "lv":
		return "lav", "Latvian"
	case "ln":
		return "lin", "Lingala"
	case "lt":
		return "lit", "Lithuanian"
	case "lb":
		return "ltz", "Luxembourgish"
	case "mk":
		return "mac", "Macedonian"
	case "mg":
		return "mlg", "Malagasy"
	case "ms":
		return "may", "Malay"
	case "ml":
		return "mal", "Malayalam"
	case "mt":
		return "mlt", "Maltese"
	case "mi":
		return "mao", "Maori"
	case "mr":
		return "mar", "Marathi"
	case "mas":
		return "mas", "Masai"
	case "nan":
		return "nan", "Min Nan"
	case "nan-TW":
		return "nan", "Min Nan (Taiwan)"
	case "lus":
		return "lus", "Mizo"
	case "mo":
		return "mol", "Moldavian"
	case "mn":
		return "mon", "Mongolian"
	case "my":
		return "bur", "Burmese"
	case "na":
		return "nau", "Nauru"
	case "nv":
		return "nav", "Navajo"
	case "ne":
		return "nep", "Nepali"
	case "no":
		return "nor", "Norwegian"
	case "fa":
		return "per", "Persian"
	case "fa-AF":
		return "per", "Persian (Afghanistan)"
	case "fa-IR":
		return "per", "Persian (Iran)"
	case "pl":
		return "pol", "Polish"
	case "pt":
		return "por", "Portuguese"
	case "pt-BR":
		return "por", "Portuguese (Brazil)"
	case "pt-PT":
		return "por", "Portuguese (Portugal)"
	case "ro":
		return "rum", "Romanian"
	case "ru":
		return "rus", "Russian"
	case "ru-Latn":
		return "rus", "Russian (Phonetic)"
	case "sr":
		return "srp", "Serbian"
	case "sr-Cyrl":
		return "srp", "Serbian (Cyrillic)"
	case "sr-Latn":
		return "srp", "Serbian (Latin)"
	case "sh":
		return "scr", "Serbo-Croatian"
	case "sk":
		return "slo", "Slovak"
	case "es":
		return "spa", "Spanish"
	case "es-419":
		return "spa", "Spanish (Latin America)"
	case "es-MX":
		return "spa", "Spanish (Mexico)"
	case "es-ES":
		return "spa", "Spanish (Spain)"
	case "es-US":
		return "spa", "Spanish (United States)"
	case "sv":
		return "swe", "Swedish"
	case "tl":
		return "tgl", "Tagalog"
	case "th":
		return "tha", "Thai"
	case "tr":
		return "tur", "Turkish"
	case "uk":
		return "ukr", "Ukrainian"
	case "ur":
		return "urd", "Urdu"
	case "vi":
		return "vie", "Vietnamese"
	default:
		return "und", "Undetermined"
	}
}
