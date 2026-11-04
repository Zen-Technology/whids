//go:build windows
// +build windows

package sysmon

import (
	"bytes"
	"encoding/xml"
	"os"
	"testing"
	"time"

	"github.com/0xrawsec/toast"
	"github.com/0xrawsec/whids/utils"
)

func init() {
	// testing on WSL makes Windows command
	// very long to respond
	DefaultTimeout = 60 * time.Second
}

func TestSysmonInfo(t *testing.T) {
	var c *Config

	// the binary is not redistributable, CI downloads it from live.sysinternals.com
	if _, err := os.Stat("./data/Sysmon64.exe"); err != nil {
		t.Skip("./data/Sysmon64.exe not found, skipping Sysmon install test")
	}

	tt := toast.FromT(t)

	// we run uninstall first
	Uninstall()
	defer Uninstall()

	_, err := NewSysmonInfo()

	tt.ExpectErr(err, ErrSysmonNotInstalled)

	tt.CheckErr(InstallOrUpdate("./data/Sysmon64.exe"))

	i, err := NewSysmonInfo()
	t.Log(utils.PrettyJsonOrPanic(i))
	tt.CheckErr(err)

	// we deserialize config
	tt.CheckErr(xml.Unmarshal([]byte(config), &c))
	// we force the good schema version to be the one of sysmon installed
	c.SchemaVersion = i.Config.Version.Schema
	xmlConfig, err := c.XML()
	tt.CheckErr(err)

	// configuring Sysmon
	tt.CheckErr(Configure(bytes.NewBuffer(xmlConfig)))

	i, err = NewSysmonInfo()
	tt.CheckErr(err)

	sha256, err := c.Sha256()
	tt.CheckErr(err)
	tt.Assert(i.Config.Hash == sha256)

	tt.CheckErr(Uninstall())
}
