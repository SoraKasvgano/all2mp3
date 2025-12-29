package main

import (
	"embed"
	"fmt"
	"image/color"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// 嵌入字体和FFmpeg资源
//
//go:embed simhei.ttf ffmpeg.exe
var embeddedFiles embed.FS

// 支持的文件格式
var supportedFormats = []string{".mp4", ".m4a", ".m4s", ".aac", ".avi", ".flac", ".wav", ".ogg", ".wmv", ".mov", ".mpg", ".mpeg", ".webm", ".mkv"}

// 自定义主题，支持中文
var _ fyne.Theme = (*chineseTheme)(nil)

type chineseTheme struct {
	baseTheme fyne.Theme
	fontPath  string
}

func (t *chineseTheme) Font(style fyne.TextStyle) fyne.Resource {
	// 尝试从嵌入资源加载中文字体
	fontData, err := embeddedFiles.ReadFile("simhei.ttf")
	if err == nil {
		// 创建临时字体文件
		tmpDir := os.TempDir()
		tmpFontPath := filepath.Join(tmpDir, "simhei.ttf")

		// 检查临时文件是否已存在
		if _, err := os.Stat(tmpFontPath); os.IsNotExist(err) {
			// 写入临时文件
			err = ioutil.WriteFile(tmpFontPath, fontData, 0644)
			if err == nil {
				// 从临时文件加载字体资源
				res, err := fyne.LoadResourceFromPath(tmpFontPath)
				if err == nil {
					return res
				}
			}
		} else if err == nil {
			// 临时文件已存在，直接加载
			res, err := fyne.LoadResourceFromPath(tmpFontPath)
			if err == nil {
				return res
			}
		}
	}

	// 如果没有找到中文字体，使用默认字体
	return t.baseTheme.Font(style)
}

func (t *chineseTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return t.baseTheme.Color(name, variant)
}

func (t *chineseTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.baseTheme.Icon(name)
}

func (t *chineseTheme) Size(name fyne.ThemeSizeName) float32 {
	return t.baseTheme.Size(name)
}

// FileInfo 文件信息结构
type FileInfo struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Ext   string `json:"ext"`
	Valid bool   `json:"valid"`
}

// ConversionResult 转换结果结构
type ConversionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

var (
	selectedFiles []string
	mu            sync.Mutex
)

// isSupportedFormat 检查文件格式是否支持
func isSupportedFormat(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, format := range supportedFormats {
		if ext == format {
			return true
		}
	}
	return false
}

// validateFiles 验证文件列表
func validateFiles(files []string) []FileInfo {
	var validFiles []FileInfo
	for _, file := range files {
		info := FileInfo{
			Path: file,
			Name: filepath.Base(file),
			Ext:  strings.ToLower(filepath.Ext(file)),
		}
		info.Valid = isSupportedFormat(file) && fileExists(file)
		validFiles = append(validFiles, info)
	}
	return validFiles
}

// fileExists 检查文件是否存在
func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// findFFmpeg 查找FFmpeg可执行文件
func findFFmpeg() (string, error) {
	// 先检查系统路径
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		return "ffmpeg", nil
	}

	// 检查当前目录
	ffmpegPath := filepath.Join(getCurrentDir(), "ffmpeg.exe")
	if fileExists(ffmpegPath) {
		return ffmpegPath, nil
	}

	// 检查子目录
	for _, subdir := range []string{"ffmpeg", "bin", "tools"} {
		ffmpegPath := filepath.Join(getCurrentDir(), subdir, "ffmpeg.exe")
		if fileExists(ffmpegPath) {
			return ffmpegPath, nil
		}
	}

	// 从嵌入资源提取FFmpeg到临时目录
	tmpDir := os.TempDir()
	tmpFFmpegPath := filepath.Join(tmpDir, "ffmpeg.exe")

	// 检查临时文件是否已存在
	if _, err := os.Stat(tmpFFmpegPath); os.IsNotExist(err) {
		// 从嵌入资源读取FFmpeg数据
		ffmpegData, err := embeddedFiles.ReadFile("ffmpeg.exe")
		if err != nil {
			return "", fmt.Errorf("无法读取嵌入的FFmpeg资源: %v", err)
		}

		// 写入临时文件
		err = ioutil.WriteFile(tmpFFmpegPath, ffmpegData, 0755) // 设置可执行权限
		if err != nil {
			return "", fmt.Errorf("无法写入临时FFmpeg文件: %v", err)
		}
	}

	return tmpFFmpegPath, nil
}

// getCurrentDir 获取当前目录
func getCurrentDir() string {
	// 获取当前执行文件的目录
	ex, _ := os.Executable()
	return filepath.Dir(ex)
}

// convertToMp3 转换单个文件为MP3
func convertToMp3(inputFile, outputFile, ffmpegPath string) error {
	cmdArgs := []string{
		"-y",
		"-i", inputFile,
		"-vn",
		"-acodec", "libmp3lame",
		"-ab", "192k",
		"-ar", "44100",
		"-ac", "2",
		outputFile,
	}

	cmd := exec.Command(ffmpegPath, cmdArgs...)
	return cmd.Run()
}

// convertFiles 转换文件列表
func convertFiles(files []string) []ConversionResult {
	var results []ConversionResult
	ffmpegPath, err := findFFmpeg()
	if err != nil {
		return []ConversionResult{{
			Success: false,
			Message: fmt.Sprintf("错误: %v", err),
		}}
	}

	for _, file := range files {
		if !isSupportedFormat(file) || !fileExists(file) {
			results = append(results, ConversionResult{
				Success: false,
				Message: fmt.Sprintf("文件不支持或不存在: %s", filepath.Base(file)),
			})
			continue
		}

		// 生成输出文件路径（与源文件相同目录）
		outputFile := strings.TrimSuffix(file, filepath.Ext(file)) + ".mp3"

		// 如果输出文件已存在，添加序号
		counter := 1
		for fileExists(outputFile) {
			base := strings.TrimSuffix(file, filepath.Ext(file))
			outputFile = fmt.Sprintf("%s_%d.mp3", base, counter)
			counter++
		}

		if err := convertToMp3(file, outputFile, ffmpegPath); err != nil {
			results = append(results, ConversionResult{
				Success: false,
				Message: fmt.Sprintf("转换失败 %s: %v", filepath.Base(file), err),
			})
		} else {
			results = append(results, ConversionResult{
				Success: true,
				Message: fmt.Sprintf("转换成功: %s", filepath.Base(outputFile)),
			})
		}
	}

	return results
}

func main() {
	// 创建应用
	a := app.New()

	// 设置自定义中文主题
	baseTheme := theme.DefaultTheme()
	a.Settings().SetTheme(&chineseTheme{baseTheme: baseTheme})

	w := a.NewWindow("音频格式转换器")
	w.Resize(fyne.NewSize(800, 900))

	// 状态变量
	var statusLabel *widget.Label
	var fileList *widget.List
	var convertButton *widget.Button
	var progressBar *canvas.Rectangle

	// 创建界面元素
	title := canvas.NewText("🎵 音频格式转换器", nil)
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter

	// 拖拽区域
	dropZone := container.NewVBox(
		canvas.NewText("📁", nil),
		canvas.NewText("拖拽文件到此处", nil),
		canvas.NewText("支持格式: MP4, M4A, M4S, AAC, AVI, FLAC, WAV, OGG, WMV, MOV, MPG, MPEG, WEBM, MKV", nil),
	)
	dropZone.Resize(fyne.NewSize(760, 200))

	// 控制按钮
	selectFileBtn := widget.NewButton("选择文件", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				defer reader.Close()
				filePath := reader.URI().Path()
				if isSupportedFormat(filePath) && fileExists(filePath) {
					mu.Lock()
					selectedFiles = append(selectedFiles, filePath)
					mu.Unlock()
					updateFileList(fileList, selectedFiles)
					convertButton.Enable()
					statusLabel.SetText(fmt.Sprintf("已选择 %d 个文件", len(selectedFiles)))
				}
			}
		}, w)
	})

	selectFolderBtn := widget.NewButton("选择文件夹", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				// 遍历文件夹中的文件
				files, _ := uri.List()
				for _, fileURI := range files {
					filePath := fileURI.Path()
					if isSupportedFormat(filePath) && fileExists(filePath) {
						mu.Lock()
						selectedFiles = append(selectedFiles, filePath)
						mu.Unlock()
					}
				}
				updateFileList(fileList, selectedFiles)
				convertButton.Enable()
				statusLabel.SetText(fmt.Sprintf("已选择 %d 个文件", len(selectedFiles)))
			}
		}, w)
	})

	clearBtn := widget.NewButton("清空列表", func() {
		mu.Lock()
		selectedFiles = make([]string, 0)
		mu.Unlock()
		updateFileList(fileList, selectedFiles)
		convertButton.Disable()
		statusLabel.SetText("等待文件选择...")
	})

	// 文件列表
	fileList = widget.NewList(
		func() int {
			return len(selectedFiles)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("文件路径")
		},
		func(id widget.ListItemID, object fyne.CanvasObject) {
			if id < len(selectedFiles) {
				file := selectedFiles[id]
				label := object.(*widget.Label)
				label.SetText(filepath.Base(file))

				// 设置颜色表示支持状态
				if isSupportedFormat(file) && fileExists(file) {
					label.Importance = widget.HighImportance
				} else {
					label.Importance = widget.MediumImportance
				}
			}
		},
	)
	fileList.Resize(fyne.NewSize(760, 600))

	// 状态标签
	statusLabel = widget.NewLabel("等待文件选择...")

	// 进度条
	progressBar = canvas.NewRectangle(&color.RGBA{R: 0, G: 0, B: 0, A: 0})
	progressBar.Resize(fyne.NewSize(760, 10))

	// 转换按钮
	convertButton = widget.NewButton("🚀 开始转换", func() {
		if len(selectedFiles) == 0 {
			dialog.ShowInformation("警告", "请先选择要转换的文件", w)
			return
		}

		// 检查FFmpeg
		_, err := findFFmpeg()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		convertButton.Disable()
		statusLabel.SetText("正在转换...")

		// 直接在主线程执行转换
		results := convertFiles(selectedFiles)

		// 更新UI
		successCount := 0
		for _, result := range results {
			if result.Success {
				successCount++
			}
		}

		// 收集结果消息
		message := ""
		for _, result := range results {
			message += result.Message + "\n"
		}

		statusLabel.SetText(fmt.Sprintf("转换完成！成功: %d/%d", successCount, len(selectedFiles)))
		convertButton.Enable()

		// 显示结果
		dialog.ShowInformation("转换结果", message, w)
	})
	convertButton.Disable()

	// 布局
	controls := container.NewHBox(
		selectFileBtn,
		selectFolderBtn,
		clearBtn,
		layout.NewSpacer(),
		convertButton,
	)

	content := container.NewVBox(
		title,
		dropZone,
		controls,
		widget.NewLabel("已选择的文件:"),
		fileList,
		widget.NewLabel("进度:"),
		progressBar,
		statusLabel,
	)

	w.SetContent(container.NewScroll(content))
	w.ShowAndRun()
}

// updateFileList 更新文件列表显示
func updateFileList(list *widget.List, files []string) {
	list.Refresh()
}
