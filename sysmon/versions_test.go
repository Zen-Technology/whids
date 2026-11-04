package sysmon

import (
	"encoding/binary"
	"errors"
	"testing"
	"unicode/utf16"
)

func utf16LE(s string, bom bool) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, 0, 2+2*len(units))
	if bom {
		out = append(out, 0xff, 0xfe)
	}
	for _, u := range units {
		out = binary.LittleEndian.AppendUint16(out, u)
	}
	return out
}

const (
	sysmon13Output = "\r\n\r\nSystem Monitor v13.34 - System activity monitor\r\n" +
		"Copyright (C) 2014-2022 Mark Russinovich and Thomas Garnier\r\n" +
		"Sysinternals - www.sysinternals.com\r\n\r\n" +
		`<manifest schemaversion="4.83" binaryversion="13.34">` + "\r\n<configuration>"

	// newer releases print a different banner/separator layout
	sysmon15Output = "System Monitor v15.15 - System activity monitor\r\n" +
		"By Mark Russinovich and Thomas Garnier\r\n" +
		"Copyright (C) 2014-2024 Microsoft Corporation\r\n" +
		"Using libxml2. libxml2 is Copyright (C) 1998-2012 Daniel Veillard. All Rights Reserved.\r\n" +
		"Sysinternals - www.sysinternals.com\r\n\r\n" +
		`<?xml version="1.0" encoding="UTF-16"?>` + "\r\n" +
		`<manifest schemaversion="4.90" binaryversion="15.15">` + "\r\n<configuration>"
)

func TestParseVersions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                    string
		out                     []byte
		version, schema, binary string
	}{
		{"v13 utf16 with BOM", utf16LE(sysmon13Output, true), "v13.34", "4.83", "13.34"},
		{"v13 ascii", []byte(sysmon13Output), "v13.34", "4.83", "13.34"},
		{"v15 utf16 without BOM", utf16LE(sysmon15Output, false), "v15.15", "4.90", "15.15"},
		{"v15 utf16 with BOM", utf16LE(sysmon15Output, true), "v15.15", "4.90", "15.15"},
		{"v15 utf8 with BOM", append([]byte{0xef, 0xbb, 0xbf}, sysmon15Output...), "v15.15", "4.90", "15.15"},
		{"manifest only", []byte(`<manifest schemaversion="4.91" binaryversion="15.16">`), "v15.16", "4.91", "15.16"},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			version, schema, bin, err := ParseVersions(c.out)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if version != c.version || schema != c.schema || bin != c.binary {
				t.Fatalf("got (%q, %q, %q), want (%q, %q, %q)", version, schema, bin, c.version, c.schema, c.binary)
			}
		})
	}
}

func TestParseVersionsUnknown(t *testing.T) {
	t.Parallel()

	for _, out := range [][]byte{
		nil,
		[]byte("Error: access denied"),
		utf16LE("System Monitor v15.15 - System activity monitor\r\n", false),
	} {
		if _, _, _, err := ParseVersions(out); !errors.Is(err, ErrSysmonVersionUnknown) {
			t.Fatalf("expected ErrSysmonVersionUnknown for %q, got %v", out, err)
		}
	}
}

func TestDecodeCommandOutputKeepsUTF8(t *testing.T) {
	t.Parallel()

	in := []byte("Système ® ok")
	if got := string(DecodeCommandOutput(in)); got != string(in) {
		t.Fatalf("utf-8 input altered: %q", got)
	}
}
