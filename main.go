package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/gousb"
)

const (
	// cmdIDExit is sent from tinfoil when all transfers are finished.
	cmdIDExit = 0
	// cmdIDFileRange is sent from tinfoil when there are additional transfers.
	cmdIDFileRange = 1
	// cmdTypeResponse is the header type.
	cmdTypeResponse = 1
)

// NSPList contains information for the payload to be sent to tinfoil.
type NSPList struct {
	// Paths is an array of paths to all NSPs to install.
	Paths []string
	// Length is the size of the entire payload.
	Length int
}

// buildNSPList populates and returns an NSPList struct built from getNSPListFromDirectory.
func buildNSPList(f []string) (l NSPList) {
	var totalLength int
	for i, path := range f {
		f[i] = path + "\n"
		totalLength += len(f[i])
	}
	l.Paths, l.Length = f, totalLength
	return l
}

// sendNSPPayload creates a payload out of an NSPList struct and sends it to the switch.
func sendNSPList(l NSPList, epOut *gousb.OutEndpoint) {
	epOut.Write([]byte("TUL0")) // Tinfoil USB List 0

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(l.Length)) // NSP list length
	epOut.Write(buf)

	buf = make([]byte, 8)
	copy(buf, strings.Repeat("\x00", 0x8)) // Padding
	epOut.Write(buf)

	fmt.Printf("Sending NSP list: %v", l)

	for _, path := range l.Paths {
		buf = make([]byte, len(path))
		copy(buf, path) // File path followed by newline
		epOut.Write(buf)
	}
}

// sendNSPFiles handles sending files to the switch.
func sendNSPFiles(l NSPList, epIn *gousb.InEndpoint, epOut *gousb.OutEndpoint, dataSize uint64) {
	buf := make([]byte, 32)
	_, err := epIn.Read(buf)
	fileRangeHeader := buf[:]
	if err != nil {
		log.Fatalf("Reading file range header failed: %v", err)
	}
	rangeSize := binary.LittleEndian.Uint64(fileRangeHeader[:8])
	rangeOffset := binary.LittleEndian.Uint64(fileRangeHeader[8:16])
	nspNameLength := binary.LittleEndian.Uint64(fileRangeHeader[16:24])
	buf = buf[:0] // reset buffer (might be unneeded, left over from resolving previous issues)

	buf = make([]byte, nspNameLength)
	_, err = epIn.Read(buf)
	if err != nil {
		log.Fatalf("Reading NSP name failed: %v", err)
	}
	nspName := string(buf)
	buf = buf[:0] // reset buffer (might be unneeded, left over from resolving previous issues)
	fmt.Printf("Range size: %d, Range offset: %d, Name length: %d, Name: %s\n",
		rangeSize, rangeOffset, nspNameLength, nspName)

	// Response headers
	epOut.Write([]byte("TUC0")) // Tinfoil USB Command 0

	buf = []byte{byte(cmdTypeResponse)}
	epOut.Write(buf)

	buf = make([]byte, 3)
	copy(buf, strings.Repeat("\x00", 0x3))
	epOut.Write(buf)

	buf = make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, cmdIDFileRange)
	epOut.Write(buf)

	buf = make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, rangeSize)
	epOut.Write(buf)

	buf = make([]byte, 12)
	copy(buf, strings.Repeat("\x00", 0xC))
	epOut.Write(buf)

	// Open file
	file, err := os.Open(nspName)
	defer file.Close()
	if err != nil {
		log.Fatalf("Error reading NSP file: %v", err)
	}

	// readSize from https://github.com/XorTroll/Tinfoil/blob/21941dd9219149e2cc598e8a967343abffc6f883/include/data/buffered_placeholder_writer.hpp#L11
	var currOffset, endOffset, readSize int64 = 0, int64(rangeSize), 8388608
	file.Seek(int64(rangeOffset), 0)
	for currOffset < endOffset {
		if currOffset+readSize >= endOffset {
			readSize = endOffset - currOffset
		}
		buf = make([]byte, readSize)
		file.Read(buf)
		epOut.Write(buf)
		currOffset += readSize
	}
}

// sendFiles wraps around sendNSPFiles and keeps the connection open.
func sendNSPFilesPoll(l NSPList, epIn *gousb.InEndpoint, epOut *gousb.OutEndpoint) {
	for {
		buf := make([]byte, 32)
		_, err := epIn.Read(buf)
		if err != nil {
			log.Fatalf("USB transfer failed: %v", err)
		}
		cmdHeader := buf[:]
		magic := cmdHeader[:4] // TUC0 (Tinfoil USB Command 0)
		fmt.Printf("Magic: %s\n", magic)
		if !bytes.Equal(magic, []byte("TUC0")) {
			continue
		}
		cmdType := cmdHeader[4:5]
		cmdID := binary.LittleEndian.Uint32(cmdHeader[8:12])
		dataSize := binary.LittleEndian.Uint64(cmdHeader[12:20])
		buf = buf[:0] // reset buffer (might be unneeded, left over from resolving previous issues)
		fmt.Printf("Cmd type: %d, Command ID: %d, Data size: %d\n", cmdType, cmdID, dataSize)
		if cmdID == cmdIDExit {
			fmt.Println("Finished transfer, exiting")
			os.Exit(0)
		} else if cmdID == cmdIDFileRange {
			sendNSPFiles(l, epIn, epOut, dataSize)
		}
	}
}

// getInOutEndpoints retrieves the in and out endpoints.
func getInOutEndpoints(intf *gousb.Interface) (in *gousb.InEndpoint, out *gousb.OutEndpoint, err error) {
	for _, desc := range intf.Setting.Endpoints {
		if desc.Direction == true {
			in, err = intf.InEndpoint(desc.Number)
		} else {
			out, err = intf.OutEndpoint(desc.Number)
		}
	}
	return in, out, err
}

// getNSPListFromDirectory list NSPs in a directory and its subdirectories.
func getNSPListFromDirectory(d string) []string {
	if _, err := os.Stat(d); os.IsNotExist(err) {
		log.Fatal("NSP directory does not exist")
	}

	var files []string

	err := filepath.Walk(d, func(path string, f os.FileInfo, err error) error {
		if filepath.Ext(path) == ".nsp" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	if len(files) == 0 {
		log.Fatal("No NSPs found, exiting")
	}
	return files
}

func main() {
	// Check args and verify path is valid
	if len(os.Args) > 2 {
		log.Fatalf("too many arguments: %d", len(os.Args))
	}
	dir := os.Args[1]
	dirStat, err := os.Stat(dir)
	if err != nil {
		log.Fatalf("directory does not exist: %s", dir)
	}
	if !dirStat.IsDir() {
		log.Fatalf("supplied path is not a directory: %s", dir)
	}

	// Initialize a new Context for Switch USB device
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Find Switch
	vid, pid := gousb.ID(0x057E), gousb.ID(0x3000) // https://github.com/XorTroll/Tinfoil/blob/21941dd9219149e2cc598e8a967343abffc6f883/source/nx/ipc/usb_comms_new.c#L71
	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == vid && desc.Product == pid
	})

	// All returned devices are now open and will need to be closed
	for _, d := range devs {
		defer d.Close()
	}
	if err != nil {
		log.Fatalf("OpenDevices(): %v", err)
	}
	if len(devs) == 0 {
		log.Fatalf("no devices found matching VID %s and PID %s", vid, pid)
	}

	// Pick the first device found
	dev := devs[0]

	// Get active configuration
	activeConfig, err := dev.ActiveConfigNum()
	if err != nil {
		log.Fatalf("failed to get active config: %v", err)
	}

	// Switch the configuration
	cfg, err := dev.Config(activeConfig)
	if err != nil {
		log.Fatalf("%s.Config(%d): %v", dev, activeConfig, err)
	}
	defer cfg.Close()

	// In the config claim interface #0 with alt setting #0
	intf, done, _ := dev.DefaultInterface()
	if err != nil {
		log.Fatalf("%s.Interface(0, 0): %v", cfg, err)
	}
	defer done()

	// Interface with tinfoil
	epIn, epOut, err := getInOutEndpoints(intf)
	NSPList := buildNSPList(getNSPListFromDirectory(dir))
	sendNSPList(NSPList, epOut)
	sendNSPFilesPoll(NSPList, epIn, epOut)
}
