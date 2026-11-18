# KVM Clipboard Sync (Go)

[![Go Version](https://img.shields.io/badge/Go-1.18%2B-00ADD8?logo=go)](https://golang.org)
[![Platform](https://img.shields.io/badge/Platform-Windows-0078D6?logo=windows)](https://microsoft.com/windows)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE.txt)

> **Instant, zero-network clipboard sharing between computers sharing a USB drive via hardware KVM switch.**

A lightweight Windows system-tray utility written in Go that automatically synchronizes the clipboard across PCs using a shared USB flash drive. Perfect for home-office setups, dual-PC streaming, and switching between Work and Personal laptops without internet, cloud, or LAN dependency.

---

## 🚀 The Problem
When using a hardware KVM switch to toggle between two computers (e.g., Work Laptop and Personal PC), you lose the ability to copy-paste text between them:
* Hardware KVM switches redirect USB peripherals physically, but possess no internal memory to transfer clipboard data.
* Network-based clipboard tools (Barrier, Synergy, KDE Connect) often fail in corporate setups due to strict VPN policies, isolated subnets, or company firewall restrictions.

## 💡 The Solution
**KVM Clipboard Sync** turns any standard USB flash drive plugged into your KVM hub into a private, high-speed clipboard relay:

```text
  [ Computer A ]                          [ Computer B ]
   (Work Laptop)                          (Personal PC)
         │                                      │
   [ Ctrl + C ]                                 │
         │                                      │
         ▼                                      ▼
    writes text                            reads text
         │                                      │
         └────────►  [ KVM SWITCH ]  ◄──────────┘
                            │
               [ Shared USB Flash Drive ]
                 (clipboard_sync.txt)
```

1. When you copy text on Computer A, it instantly writes to a sync file on the shared USB flash drive.
2. When you toggle your KVM switch to Computer B, the tool detects the USB drive, reads the sync file, and updates Computer B's clipboard automatically.

**No LAN, no Wi-Fi, no cloud services, and zero configuration required.**

---

## ✨ Features
*   **Zero Dependencies:** Portable single `.exe` binary. No installers, runtimes, or background services required.
*   **System Tray Integration:** Runs quietly in the Windows notification area with negligible memory (~3-5 MB) and 0% CPU at idle.
*   **One-Click Windows Auto-Start:** Toggle `[x] Run on Windows Startup` directly from the tray menu (stored in user registry `HKCU\...\Run`).
*   **Smart Filtered Drive Menu:** The tray menu displays mounted, valid drive letters on your machine, with a manual `Rescan Drives` option on demand.
*   **Reliable Auto-Save:** Remembers your selected drive in `settings.txt` adjacent to the executable — seamlessly compatible with Windows Startup and Task Scheduler.
*   **Flash Storage Protection:** Built-in 2 MB safety cap prevents USB controller wear and system lockups when copying massive database dumps or log files.
*   **Single-Instance Lock:** Built-in loopback socket lock prevents duplicate processes.
*   **Bi-directional Smart Sync:** Sync triggers only when clipboard or file content genuinely changes, avoiding unnecessary disk writes.

---

## 🛠️ Quick Start

1. **Download** the latest `sync.exe` from [Releases](https://github.com/alexanderisaev1981-droid/kvm-clipboard-sync/releases) (or build from source).
2. **Plug in** your shared USB flash drive into the KVM switch USB hub.
3. **Run** `sync.exe` on **both** computers.
4. **Right-click** the tray icon and select your shared USB drive letter (e.g., `E:` or `Z:`).
5. **Enable Auto-Start:** check `Запускать при старте Windows` in the tray menu so it runs automatically every time.

---

## 🏗️ Building from Source

Requirements:
*   Go 1.18+
*   GCC (MinGW-w64 or TDM-GCC) for Windows system tray CGO bindings.

```bash
# Clone the repository
git clone https://github.com/alexanderisaev1981-droid/kvm-clipboard-sync.git
cd kvm-clipboard-sync

# Build optimized binary (hides console window, strips debug symbols)
go build -ldflags "-H=windowsgui -s -w" -o sync.exe main.go
```

---

## 📄 License
Distributed under the MIT License. See `LICENSE.txt` for details.
