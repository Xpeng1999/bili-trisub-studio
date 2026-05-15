package app

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type webJob struct {
	ID           string    `json:"id"`
	Platform     string    `json:"platform"`
	URL          string    `json:"url"`
	OutputDir    string    `json:"outputDir"`
	StreamFormat string    `json:"streamFormat,omitempty"`
	LLMBaseURL   string    `json:"-"`
	LLMAPIKey    string    `json:"-"`
	LLMModel     string    `json:"-"`
	Status       string    `json:"status"`
	Progress     int       `json:"progress"`
	Logs         []string  `json:"logs"`
	Error        string    `json:"error,omitempty"`
	VideoURL     string    `json:"videoUrl,omitempty"`
	SubtitleURL  string    `json:"subtitleUrl,omitempty"`
	SubtitlePath string    `json:"subtitlePath,omitempty"`
	StartedAt    time.Time `json:"startedAt"`
	EndedAt      time.Time `json:"endedAt,omitempty"`
}

type triPayload struct {
	Version  int          `json:"version"`
	Segments []triSegment `json:"segments"`
}

type triSegment struct {
	Index  int     `json:"index"`
	Start  float64 `json:"start"`
	End    float64 `json:"end"`
	ZH     string  `json:"zh"`
	EN     string  `json:"en"`
	Pinyin string  `json:"pinyin"`
}

type webServer struct {
	mu   sync.Mutex
	jobs map[string]*webJob
	cwd  string
	exe  string
}

type localLLMDefaults struct {
	BaseURL string `json:"baseUrl"`
	Model   string `json:"model"`
	APIKey  string `json:"apiKey"`
}

// StartWebServer starts the local browser interface.
func StartWebServer(addr string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	s := &webServer{
		jobs: map[string]*webJob{},
		cwd:  wd,
		exe:  exe,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.index)
	mux.HandleFunc("/api/jobs", s.createJob)
	mux.HandleFunc("/api/jobs/", s.getJob)
	mux.HandleFunc("/api/dirs", s.listDirs)
	mux.HandleFunc("/api/pinyin", s.generatePinyin)
	mux.HandleFunc("/api/subtitles/save", s.saveSubtitles)
	mux.HandleFunc("/media", s.media)

	fmt.Printf("Web UI running at http://%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *webServer) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	defaultsJSON, _ := json.Marshal(s.loadLocalLLMDefaults())
	html := strings.Replace(webIndexHTML, "__LOCAL_LLM_DEFAULTS__", string(defaultsJSON), 1)
	_, _ = io.WriteString(w, html)
}

func (s *webServer) loadLocalLLMDefaults() localLLMDefaults {
	path := filepath.Join(s.cwd, "local_llm_config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return localLLMDefaults{}
	}
	var defaults localLLMDefaults
	if err := json.Unmarshal(data, &defaults); err != nil {
		return localLLMDefaults{}
	}
	defaults.BaseURL = strings.TrimSpace(defaults.BaseURL)
	defaults.Model = strings.TrimSpace(defaults.Model)
	defaults.APIKey = strings.TrimSpace(defaults.APIKey)
	return defaults
}

func (s *webServer) createJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Platform   string `json:"platform"`
		URL        string `json:"url"`
		OutputDir  string `json:"outputDir"`
		Quality    string `json:"quality"`
		LLMBaseURL string `json:"llmBaseUrl"`
		LLMAPIKey  string `json:"llmApiKey"`
		LLMModel   string `json:"llmModel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	platform := strings.TrimSpace(req.Platform)
	videoURL := strings.TrimSpace(req.URL)
	outputDir := strings.TrimSpace(req.OutputDir)
	streamFormat, err := streamFormatFromQuality(req.Quality)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	llmBaseURL := strings.TrimSpace(req.LLMBaseURL)
	llmAPIKey := strings.TrimSpace(req.LLMAPIKey)
	llmModel := strings.TrimSpace(req.LLMModel)
	if platform != "bilibili" {
		http.Error(w, "当前版本仅支持 B站 视频", http.StatusBadRequest)
		return
	}
	if videoURL == "" {
		http.Error(w, "请粘贴视频链接", http.StatusBadRequest)
		return
	}
	if !platformMatchesURL(platform, videoURL) {
		http.Error(w, "链接和选择的平台不匹配", http.StatusBadRequest)
		return
	}
	if outputDir == "" {
		outputDir = defaultDownloadDir()
	}
	if llmBaseURL == "" || llmAPIKey == "" || llmModel == "" {
		http.Error(w, "请填写大模型 API 地址、API Key 和模型名称", http.StatusBadRequest)
		return
	}
	if parsed, err := neturl.ParseRequestURI(llmBaseURL); err != nil || parsed.Scheme == "" || parsed.Host == "" {
		http.Error(w, "大模型 API 地址格式不正确", http.StatusBadRequest)
		return
	}
	info, err := os.Stat(outputDir)
	if err != nil || !info.IsDir() {
		http.Error(w, "下载目录不存在", http.StatusBadRequest)
		return
	}

	id := newID()
	childDir := filepath.Join(outputDir, childFolderName(platform, videoURL))
	if err := os.MkdirAll(childDir, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jobLogs := []string{"任务已创建", "输出目录: " + childDir}

	job := &webJob{
		ID:           id,
		Platform:     platform,
		URL:          videoURL,
		OutputDir:    childDir,
		StreamFormat: streamFormat,
		LLMBaseURL:   llmBaseURL,
		LLMAPIKey:    llmAPIKey,
		LLMModel:     llmModel,
		Status:       "running",
		Progress:     3,
		StartedAt:    time.Now(),
		Logs:         jobLogs,
	}

	s.mu.Lock()
	s.jobs[id] = job
	s.mu.Unlock()

	go s.runJob(job)
	writeJSON(w, job)
}

func (s *webServer) getJob(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
	s.mu.Lock()
	job := s.jobs[id]
	s.mu.Unlock()
	if job == nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, job)
}

func (s *webServer) listDirs(w http.ResponseWriter, r *http.Request) {
	dir := strings.TrimSpace(r.URL.Query().Get("path"))
	if dir == "" {
		dir = defaultDownloadDir()
	}
	dir = filepath.Clean(os.ExpandEnv(dir))
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		http.Error(w, "目录不存在", http.StatusBadRequest)
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	dirs := []string{}
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			dirs = append(dirs, entry.Name())
		}
	}
	sort.Strings(dirs)

	writeJSON(w, map[string]any{
		"path":   dir,
		"parent": filepath.Dir(dir),
		"dirs":   dirs,
	})
}

func (s *webServer) runJob(job *webJob) {
	args := []string{
		"--silent",
		"--wait-subtitle",
		"--output-path", job.OutputDir,
	}
	if job.StreamFormat != "" {
		args = append(args, "--stream-format", job.StreamFormat)
	}
	args = append(args, job.URL)
	cmd := exec.Command(s.exe, args...)
	cmd.Dir = s.cwd
	cmd.Env = append(os.Environ(),
		"LUX_WHISPERX_SUB_DIR="+filepath.Join(s.cwd, "whisperx_Sub"),
		"LUX_LLM_BASE_URL="+job.LLMBaseURL,
		"LUX_LLM_API_KEY="+job.LLMAPIKey,
		"LUX_LLM_MODEL="+job.LLMModel,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.failJob(job, err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.failJob(job, err)
		return
	}
	if err := cmd.Start(); err != nil {
		s.failJob(job, err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go s.scanJobOutput(job, stdout, &wg)
	go s.scanJobOutput(job, stderr, &wg)
	wg.Wait()

	err = cmd.Wait()
	videoPath, subtitlePath := findPreviewFiles(job.OutputDir)
	s.mu.Lock()
	defer s.mu.Unlock()
	job.EndedAt = time.Now()
	if videoPath != "" {
		job.VideoURL = "/media?path=" + urlQueryEscape(videoPath)
	}
	if subtitlePath != "" {
		job.SubtitleURL = "/media?path=" + urlQueryEscape(subtitlePath)
		job.SubtitlePath = subtitlePath
	}
	if err != nil {
		job.Status = "failed"
		job.Error = err.Error()
		job.Progress = min(job.Progress, 95)
		job.Logs = append(job.Logs, "任务失败: "+err.Error())
		return
	}
	job.Status = "done"
	job.Progress = 100
	job.Logs = append(job.Logs, "任务完成")
}

func (s *webServer) media(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

func (s *webServer) generatePinyin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pinyin, err := s.pinyin(req.Text)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, pinyin)
}

func (s *webServer) saveSubtitles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Path     string       `json:"path"`
		Segments []triSegment `json:"segments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	path := filepath.Clean(req.Path)
	if !strings.HasSuffix(strings.ToLower(path), "_tri.json") {
		http.Error(w, "只能保存 *_tri.json 字幕文件", http.StatusBadRequest)
		return
	}
	if len(req.Segments) == 0 {
		http.Error(w, "没有可保存的字幕段落", http.StatusBadRequest)
		return
	}
	segments, err := s.normalizeSegments(req.Segments)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := writeTriFiles(path, segments); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "path": path, "segments": len(segments)})
}

type pinyinResponse struct {
	Text   string `json:"text"`
	Pinyin string `json:"pinyin"`
}

func (s *webServer) pinyin(text string) (pinyinResponse, error) {
	var resp pinyinResponse
	if err := s.runTriEdit("pinyin", map[string]string{"text": text}, &resp); err != nil {
		return pinyinResponse{}, err
	}
	return resp, nil
}

func (s *webServer) normalizeSegments(segments []triSegment) ([]triSegment, error) {
	var resp struct {
		Segments []triSegment `json:"segments"`
	}
	if err := s.runTriEdit("normalize", map[string]any{"segments": segments}, &resp); err != nil {
		return nil, err
	}
	return resp.Segments, nil
}

func (s *webServer) runTriEdit(command string, payload any, response any) error {
	script := filepath.Join(s.cwd, "whisperx_Sub", "tri_edit.py")
	python := findWebPython(s.cwd)
	cmd := exec.Command(python, script, command)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	var out strings.Builder
	var stderr strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := json.NewEncoder(stdin).Encode(payload); err != nil {
		stdin.Close()
		return err
	}
	stdin.Close()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}
	if err := json.Unmarshal([]byte(out.String()), response); err != nil {
		return err
	}
	return nil
}

func findWebPython(root string) string {
	candidates := []string{}
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			filepath.Join(root, ".venv", "Scripts", "python.exe"),
			filepath.Join(root, "venv", "Scripts", "python.exe"),
			"py",
			"python",
		)
	} else {
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates, filepath.Join(home, "miniconda3", "envs", "whisperx_env", "bin", "python"))
		}
		candidates = append(candidates,
			filepath.Join(root, ".venv", "bin", "python"),
			filepath.Join(root, "venv", "bin", "python"),
			"python3",
			"python",
		)
	}
	for _, candidate := range candidates {
		if strings.Contains(candidate, string(filepath.Separator)) {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
			continue
		}
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	return "python"
}

func (s *webServer) scanJobOutput(job *webJob, r io.Reader, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		s.mu.Lock()
		job.Logs = append(job.Logs, line)
		job.Progress = estimateProgress(job.Progress, line)
		if len(job.Logs) > 400 {
			job.Logs = job.Logs[len(job.Logs)-400:]
		}
		s.mu.Unlock()
	}
}

func (s *webServer) failJob(job *webJob, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job.Status = "failed"
	job.Error = err.Error()
	job.EndedAt = time.Now()
	job.Logs = append(job.Logs, "任务失败: "+err.Error())
}

func estimateProgress(current int, line string) int {
	next := current
	switch {
	case strings.Contains(line, "Downloading captions"):
		next = 18
	case strings.Contains(line, "Chinese SRT"):
		next = 72
	case strings.Contains(line, "Translating"):
		next = 82
	case strings.Contains(line, "English SRT"):
		next = 88
	case strings.Contains(line, "Generating pinyin"):
		next = 94
	case strings.Contains(line, "Pinyin SRT"):
		next = 98
	case strings.Contains(line, "Tri-language JSON"):
		next = 99
	case strings.Contains(line, "[subtitle]"):
		next = max(current, 65)
	default:
		next = min(current+1, 60)
	}
	return max(current, next)
}

func platformMatchesURL(platform, videoURL string) bool {
	lower := strings.ToLower(videoURL)
	if platform == "bilibili" {
		return strings.Contains(lower, "bilibili.com") || strings.Contains(lower, "b23.tv") || strings.Contains(videoURL, "BV")
	}
	return false
}

func streamFormatFromQuality(quality string) (string, error) {
	switch strings.TrimSpace(quality) {
	case "", "auto":
		return "", nil
	case "360":
		return "360P", nil
	case "480":
		return "480P", nil
	case "720":
		return "720P", nil
	case "1080":
		return "1080P", nil
	default:
		return "", fmt.Errorf("不支持的视频清晰度: %s", quality)
	}
}

func childFolderName(platform, videoURL string) string {
	id := extractVideoID(videoURL)
	if id == "" {
		id = time.Now().Format("20060102-150405")
	}
	return sanitizeFolderName(platform + "-" + id)
}

func extractVideoID(videoURL string) string {
	if match := regexp.MustCompile(`BV[0-9A-Za-z]+`).FindString(videoURL); match != "" {
		return match
	}
	if match := regexp.MustCompile(`video/([0-9]+)`).FindStringSubmatch(videoURL); len(match) == 2 {
		return match[1]
	}
	if match := regexp.MustCompile(`modal_id=([0-9]+)`).FindStringSubmatch(videoURL); len(match) == 2 {
		return match[1]
	}
	return ""
}

func sanitizeFolderName(name string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "'", "<", "-", ">", "-", "|", "-")
	name = strings.TrimSpace(replacer.Replace(name))
	if name == "" {
		return "video-" + time.Now().Format("20060102-150405")
	}
	return name + "-" + time.Now().Format("20060102-150405")
}

func defaultDownloadDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "Downloads")
	}
	return "."
}

func newID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf[:])
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func writeTriFiles(triPath string, segments []triSegment) error {
	for i := range segments {
		segments[i].Index = i + 1
		segments[i].ZH = strings.TrimSpace(segments[i].ZH)
		segments[i].EN = strings.TrimSpace(segments[i].EN)
		segments[i].Pinyin = strings.TrimSpace(segments[i].Pinyin)
		if segments[i].End < segments[i].Start {
			segments[i].End = segments[i].Start
		}
	}
	payload := triPayload{Version: 1, Segments: segments}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(triPath, data, 0644); err != nil {
		return err
	}

	base := strings.TrimSuffix(triPath, "_tri.json")
	files := map[string]string{
		base + "_aligned_zh.srt":     "zh",
		base + "_aligned_en.srt":     "en",
		base + "_aligned_pinyin.srt": "pinyin",
		base + "_zh.srt":             "zh",
		base + "_en.srt":             "en",
		base + "_pinyin.srt":         "pinyin_bilingual",
	}
	for path, kind := range files {
		if err := os.WriteFile(path, []byte(renderSRT(segments, kind)), 0644); err != nil {
			return err
		}
	}
	return nil
}

func renderSRT(segments []triSegment, kind string) string {
	var blocks []string
	for i, segment := range segments {
		var text string
		switch kind {
		case "zh":
			text = segment.ZH
		case "en":
			text = segment.EN
		case "pinyin":
			text = segment.Pinyin
		case "pinyin_bilingual":
			text = strings.TrimSpace(segment.ZH + "\n" + segment.Pinyin)
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		blocks = append(blocks, fmt.Sprintf(
			"%d\n%s --> %s\n%s",
			i+1,
			formatSRTTime(segment.Start),
			formatSRTTime(segment.End),
			text,
		))
	}
	return strings.Join(blocks, "\n\n")
}

func formatSRTTime(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	totalMillis := int64(seconds*1000 + 0.5)
	hours := totalMillis / 3600000
	totalMillis %= 3600000
	minutes := totalMillis / 60000
	totalMillis %= 60000
	secs := totalMillis / 1000
	millis := totalMillis % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, secs, millis)
}

func urlQueryEscape(path string) string {
	return neturl.QueryEscape(path)
}

func findPreviewFiles(dir string) (string, string) {
	var videoPath string
	var subtitlePath string
	videoExts := map[string]bool{".mp4": true, ".mkv": true, ".mov": true, ".webm": true, ".flv": true}
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		name := strings.ToLower(filepath.Base(path))
		if videoPath == "" && videoExts[ext] && !strings.HasSuffix(name, ".download") {
			videoPath = path
		}
		if strings.HasSuffix(name, "_tri.json") {
			subtitlePath = path
		}
		return nil
	})
	return videoPath, subtitlePath
}

const webIndexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>coding-lux 字幕下载器</title>
  <style>
    :root { color-scheme: light; --ink:#1f2933; --muted:#687385; --line:#d9dee7; --panel:#ffffff; --bg:#f5f6f8; --accent:#176b61; --dark:#34383e; --soft:#eef1f5; --mark:#f9d9e1; }
    * { box-sizing:border-box; }
    body { margin:0; min-height:100vh; font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif; color:var(--ink); background:var(--bg); }
    main { width:min(1120px, calc(100% - 28px)); margin:24px auto 40px; }
    h1 { margin:0 0 6px; font-size:27px; font-weight:760; letter-spacing:0; }
    .sub { margin:0 0 20px; color:var(--muted); }
    .shell { background:var(--panel); border:1px solid var(--line); border-radius:8px; overflow:hidden; box-shadow:0 10px 28px rgba(20, 31, 44, .07); }
    .form { display:grid; grid-template-columns:170px 1fr; gap:16px; padding:20px; border-bottom:1px solid var(--line); }
    label { display:block; font-size:13px; font-weight:720; margin-bottom:7px; color:#354052; }
    select, input { width:100%; height:42px; border:1px solid #cbd3df; border-radius:6px; padding:0 12px; font-size:15px; background:white; color:var(--ink); }
    .url { grid-column:1 / -1; }
    .llm { grid-column:1 / -1; display:grid; grid-template-columns:1.3fr 1fr 1fr; gap:12px; }
    .dir { grid-column:1 / -1; display:grid; grid-template-columns:1fr auto; gap:10px; align-items:end; }
    button { height:42px; border:0; border-radius:6px; padding:0 15px; font-size:14px; font-weight:760; cursor:pointer; color:white; background:var(--accent); }
    button.secondary { color:#222a35; background:#e4e8ee; }
    button.icon { width:52px; height:52px; padding:0; border-radius:50%; display:grid; place-items:center; color:#4d535c; background:#eceef1; font-size:26px; }
    button.icon.dark { color:white; background:#45484d; }
    button:disabled { opacity:.55; cursor:not-allowed; }
    .actions { grid-column:1 / -1; display:flex; align-items:center; justify-content:space-between; gap:12px; }
    .hint { color:var(--muted); font-size:13px; }
    .status { padding:18px 20px; border-bottom:1px solid var(--line); }
    .bar { height:13px; border-radius:999px; overflow:hidden; background:#e5e8ed; border:1px solid #d4dae3; }
    .fill { width:0%; height:100%; background:linear-gradient(90deg, #176b61, #2d6cdf); transition:width .25s ease; }
    .meta { display:flex; justify-content:space-between; margin:9px 0 12px; color:var(--muted); font-size:13px; }
    details { margin-top:12px; }
    summary { cursor:pointer; color:#424b57; font-weight:700; }
    pre { margin:10px 0 0; min-height:140px; max-height:300px; overflow:auto; padding:14px; background:#101828; color:#e7eef7; border-radius:6px; line-height:1.45; font-size:13px; white-space:pre-wrap; }
    .preview { display:none; padding:20px; background:#f7f8fa; }
    .preview.ready { display:block; }
    .video-wrap { position:relative; background:#111; border-radius:8px; overflow:hidden; aspect-ratio:16 / 9; }
    video { width:100%; height:100%; display:block; object-fit:contain; background:#111; }
    .controls { display:grid; grid-template-columns:repeat(4, 1fr); gap:12px; align-items:start; padding:18px 6px 16px; }
    .control { display:grid; justify-items:center; gap:7px; color:#555d68; font-size:14px; min-width:0; }
    .control span { white-space:nowrap; }
    .rate { height:52px; display:grid; place-items:center; font-size:18px; }
    .subtitle-card { min-height:320px; background:white; border:1px solid #e2e6ed; border-radius:8px; box-shadow:0 2px 10px rgba(15,23,42,.08); padding:24px; display:grid; gap:18px; }
    .subtitle-tools { display:flex; align-items:center; justify-content:space-between; color:#737b86; font-size:15px; }
    .counter { display:flex; gap:14px; align-items:center; }
    .tool-actions { display:flex; gap:10px; align-items:center; }
    .editor { display:grid; gap:14px; }
    .field { display:grid; gap:7px; }
    .field-head { display:flex; justify-content:space-between; gap:10px; align-items:center; color:#4b5563; font-size:13px; font-weight:720; }
    textarea { width:100%; min-height:76px; resize:vertical; border:1px solid #cfd6e2; border-radius:6px; padding:10px 12px; font:16px/1.45 -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif; color:#202734; }
    textarea.en-edit { font-size:18px; font-weight:650; }
    .pinyin-box { min-height:48px; padding:11px 12px; border:1px solid #e0e5ec; border-radius:6px; background:#f8fafc; color:#606b7a; line-height:1.55; overflow-wrap:anywhere; }
    .save-state { color:#687385; font-size:13px; }
    dialog { width:min(720px, calc(100% - 28px)); border:1px solid var(--line); border-radius:8px; padding:0; }
    dialog::backdrop { background:rgba(15,23,42,.32); }
    .dialog-head, .dialog-foot { padding:14px 16px; display:flex; gap:10px; align-items:center; border-bottom:1px solid var(--line); }
    .dialog-foot { border-top:1px solid var(--line); border-bottom:0; justify-content:flex-end; }
    .dirs { padding:10px 16px; max-height:420px; overflow:auto; }
    .dir-row { width:100%; height:38px; margin:3px 0; display:flex; align-items:center; border:1px solid transparent; background:white; color:var(--ink); text-align:left; font-weight:650; }
    .dir-row:hover { border-color:#cbd5e1; background:#f8fafc; }
    @media (max-width:820px) { .form { grid-template-columns:1fr; } .llm { grid-template-columns:1fr; } .dir { grid-template-columns:1fr; } .actions { align-items:stretch; flex-direction:column; } button { width:100%; } .controls { grid-template-columns:repeat(2, 1fr); } button.icon { width:48px; height:48px; } .subtitle-card { padding:18px; } .subtitle-tools { align-items:flex-start; flex-direction:column; } }
  </style>
</head>
<body>
  <main>
    <h1>coding-lux 字幕下载器</h1>
    <p class="sub">下载完成后在下方预览视频，三语字幕显示在视频外的卡片中。</p>
    <section class="shell">
      <div class="form">
        <div>
          <label for="platform">平台</label>
          <select id="platform">
            <option value="bilibili">B站</option>
          </select>
        </div>
        <div>
          <label for="quality">视频清晰度</label>
          <select id="quality">
            <option value="auto">自动最高</option>
            <option value="1080">1080P</option>
            <option value="720">720P</option>
            <option value="480">480P</option>
            <option value="360">360P</option>
          </select>
        </div>
        <div class="url">
          <label for="url">视频链接</label>
          <input id="url" placeholder="https://www.bilibili.com/video/BV...">
        </div>
        <div class="llm">
          <div>
            <label for="llmBaseUrl">大模型 API 地址</label>
            <input id="llmBaseUrl" placeholder="https://api.example.com/v1">
          </div>
          <div>
            <label for="llmModel">模型名称</label>
            <input id="llmModel" placeholder="填写服务商提供的模型名称">
          </div>
          <div>
            <label for="llmApiKey">API Key</label>
            <input id="llmApiKey" type="password" autocomplete="off" placeholder="sk-...">
          </div>
        </div>
        <div class="dir">
          <div>
            <label for="outputDir">下载目录</label>
            <input id="outputDir">
          </div>
          <button class="secondary" id="browse" type="button">选择目录</button>
        </div>
        <div class="actions">
          <span class="hint" id="hint">任务运行时请保持这个窗口和后端服务开启。</span>
          <button id="start">下载 + 添加字幕</button>
        </div>
      </div>
      <div class="status">
        <div class="bar"><div class="fill" id="fill"></div></div>
        <div class="meta"><span id="state">等待开始</span><span id="percent">0%</span></div>
        <details>
          <summary>任务日志</summary>
          <pre id="logs">暂无日志</pre>
        </details>
      </div>
      <section class="preview" id="preview">
        <div class="video-wrap">
          <video id="video" controls playsinline></video>
        </div>
        <div class="controls">
          <div class="control"><button class="icon" id="replay" title="重放本句">↻</button><span>重放</span></div>
          <div class="control"><button class="icon dark" id="prev" title="上一句">←</button><span>上一句</span></div>
          <div class="control"><button class="icon dark" id="play" title="播放">▶</button><span>播放</span></div>
          <div class="control"><button class="icon dark" id="next" title="下一句">→</button><span>下一句</span></div>
        </div>
        <div class="subtitle-card">
          <div class="subtitle-tools">
            <div class="counter"><span id="captionIndex">0 / 0</span><span id="captionTime">00:00 - 00:00</span><span>字幕校对</span></div>
            <div class="tool-actions">
              <span class="save-state" id="saveState">未修改</span>
              <button class="secondary" id="confirmZh" type="button">确认中文并生成拼音</button>
              <button id="saveSubtitles" type="button">保存字幕文件</button>
            </div>
          </div>
          <div class="editor">
            <div class="field">
              <div class="field-head"><span>中文字幕</span><span>确认后更新拼音</span></div>
              <textarea id="captionZh" placeholder="中文字幕"></textarea>
            </div>
            <div class="field">
              <div class="field-head"><span>英文字幕</span><span>可直接校对</span></div>
              <textarea class="en-edit" id="captionEn" placeholder="English subtitle"></textarea>
            </div>
            <div class="field">
              <div class="field-head"><span>拼音字幕</span><span>由中文字幕生成</span></div>
              <div class="pinyin-box" id="captionPinyin">等待字幕数据</div>
            </div>
          </div>
        </div>
      </section>
    </section>
  </main>

  <dialog id="dirDialog">
    <div class="dialog-head">
      <input id="dirPath">
      <button class="secondary" id="goDir">打开</button>
    </div>
    <div class="dirs" id="dirs"></div>
    <div class="dialog-foot">
      <button class="secondary" id="closeDir">取消</button>
      <button id="chooseDir">选择当前目录</button>
    </div>
  </dialog>

  <script>
    const platform = document.querySelector("#platform");
    const quality = document.querySelector("#quality");
    const url = document.querySelector("#url");
    const outputDir = document.querySelector("#outputDir");
    const llmBaseUrl = document.querySelector("#llmBaseUrl");
    const llmModel = document.querySelector("#llmModel");
    const llmApiKey = document.querySelector("#llmApiKey");
    const start = document.querySelector("#start");
    const fill = document.querySelector("#fill");
    const state = document.querySelector("#state");
    const percent = document.querySelector("#percent");
    const logs = document.querySelector("#logs");
    const hint = document.querySelector("#hint");
    const dialog = document.querySelector("#dirDialog");
    const dirPath = document.querySelector("#dirPath");
    const dirs = document.querySelector("#dirs");
    const preview = document.querySelector("#preview");
    const video = document.querySelector("#video");
    const play = document.querySelector("#play");
    const captionIndex = document.querySelector("#captionIndex");
    const captionTime = document.querySelector("#captionTime");
    const captionEn = document.querySelector("#captionEn");
    const captionZh = document.querySelector("#captionZh");
    const captionPinyin = document.querySelector("#captionPinyin");
    const confirmZh = document.querySelector("#confirmZh");
    const saveSubtitles = document.querySelector("#saveSubtitles");
    const saveState = document.querySelector("#saveState");

    let segments = [];
    let activeIndex = -1;
    let subtitlePath = "";
    let dirty = false;
    outputDir.value = "";
    const localLlmDefaults = __LOCAL_LLM_DEFAULTS__;
    llmBaseUrl.value = localLlmDefaults.baseUrl || "";
    llmModel.value = localLlmDefaults.model || "";
    llmApiKey.value = localLlmDefaults.apiKey || "";

    async function api(path, options) {
      const res = await fetch(path, options);
      if (!res.ok) throw new Error(await res.text());
      return res.json();
    }

    async function loadDirs(path = "") {
      const data = await api("/api/dirs" + (path ? "?path=" + encodeURIComponent(path) : ""));
      dirPath.value = data.path;
      dirs.innerHTML = "";
      const parent = document.createElement("button");
      parent.className = "dir-row";
      parent.textContent = "上一级: " + data.parent;
      parent.onclick = () => loadDirs(data.parent);
      dirs.appendChild(parent);
      for (const name of data.dirs) {
        const row = document.createElement("button");
        row.className = "dir-row";
        row.textContent = name;
        row.onclick = () => loadDirs(data.path.replace(/\/$/, "") + "/" + name);
        dirs.appendChild(row);
      }
    }

    document.querySelector("#browse").onclick = async () => {
      dialog.showModal();
      try { await loadDirs(outputDir.value); } catch (e) { hint.textContent = e.message.trim(); }
    };
    document.querySelector("#goDir").onclick = () => loadDirs(dirPath.value).catch(e => hint.textContent = e.message.trim());
    document.querySelector("#closeDir").onclick = () => dialog.close();
    document.querySelector("#chooseDir").onclick = () => {
      outputDir.value = dirPath.value;
      dialog.close();
    };

    start.onclick = async () => {
      if (!llmBaseUrl.value.trim() || !llmModel.value.trim() || !llmApiKey.value.trim()) {
        state.textContent = "参数不完整";
        logs.textContent = "请先填写大模型 API 地址、模型名称和 API Key。";
        return;
      }
      start.disabled = true;
      logs.textContent = "任务提交中...";
      state.textContent = "运行中";
      preview.classList.remove("ready");
      try {
        const job = await api("/api/jobs", {
          method: "POST",
          headers: {"Content-Type": "application/json"},
          body: JSON.stringify({
            platform: platform.value,
            quality: quality.value,
            url: url.value,
            outputDir: outputDir.value,
            llmBaseUrl: llmBaseUrl.value,
            llmModel: llmModel.value,
            llmApiKey: llmApiKey.value
          })
        });
        poll(job.id);
      } catch (e) {
        start.disabled = false;
        state.textContent = "启动失败";
        logs.textContent = e.message.trim();
      }
    };

    async function poll(id) {
      const job = await api("/api/jobs/" + id);
      fill.style.width = job.progress + "%";
      percent.textContent = job.progress + "%";
      state.textContent = job.status === "done" ? "完成" : job.status === "failed" ? "失败" : "运行中";
      logs.textContent = (job.logs || []).join("\n");
      logs.scrollTop = logs.scrollHeight;
      if (job.status === "running") {
        setTimeout(() => poll(id).catch(e => logs.textContent += "\n" + e.message), 1000);
      } else {
        start.disabled = false;
        hint.textContent = "输出目录: " + job.outputDir;
        if (job.videoUrl) await loadPreview(job);
      }
    }

    async function loadPreview(job) {
      preview.classList.add("ready");
      video.src = job.videoUrl;
      subtitlePath = job.subtitlePath || "";
      segments = [];
      activeIndex = -1;
      if (job.subtitleUrl) {
        const data = await api(job.subtitleUrl);
        segments = normalizeSegments(data.segments || []);
      }
      renderCaption(0);
    }

    function formatTime(seconds) {
      seconds = Math.max(0, Math.floor(seconds || 0));
      const m = String(Math.floor(seconds / 60)).padStart(2, "0");
      const s = String(seconds % 60).padStart(2, "0");
      return m + ":" + s;
    }

    function findSegmentIndex(time) {
      if (!segments.length) return -1;
      let left = 0, right = segments.length - 1;
      while (left <= right) {
        const mid = Math.floor((left + right) / 2);
        const seg = segments[mid];
        if (time < seg.start) right = mid - 1;
        else if (time >= seg.end) left = mid + 1;
        else return mid;
      }
      return Math.max(0, Math.min(segments.length - 1, left));
    }

    function normalizeSegments(input) {
      const out = [];
      let lastEnd = 0;
      for (const item of input) {
        const zh = (item.zh || "").trim();
        if (/^[\\d:：,，.。]{1,2}$/.test(zh)) continue;
        const start = Math.max(Number(item.start) || 0, lastEnd);
        const end = Math.max(Number(item.end) || 0, start + 0.55);
        out.push({
          index: out.length + 1,
          start,
          end,
          zh,
          en: item.en || "",
          pinyin: item.pinyin || ""
        });
        lastEnd = end;
      }
      return out;
    }

    function renderCaption(index) {
      persistCurrentEdits();
      if (!segments.length) {
        captionIndex.textContent = "0 / 0";
        captionTime.textContent = "00:00 - 00:00";
        captionEn.value = "";
        captionEn.placeholder = "等待字幕数据";
        captionZh.value = "";
        captionPinyin.textContent = "";
        return;
      }
      index = Math.max(0, Math.min(segments.length - 1, index));
      const seg = segments[index];
      activeIndex = index;
      captionIndex.textContent = (index + 1) + " / " + segments.length;
      captionTime.textContent = formatTime(seg.start) + " - " + formatTime(seg.end);
      captionEn.value = seg.en || "";
      captionZh.value = seg.zh || "";
      captionPinyin.textContent = seg.pinyin || "确认中文字幕后生成拼音";
    }

    video.addEventListener("timeupdate", () => {
      const index = findSegmentIndex(video.currentTime);
      if (index !== activeIndex) renderCaption(index);
    });
    video.addEventListener("play", () => play.textContent = "Ⅱ");
    video.addEventListener("pause", () => play.textContent = "▶");

    play.onclick = () => video.paused ? video.play() : video.pause();
    document.querySelector("#prev").onclick = () => seekLine(activeIndex - 1);
    document.querySelector("#next").onclick = () => seekLine(activeIndex + 1);
    document.querySelector("#replay").onclick = () => seekLine(activeIndex);
    captionZh.addEventListener("input", () => markDirty("中文字幕已修改，确认后会更新拼音"));
    captionEn.addEventListener("input", () => markDirty("英文字幕已修改"));
    confirmZh.onclick = async () => {
      if (activeIndex < 0 || !segments.length) return;
      persistCurrentEdits();
      confirmZh.disabled = true;
      saveState.textContent = "正在生成拼音...";
      try {
        const resp = await api("/api/pinyin", {
          method: "POST",
          headers: {"Content-Type": "application/json"},
          body: JSON.stringify({ text: segments[activeIndex].zh || "" })
        });
        segments[activeIndex].zh = resp.text || segments[activeIndex].zh || "";
        segments[activeIndex].pinyin = resp.pinyin || "";
        captionZh.value = segments[activeIndex].zh;
        captionPinyin.textContent = segments[activeIndex].pinyin || "无拼音内容";
        markDirty("拼音已更新，记得保存");
      } catch (e) {
        saveState.textContent = e.message.trim();
      } finally {
        confirmZh.disabled = false;
      }
    };
    saveSubtitles.onclick = async () => {
      persistCurrentEdits();
      if (!subtitlePath) {
        saveState.textContent = "没有找到字幕文件路径";
        return;
      }
      saveSubtitles.disabled = true;
      saveState.textContent = "正在保存...";
      try {
        await api("/api/subtitles/save", {
          method: "POST",
          headers: {"Content-Type": "application/json"},
          body: JSON.stringify({ path: subtitlePath, segments })
        });
        dirty = false;
        saveState.textContent = "已保存到字幕文件";
      } catch (e) {
        saveState.textContent = e.message.trim();
      } finally {
        saveSubtitles.disabled = false;
      }
    };

    function markDirty(message) {
      dirty = true;
      saveState.textContent = message || "有未保存修改";
      if (activeIndex >= 0) persistCurrentEdits();
    }

    function persistCurrentEdits() {
      if (activeIndex < 0 || !segments.length) return;
      segments[activeIndex].zh = captionZh.value;
      segments[activeIndex].en = captionEn.value;
    }

    function seekLine(index) {
      if (!segments.length) return;
      persistCurrentEdits();
      index = Math.max(0, Math.min(segments.length - 1, index < 0 ? 0 : index));
      video.currentTime = Math.max(0, segments[index].start + 0.01);
      renderCaption(index);
    }
  </script>
</body>
</html>`
