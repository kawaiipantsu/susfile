package analyze

import (
	"bytes"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"fmt"
	"io"
)

// maxSectionEntropyBytes caps how much of a section we read to measure its
// entropy, so a header claiming a huge section cannot make us allocate it.
const maxSectionEntropyBytes = 1 << 20

// structurePass fills r.Magic, r.MIME/MIMETree/ExtGuess and r.BinFmt from the
// buffered prefix. Every parser is wrapped so a crafted file yields warnings,
// never a panic.
func structurePass(buf []byte, r *Result) {
	r.Magic = matchMagic(buf)
	r.MIME, r.MIMETree, r.ExtGuess = detectMIME(buf)
	if r.ExtGuess == "" && r.Magic.Ext != "" {
		r.ExtGuess = r.Magic.Ext
	}

	bf, warns := detectBinFmt(buf)
	r.BinFmt = bf
	for _, w := range warns {
		r.warn("%s", w)
	}
}

// detectBinFmt tries ELF, then PE, then Mach-O. It returns nil if none parse.
func detectBinFmt(buf []byte) (bf *BinFmt, warns []string) {
	defer func() {
		if p := recover(); p != nil {
			warns = append(warns, fmt.Sprintf("executable parser panicked: %v", p))
			bf = nil
		}
	}()

	switch {
	case len(buf) >= 4 && string(buf[:4]) == "\x7fELF":
		return parseELF(buf)
	case len(buf) >= 2 && string(buf[:2]) == "MZ":
		return parsePE(buf)
	case len(buf) >= 4 && isMachOMagic(buf):
		return parseMachO(buf)
	}
	return nil, nil
}

func isMachOMagic(buf []byte) bool {
	switch string(buf[:4]) {
	case "\xcf\xfa\xed\xfe", "\xce\xfa\xed\xfe", "\xfe\xed\xfa\xcf", "\xfe\xed\xfa\xce":
		return true
	}
	return false
}

func parseELF(buf []byte) (*BinFmt, []string) {
	var warns []string
	f, err := elf.NewFile(bytes.NewReader(buf))
	if err != nil {
		return nil, []string{fmt.Sprintf("ELF header present but unparseable: %v", err)}
	}
	bf := &BinFmt{
		Format:  "ELF",
		Machine: elfMachine(f.Machine),
		Type:    elfType(f.Type),
	}
	switch f.Class {
	case elf.ELFCLASS32:
		bf.Bits = 32
	case elf.ELFCLASS64:
		bf.Bits = 64
	}
	switch f.Data {
	case elf.ELFDATA2LSB:
		bf.Endian = "little"
	case elf.ELFDATA2MSB:
		bf.Endian = "big"
	}

	var hasSym bool
	for _, s := range f.Sections {
		if s.Type == elf.SHT_NULL || s.Size == 0 {
			continue
		}
		if s.Name == ".symtab" {
			hasSym = true
		}
		bf.Sections = append(bf.Sections, Section{
			Name:    s.Name,
			Offset:  int64(s.Offset),
			Size:    int64(s.Size),
			Entropy: sectionEntropy(s.Open(), int64(s.Size)),
			Flags:   elfFlags(s.Flags),
		})
	}
	bf.Stripped = !hasSym

	if p := elfInterp(f); p != "" {
		bf.Interp = p
	}
	if syms, err := f.ImportedSymbols(); err == nil {
		bf.NumImports = len(syms)
	} else {
		warns = append(warns, fmt.Sprintf("ELF imports unreadable: %v", err))
	}
	return bf, warns
}

func elfMachine(m elf.Machine) string {
	switch m {
	case elf.EM_X86_64:
		return "x86-64"
	case elf.EM_386:
		return "x86"
	case elf.EM_AARCH64:
		return "ARM64"
	case elf.EM_ARM:
		return "ARM"
	case elf.EM_RISCV:
		return "RISC-V"
	case elf.EM_PPC64:
		return "PowerPC64"
	case elf.EM_MIPS:
		return "MIPS"
	case elf.EM_S390:
		return "S/390"
	case elf.EM_LOONGARCH:
		return "LoongArch"
	}
	return m.String()
}

func elfType(t elf.Type) string {
	switch t {
	case elf.ET_REL:
		return "relocatable object"
	case elf.ET_EXEC:
		return "executable"
	case elf.ET_DYN:
		return "shared object / PIE"
	case elf.ET_CORE:
		return "core dump"
	}
	return t.String()
}

func elfFlags(f elf.SectionFlag) string {
	s := "r"
	if f&elf.SHF_WRITE != 0 {
		s += "w"
	}
	if f&elf.SHF_EXECINSTR != 0 {
		s += "x"
	}
	return s
}

func elfInterp(f *elf.File) string {
	for _, p := range f.Progs {
		if p.Type != elf.PT_INTERP {
			continue
		}
		n := p.Filesz
		if n == 0 || n > 4096 {
			return ""
		}
		b := make([]byte, n)
		if _, err := p.ReadAt(b, 0); err != nil && err != io.EOF {
			return ""
		}
		return string(bytes.TrimRight(b, "\x00"))
	}
	return ""
}

func parsePE(buf []byte) (*BinFmt, []string) {
	var warns []string
	f, err := pe.NewFile(bytes.NewReader(buf))
	if err != nil {
		return nil, []string{fmt.Sprintf("PE header present but unparseable: %v", err)}
	}
	bf := &BinFmt{
		Format:  "PE",
		Machine: peMachine(f.Machine),
		Endian:  "little",
	}
	if f.OptionalHeader != nil {
		switch oh := f.OptionalHeader.(type) {
		case *pe.OptionalHeader64:
			bf.Bits = 64
			bf.Type = peType(f.Characteristics, oh.Subsystem)
		case *pe.OptionalHeader32:
			bf.Bits = 32
			bf.Type = peType(f.Characteristics, oh.Subsystem)
		}
	}
	if f.Characteristics&pe.IMAGE_FILE_DLL != 0 {
		bf.Type = "dynamic library (DLL)"
	}

	for _, s := range f.Sections {
		if s.Size == 0 {
			continue
		}
		bf.Sections = append(bf.Sections, Section{
			Name:    s.Name,
			Offset:  int64(s.Offset),
			Size:    int64(s.Size),
			Entropy: sectionEntropy(s.Open(), int64(s.Size)),
			Flags:   peFlags(s.Characteristics),
		})
	}
	bf.Stripped = len(f.Symbols) == 0

	if syms, err := f.ImportedSymbols(); err == nil {
		bf.NumImports = len(syms)
	} else {
		warns = append(warns, fmt.Sprintf("PE imports unreadable: %v", err))
	}
	return bf, warns
}

func peMachine(m uint16) string {
	switch m {
	case pe.IMAGE_FILE_MACHINE_AMD64:
		return "x86-64"
	case pe.IMAGE_FILE_MACHINE_I386:
		return "x86"
	case pe.IMAGE_FILE_MACHINE_ARM64:
		return "ARM64"
	case pe.IMAGE_FILE_MACHINE_ARMNT:
		return "ARM Thumb-2"
	}
	return fmt.Sprintf("machine 0x%04x", m)
}

func peType(ch, subsystem uint16) string {
	if ch&pe.IMAGE_FILE_DLL != 0 {
		return "dynamic library (DLL)"
	}
	const (
		subNative = 1
		subGUI    = 2
		subCUI    = 3
	)
	switch subsystem {
	case subNative:
		return "native executable"
	case subGUI:
		return "GUI executable"
	case subCUI:
		return "console executable"
	}
	return "executable"
}

func peFlags(c uint32) string {
	s := ""
	if c&0x40000000 != 0 {
		s += "r"
	}
	if c&0x80000000 != 0 {
		s += "w"
	}
	if c&0x20000000 != 0 {
		s += "x"
	}
	if s == "" {
		s = "-"
	}
	return s
}

func parseMachO(buf []byte) (*BinFmt, []string) {
	f, err := macho.NewFile(bytes.NewReader(buf))
	if err != nil {
		return nil, []string{fmt.Sprintf("Mach-O header present but unparseable: %v", err)}
	}
	bf := &BinFmt{
		Format:  "Mach-O",
		Machine: f.Cpu.String(),
		Type:    machoType(f.Type),
		Endian:  "little",
	}
	if f.Magic == macho.Magic64 {
		bf.Bits = 64
	} else {
		bf.Bits = 32
	}
	for _, s := range f.Sections {
		if s.Size == 0 {
			continue
		}
		bf.Sections = append(bf.Sections, Section{
			Name:    s.Seg + "," + s.Name,
			Offset:  int64(s.Offset),
			Size:    int64(s.Size),
			Entropy: sectionEntropy(s.Open(), int64(s.Size)),
		})
	}
	if syms := f.Symtab; syms != nil {
		bf.NumImports = len(syms.Syms)
		bf.Stripped = len(syms.Syms) == 0
	} else {
		bf.Stripped = true
	}
	return bf, nil
}

func machoType(t macho.Type) string {
	switch t {
	case macho.TypeObj:
		return "relocatable object"
	case macho.TypeExec:
		return "executable"
	case macho.TypeDylib:
		return "dynamic library"
	case macho.TypeBundle:
		return "bundle"
	}
	return fmt.Sprintf("type %d", int(t))
}

// sectionEntropy reads up to maxSectionEntropyBytes from rd and returns the
// Shannon entropy of what it got. declaredSize only bounds the allocation.
func sectionEntropy(rd io.Reader, declaredSize int64) float64 {
	n := declaredSize
	if n <= 0 {
		return 0
	}
	if n > maxSectionEntropyBytes {
		n = maxSectionEntropyBytes
	}
	b, err := io.ReadAll(io.LimitReader(rd, n))
	if len(b) == 0 && err != nil {
		return 0
	}
	return entropyOf(b)
}
