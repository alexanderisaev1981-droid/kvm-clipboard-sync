package main

import (
	"embed"
	"io/ioutil"
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

const lockPort = "127.0.0.1:54321"
const syncFileName = "clipboard_sync.txt"
const configFileName = "settings.txt" // Имя файла, где храним букву

// Переменные состояния
var (
	currentDrive = "Z:" // Значение по умолчанию
	driveMutex   sync.RWMutex
)

//go:embed icon.ico
var iconFs embed.FS

func main() {
	// 1. ЗАЩИТА ОТ ПОВТОРНОГО ЗАПУСКА
	listener, err := net.Listen("tcp", lockPort)
	if err != nil {
		return
	}
	defer listener.Close()

	// 2. ЗАГРУЗКА НАСТРОЕК
	// Пытаемся прочитать файл settings.txt рядом с exe
	// Если файла нет или ошибка — останется Z:
	if data, err := ioutil.ReadFile(configFileName); err == nil {
		// TrimSpace убирает лишние пробелы и переносы строк, если они там есть
		savedLetter := strings.TrimSpace(string(data))
		// Простая проверка, что там похоже на букву диска (длина 2, например "F:")
		if len(savedLetter) == 2 {
			currentDrive = savedLetter
		}
	}

	// 3. ЗАПУСК ТРЕЯ
	systray.Run(onReady, onExit)
}

func onReady() {
	// Иконка и описание
	iconData, _ := iconFs.ReadFile("icon.ico")
	systray.SetIcon(iconData)
	systray.SetTitle("KVM Clipboard Sync")
	systray.SetTooltip("Синхронизация буфера")

	// --- МЕНЮ ВЫБОРА ДИСКА ---
	mDrives := systray.AddMenuItem("Выбрать диск", "Текущий диск: "+currentDrive)

	driveItems := make(map[string]*systray.MenuItem)

	// Генерируем пункты от C: до Z:
	for i := 'C'; i <= 'Z'; i++ {
		letter := string(i) + ":"
		item := mDrives.AddSubMenuItem(letter, "Использовать диск "+letter)
		driveItems[letter] = item

		// Ставим галочку, если этот диск был загружен из настроек
		if letter == currentDrive {
			item.Check()
		}

		localLetter := letter

		// Обработчик клика
		go func() {
			for {
				<-item.ClickedCh // Ждем клика

				// 1. Обновляем в памяти
				driveMutex.Lock()
				currentDrive = localLetter
				driveMutex.Unlock()

				// 2. СОХРАНЯЕМ В ФАЙЛ
				// Игнорируем ошибки записи, так как это не критично
				_ = ioutil.WriteFile(configFileName, []byte(localLetter), 0644)

				// 3. Обновляем галочки в меню
				for l, menuItem := range driveItems {
					if l == localLetter {
						menuItem.Check()
					} else {
						menuItem.Uncheck()
					}
				}

				// Обновляем подсказку главного пункта
				mDrives.SetTooltip("Текущий диск: " + localLetter)
			}
		}()
	}

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
	return filepath.Join(currentDrive+"\\", syncFileName)
}

func runSyncLoop() {
	lastClipboardContent := ""
	lastFileContent := ""

	for {
		time.Sleep(1 * time.Second)

		currentPath := getFullSyncPath()

		// 1. Проверка доступности
		if _, err := os.Stat(currentPath); os.IsNotExist(err) {
			dir := filepath.Dir(currentPath)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				continue
			}
		}

		// 2. Читаем буфер
		currClip, err := clipboard.ReadAll()
		if err != nil {
			continue
		}

		// 3. Читаем файл
		fileBytes, err := ioutil.ReadFile(currentPath)
		currFile := string(fileBytes)
		if err != nil {
			currFile = ""
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
			if currClip != currFile {
				err := ioutil.WriteFile(currentPath, []byte(currClip), 0644)
				if err == nil {
					lastFileContent = currClip
				}
			}
			lastClipboardContent = currClip
		}
	}
}
