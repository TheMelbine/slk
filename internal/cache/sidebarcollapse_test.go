package cache

import "testing"

func TestSectionCollapse_RoundTrip(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.SetSectionCollapsed("T1", "L1", true); err != nil {
		t.Fatal(err)
	}
	if err := db.SetSectionCollapsed("T1", "Channels", false); err != nil {
		t.Fatal(err)
	}
	if err := db.SetSectionCollapsed("T2", "L1", false); err != nil {
		t.Fatal(err)
	}
	if err := db.SetSectionCollapsed("T1", "L1", false); err != nil {
		t.Fatal(err)
	}
	got, err := db.SectionCollapse("T1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["L1"] || got["Channels"] {
		t.Errorf("T1 = %v, want L1 and Channels expanded", got)
	}
	other, _ := db.SectionCollapse("T2")
	if len(other) != 1 {
		t.Errorf("T2 = %v, want its own one row", other)
	}
}
