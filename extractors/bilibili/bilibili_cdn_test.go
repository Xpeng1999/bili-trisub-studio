package bilibili

import "testing"

func TestDashStreamURLPreference(t *testing.T) {
	stream := dashStream{
		BaseURL: "https://wfm4966c.edge.mountaintoys.cn:4483/upgcxcode/video.m4s?os=mcdn&mcdnid=50051224",
		BackupURL: []string{
			"https://upos-sz-mirrorcos.bilivideo.com/upgcxcode/video.m4s?os=cos",
			"https://upos-sz-mirrorcos.bilivideo.com/upgcxcode/video.m4s?os=cos",
		},
	}

	urls := dashStreamURLs(stream)
	if len(urls) != 2 {
		t.Fatalf("dashStreamURLs() len = %d, want 2", len(urls))
	}
	if urls[0] != stream.BackupURL[0] {
		t.Fatalf("dashStreamURLs()[0] = %q, want backup bilivideo URL", urls[0])
	}
}
