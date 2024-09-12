package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/augustoroman/hexdump"
	"github.com/google/subcommands"
	"github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"os"
)

const (
	KB = int64(1024)
	MB = 1024 * KB
	GB = 1024 * MB
)

type cmdWriteAt struct {
	ofs int64
	len int64
}

func (cmd *cmdWriteAt) Name() string {
	return "writeat"
}
func (cmd *cmdWriteAt) Synopsis() string { return "write to file using os.file.WriteAt" }
func (cmd *cmdWriteAt) Usage() string {
	return `writeat -ofs offset [-len length]
`
}
func (cmd *cmdWriteAt) SetFlags(f *flag.FlagSet) {
	f.Int64Var(&cmd.ofs, "ofs", 0, "offset(in 4K)")
	f.Int64Var(&cmd.len, "len", 1, "length(in 4K)")
}
func (cmd *cmdWriteAt) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if len(f.Args()) != 1 {
		fmt.Printf("Usage: writeat [-ofs offset] [-len length] file\n")
		return subcommands.ExitUsageError
	}

	fileName := f.Arg(0)
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Printf("open file err! %v", err)
		return subcommands.ExitFailure
	}
	defer file.Close()

	buf := randomBlock(4 * KB)
	offset := cmd.ofs * 4 * KB

	for i := int64(0); i < cmd.len; i++ {
		n, err := file.WriteAt(buf, offset)
		if err != nil {
			logrus.Printf("write to file err! %v", err)
			return subcommands.ExitFailure
		}
		offset += int64(n)
	}
	logrus.Println("write ok!", "offset", cmd.ofs*4*KB, "len", cmd.len*4*KB)

	return subcommands.ExitSuccess

}

type cmdPunchHole struct {
	ofs int64
	len int64
}

func (cmd *cmdPunchHole) Name() string {
	return "punchhole"
}
func (cmd *cmdPunchHole) Synopsis() string { return "punch hole using fAllocat" }
func (cmd *cmdPunchHole) Usage() string {
	return `punchhole -ofs offset [-len length]
`
}
func (cmd *cmdPunchHole) SetFlags(f *flag.FlagSet) {
	f.Int64Var(&cmd.ofs, "ofs", 0, "offset(in 4K)")
	f.Int64Var(&cmd.len, "len", 1, "length(in 4K)")
}
func (cmd *cmdPunchHole) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if len(f.Args()) != 1 {
		f.Usage()
		return subcommands.ExitUsageError
	}

	fileName := f.Arg(0)
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Printf("open file err! %v", err)
		return subcommands.ExitFailure
	}
	defer file.Close()
	offset := cmd.ofs * 4 * KB
	length := cmd.len * 4 * KB

	fiefile := NewFiemapFile(file)
	err = fiefile.PunchHole(offset, length)
	if err != nil {
		logrus.Printf("punch hole err! %v", err)
		return subcommands.ExitFailure
	}

	logrus.Println("punch ok!", "offset", cmd.ofs*4*KB, "len", cmd.len*4*KB)

	return subcommands.ExitSuccess

}

type cmdReadAt struct {
	ofs int64
	len int64
}

func (cmd *cmdReadAt) Name() string {
	return "readat"
}
func (cmd *cmdReadAt) Synopsis() string { return "write to file using os.file.ReadAt" }
func (cmd *cmdReadAt) Usage() string {
	return `readAt -ofs offset [-len length]
`
}
func (cmd *cmdReadAt) SetFlags(f *flag.FlagSet) {
	f.Int64Var(&cmd.ofs, "ofs", 0, "offset(in bytes)")
	f.Int64Var(&cmd.len, "len", 128, "length(in bytes)")
}
func (cmd *cmdReadAt) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if len(f.Args()) != 1 {
		fmt.Printf("Usage: readat [-ofs offset] [-len length] file\n")
		return subcommands.ExitFailure
	}

	fileName := f.Arg(0)
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		logrus.Printf("open file err! %v", err)
		return subcommands.ExitFailure
	}
	defer file.Close()
	rlen := cmd.len
	if rlen > 128 {
		rlen = 128
	}

	buf := make([]byte, rlen)
	_, err = file.ReadAt(buf, cmd.ofs)
	if err != nil {
		fmt.Printf("read file err! %v", err)
		return subcommands.ExitFailure
	}

	fmt.Printf(hexdump.Dump(buf))

	return subcommands.ExitSuccess
}

type cmdFieMap struct {
	ofs uint64
	len uint64
}

func (cmd *cmdFieMap) Name() string {
	return "fiemap"
}
func (cmd *cmdFieMap) Synopsis() string { return "dump fiemap to file" }
func (cmd *cmdFieMap) Usage() string {
	return `fiemap -ofs offset [-len length]
`
}
func (cmd *cmdFieMap) SetFlags(f *flag.FlagSet) {
	f.Uint64Var(&cmd.ofs, "ofs", 0, "offset(in bytes)")
	f.Uint64Var(&cmd.len, "len", 0, "length(in bytes)")
}
func (cmd *cmdFieMap) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if len(f.Args()) != 1 {
		fmt.Printf("Usage: fiemap [-ofs offset] [-len length] file\n")
		return subcommands.ExitFailure
	}

	fileName := f.Arg(0)
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		logrus.Printf("open file err! %v", err)
		return subcommands.ExitFailure
	}
	defer file.Close()

	exts, err := NewFiemapFile(file).FieMap(cmd.ofs, cmd.len)
	if err != nil {
		logrus.Printf("get fiemap err! %v", err)
		return subcommands.ExitFailure
	}
	fmt.Printf("%5s\t%16s\t%16s\t%16s\t%04s\n", "#", "Logical", "Physical", "Length", "Flags")
	for i, ext := range exts {
		fmt.Printf("%05x\t%016x\t%016x\t%016x\t%04x\n", i, ext.Logical, ext.Physical, ext.Length, ext.Flags)
	}

	return subcommands.ExitSuccess
}

type cmdSeekWrite struct {
	ofs int64
	len int64
}

func (cmd *cmdSeekWrite) Name() string {
	return "seekwrite"
}
func (cmd *cmdSeekWrite) Synopsis() string { return "write to file using seek+write" }
func (cmd *cmdSeekWrite) Usage() string {
	return `seekwrite -ofs offset [-len length]
`
}
func (cmd *cmdSeekWrite) SetFlags(f *flag.FlagSet) {
	f.Int64Var(&cmd.ofs, "ofs", 0, "offset(in 4K)")
	f.Int64Var(&cmd.len, "len", 1, "length(in 4K)")
}
func (cmd *cmdSeekWrite) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if len(f.Args()) != 1 {
		fmt.Printf("Usage: seekwrite [-ofs offset] [-len length] file\n")
		return subcommands.ExitFailure
	}

	fileName := f.Arg(0)
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Fatal("open file err! %v", err)
	}
	defer file.Close()

	buf := randomBlock(4 * KB)
	offset := cmd.ofs * 4 * KB
	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		logrus.Fatal("seek file err! %v", err)
	}

	for i := int64(0); i < cmd.len; i++ {
		n, err := file.Write(buf)
		if err != nil {
			logrus.Fatal("write to file err! %v", err)
		}
		offset += int64(n)
	}
	logrus.Println("write ok!", "from:", cmd.ofs*4*KB, "len:", cmd.len*4*KB, "end:", offset)

	return subcommands.ExitSuccess
}

type cmdSeekHole struct {
	ofs int64
}

func (cmd *cmdSeekHole) Name() string {
	return "seekhole"
}
func (cmd *cmdSeekHole) Synopsis() string { return "seek hole by offset" }
func (cmd *cmdSeekHole) Usage() string {
	return `seekhole -ofs offset
`
}
func (cmd *cmdSeekHole) SetFlags(f *flag.FlagSet) {
	f.Int64Var(&cmd.ofs, "ofs", 0, "offset")
}
func (cmd *cmdSeekHole) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if len(f.Args()) != 1 {
		fmt.Printf("Usage: seekhole [-ofs offset] file\n")
		return subcommands.ExitUsageError
	}

	fileName := f.Arg(0)
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Printf("open file err! %v", err)
		return subcommands.ExitFailure
	}
	defer file.Close()

	fiefile := NewFiemapFile(file)
	holeOffset, err := fiefile.Seek(cmd.ofs, SEEK_HOLE)
	if err != nil {
		logrus.Printf("seek file err! %v", err)
		return subcommands.ExitFailure
	}
	logrus.Println("offset:", holeOffset)
	return subcommands.ExitSuccess
}

type cmdSeekData struct {
	ofs int64
}

func (cmd *cmdSeekData) Name() string {
	return "seekdata"
}
func (cmd *cmdSeekData) Synopsis() string { return "seek hole by offset" }
func (cmd *cmdSeekData) Usage() string {
	return `seekdata -ofs offset
`
}
func (cmd *cmdSeekData) SetFlags(f *flag.FlagSet) {
	f.Int64Var(&cmd.ofs, "ofs", 0, "offset")
}
func (cmd *cmdSeekData) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if len(f.Args()) != 1 {
		fmt.Printf("Usage: seekdata [-ofs offset] file\n")
		return subcommands.ExitUsageError
	}

	fileName := f.Arg(0)
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Printf("open file err! %v", err)
		return subcommands.ExitFailure
	}
	defer file.Close()

	fiefile := NewFiemapFile(file)
	dataOffset, err := fiefile.Seek(cmd.ofs, SEEK_DATA)
	if err != nil {
		logrus.Printf("seek file err! %v", err)
		return subcommands.ExitFailure
	}
	logrus.Println("offset:", dataOffset)
	return subcommands.ExitSuccess
}

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(&cmdFieMap{}, "")
	subcommands.Register(&cmdPunchHole{}, "")
	subcommands.Register(&cmdReadAt{}, "")
	subcommands.Register(&cmdSeekWrite{}, "")
	subcommands.Register(&cmdWriteAt{}, "")
	subcommands.Register(&cmdSeekHole{}, "")
	subcommands.Register(&cmdSeekData{}, "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}

func randomBlock(n int64) []byte {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	char := letterBytes[rand.Intn(len(letterBytes))]
	b := make([]byte, n)
	for i := range b {
		b[i] = char
	}
	return b
}
