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
)

// --- ГЛОБАЛЬНЫЕ НАСТРОЙКИ ---

const (
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
)

//go:embed icon.ico
var iconFs embed.FS

// getConfigFilePath возвращает надежный путь к файлу конфигурации рядом с exe
func getConfigFilePath() string {
	return filepath.Join(appDir, configFileName)
}

func initAppDir() {
	if exePath, err := os.Executable(); err == nil {
		appDir = filepath.Dir(exePath)
	}
}

// isDriveAvailable проверяет, существует ли корень диска в системе
func isDriveAvailable(driveLetter string) bool {
	root := driveLetter + `\`
	_, err := os.Stat(root)
	return err == nil
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
	// Читаем файл settings.txt рядом с исполняемым файлом
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
	// Скрываем те, которых сейчас нет в системе
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

	// Пункт меню для ручного принудительного сканирования
	mRefreshDrives := systray.AddMenuItem("Обновить список дисков", "Пересканировать подключенные диски и флешки")
	go func() {
		for range mRefreshDrives.ClickedCh {
			updateDriveMenuVisibility()
		}
	}()

	// Фоновый таймер: автоматически сканирует диски каждые 3 секунды
	// чтобы при вставке флешки она сразу появлялась в меню
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
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
