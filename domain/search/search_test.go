package search_test

import (
	"reflect"
	"testing"
	"time"

	"cwdgo/domain/recentfolders"
	"cwdgo/domain/search"
)

func entry(path string) recentfolders.Entry {
	return recentfolders.Entry{Path: path, LastUsed: time.Now()}
}

func paths(es []recentfolders.Entry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Path
	}
	return out
}

func TestEmptyQueryReturnsAllInInputOrder(t *testing.T) {
	in := []recentfolders.Entry{entry(`C:\a`), entry(`C:\b`)}
	got := search.Search(in, "")
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("Search(_, \"\") = %v, want %v", paths(got), paths(in))
	}
}

func TestNoMatchReturnsEmpty(t *testing.T) {
	in := []recentfolders.Entry{entry(`C:\Users\Documents`), entry(`D:\work`)}
	if got := search.Search(in, "zzzzzz"); len(got) != 0 {
		t.Fatalf("Search = %v, want no matches", paths(got))
	}
}

func TestMatchesFolderNameCaseInsensitively(t *testing.T) {
	in := []recentfolders.Entry{
		entry(`C:\Users\jerem\Documents`),
		entry(`D:\work\proj`),
	}
	got := search.Search(in, "DOCUMENT")
	if len(got) != 1 || got[0].Path != `C:\Users\jerem\Documents` {
		t.Fatalf("Search(%q) = %v, want Documents", "DOCUMENT", paths(got))
	}
}

func TestMatchesFullPathWhenNameDoesNot(t *testing.T) {
	in := []recentfolders.Entry{
		entry(`C:\Users\jerem\Documents`), // name "Documents" doesn't match; path does
	}
	got := search.Search(in, "users")
	if len(got) != 1 || got[0].Path != `C:\Users\jerem\Documents` {
		t.Fatalf("Search(%q) = %v, want path match", "users", paths(got))
	}
}

func TestFuzzyPartialInputHitsName(t *testing.T) {
	in := []recentfolders.Entry{
		entry(`C:\Users\jerem\Documents`),
		entry(`D:\other`),
	}
	got := search.Search(in, "dcmnt") // missing letters, wrong order of letters skipped
	if len(got) != 1 || got[0].Path != `C:\Users\jerem\Documents` {
		t.Fatalf("Search(%q) = %v, want Documents", "dcmnt", paths(got))
	}
}

func TestExactNameMatchRanksAboveFuzzyNameMatch(t *testing.T) {
	in := []recentfolders.Entry{
		entry(`C:\a\documents`),
		entry(`C:\b\docs`),
	}
	got := search.Search(in, "docs")
	want := []string{`C:\b\docs`, `C:\a\documents`}
	if !reflect.DeepEqual(paths(got), want) {
		t.Fatalf("Search(%q) = %v, want %v", "docs", paths(got), want)
	}
}

func TestNameMatchRanksAbovePathOnlyMatch(t *testing.T) {
	in := []recentfolders.Entry{
		entry(`C:\Program Files\SomeApp`), // name "SomeApp"; path contains "pro"
		entry(`D:\work\proj`),             // name "proj" starts with "pro"
	}
	got := search.Search(in, "pro")
	want := []string{`D:\work\proj`, `C:\Program Files\SomeApp`}
	if !reflect.DeepEqual(paths(got), want) {
		t.Fatalf("Search(%q) = %v, want %v", "pro", paths(got), want)
	}
}

func TestEarlierMatchRanksAboveLaterMatch(t *testing.T) {
	in := []recentfolders.Entry{
		entry(`C:\a\MyDocuments`), // "doc" at index 2
		entry(`C:\b\Documents`),   // "doc" at index 0
	}
	got := search.Search(in, "doc")
	want := []string{`C:\b\Documents`, `C:\a\MyDocuments`}
	if !reflect.DeepEqual(paths(got), want) {
		t.Fatalf("Search(%q) = %v, want %v", "doc", paths(got), want)
	}
}

func TestContiguousMatchRanksAboveGappedMatch(t *testing.T) {
	in := []recentfolders.Entry{
		entry(`C:\a\axbxc`), // gapped: a x b x c
		entry(`C:\b\abcx`),  // contiguous
	}
	got := search.Search(in, "abc")
	want := []string{`C:\b\abcx`, `C:\a\axbxc`}
	if !reflect.DeepEqual(paths(got), want) {
		t.Fatalf("Search(%q) = %v, want %v", "abc", paths(got), want)
	}
}

func TestEqualMatchesKeepInputOrder(t *testing.T) {
	in := []recentfolders.Entry{
		entry(`C:\a\project`),
		entry(`C:\b\project`),
	}
	got := search.Search(in, "project")
	want := []string{`C:\a\project`, `C:\b\project`}
	if !reflect.DeepEqual(paths(got), want) {
		t.Fatalf("Search(%q) = %v, want %v (stable)", "project", paths(got), want)
	}
}

func TestMatchesChineseFolderNames(t *testing.T) {
	in := []recentfolders.Entry{
		entry(`C:\资料\项目文档`),
		entry(`D:\work`),
	}
	got := search.Search(in, "文档")
	if len(got) != 1 || got[0].Path != `C:\资料\项目文档` {
		t.Fatalf("Search(%q) = %v, want 项目文档", "文档", paths(got))
	}
}

func TestSearchDoesNotMutateInput(t *testing.T) {
	in := []recentfolders.Entry{entry(`C:\a\docs`), entry(`C:\b\documents`)}
	before := append([]recentfolders.Entry(nil), in...)
	search.Search(in, "docs")
	if !reflect.DeepEqual(in, before) {
		t.Fatal("Search mutated its input")
	}
}

// Filter keeps input order, unlike Search which ranks by fuzzy score. The
// panel shows Recent Folders newest-first and must not reshuffle recorded
// items when several match the query.
func TestFilterKeepsInputOrderAcrossDifferentScores(t *testing.T) {
	in := []recentfolders.Entry{
		entry(`C:\a\MyDocuments`), // fuzzy match, later start
		entry(`C:\b\Documents`),   // prefix match, better score
	}
	got := search.Filter(in, "doc")
	// Same set, but input order preserved (NOT ranked best-first).
	want := []string{`C:\a\MyDocuments`, `C:\b\Documents`}
	if !reflect.DeepEqual(paths(got), want) {
		t.Fatalf("Filter(%q) = %v, want %v (input order)", "doc", paths(got), want)
	}
}

func TestFilterEmptyQueryReturnsAllInInputOrder(t *testing.T) {
	in := []recentfolders.Entry{entry(`C:\a`), entry(`C:\b`)}
	got := search.Filter(in, "")
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("Filter empty = %v, want %v", paths(got), paths(in))
	}
}

func TestFilterFuzzyStillMatches(t *testing.T) {
	in := []recentfolders.Entry{
		entry(`C:\Users\jerem\Documents`),
		entry(`D:\other`),
	}
	got := search.Filter(in, "dcmnt")
	if len(got) != 1 || got[0].Path != `C:\Users\jerem\Documents` {
		t.Fatalf("Filter(%q) = %v, want Documents", "dcmnt", paths(got))
	}
}

func TestFilterNoMatchReturnsEmpty(t *testing.T) {
	in := []recentfolders.Entry{entry(`C:\docs`)}
	if got := search.Filter(in, "zzzz"); len(got) != 0 {
		t.Fatalf("Filter = %v, want empty", paths(got))
	}
}
