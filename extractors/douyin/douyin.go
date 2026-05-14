package douyin

import (
	"bytes"
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	netURL "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dop251/goja"
	"github.com/pkg/errors"

	"github.com/iawia002/lux/extractors"
	"github.com/iawia002/lux/request"
	"github.com/iawia002/lux/utils"
)

func init() {
	e := New()
	extractors.Register("douyin", e)
	extractors.Register("iesdouyin", e)
}

//go:embed sign.js
var script string

type extractor struct{}

// New returns a douyin extractor.
func New() extractors.Extractor {
	return &extractor{}
}

// Extract is the main function to extract the data.
func (e *extractor) Extract(url string, option extractors.Options) ([]*extractors.Data, error) {
	if strings.Contains(url, "v.douyin.com") {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		c := http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		resp, err := c.Do(req)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		defer resp.Body.Close() // nolint
		url = resp.Header.Get("location")
	}

	itemIds := utils.MatchOneOf(url, `/video/(\d+)`)
	if len(itemIds) == 0 {
		return nil, errors.New("unable to get video ID")
	}
	if itemIds == nil || len(itemIds) < 2 {
		return nil, errors.WithStack(extractors.ErrURLParseFailed)
	}
	itemId := itemIds[len(itemIds)-1]

	cookie := ""
	if strings.TrimSpace(option.Cookie) == "" {
		// Dynamic cookies are enough for older Douyin responses. Newer responses
		// may still require the user's fresh browser cookies, which request.Request
		// applies globally when --cookie is provided.
		var err error
		cookie, err = createCookie()
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	jsonData, err := fetchAwemeDetail(itemId, url, cookie)
	if err != nil {
		if data, fallbackErr := extractWithYTDLP(url, option.Cookie); fallbackErr == nil {
			return data, nil
		} else if strings.TrimSpace(option.Cookie) != "" {
			return nil, errors.Errorf("%v；yt-dlp 兜底也失败：%v", err, fallbackErr)
		}
		return nil, errors.WithStack(err)
	}
	var douyin douyinData
	if err = json.Unmarshal([]byte(jsonData), &douyin); err != nil {
		return nil, errors.WithStack(err)
	}
	if douyin.StatusCode != 0 {
		return nil, errors.Errorf("douyin detail api returned status_code %d", douyin.StatusCode)
	}
	if douyin.AwemeDetail.AwemeID == "" {
		return nil, errors.New("douyin detail api did not return aweme_detail")
	}

	urlData := make([]*extractors.Part, 0)
	var douyinType extractors.DataType
	var totalSize int64
	// AwemeType: 0:video 68:image
	if douyin.AwemeDetail.AwemeType == 68 {
		douyinType = extractors.DataTypeImage
		for _, img := range douyin.AwemeDetail.Images {
			realURL := img.URLList[len(img.URLList)-1]
			size, err := request.Size(realURL, url)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			totalSize += size
			_, ext, err := utils.GetNameAndExt(realURL)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			urlData = append(urlData, &extractors.Part{
				URL:  realURL,
				Size: size,
				Ext:  ext,
			})
		}
	} else {
		douyinType = extractors.DataTypeVideo
		realURL := selectVideoURL(douyin)
		if realURL == "" {
			return nil, errors.New("douyin detail api did not return a playable video url")
		}
		totalSize, err = request.Size(realURL, url)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		urlData = append(urlData, &extractors.Part{
			URL:  realURL,
			Size: totalSize,
			Ext:  "mp4",
		})
	}
	streams := map[string]*extractors.Stream{
		"default": {
			Parts: urlData,
			Size:  totalSize,
		},
	}

	return []*extractors.Data{
		{
			Site:    "抖音 douyin.com",
			Title:   douyin.AwemeDetail.Desc,
			Type:    douyinType,
			Streams: streams,
			URL:     url,
		},
	}, nil
}

type ytDLPData struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Ext         string `json:"ext"`
	WebpageURL  string `json:"webpage_url"`
	Filesize    int64  `json:"filesize"`
	Formats     []struct {
		URL            string `json:"url"`
		Ext            string `json:"ext"`
		VCodec         string `json:"vcodec"`
		ACodec         string `json:"acodec"`
		Filesize       int64  `json:"filesize"`
		FilesizeApprox int64  `json:"filesize_approx"`
	} `json:"formats"`
}

func extractWithYTDLP(videoURL, cookieOption string) ([]*extractors.Data, error) {
	python, err := findPythonForYTDLP()
	if err != nil {
		return nil, err
	}
	args := []string{"-m", "yt_dlp", "--dump-json", "--no-playlist", "--no-warnings"}
	if cookieFile, cleanup, err := ytdlpCookieFile(cookieOption); err != nil {
		return nil, err
	} else if cookieFile != "" {
		defer cleanup()
		args = append(args, "--cookies", cookieFile)
	}
	args = append(args, videoURL)

	cmd := exec.Command(python, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, errors.New(msg)
	}
	var item ytDLPData
	if err := json.Unmarshal(out, &item); err != nil {
		return nil, err
	}
	realURL, ext, size := selectYTDLPURL(item)
	if realURL == "" {
		return nil, errors.New("yt-dlp did not return a playable video url")
	}
	if ext == "" {
		ext = "mp4"
	}
	title := strings.TrimSpace(item.Title)
	if title == "" {
		title = strings.TrimSpace(item.Description)
	}
	if title == "" {
		title = "douyin-" + filepath.Base(videoURL)
	}
	streams := map[string]*extractors.Stream{
		"default": {
			Parts: []*extractors.Part{{URL: realURL, Size: size, Ext: ext}},
			Size:  size,
		},
	}
	return []*extractors.Data{{
		Site:    "抖音 douyin.com",
		Title:   title,
		Type:    extractors.DataTypeVideo,
		Streams: streams,
		URL:     videoURL,
	}}, nil
}

func findPythonForYTDLP() (string, error) {
	candidates := []string{
		filepath.Join(os.Getenv("HOME"), "miniconda3", "envs", "whisperx_env", "bin", "python"),
		"python3",
		"python",
	}
	for _, candidate := range candidates {
		if strings.Contains(candidate, string(os.PathSeparator)) {
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
			continue
		}
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", errors.New("python not found for yt-dlp fallback")
}

func ytdlpCookieFile(cookieOption string) (string, func(), error) {
	cookieOption = strings.TrimSpace(cookieOption)
	if cookieOption == "" {
		return "", func() {}, nil
	}
	if data, err := os.ReadFile(cookieOption); err == nil {
		content := strings.TrimSpace(string(data))
		if strings.HasPrefix(content, "# Netscape HTTP Cookie File") {
			return cookieOption, func() {}, nil
		}
		return writeYTDLPNetscapeCookies(content)
	}
	return writeYTDLPNetscapeCookies(cookieOption)
}

func writeYTDLPNetscapeCookies(cookieHeader string) (string, func(), error) {
	file, err := os.CreateTemp("", "coding-lux-ytdlp-cookies-*.txt")
	if err != nil {
		return "", func() {}, err
	}
	path := file.Name()
	var builder strings.Builder
	builder.WriteString("# Netscape HTTP Cookie File\n")
	expiry := "2147483647"
	for _, part := range strings.Split(cookieHeader, ";") {
		part = strings.TrimSpace(part)
		if part == "" || !strings.Contains(part, "=") {
			continue
		}
		name, value, _ := strings.Cut(part, "=")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if name == "" {
			continue
		}
		for _, domain := range []string{".douyin.com", ".iesdouyin.com"} {
			builder.WriteString(fmt.Sprintf("%s\tTRUE\t/\tTRUE\t%s\t%s\t%s\n", domain, expiry, name, value))
		}
	}
	if _, err := file.WriteString(builder.String()); err != nil {
		file.Close()
		os.Remove(path) // nolint
		return "", func() {}, err
	}
	if err := file.Close(); err != nil {
		os.Remove(path) // nolint
		return "", func() {}, err
	}
	return path, func() { os.Remove(path) }, nil
}

func selectYTDLPURL(item ytDLPData) (string, string, int64) {
	if item.URL != "" {
		return item.URL, item.Ext, item.Filesize
	}
	for i := len(item.Formats) - 1; i >= 0; i-- {
		format := item.Formats[i]
		if format.URL == "" || format.VCodec == "none" {
			continue
		}
		size := format.Filesize
		if size == 0 {
			size = format.FilesizeApprox
		}
		return format.URL, format.Ext, size
	}
	return "", "", 0
}

func fetchAwemeDetail(itemId, refer, cookie string) (string, error) {
	userAgents := []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Linux; Android 12; Pixel 6 Build/SP2A.220505.002) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Mobile Safari/537.36",
	}
	apis := []string{
		douyinDetailAPI(itemId, true),
		douyinDetailAPI(itemId, false),
		"https://www.douyin.com/aweme/v1/web/aweme/detail/?aweme_id=" + itemId,
	}

	var lastErr error
	emptyResponses := 0
	for _, api := range apis {
		for _, userAgent := range userAgents {
			signedAPI, err := signAPI(api, userAgent)
			if err != nil {
				return "", err
			}
			headers := douyinHeaders(cookie, userAgent, refer)
			jsonData, err := request.Get(signedAPI, refer, headers)
			if err != nil {
				lastErr = err
				continue
			}
			jsonData = strings.TrimSpace(jsonData)
			if jsonData == "" {
				emptyResponses++
				lastErr = errors.New("douyin detail api returned an empty response")
				continue
			}
			if !strings.HasPrefix(jsonData, "{") {
				lastErr = errors.Errorf("douyin detail api returned non-json response: %.80s", jsonData)
				continue
			}
			return jsonData, nil
		}
	}

	if pageData, err := fetchDetailFromPage(refer, cookie); err == nil && pageData != "" {
		return pageData, nil
	} else if err != nil {
		lastErr = err
	}
	if emptyResponses > 0 {
		return "", errors.New("抖音需要 fresh cookies：请先在浏览器打开 douyin.com，再把 Cookie 粘贴到前端的“抖音 Cookie”输入框后重试")
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", errors.New("douyin detail api failed")
}

func douyinDetailAPI(itemId string, full bool) string {
	if !full {
		return "https://www.douyin.com/aweme/v1/web/aweme/detail/?aweme_id=" + itemId + "&aid=6383&device_platform=webapp"
	}
	values := netURL.Values{}
	values.Set("device_platform", "webapp")
	values.Set("aid", "6383")
	values.Set("channel", "channel_pc_web")
	values.Set("aweme_id", itemId)
	values.Set("update_version_code", "170400")
	values.Set("pc_client_type", "1")
	values.Set("version_code", "290100")
	values.Set("version_name", "29.1.0")
	values.Set("cookie_enabled", "true")
	values.Set("screen_width", "1920")
	values.Set("screen_height", "1080")
	values.Set("browser_language", "zh-CN")
	values.Set("browser_platform", "MacIntel")
	values.Set("browser_name", "Chrome")
	values.Set("browser_version", "124.0.0.0")
	values.Set("browser_online", "true")
	values.Set("engine_name", "Blink")
	values.Set("engine_version", "124.0.0.0")
	values.Set("os_name", "Mac OS")
	values.Set("os_version", "10.15.7")
	values.Set("cpu_core_num", "8")
	values.Set("device_memory", "8")
	values.Set("platform", "PC")
	values.Set("downlink", "10")
	values.Set("effective_type", "4g")
	values.Set("round_trip_time", "50")
	return "https://www.douyin.com/aweme/v1/web/aweme/detail/?" + values.Encode()
}

func signAPI(api, userAgent string) (string, error) {
	query, err := netURL.Parse(api)
	if err != nil {
		return "", errors.WithStack(extractors.ErrURLQueryParamsParseFailed)
	}
	vm := goja.New()
	_, _ = vm.RunString(script)
	sign, err := vm.RunString(fmt.Sprintf("sign('%s', '%s')", query.RawQuery, userAgent))
	if err != nil {
		return "", errors.WithStack(err)
	}
	return fmt.Sprintf("%s&X-Bogus=%s", api, sign), nil
}

func douyinHeaders(cookie, userAgent, refer string) map[string]string {
	return map[string]string{
		"Accept":             "application/json, text/plain, */*",
		"Accept-Language":    "zh-CN,zh;q=0.9,en;q=0.8",
		"Cookie":             cookie,
		"Referer":            refer,
		"Sec-Fetch-Dest":     "empty",
		"Sec-Fetch-Mode":     "cors",
		"Sec-Fetch-Site":     "same-origin",
		"User-Agent":         userAgent,
		"X-Requested-With":   "XMLHttpRequest",
		"sec-ch-ua":          `"Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99"`,
		"sec-ch-ua-mobile":   "?0",
		"sec-ch-ua-platform": `"macOS"`,
	}
}

func fetchDetailFromPage(pageURL, cookie string) (string, error) {
	headers := douyinHeaders(cookie, "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36", pageURL)
	res, err := request.Request(http.MethodGet, pageURL, nil, headers)
	if err != nil {
		return "", err
	}
	defer res.Body.Close() // nolint
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", errors.WithStack(err)
	}
	page := string(body)
	for _, pattern := range []string{
		`<script id="RENDER_DATA" type="application/json">([^<]+)</script>`,
		`<script id="ROUTER_DATA" type="application/json">([^<]+)</script>`,
	} {
		match := regexp.MustCompile(pattern).FindStringSubmatch(page)
		if len(match) != 2 {
			continue
		}
		decoded, err := netURL.QueryUnescape(match[1])
		if err != nil {
			decoded = match[1]
		}
		if detail := extractAwemeDetailJSON(decoded); detail != "" {
			return `{"status_code":0,"aweme_detail":` + detail + `}`, nil
		}
	}
	return "", errors.New("douyin page did not contain aweme detail json")
}

func extractAwemeDetailJSON(data string) string {
	keys := []string{`"aweme_detail":`, `"awemeDetail":`, `"aweme":`}
	for _, key := range keys {
		idx := strings.Index(data, key)
		if idx == -1 {
			continue
		}
		start := idx + len(key)
		for start < len(data) && (data[start] == ' ' || data[start] == '\n' || data[start] == '\t') {
			start++
		}
		if start >= len(data) || data[start] != '{' {
			continue
		}
		if obj := balancedJSONObject(data[start:]); obj != "" {
			return obj
		}
	}
	return ""
}

func balancedJSONObject(data string) string {
	depth := 0
	inString := false
	escaped := false
	for i, r := range data {
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '"' {
				inString = false
			}
			continue
		}
		switch r {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return data[:i+1]
			}
		}
	}
	return ""
}

func selectVideoURL(douyin douyinData) string {
	if urls := douyin.AwemeDetail.Video.PlayAddr.URLList; len(urls) > 0 {
		return urls[0]
	}
	if urls := douyin.AwemeDetail.Video.PlayAddrH264.URLList; len(urls) > 0 {
		return urls[0]
	}
	for _, item := range douyin.AwemeDetail.Video.BitRate {
		if len(item.PlayAddr.URLList) > 0 {
			return item.PlayAddr.URLList[0]
		}
	}
	return ""
}

func createCookie() (string, error) {
	v1, err := msToken(107)
	if err != nil {
		return "", err
	}
	v2, err := ttwid()
	if err != nil {
		return "", err
	}
	v3 := "324fb4ea4a89c0c05827e18a1ed9cf9bf8a17f7705fcc793fec935b637867e2a5a9b8168c885554d029919117a18ba69"
	v4 := "eyJiZC10aWNrZXQtZ3VhcmQtdmVyc2lvbiI6MiwiYmQtdGlja2V0LWd1YXJkLWNsaWVudC1jc3IiOiItLS0tLUJFR0lOIENFUlRJRklDQVRFIFJFUVVFU1QtLS0tLVxyXG5NSUlCRFRDQnRRSUJBREFuTVFzd0NRWURWUVFHRXdKRFRqRVlNQllHQTFVRUF3d1BZbVJmZEdsamEyVjBYMmQxXHJcbllYSmtNRmt3RXdZSEtvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUVKUDZzbjNLRlFBNUROSEcyK2F4bXAwNG5cclxud1hBSTZDU1IyZW1sVUE5QTZ4aGQzbVlPUlI4NVRLZ2tXd1FJSmp3Nyszdnc0Z2NNRG5iOTRoS3MvSjFJc3FBc1xyXG5NQ29HQ1NxR1NJYjNEUUVKRGpFZE1Cc3dHUVlEVlIwUkJCSXdFSUlPZDNkM0xtUnZkWGxwYmk1amIyMHdDZ1lJXHJcbktvWkl6ajBFQXdJRFJ3QXdSQUlnVmJkWTI0c0RYS0c0S2h3WlBmOHpxVDRBU0ROamNUb2FFRi9MQnd2QS8xSUNcclxuSURiVmZCUk1PQVB5cWJkcytld1QwSDZqdDg1czZZTVNVZEo5Z2dmOWlmeTBcclxuLS0tLS1FTkQgQ0VSVElGSUNBVEUgUkVRVUVTVC0tLS0tXHJcbiJ9"
	cookie := fmt.Sprintf("msToken=%s;ttwid=%s;odin_tt=%s;bd_ticket_guard_client_data=%s;", v1, v2, v3, v4)
	return cookie, nil
}

func msToken(length int) (string, error) {
	const characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	token := make([]byte, length)
	for i, b := range randomBytes {
		token[i] = characters[int(b)%len(characters)]
	}
	return string(token), nil
}

func ttwid() (string, error) {
	body := map[string]interface{}{
		"aid":           1768,
		"union":         true,
		"needFid":       false,
		"region":        "cn",
		"cbUrlProtocol": "https",
		"service":       "www.ixigua.com",
		"migrate_info":  map[string]string{"ticket": "", "source": "node"},
	}
	bytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	payload := strings.NewReader(string(bytes))
	resp, err := request.Request(http.MethodPost, "https://ttwid.bytedance.com/ttwid/union/register/", payload, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() // nolint
	cookie := resp.Header.Get("Set-Cookie")
	re := regexp.MustCompile(`ttwid=([^;]+)`)
	if match := re.FindStringSubmatch(cookie); match != nil {
		return match[1], nil
	}
	return "", errors.New("douyin ttwid request failed")
}
