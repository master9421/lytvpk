package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"vpk-manager/parser"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ConflictGroup struct {
	VpkFiles []string `json:"vpk_files"`
	Files    []string `json:"files"`
	Severity string   `json:"severity"` // "critical", "warning", "info"
}

type ConflictResult struct {
	TotalConflicts int             `json:"total_conflicts"`
	ConflictGroups []ConflictGroup `json:"conflict_groups"`
}

// getConflictSeverity 判断文件冲突严重程度
func getConflictSeverity(filePath string) string {
	lower := strings.ToLower(filePath)
	lower = strings.ReplaceAll(lower, "\\", "/")

	// 🔴 严重
	// 完全匹配
	if lower == "particles/particles_manifest.txt" {
		return "critical"
	}
	if lower == "scripts/soundmixers.txt" {
		return "critical"
	}
	// 后缀匹配
	if strings.HasSuffix(lower, ".bsp") || strings.HasSuffix(lower, ".nav") {
		return "critical"
	}
	// 前缀+后缀匹配
	if strings.HasPrefix(lower, "missions/") && strings.HasSuffix(lower, ".txt") {
		return "critical"
	}
	if strings.HasPrefix(lower, "scripts/") && strings.HasSuffix(lower, ".txt") {
		// 特殊情况：vscripts 属于告警
		if strings.HasPrefix(lower, "scripts/vscripts/") {
			return "warning"
		}
		return "critical"
	}

	// 🟡 告警
	if lower == "sound/sound.cache" {
		return "warning"
	}
	if strings.HasSuffix(lower, ".phy") {
		return "warning"
	}
	if strings.HasPrefix(lower, "resource/") && strings.HasSuffix(lower, ".res") {
		return "warning"
	}
	if strings.HasPrefix(lower, "scripts/vscripts/") {
		return "warning"
	}
	if strings.HasSuffix(lower, ".vscript") || strings.HasSuffix(lower, ".nut") || strings.HasSuffix(lower, ".nuc") {
		return "warning"
	}
	if strings.HasSuffix(lower, ".db") {
		return "warning"
	}
	if strings.HasSuffix(lower, ".vtx") || strings.HasSuffix(lower, ".vvd") {
		return "warning"
	}
	if strings.HasSuffix(lower, ".ttf") || strings.HasSuffix(lower, ".otf") {
		return "warning"
	}

	// 🟢 一般 (其他所有文件)
	return "info"
}

// CheckConflicts 检测VPK文件冲突
func (a *App) CheckConflicts() (*ConflictResult, error) {
	if a.rootDir == "" {
		return nil, fmt.Errorf("未选择L4D2目录")
	}

	// a.rootDir 已经是 addons 目录
	addonsDir := a.rootDir
	workshopDir := filepath.Join(addonsDir, "workshop")

	var vpkPaths []string

	// 扫描 addons 目录
	entries, err := os.ReadDir(addonsDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".vpk") {
				vpkPaths = append(vpkPaths, filepath.Join(addonsDir, entry.Name()))
			}
		}
	}

	// 扫描 workshop 目录
	entries, err = os.ReadDir(workshopDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".vpk") {
				vpkPaths = append(vpkPaths, filepath.Join(workshopDir, entry.Name()))
			}
		}
	}

	totalFiles := len(vpkPaths)
	if totalFiles == 0 {
		return &ConflictResult{}, nil
	}

	// 发送开始事件
	runtime.EventsEmit(a.ctx, "conflict_check_progress", ProgressInfo{
		Current: 0,
		Total:   totalFiles,
		Message: "开始扫描冲突...",
	})

	// 文件路径 -> VPK列表
	fileMap := make(map[string][]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 进度计数器
	var processedCount int
	var countMu sync.Mutex

	// 使用协程池并发处理
	for _, path := range vpkPaths {
		wg.Add(1)
		p := path // capture loop variable

		err := a.goroutinePool.Submit(func() {
			defer wg.Done()

			files, err := parser.GetVPKFileList(p)

			countMu.Lock()
			processedCount++
			current := processedCount
			countMu.Unlock()

			// 每5个文件或者最后一个文件发送一次进度，避免事件过多
			if current%5 == 0 || current == totalFiles {
				runtime.EventsEmit(a.ctx, "conflict_check_progress", ProgressInfo{
					Current: current,
					Total:   totalFiles,
					Message: fmt.Sprintf("正在分析: %s", filepath.Base(p)),
				})
			}

			if err != nil {
				return
			}

			// 计算相对路径作为显示名称，以便区分 workshop 和 根目录的文件
			relPath, err := filepath.Rel(a.rootDir, p)
			if err != nil {
				relPath = filepath.Base(p)
			}
			vpkName := filepath.ToSlash(relPath)

			mu.Lock()
			for _, f := range files {
				// 归一化 VPK 内部文件路径，确保跨平台兼容性
				f = strings.ReplaceAll(f, "\\", "/")
				f = strings.TrimSpace(f)
				lowerF := strings.ToLower(f)

				if lowerF == "addoninfo.txt" || lowerF == "" || lowerF == "addonimage.vtf" || lowerF == "addonimage.jpg" {
					continue
				}
				// 忽略开发残留和临时文件
				if strings.HasPrefix(lowerF, "materials/dev/") || strings.HasPrefix(lowerF, "materials/temp/") {
					continue
				}
				fileMap[lowerF] = append(fileMap[lowerF], vpkName)
			}
			mu.Unlock()
		})

		if err != nil {
			wg.Done() // Submit failed
		}
	}

	wg.Wait()

	// 分析冲突
	runtime.EventsEmit(a.ctx, "conflict_check_progress", ProgressInfo{
		Current: totalFiles,
		Total:   totalFiles,
		Message: "正在整理冲突结果...",
	})

	// VPK组合 -> 冲突文件列表
	// key: "vpk1.vpk|vpk2.vpk" (sorted)
	conflictMap := make(map[string][]string)

	for f, vpks := range fileMap {
		if len(vpks) > 1 {
			// 排序以生成唯一key
			sort.Strings(vpks)
			key := strings.Join(vpks, "|")
			conflictMap[key] = append(conflictMap[key], f)
		}
	}

	var groups []ConflictGroup
	for key, files := range conflictMap {
		vpks := strings.Split(key, "|")
		sort.Strings(files) // 文件列表也排序

		// 计算严重程度
		severity := "info"
		for _, f := range files {
			s := getConflictSeverity(f)
			if s == "critical" {
				severity = "critical"
				break // 已经是最高级别，无需继续
			}
			if s == "warning" {
				severity = "warning"
			}
		}

		groups = append(groups, ConflictGroup{
			VpkFiles: vpks,
			Files:    files,
			Severity: severity,
		})
	}

	// 按严重程度和冲突数量排序 groups
	sort.Slice(groups, func(i, j int) bool {
		// 严重程度优先级: critical > warning > info
		severityOrder := map[string]int{"critical": 3, "warning": 2, "info": 1}
		si := severityOrder[groups[i].Severity]
		sj := severityOrder[groups[j].Severity]

		if si != sj {
			return si > sj
		}
		return len(groups[i].Files) > len(groups[j].Files)
	})

	return &ConflictResult{
		TotalConflicts: len(groups),
		ConflictGroups: groups,
	}, nil
}
