package main

import (
	"embed"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/getlantern/systray"
	"golang.org/x/sys/windows/registry"
)

// --- ГЛОБАЛЬНЫЕ НАСТРОЙКИ ---

const (
	appName        = "KVMClipboardSync"
	lockPort       = "127.0.0.1:54321"
	syncFileName   = "clipboard_sync.txt"
	configFileName = "settings.txt"
	maxSyncBytes   = 2 * 1024 * 1024 // 2 МБ максимальный размер для синхронизации
)

// Переменные состояния
var (
	currentDrive = "Z:" // Значение по умолчанию
	driveMutex   sync.RWMutex
	appDir       = "."
	exePath      = ""
)

//go:embed icon.ico
var iconFs embed.FS

// getConfigFilePath возвращает надежный путь к файлу конфигурации рядом с exe
func getConfigFilePath() string {
	return filepath.Join(appDir, configFileName)
}

func initAppDir() {
	if p, err := os.Executable(); err == nil {
		exePath = p
		appDir = filepath.Dir(p)
	}
}

// isDriveAvailable проверяет, существует ли корень диска в системе
func isDriveAvailable(driveLetter string) bool {
	root := driveLetter + `\`
	_, err := os.Stat(root)
	return err == nil
}

// --- УПРАВЛЕНИЕ АВТОЗАПУСКОМ ЧЕРЕЗ РЕЕСТР WINDOWS ---

func isAutoStartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	val, _, err := k.GetStringValue(appName)
	return err == nil && val != ""
}

func setAutoStart(enable bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if enable {
		if exePath == "" {
			initAppDir()
		}
		return k.SetStringValue(appName, `"`+exePath+`"`)
	}
	return k.DeleteValue(appName)
}

func main() {
	initAppDir()

	// 1. ЗАЩИТА ОТ ПОВТОРНОГО ЗАПУСКА
	listener, err := net.Listen("tcp", lockPort)
	if err != nil {
		return
	}
	defer listener.Close()

	// 2. ЗАГРУЗКА НАСТРОЕК
	if data, err := os.ReadFile(getConfigFilePath()); err == nil {
		savedLetter := strings.TrimSpace(string(data))
		if len(savedLetter) == 2 && savedLetter[1] == ':' {
			currentDrive = strings.ToUpper(savedLetter)
		}
	}

	// 3. ЗАПУСК ТРЕЯ
	systray.Run(onReady, onExit)
}

func onReady() {
	iconData, _ := iconFs.ReadFile("icon.ico")
	systray.SetIcon(iconData)
	systray.SetTitle("KVM Clipboard Sync")
	systray.SetTooltip("Синхронизация буфера")

	// --- МЕНЮ ВЫБОРА ДИСКА ---
	mDrives := systray.AddMenuItem("Выбрать диск", "Текущий диск: "+currentDrive)

	driveItems := make(map[string]*systray.MenuItem)
	selectCh := make(chan string, 16)

	// Создаем элементы меню для всех букв от C: до Z:
	for i := 'C'; i <= 'Z'; i++ {
		letter := string(i) + ":"
		item := mDrives.AddSubMenuItem(letter, "Использовать диск "+letter)
		driveItems[letter] = item

		if letter == currentDrive {
			item.Check()
		}

		if !isDriveAvailable(letter) && letter != currentDrive {
			item.Hide()
		}

		localLetter := letter
		go func() {
			for range item.ClickedCh {
				selectCh <- localLetter
			}
		}()
	}

	// Функция обновления видимости дисков в меню
	updateDriveMenuVisibility := func() {
		driveMutex.RLock()
		active := currentDrive
		driveMutex.RUnlock()

		for letter, item := range driveItems {
			if isDriveAvailable(letter) || letter == active {
				item.Show()
			} else {
				item.Hide()
			}
		}
	}

	// Пункт меню для ручного пересканирования дисков
	mRefreshDrives := systray.AddMenuItem("Обновить список дисков", "Пересканировать подключенные диски и флешки")
	go func() {
		for range mRefreshDrives.ClickedCh {
			updateDriveMenuVisibility()
		}
	}()

	// Единый диспетчер смены диска
	go func() {
		for chosenLetter := range selectCh {
			driveMutex.Lock()
			currentDrive = chosenLetter
			driveMutex.Unlock()

			// Сохраняем в файл настроек рядом с бинарником
			_ = os.WriteFile(getConfigFilePath(), []byte(chosenLetter), 0644)

			for l, menuItem := range driveItems {
				if l == chosenLetter {
					menuItem.Check()
				} else {
					menuItem.Uncheck()
				}
			}

			mDrives.SetTooltip("Текущий диск: " + chosenLetter)
		}
	}()

	systray.AddSeparator()

	// --- ПУНКТ АВТОЗАПУСКА С WINDOWS ---
	autoStartEnabled := isAutoStartEnabled()
	mAutoStart := systray.AddMenuItemCheckbox("Запускать при старте Windows", "Включить или отключить автозапуск программы", autoStartEnabled)

	go func() {
		for range mAutoStart.ClickedCh {
			if mAutoStart.Checked() {
				if err := setAutoStart(false); err == nil {
					mAutoStart.Uncheck()
				}
			} else {
				if err := setAutoStart(true); err == nil {
					mAutoStart.Check()
				}
			}
		}
	}()

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Выход", "Закрыть программу")
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()

	go runSyncLoop()
}

func onExit() {
	// Очистка при выходе
}

func getFullSyncPath() string {
	driveMutex.RLock()
	defer driveMutex.RUnlock()
	return filepath.Join(currentDrive+`\`, syncFileName)
}

func runSyncLoop() {
	lastClipboardContent := ""
	lastFileContent := ""

	for {
		time.Sleep(1 * time.Second)

		currentPath := getFullSyncPath()

		// 1. Проверка доступности диска/папки
		dir := filepath.Dir(currentPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		// 2. Читаем буфер
		currClip, err := clipboard.ReadAll()
		if err != nil {
			continue
		}

		// 3. Читаем файл, если он существует
		var currFile string
		if info, err := os.Stat(currentPath); err == nil {
			// Защита от слишком больших файлов на флешке
			if info.Size() <= maxSyncBytes {
				if fileBytes, err := os.ReadFile(currentPath); err == nil {
					currFile = string(fileBytes)
				}
			}
		}

		// А: Флешка -> Буфер
		if currFile != lastFileContent && currFile != "" && currFile != currClip {
			clipboard.WriteAll(currFile)
			lastFileContent = currFile
			lastClipboardContent = currFile
			continue
		}

		// Б: Буфер -> Флешка
		if currClip != lastClipboardContent && currClip != "" {
			// Проверка размера текста в буфере обмена (до 2 МБ)
			if len([]byte(currClip)) <= maxSyncBytes && currClip != currFile {
				if err := os.WriteFile(currentPath, []byte(currClip), 0644); err == nil {
					lastFileContent = currClip
				}
			}
			lastClipboardContent = currClip
		}
	}
}
