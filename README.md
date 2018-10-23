# TinfoilUSBGo
TinfoilUSBGo is a Golang rewrite of the Python Tinfoil USB installer script: 
[https://github.com/XorTroll/Tinfoil/blob/master/tools/usb_install_pc.py](https://github.com/XorTroll/Tinfoil/blob/master/tools/usb_install_pc.py).

Tinfoil is a Nintendo Switch homebrew application used to manage titles. The
existing Python 3 script was written to interface with Tinfoil's USB
installation. This rewrite intends to eliminate the Python 3 dependency for
those who do not wish to install it (mainly for Windows users).

# Setup
### Windows
To interface with the Nintendo Switch, the libusbK driver is required.

1. Download Zadig: [https://zadig.akeo.ie/](https://zadig.akeo.ie/).
2. With your Nintendo Switch plugged in and on the Tinfoil USB install menu, choose
"List All Devices" under the options menu in Zadig, and select libnx USB comms.
3. Choose libusbK from the driver list and click the "Replace Driver" button.

### macOS
Under construction.

### Linux
Under construction.

# Usage
### Windows
1. Plug in your Nintendo Switch and navigate to Tinfoil's USB install page.
2. Open `cmd` or `PowerShell` and navigate to the downloaded `exe`.
3. Run the application using `.\tinfoilusbgo-windows-10.0-amd64.exe [nsp_path]`.

# Build
TinfoilUSBGo uses [xgo](https://github.com/karalabe/xgo) to cross-compile to
Windows and Linux. Xgo requires `docker` installed to build and run.

### Windows
```sh
make build-windows
```

### macOS
```sh
make build-osx
```

### Linux
Under construction.