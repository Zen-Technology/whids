//go:build windows
// +build windows

package sysmon

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xrawsec/golang-utils/crypto/file"
	"github.com/0xrawsec/golang-utils/fsutil"
	"github.com/0xrawsec/golang-win32/win32/advapi32"
	"github.com/0xrawsec/whids/utils"
	"github.com/0xrawsec/whids/utils/command"
)

const (
	servicesPath = `HKLM\SYSTEM\CurrentControlSet\Services`
	driverPath   = `C:\Windows`
)

var (
	DefaultTimeout = 5 * time.Second
)

// stdCmdOutput normalises Sysmon console output (UTF-16LE with or without
// BOM, or UTF-8) to UTF-8.
func stdCmdOutput(b []byte) []byte {
	return DecodeCommandOutput(b)
}

// Versions runs `sysmon -s` on image and returns the Sysmon version, the
// configuration schema version and the binary version. It fails when the
// schema version cannot be determined, as configuring Sysmon without it
// makes recent releases crash.
func Versions(image string) (version, schema, binary string, err error) {
	var out []byte

	if !fsutil.IsFile(image) {
		err = fmt.Errorf("sysmon image not found: %s", image)
		return
	}

	c := command.CommandTimeout(
		DefaultTimeout,
		image,
		"-s",
	)
	defer c.Terminate()

	if out, err = c.CombinedOutput(); err != nil {
		err = fmt.Errorf("failed to run sysmon schema command: %s", err)
		return
	}

	return ParseVersions(out)
}

func InstallOrUpdate(image string) (err error) {
	var out []byte

	// we first uninstall existing installation
	// if Sysmon is not installed yet we should not get any error
	if err = Uninstall(); err != nil && !errors.Is(err, ErrSysmonNotInstalled) {
		return
	}

	// we run install
	c := command.CommandTimeout(DefaultTimeout, image, "-accepteula", "-i")
	defer c.Terminate()
	if out, err = c.CombinedOutput(); err != nil {
		out = stdCmdOutput(out)
		return fmt.Errorf("failed to install sysmon: %w\n%s", err, out)
	}

	return
}

func Configure(r io.Reader) (err error) {
	var tmp, config string
	var i *Info

	// retrieve sysmon information
	if i, err = NewSysmonInfo(); err != nil {
		return
	}

	image := i.Service.Image

	if !fsutil.IsFile(image) {
		return ErrSysmonNotInstalled
	}

	if tmp, err = utils.HidsMkTmpDir(); err != nil {
		return fmt.Errorf("failed to create tmp dir: %w", err)
	}
	// remove temporary file
	defer os.RemoveAll(tmp)

	config = filepath.Join(tmp, "sysmon.xml")
	if err = utils.HidsWriteReader(config, r, false); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	c := command.CommandTimeout(DefaultTimeout, image, "-c", config)
	defer c.Terminate()
	if err = c.Run(); err != nil {
		return fmt.Errorf("command to configure sysmon failed: %w", err)
	}

	return
}

func Uninstall() (err error) {
	var i *Info
	var out []byte

	// retrieve sysmon information
	if i, err = NewSysmonInfo(); err != nil {
		return
	}

	image := i.Service.Image

	//means sysmon is already installed
	if fsutil.IsFile(image) {
		// we uninstall it
		// force flag forces all components to uninstall
		c := command.CommandTimeout(DefaultTimeout, image, "-u", "force")
		defer c.Terminate()

		if out, err = c.CombinedOutput(); err != nil {
			out = stdCmdOutput(out)
			return fmt.Errorf("failed to uninstall sysmon: %w\n%s", err, string(out))
		}

		if err = os.Remove(image); err != nil {
			return fmt.Errorf("failed to remove sysmon binary: %w", err)
		}
	}

	// we don't return any error if Sysmon is not yet installed
	return
}

func findMatchingSysmonServiceKeys() (keys []string) {
	keys = make([]string, 0)

	if skeys, err := advapi32.RegEnumKeys(servicesPath); err != nil {
		return
	} else {
		for _, skey := range skeys {
			fullSubKey := utils.RegJoin(servicesPath, skey)
			descValue := utils.RegJoin(fullSubKey, "Description")
			if desc := utils.RegValueToString(descValue); desc != "" {
				if strings.EqualFold(desc, "System Monitor service") {
					keys = append(keys, skey)
				}
			}
		}
	}
	return
}

func NewSysmonInfo() (i *Info, err error) {
	var sysmonRegPath string

	i = &Info{}

	// validate that we find only one entry in registry
	keys := findMatchingSysmonServiceKeys()

	// sysmon is not installed
	if len(keys) == 0 {
		err = ErrSysmonNotInstalled
		return
	}

	// more than one key found -> not normal
	if len(keys) > 1 {
		err = fmt.Errorf("more than one key looking like Sysmon: %v", keys)
		return
	}

	i.Service.Name = keys[0]

	sysmonRegPath = utils.RegJoin(servicesPath, i.Service.Name)
	i.Service.Image = utils.RegValueToString(sysmonRegPath, "ImagePath")
	i.Service.Sha256, _ = file.Sha256(i.Service.Image)

	i.Driver.Name = i.DriverName()
	i.Driver.Image = filepath.Join(driverPath, utils.RegValueToString(i.DriverRegistry(), "ImagePath"))
	i.Driver.Sha256, _ = file.Sha256(i.Driver.Image)

	i.Config.Hash = i.ConfigHash()

	// parse schema and populate version information
	err = i.parseSchema()

	return
}

func (i *Info) ServiceRegistry() string {
	return utils.RegJoin(servicesPath, i.Service.Name)
}

func (i *Info) ServiceParametersRegistry() string {
	return utils.RegJoin(servicesPath, i.Service.Name, "Parameters")
}

func (i *Info) DriverName() string {
	if i.Driver.Name != "" {
		return i.Driver.Name
	}
	return utils.RegValueToString(i.ServiceParametersRegistry(), "DriverName")
}

func (i *Info) DriverRegistry() string {
	return utils.RegJoin(servicesPath, i.DriverName())
}

func (i *Info) DriverParametersRegistry() string {
	return utils.RegJoin(i.DriverRegistry(), "Parameters")
}

func (i *Info) ConfigHash() string {
	if i.Config.Hash != "" {
		return i.Config.Hash
	}
	hash := utils.RegValueToString(i.DriverParametersRegistry(), "ConfigHash")
	if i := strings.Index(hash, "="); i > -1 && i+1 < len(hash) {
		hash = strings.ToLower(hash[i+1:])
	}
	return hash
}

func (i *Info) parseSchema() (err error) {
	var version, schema, binary string

	version, schema, binary, err = Versions(i.Service.Image)
	i.Version = version
	i.Config.Version.Schema = schema
	i.Config.Version.Binary = binary

	return
}
