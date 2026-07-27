package insight

import "testing"

// The sizes come back as text from two different tools, and misreading one
// means a headline figure that is wrong by a factor of a thousand.
func TestDockerSizesAreRead(t *testing.T) {
	// Through a function so Go treats these as runtime values: a constant
	// float with a fractional part cannot be converted to int64 directly.
	scaled := func(value float64, scale int64) int64 { return int64(value * float64(scale)) }

	cases := map[string]int64{
		"3.875GB (97%)": scaled(3.875, 1<<30),
		"1.424GB":       scaled(1.424, 1<<30),
		"389.1kB":       scaled(389.1, 1<<10),
		"0B":            0,
		"":              0,
		"nonsense":      0,
	}

	for input, want := range cases {
		if got := dockerSize(input); got != want {
			t.Errorf("dockerSize(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestJournalSizeIsRead(t *testing.T) {
	text := "Archived and active journals take up 24.0M in the file system."
	if got, want := journalSize(text), int64(24*(1<<20)); got != want {
		t.Errorf("journalSize = %d, want %d", got, want)
	}

	if got := journalSize("nothing useful here"); got != 0 {
		t.Errorf("journalSize of unparseable text = %d, want 0", got)
	}
}

// Only the commands written down here can ever run, and each must be one that
// cannot delete something the owner put on the machine on purpose.
func TestOnlyKnownReclaimsHaveCommands(t *testing.T) {
	for _, id := range []string{"docker-images", "docker-build-cache", "apt-cache", "journal"} {
		if len(Commands(id)) == 0 {
			t.Errorf("%s has no command", id)
		}
		if Label(id) == "Reclaim space" {
			t.Errorf("%s has no label of its own", id)
		}
	}

	for _, unknown := range []string{"", "docker-volumes", "rm -rf /", "docker system prune --volumes"} {
		if cmds := Commands(unknown); cmds != nil {
			t.Errorf("Commands(%q) returned %v; anything not written down must return nothing", unknown, cmds)
		}
	}
}
