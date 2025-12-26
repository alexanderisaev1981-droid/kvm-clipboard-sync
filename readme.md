# KVM Clipboard Sync (Go)

A lightweight, system-tray utility written in Go (Golang) to synchronize the clipboard between two computers sharing a USB drive (typically via a KVM switch).

## 🚀 The Problem
If you use a hardware KVM switch to toggle between two computers (e.g., Work Laptop and Home PC), you lose the ability to copy-paste text between them. KVMs switch USB devices physically, but they don't have internal memory for clipboard sharing.

## 💡 The Solution
This tool runs in the background on both computers.
1. When you copy text, it saves it to a hidden file on your shared USB flash drive.
2. When you switch the KVM to the other computer, the tool detects the USB drive, reads the file, and puts the text into the clipboard.

**It works automatically and instantly.**

## ✨ Features
*   **Zero Dependencies:** Single `.exe` file. No installation required.
*   **Tray Icon:** Runs silently in the system tray.
*   **Drive Selection:** Easily select your USB drive letter from the tray menu.
*   **Smart Sync:** Only updates when the clipboard content actually changes.
*   **Auto-Save:** Remembers your selected drive letter for the next reboot.
*   **Lightweight:** Consumes minimal RAM (~3-5MB) and 0% CPU when idle.

## 🛠️ Installation & Usage

1. **Download** the latest release (or build it yourself).
2. **Plug in** your shared USB drive (via KVM).
3. **Run** `sync.exe` on **both** computers.
4. **Right-click** the tray icon and select your USB drive letter (e.g., `Z:`).
5. Add `sync.exe` to your Windows Startup folder to run it automatically.

## 🏗️ Building from Source

Requirements:
*   Go 1.16+
*   GCC (MinGW or TDM-GCC) for Windows system tray support.

```bash
# 1. Clone the repository
git clone https://github.com/YOUR_USERNAME/kvm-clipboard-sync.git

# 2. Go to directory
cd kvm-clipboard-sync

# 3. Build (Hide console window)
go build -ldflags -H=windowsgui -o sync.exe main.go