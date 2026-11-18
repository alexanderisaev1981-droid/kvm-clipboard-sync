# KVM Clipboard Sync (Go)

A lightweight, system-tray utility written in Go (Golang) to synchronize the clipboard between two computers sharing a USB drive (typically via a KVM switch).

## 🚀 The Problem
If you use a hardware KVM switch to toggle between two computers (e.g., Work Laptop and Home PC), you lose the ability to copy-paste text between them. KVMs switch USB devices physically, but they don't have internal memory for clipboard sharing.

## 💡 The Solution
This tool runs in the background on both computers.
1. When you copy text, it saves it to a file on your shared USB flash drive.
2. When you switch the KVM to the other computer, the tool detects the USB drive, reads the file, and puts the text into the clipboard.

**It works automatically and instantly.**

## ✨ Features
*   **Zero Dependencies:** Single `.exe` file. No installation required.
*   **Tray Icon:** Runs silently in the Windows system tray.
*   **Filtered Drive Selection:** The tray menu dynamically discovers and displays only active/mounted drives in your system.
*   **Reliable Auto-Save:** Automatically persists your selected drive letter in `settings.txt` located right next to the executable, ensuring proper operation with Windows Startup, Task Scheduler, or shortcuts.
*   **Optimized Resource Usage:** Consumes minimal RAM (~3-5MB), negligible CPU, and uses a centralized channel-based event dispatcher.
*   **Flash Drive & Memory Protection:** Implements a safety limit (2 MB) for synchronized text to prevent high I/O lag and wear on USB flash storage from accidental huge copies (e.g. database dumps).
*   **Single-Instance Lock:** Built-in loopback socket lock prevents accidental duplicate processes from running simultaneously.
*   **Smart Sync:** Bi-directional sync triggers only when the clipboard or file content actually changes.

## 🛠️ Installation & Usage

1. **Download** the latest release (or build it yourself).
2. **Plug in** your shared USB drive (via KVM).
3. **Run** `sync.exe` on **both** computers.
4. **Right-click** the tray icon and select your USB drive letter (e.g., `Z:`).
5. Add `sync.exe` to your Windows Startup folder (`shell:startup`) to run it automatically on boot.

## 🏗️ Building from Source

Requirements:
*   Go 1.18+
*   GCC (MinGW-w64 or TDM-GCC) for Windows system tray support (`cgo`).

```bash
# 1. Clone the repository
git clone https://github.com/alexanderisaev1981-droid/kvm-clipboard-sync.git

# 2. Go to directory
cd kvm-clipboard-sync

# 3. Build (Hide console window)
go build -ldflags "-H=windowsgui -s -w" -o sync.exe main.go
```
