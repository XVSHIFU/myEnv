package core

import (
	"golang.org/x/sys/windows/registry"
	"path/filepath"
)

func registeredInventory() ([]inventoryCandidate, []string) {
	out := []inventoryCandidate{}
	warnings := []string{}
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		for _, view := range []uint32{registry.WOW64_64KEY, registry.WOW64_32KEY} {
			key, err := registry.OpenKey(root, `Software\Python`, registry.READ|view)
			if err != nil {
				if err != registry.ErrNotExist {
					warnings = append(warnings, "Python registry: "+err.Error())
				}
				continue
			}
			companies, err := key.ReadSubKeyNames(256)
			key.Close()
			if err != nil && len(companies) == 0 {
				continue
			}
			for _, company := range companies {
				base := `Software\Python\` + company
				key, err = registry.OpenKey(root, base, registry.READ|view)
				if err != nil {
					continue
				}
				tags, _ := key.ReadSubKeyNames(256)
				key.Close()
				for _, tag := range tags {
					key, err = registry.OpenKey(root, base+`\`+tag+`\InstallPath`, registry.READ|view)
					if err != nil {
						continue
					}
					entry, _, e := key.GetStringValue("ExecutablePath")
					if e != nil {
						dir, _, e := key.GetStringValue("")
						if e == nil {
							entry = filepath.Join(dir, "python.exe")
						}
					}
					key.Close()
					if filepath.IsAbs(entry) {
						out = append(out, inventoryCandidate{"python", entry, "Windows Python registry: " + company + "/" + tag})
					}
				}
			}
		}
	}
	return out, warnings
}
