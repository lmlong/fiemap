package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	FiemapSize = 32 // sizeof(struct fiemap)
	ExtentSize = 56 // sizeof(struct Extent)

	// FS_IOC_FIEMAP is defined in <linux/fs.h>:
	FS_IOC_FIEMAP = 3223348747

	// FIEMAP_MAX_OFFSET Defined in <linux/fiemap.h>:
	FIEMAP_MAX_OFFSET            = ^uint64(0)
	FIEMAP_FLAG_SYNC             = 0x0001 // sync file data before map
	FIEMAP_FLAG_XATTR            = 0x0002 // map extended attribute tree
	FIEMAP_FLAG_CACHE            = 0x0004 // request caching of the extents
	FIEMAP_FLAGS_COMPAT          = (FIEMAP_FLAG_SYNC | FIEMAP_FLAG_XATTR)
	FIEMAP_EXTENT_LAST           = 0x0001 // Last extent in file.
	FIEMAP_EXTENT_UNKNOWN        = 0x0002 // Data location unknown.
	FIEMAP_EXTENT_DELALLOC       = 0x0004 // Location still pending. Sets EXTENT_UNKNOWN.
	FIEMAP_EXTENT_ENCODED        = 0x0008 // Data can not be read while fs is unmounted
	FIEMAP_EXTENT_DATA_ENCRYPTED = 0x0080 // Data is encrypted by fs. Sets EXTENT_NO_BYPASS.
	FIEMAP_EXTENT_NOT_ALIGNED    = 0x0100 // Extent offsets may not be block aligned.
	FIEMAP_EXTENT_DATA_INLINE    = 0x0200 // Data mixed with metadata. Sets EXTENT_NOT_ALIGNED.
	FIEMAP_EXTENT_DATA_TAIL      = 0x0400 // Multiple files in block. Sets EXTENT_NOT_ALIGNED.
	FIEMAP_EXTENT_UNWRITTEN      = 0x0800 // Space allocated, but no data (i.e. zero).
	FIEMAP_EXTENT_MERGED         = 0x1000 // File does not natively support extents. Result merged for efficiency.
	FIEMAP_EXTENT_SHARED         = 0x2000 // Space shared with other files.

	// FALLOC_FL_KEEP_SIZE Defined in <linux/falloc.h>:
	FALLOC_FL_KEEP_SIZE    = 0x01 // default is extend size
	FALLOC_FL_PUNCH_HOLE   = 0x02 // de-allocates range
	FALLOC_FL_NO_HIDE_STAE = 0x04 // reserved codepoint
)

const (
	SEEK_SET  = 0
	SEEK_CUR  = 1
	SEEK_END  = 2
	SEEK_DATA = 3
	SEEK_HOLE = 4
)

// based on struct fiemap from <linux/fiemap.h>
type fiemap struct {
	Start         uint64 // logical offset (inclusive) at which to start mapping (in)
	Length        uint64 // logical length of mapping which userspace wants (in)
	Flags         uint32 // FIEMAP_FLAG_* flags for request (in/out)
	MappedExtents uint32 // number of extents that were mapped (out)
	ExtentCount   uint32 // size of fm_extents array (in)
	Reserved      uint32
	// Extents [0]Extent // array of mapped extents (out)
}

// Extent : based on struct fiemap_extent from <linux/fiemap.h>
type Extent struct {
	Logical    uint64 // logical offset in bytes for the start of the extent from the beginning of the file
	Physical   uint64 // physical offset in bytes for the start	of the extent from the beginning of the disk
	Length     uint64 // length in bytes for this extent
	Reserved64 [2]uint64
	Flags      uint32 // FIEMAP_EXTENT_* flags for this extent
	Reserved   [3]uint32
}

// FiemapFile creates a new type by wrapping up os.File
type FiemapFile struct {
	*os.File
}

// NewFiemapFile : return a new FibmapFile
func NewFiemapFile(f *os.File) FiemapFile {
	return FiemapFile{f}
}

func (f FiemapFile) FieMap(offset, length uint64) ([]Extent, error) {
	isEof := false
	extents := make([]Extent, 0)
	if length == 0 {
		length = FIEMAP_MAX_OFFSET
	}
	for !isEof {
		exts, err := f.fiemap(offset, length)
		if err != nil {
			return nil, err
		}
		extents = append(extents, exts...)

		for i := range exts {
			if FIEMAP_EXTENT_LAST&exts[i].Flags == FIEMAP_EXTENT_LAST {
				isEof = true
			}
			offset += exts[i].Length
			length -= exts[i].Length
		}
	}
	return extents, nil
}

func (f FiemapFile) fiemap(start uint64, length uint64) ([]Extent, error) {
	fmt.Printf("fiemap: start=%d length=%d\n", start, length)
	numExts := 250
	extBuf := make([]Extent, numExts+1)
	ptr := unsafe.Pointer(uintptr(unsafe.Pointer(&extBuf[0])) + (ExtentSize - FiemapSize))

	t := (*fiemap)(ptr)
	t.Start = start
	t.Length = length
	t.Flags = FIEMAP_FLAG_SYNC
	t.ExtentCount = uint32(numExts)

	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), FS_IOC_FIEMAP, uintptr(ptr))
	if err != 0 {
		fmt.Printf("syscall error:%v\n", err)
		return nil, err
	}
	return extBuf[1 : 1+t.MappedExtents], nil
}

// Fallocate : allocate using fallocate
func (f FiemapFile) Fallocate(offset int64, length int64) error {
	return syscall.Fallocate(int(f.Fd()), 0, offset, length)
}

// PunchHole : punch hole using fallocate
func (f FiemapFile) PunchHole(offset int64, length int64) error {
	return syscall.Fallocate(int(f.Fd()), FALLOC_FL_KEEP_SIZE|FALLOC_FL_PUNCH_HOLE, offset, length)
}
