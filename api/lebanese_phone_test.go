package api

import "testing"

func TestParseLebanesePhone(t *testing.T) {
	cases := []struct {
		in   string
		want phoneKind
	}{
		{"03 123 456", phoneMobile},
		{"3123456", phoneMobile},
		{"+961 3 123 456", phoneMobile},
		{"00961-71-123-456", phoneMobile},
		{"71123456", phoneMobile},
		{"081 123 456", phoneMobile},
		{"76123456", phoneMobile},
		{"01 234 567", phoneLandline},
		{"+961 1 234 567", phoneLandline},
		{"09 123 456", phoneLandline},
		{"٠٣١٢٣٤٥٦", phoneMobile},
		{"", phoneInvalid},
		{"123", phoneInvalid},
		{"0312345", phoneInvalid},
		{"031234567", phoneInvalid},
		{"72123456", phoneInvalid},
		{"02 123 456", phoneInvalid},
		{"+44 7911 123456", phoneInvalid},
		{"03-abc-456", phoneInvalid},
	}
	for _, c := range cases {
		if _, got := parseLebanesePhone(c.in); got != c.want {
			t.Errorf("parseLebanesePhone(%q) = %v, want %v", c.in, got, c.want)
		}
	}
	if phoneIdentity("+961 3 123 456") != phoneIdentity("03123456") {
		t.Error("same number written two ways should share an identity")
	}
}

func TestDedupeClientExportFiles(t *testing.T) {
	row := func(id int64, phone string, spent int64) clientExportRow {
		return clientExportRow{ID: id, Phone: phone, TotalSpent: spent}
	}
	files := []clientExportFile{
		{name: "a", rows: []clientExportRow{row(1, "03 123 456", 10), row(2, "71 111 111", 5), row(3, "71111111", 9), row(4, "", 1), row(5, "", 1)}},
		{name: "b", rows: []clientExportRow{row(6, "+961 3 123 456", 7), row(7, "70 222 222", 2)}},
	}
	dedupeClientExportFiles(files)

	ids := func(f clientExportFile) []int64 {
		var out []int64
		for _, r := range f.rows {
			out = append(out, r.ID)
		}
		return out
	}
	// 03123456 is in both files, so it is gone from both; 71111111 appears
	// twice in a, so only the bigger spender (3) stays; blank phones stay.
	if got := ids(files[0]); len(got) != 3 || got[0] != 3 || got[1] != 4 || got[2] != 5 {
		t.Errorf("file a = %v, want [3 4 5]", got)
	}
	if got := ids(files[1]); len(got) != 1 || got[0] != 7 {
		t.Errorf("file b = %v, want [7]", got)
	}
}
