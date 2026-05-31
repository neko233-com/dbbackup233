package cmd

import (
	"runtime"
	"strings"
	"testing"
)

func TestSelectReleaseAsset(t *testing.T) {
	want := releaseAssetName()
	release := githubRelease{TagName: "v1.2.3"}
	release.Assets = append(release.Assets, struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	}{Name: "other", BrowserDownloadURL: "https://example.invalid/other"})
	release.Assets = append(release.Assets, struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	}{Name: want, BrowserDownloadURL: "https://example.invalid/" + want})

	url, name, err := selectReleaseAsset(release)
	if err != nil {
		t.Fatal(err)
	}
	if name != want || url == "" {
		t.Fatalf("unexpected asset name=%s url=%s", name, url)
	}
}

func TestReleaseAssetNameIncludesPlatform(t *testing.T) {
	got := releaseAssetName()
	if !strings.Contains(got, runtime.GOOS) || !strings.Contains(got, runtime.GOARCH) {
		t.Fatalf("asset %q does not include platform", got)
	}
}
