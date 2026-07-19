package services

import (
	"strings"
	"testing"
)

func TestParseSamehaPlayerOptions(t *testing.T) {
	html := `
	<div id="player-option-1" class="east_player_option" data-post="51258" data-nume="1" data-type="schtml"><span>Blogspot </span></div>
	<div id="player-option-4" class="east_player_option" data-post="51258" data-nume="4" data-type="schtml"><span>Wibufile  720p</span></div>
	<div id="player-option-5" class="east_player_option" data-post="51258" data-nume="5" data-type="schtml"><span>Wibufile 1080p</span></div>
	`
	mirrors := parseSamehaPlayerOptions(html)
	if len(mirrors) < 3 {
		t.Fatalf("expected >=3 mirrors, got %d", len(mirrors))
	}
	// Best should be 1080p Wibufile
	if !strings.Contains(strings.ToLower(mirrors[0].Label), "1080") {
		t.Errorf("best mirror should be 1080p, got %q score=%d", mirrors[0].Label, mirrors[0].Score)
	}
	if mirrors[0].PostID != 51258 {
		t.Errorf("post id = %d", mirrors[0].PostID)
	}
}

func TestScoreSamehaMirror(t *testing.T) {
	if scoreSamehaMirror("Wibufile 1080p") <= scoreSamehaMirror("Blogspot") {
		t.Error("1080 wibufile should score higher than blogspot")
	}
}

func TestSamehaAjaxURL(t *testing.T) {
	u := samehaAjaxURL("https://v2.samehadaku.how/foo/")
	if u != "https://v2.samehadaku.how/wp-admin/admin-ajax.php" {
		t.Errorf("got %s", u)
	}
}
