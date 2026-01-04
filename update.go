package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

// 全局变量存储下载地址，避免 DoUpdate 时再次请求 API 导致速率限制或网络错误
var pendingUpdateURL string

// UpdateInfo 返回给前端的结构体
type UpdateInfo struct {
	HasUpdate   bool   `json:"has_update"`
	LatestVer   string `json:"latest_ver"`
	CurrentVer  string `json:"current_ver"`
	ReleaseNote string `json:"release_note"`
	DownloadURL string `json:"download_url"`
	Error       string `json:"error,omitempty"`
}

// GithubRelease 简化的 GitHub Release 结构
type GithubRelease struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	Assets  []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
		Name               string `json:"name"`
	} `json:"assets"`
}

// fetchLatestRelease 获取最新版本信息
func fetchLatestRelease(repo string) (*GithubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 必须设置 User-Agent，否则 GitHub API 会拒绝
	req.Header.Set("User-Agent", "vpk-manager-updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// CheckUpdate 检查更新
func (a *App) CheckUpdate() UpdateInfo {
	// 1. 解析当前版本
	vCurrent, err := semver.ParseTolerant(AppVersion)
	if err != nil {
		return UpdateInfo{Error: "当前版本号格式错误: " + err.Error()}
	}

	// 2. 检测 GitHub 最新版本
	release, err := fetchLatestRelease(GithubRepo)
	if err != nil {
		return UpdateInfo{Error: "检查更新失败: " + err.Error()}
	}

	// 3. 解析最新版本号
	vLatest, err := semver.ParseTolerant(release.TagName)
	if err != nil {
		return UpdateInfo{Error: "最新版本号格式错误: " + err.Error()}
	}

	// 4. 比较版本
	if vLatest.GT(vCurrent) {
		// 预先解析下载地址
		var url string
		suffix := fmt.Sprintf("%s_%s.zip", runtime.GOOS, runtime.GOARCH)

		// 优先匹配精确架构
		for _, asset := range release.Assets {
			if strings.HasSuffix(asset.Name, suffix) {
				url = asset.BrowserDownloadURL
				break
			}
		}

		// 降级匹配任意 zip
		if url == "" {
			for _, asset := range release.Assets {
				if strings.HasSuffix(asset.Name, ".zip") {
					url = asset.BrowserDownloadURL
					break
				}
			}
		}

		// 存入全局变量
		pendingUpdateURL = url

		return UpdateInfo{
			HasUpdate:   true,
			LatestVer:   vLatest.String(),
			CurrentVer:  AppVersion,
			ReleaseNote: release.Body,
			DownloadURL: url,
		}
	}

	return UpdateInfo{
		HasUpdate:  false,
		CurrentVer: AppVersion,
		LatestVer:  vLatest.String(),
	}
}

// DoUpdate 执行更新
func (a *App) DoUpdate() string {
	downloadURL := pendingUpdateURL

	// 如果缓存为空，尝试重新获取
	if downloadURL == "" {
		release, err := fetchLatestRelease(GithubRepo)
		if err != nil {
			return "更新检测失败: " + err.Error()
		}

		suffix := fmt.Sprintf("%s_%s.zip", runtime.GOOS, runtime.GOARCH)
		for _, asset := range release.Assets {
			if strings.HasSuffix(asset.Name, suffix) {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}

		if downloadURL == "" {
			for _, asset := range release.Assets {
				if strings.HasSuffix(asset.Name, ".zip") {
					downloadURL = asset.BrowserDownloadURL
					break
				}
			}
		}
	}

	if downloadURL == "" {
		return "未找到适合当前系统的更新包"
	}

	// 获取当前执行文件路径
	exe, err := os.Executable()
	if err != nil {
		return "无法获取程序路径"
	}

	// 执行更新
	if err := selfupdate.UpdateTo(downloadURL, exe); err != nil {
		return "更新安装失败: " + err.Error()
	}

	return "success"
}
