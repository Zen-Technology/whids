package sysmon

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
	"unicode/utf16"
)

var (
	// ErrSysmonVersionUnknown is returned when the output of `sysmon -s`
	// does not contain a schema version. Configuring Sysmon without one
	// makes recent releases crash (STATUS_STACK_BUFFER_OVERRUN).
	ErrSysmonVersionUnknown = errors.New("cannot determine Sysmon schema version")

	// Each value is matched on its own: Sysmon releases changed the banner
	// and the text between the banner and the schema manifest over time.
	bannerVersionRe  = regexp.MustCompile(`System\s+Monitor\s+(v\d+(?:\.\d+)*)`)
	manifestSchemaRe = regexp.MustCompile(`schemaversion\s*=\s*"(\d+(?:\.\d+)*)"`)
	binaryVersionRe  = regexp.MustCompile(`binaryversion\s*=\s*"(\d+(?:\.\d+)*)"`)
)

// DecodeCommandOutput normalises console output of Sysinternals tools to
// UTF-8. Depending on the release and on whether output is redirected,
// Sysmon writes UTF-16LE with a BOM, UTF-16LE without a BOM, or UTF-8/ASCII.
func DecodeCommandOutput(b []byte) []byte {
	switch {
	case len(b) >= 2 && b[0] == 0xff && b[1] == 0xfe:
		return decodeUTF16LE(b[2:])
	case looksUTF16LE(b):
		return decodeUTF16LE(b)
	default:
		return bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf})
	}
}

// looksUTF16LE reports whether b is mostly ASCII encoded as UTF-16LE, i.e.
// most bytes at odd offsets are NUL.
func looksUTF16LE(b []byte) bool {
	n := len(b)
	if n > 1024 {
		n = 1024
	}
	n &^= 1
	if n < 4 {
		return false
	}
	zeros := 0
	for i := 1; i < n; i += 2 {
		if b[i] == 0 {
			zeros++
		}
	}
	return zeros*10 >= (n/2)*7 // at least 70% of code units are ASCII
}

func decodeUTF16LE(b []byte) []byte {
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(b[2*i:])
	}
	return []byte(string(utf16.Decode(units)))
}

// ParseVersions extracts the Sysmon version (e.g. v15.15), the configuration
// schema version and the binary version from the output of `sysmon -s`.
func ParseVersions(out []byte) (version, schema, binary string, err error) {
	text := DecodeCommandOutput(out)

	if m := bannerVersionRe.FindSubmatch(text); m != nil {
		version = string(m[1])
	}
	if m := manifestSchemaRe.FindSubmatch(text); m != nil {
		schema = string(m[1])
	}
	if m := binaryVersionRe.FindSubmatch(text); m != nil {
		binary = string(m[1])
	}

	// the manifest's binary version matches the banner version
	if version == "" && binary != "" {
		version = "v" + binary
	}

	if schema == "" {
		head := text
		if len(head) > 300 {
			head = head[:300]
		}
		err = fmt.Errorf("%w from `sysmon -s` output: %q", ErrSysmonVersionUnknown, head)
	}
	return
}
