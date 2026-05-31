package cmd

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func updateCmd() *cobra.Command {
	var dryRun bool
	var checkOnly bool
	var repo string
	var tag string
	c := &cobra.Command{
		Use:   "update",
		Short: "Upgrade dbbackup233 from the latest GitHub Release",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd.Context(), cmd, repo, tag, dryRun, checkOnly)
		},
	}
	c.Flags().BoolVar(&checkOnly, "check", false, "check the selected release without replacing the binary")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print update plan without replacing the binary")
	c.Flags().StringVar(&repo, "repo", "neko233-com/dbbackup233", "GitHub owner/repo")
	c.Flags().StringVar(&tag, "tag", "", "release tag to install; when empty, latest release is queried")
	return c
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func runUpdate(ctx context.Context, cmd *cobra.Command, repo, tag string, dryRun, checkOnly bool) error {
	if tag != "" {
		asset := releaseAssetName()
		url := "https://github.com/" + repo + "/releases/download/" + tag + "/" + asset
		current, err := os.Executable()
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "tag: %s\nasset: %s\nurl: %s\nreplace: %s\nmode: hot-swap-next-run\n", tag, asset, url, current)
		if dryRun || checkOnly {
			return nil
		}
		tmp, err := downloadAsset(ctx, url)
		if err != nil {
			return err
		}
		defer os.Remove(tmp)
		return replaceExecutable(tmp, current)
	}

	url := "https://api.github.com/repos/" + repo + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub release lookup failed: %s; retry with --tag vX.Y.Z to avoid the GitHub API", resp.Status)
	}
	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return err
	}
	assetURL, assetName, err := selectReleaseAsset(release)
	if err != nil {
		return err
	}
	current, err := os.Executable()
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "latest: %s\nasset: %s\nreplace: %s\nmode: hot-swap-next-run\n", release.TagName, assetName, current)
	if dryRun || checkOnly {
		return nil
	}
	tmp, err := downloadAsset(ctx, assetURL)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	return replaceExecutable(tmp, current)
}

func selectReleaseAsset(release githubRelease) (string, string, error) {
	needle := releaseAssetName()
	for _, asset := range release.Assets {
		if asset.Name == needle {
			return asset.BrowserDownloadURL, asset.Name, nil
		}
	}
	return "", "", fmt.Errorf("no release asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
}

func releaseAssetName() string {
	asset := "dbbackup233-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		asset += ".exe"
	}
	return asset
}

func downloadAsset(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}
	tmp, err := os.CreateTemp("", "dbbackup233-update-*")
	if err != nil {
		return "", err
	}
	defer tmp.Close()
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return "", err
	}
	return tmp.Name(), nil
}

func replaceExecutable(src, dst string) error {
	if runtime.GOOS == "windows" {
		next := dst + ".new"
		if err := copyPath(src, next); err != nil {
			return err
		}
		script := fmt.Sprintf("Start-Sleep -Seconds 1; Move-Item -Force %q %q", next, dst)
		return exec.Command("powershell", "-NoProfile", "-Command", script).Start()
	}
	if err := copyPath(src, dst); err != nil {
		return err
	}
	return os.Chmod(dst, 0o755)
}

func copyPath(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func unzipSingleBinary(zipPath, name string) (string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	for _, file := range reader.File {
		if strings.EqualFold(filepath.Base(file.Name), name) {
			rc, err := file.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			tmp, err := os.CreateTemp("", "dbbackup233-bin-*")
			if err != nil {
				return "", err
			}
			defer tmp.Close()
			if _, err := io.Copy(tmp, rc); err != nil {
				return "", err
			}
			return tmp.Name(), nil
		}
	}
	return "", fmt.Errorf("binary %q not found in zip", name)
}
