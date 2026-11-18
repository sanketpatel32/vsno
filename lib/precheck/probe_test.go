package precheck

import (
	"os"
	"path/filepath"
	"testing"
)

func loadFixture(t *testing.T, name string) NodeHardware {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	hw, err := ParseProbe(string(b))
	if err != nil {
		t.Fatalf("ParseProbe(%s): %v", name, err)
	}
	return hw
}

func findDisk(hw NodeHardware, name string) *Disk {
	for i := range hw.Disks {
		if hw.Disks[i].Name == name {
			return &hw.Disks[i]
		}
	}
	return nil
}

func TestParseProbeBareMetalEmptyDisk(t *testing.T) {
	hw := loadFixture(t, "probe-baremetal-empty-disk.txt")
	if !hw.BareMetal() || !hw.KVM {
		t.Fatalf("want bare metal with KVM, got virt=%q kvm=%v", hw.Virt, hw.KVM)
	}
	if hw.Threads != 104 || hw.Product != "Example Server" {
		t.Fatalf("threads=%d product=%q", hw.Threads, hw.Product)
	}
	if len(hw.Disks) != 2 {
		t.Fatalf("want 2 whole disks (loops skipped), got %d", len(hw.Disks))
	}
	sda, sdb := findDisk(hw, "sda"), findDisk(hw, "sdb")
	if sda == nil || sda.Empty() {
		t.Fatalf("sda is the OS disk and must not be empty: %+v", sda)
	}
	if sdb == nil || !sdb.Empty() {
		t.Fatalf("sdb should be empty: %+v", sdb)
	}
	if got, want := sdb.PreferredByID(), "/dev/disk/by-id/wwn-0x5000000000000002"; got != want {
		t.Fatalf("PreferredByID = %q, want %q", got, want)
	}
	if sdb.SizeGB() != 1919 {
		t.Fatalf("SizeGB = %d", sdb.SizeGB())
	}
}

func TestParseProbeBareMetalLVMSDisk(t *testing.T) {
	hw := loadFixture(t, "probe-baremetal-lvms-disk.txt")
	sdb := findDisk(hw, "sdb")
	if sdb == nil || sdb.Empty() || !sdb.LVMMember() {
		t.Fatalf("sdb should be an LVM member, not empty: %+v", sdb)
	}
	// lvm-pv-uuid aliases must never be chosen.
	if got := sdb.PreferredByID(); got != "/dev/disk/by-id/wwn-0x5000000000000002" {
		t.Fatalf("PreferredByID = %q", got)
	}
}

func TestParseProbeVMWorker(t *testing.T) {
	hw := loadFixture(t, "probe-vm-worker.txt")
	if hw.BareMetal() {
		t.Fatal("VM worker must not be reported as bare metal")
	}
	if !hw.KVM {
		t.Fatal("fixture has nested KVM exposed; KVM should be true")
	}
	vdb := findDisk(hw, "vdb")
	if vdb == nil || !vdb.Empty() || vdb.SizeGB() != 107 {
		t.Fatalf("vdb should be an empty ~107GB disk: %+v", vdb)
	}
	if findDisk(hw, "sr0") != nil {
		t.Fatal("rom devices must be skipped")
	}
}

func TestParseProbeRejectsGarbage(t *testing.T) {
	if _, err := ParseProbe("hello\nworld"); err == nil {
		t.Fatal("expected an error for output without VIRT/NPROC")
	}
}

func TestFlexTypes(t *testing.T) {
	out := `VIRT=none
NPROC=8
LSBLK={"blockdevices":[{"name":"nvme0n1","size":"1000204886016","type":"disk","fstype":null,"pttype":null,"mountpoint":null,"rm":"0"}]}
BYID=nvme0n1 nvme-eui.0025385a91b04a0b
BYID=nvme0n1 nvme-Samsung_SSD_970`
	hw, err := ParseProbe(out)
	if err != nil {
		t.Fatal(err)
	}
	d := findDisk(hw, "nvme0n1")
	if d == nil || !d.Empty() || d.SizeGB() != 1000 {
		t.Fatalf("unexpected disk %+v", d)
	}
	if got := d.PreferredByID(); got != "/dev/disk/by-id/nvme-eui.0025385a91b04a0b" {
		t.Fatalf("PreferredByID = %q", got)
	}
}
