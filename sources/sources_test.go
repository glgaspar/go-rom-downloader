package sources

import (
	"fmt"
	"testing"
)

func TestCoolrom(t *testing.T) {
	src := NewCoolromSource()
	roms := src.Lookup("sonic")
	fmt.Printf("Coolrom lookup 'sonic' found %d results\n", len(roms))
	for i, r := range roms {
		if i < 5 {
			fmt.Printf("  - %s (%s): %s\n", r.Name, r.Console, r.URL)
		}
	}
	if len(roms) > 0 {
		rom := &roms[0]
		dl := src.GetDownloadLink(rom)
		fmt.Printf("Coolrom download link for '%s': %s\n", rom.Name, dl)
	}
}

func TestEmuparadise(t *testing.T) {
	src := NewEmuparadiseSource()
	roms := src.Lookup("sonic")
	fmt.Printf("Emuparadise lookup 'sonic' found %d results\n", len(roms))
	for i, r := range roms {
		if i < 5 {
			fmt.Printf("  - %s (%s): %s\n", r.Name, r.Console, r.URL)
		}
	}
	if len(roms) > 0 {
		rom := &roms[0]
		dl := src.GetDownloadLink(rom)
		fmt.Printf("Emuparadise download link for '%s': %s\n", rom.Name, dl)
	}
}

func TestConsoleroms(t *testing.T) {
	src := NewConsoleromsSource()
	roms := src.Lookup("sonic")
	fmt.Printf("Consoleroms lookup 'sonic' found %d results\n", len(roms))
	for i, r := range roms {
		if i < 5 {
			fmt.Printf("  - %s (%s): %s\n", r.Name, r.Console, r.URL)
		}
	}
	if len(roms) > 0 {
		rom := &roms[0]
		dl := src.GetDownloadLink(rom)
		fmt.Printf("Consoleroms download link for '%s': %s\n", rom.Name, dl)
	}
}

func TestRomsgames(t *testing.T) {
	src := NewRomsgamesSource()
	roms := src.Lookup("sonic")
	fmt.Printf("Romsgames lookup 'sonic' found %d results\n", len(roms))
	for i, r := range roms {
		if i < 5 {
			fmt.Printf("  - %s (%s): %s\n", r.Name, r.Console, r.URL)
		}
	}
	if len(roms) > 0 {
		rom := &roms[0]
		dl := src.GetDownloadLink(rom)
		fmt.Printf("Romsgames download link for '%s': %s\n", rom.Name, dl)
	}
}


