package wnram

import (
	"path"
	"runtime"
	"slices"
	"testing"
)

const PathToWordnetDataFiles = "./data"

func sourceCodeRelPath(suffix string) string {
	_, fileName, _, _ := runtime.Caller(1)
	return path.Join(path.Dir(fileName), suffix)
}

var wnInstance *Handle
var wnErr error

func init() {
	wnInstance, wnErr = New(sourceCodeRelPath(PathToWordnetDataFiles))
}

func TestParsing(t *testing.T) {
	if wnErr != nil {
		t.Fatalf("Can't initialize: %s", wnErr)
	}
}

func TestBasicLookup(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "good"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	gotAdjective := false
	for _, f := range found {
		if f.POS() == Adjective {
			gotAdjective = true
			break
		}
	}

	if !gotAdjective {
		t.Errorf("couldn't find basic adjective form for good")
	}
}

func TestPluralExceptionLookup(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "wolves"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	gotNoun := false
	for _, f := range found {
		if f.POS() == Noun {
			gotNoun = true
			break
		}
	}

	if !gotNoun {
		t.Errorf("couldn't find exception plural noun form for wolves")
	}
}

func TestPluralLookup(t *testing.T) {
	tests := []struct {
		plural   string
		singular string
		pos      PartOfSpeech
	}{
		// Noun examples
		{"dogs", "dog", Noun},
		{"cars", "car", Noun},
		{"houses", "house", Noun},
		// Verb examples
		{"runs", "run", Verb},
		{"flies", "fly", Verb},
		{"plays", "play", Verb},
		// Adjective examples
		{"faster", "fast", Adjective},
		{"stronger", "strong", Adjective},
	}

	for _, tt := range tests {
		found, err := wnInstance.Lookup(Criteria{Matching: tt.plural})
		if err != nil {
			t.Errorf("Lookup(%q) failed: %v", tt.plural, err)
			continue
		}
		if len(found) == 0 {
			t.Errorf("couldn't find %v form for %q", tt.pos, tt.plural)
		}
	}
}

func TestLemma(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "awesome", POS: []PartOfSpeech{Adjective}})
	if err != nil {
		t.Fatalf("%s", err)
	}

	if len(found) != 1 {
		for _, f := range found {
			f.Dump()
		}
		t.Fatalf("expected one synonym cluster for awesome, got %d", len(found))
	}

	if found[0].Lemma() != "amazing" {
		t.Errorf("incorrect lemma for awesome (%s)", found[0].Lemma())
	}
}

func setContains(haystack, needles []string) bool {
	for _, n := range needles {
		found := slices.Contains(haystack, n)
		if !found {
			return false
		}
	}
	return true
}

func TestSynonyms(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "yummy", POS: []PartOfSpeech{Adjective}})
	if err != nil {
		t.Fatalf("%s", err)
	}

	if len(found) != 1 {
		for _, f := range found {
			f.Dump()
		}
		t.Fatalf("expected one synonym cluster for yummy, got %d", len(found))
	}

	syns := found[0].Synonyms()
	if !setContains(syns, []string{"delicious", "delectable"}) {
		t.Errorf("missing synonyms for yummy")
	}
}

func TestAntonyms(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "good", POS: []PartOfSpeech{Adjective}})
	if err != nil {
		t.Fatalf("%s", err)
	}

	var antonyms []string
	for _, f := range found {
		as := f.Related(Antonym)
		for _, a := range as {
			antonyms = append(antonyms, a.Word())
		}
	}

	if !setContains(antonyms, []string{"bad", "evil"}) {
		t.Errorf("missing antonyms for good")
	}
}

func TestHypernyms(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "jab", POS: []PartOfSpeech{Noun}})
	if err != nil {
		t.Fatalf("%s", err)
	}

	var hypernyms []string
	for _, f := range found {
		as := f.Related(Hypernym)
		for _, a := range as {
			hypernyms = append(hypernyms, a.Word())
		}
	}

	if !setContains(hypernyms, []string{"punch"}) {
		t.Errorf("missing hypernyms for jab (expected punch, got %v)", hypernyms)
	}
}

func TestHyponyms(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "food", POS: []PartOfSpeech{Noun}})
	if err != nil {
		t.Fatalf("%s", err)
	}

	var hyponyms []string
	for _, f := range found {
		as := f.Related(Hyponym)
		for _, a := range as {
			hyponyms = append(hyponyms, a.Word())
		}
	}

	expected := []string{"chocolate", "cheese", "pasta", "leftovers"}
	if !setContains(hyponyms, expected) {
		t.Errorf("missing hyponyms for candy (expected %v, got %v)", expected, hyponyms)
	}
}

func TestIterate(t *testing.T) {
	count := 0
	err := wnInstance.Iterate(PartOfSpeechList{Noun}, func(l Lookup) error {
		count++
		return nil
	})

	if err != nil {
		t.Fatalf("Iterate failed: %v", err)
	}

	if count != 82192 {
		t.Errorf("Missing nouns!")
	}
}
func TestWordbase(t *testing.T) {
	tests := []struct {
		word     string
		ender    int
		expected string
	}{
		// Noun suffixes
		{"dogs", 0, "dog"},
		{"buses", 1, "bus"},
		// Verb suffixes
		{"runs", 8, "run"},
		{"flies", 9, "fly"},
		// Adjective suffixes
		{"faster", 16, "fast"},  // "er" -> ""
		{"fastest", 17, "fast"}, // "est" -> ""
	}

	for _, tt := range tests {
		got := wordbase(tt.word, tt.ender)
		if got != tt.expected {
			t.Errorf("wordbase(%q, %d) = %q; want %q", tt.word, tt.ender, got, tt.expected)
		}
	}
}

func TestMorphword(t *testing.T) {
	tests := []struct {
		word     string
		pos      PartOfSpeech
		expected string
	}{
		// Noun cases
		{"dogs", Noun, "dog"},
		{"buses", Noun, "bus"},
		{"boxes", Noun, "box"},
		{"handful", Noun, "hand"},
		{"men", Noun, "man"},
		{"ladies", Noun, "lady"},
		{"fullness", Noun, ""}, // "ss" ending returns ""
		{"a", Noun, ""},        // too short returns ""
		// Verb cases
		{"runs", Verb, "run"},
		{"flies", Verb, "fly"},
		{"played", Verb, "play"},
		{"playing", Verb, "play"},
		// Adjective cases
		{"faster", Adjective, "fast"},
		{"fastest", Adjective, "fast"},
		{"stronger", Adjective, "strong"},
		{"strongest", Adjective, "strong"},
		// Adverb cases (should not change)
		{"quickly", Adverb, ""},
		{"slowly", Adverb, ""},
	}

	for _, tt := range tests {
		got := wnInstance.MorphWord(tt.word, tt.pos)
		if got != tt.expected {
			t.Errorf("morphword(%q, %v) = %q; want %q", tt.word, tt.pos, got, tt.expected)
		}
	}
}
