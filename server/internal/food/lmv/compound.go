package lmv

import (
	"sort"
	"strings"
	"unicode"
)

// Swedish writes compounds as one word: kycklingfärs, falukorv, bröstfilé.
// Postgres stems Swedish but never splits a compound, so a search for
// "kyckling" misses "kycklingfärs" and "korv" misses "falukorv". The release
// itself says which words exist, since most also stand alone in some name, and
// a compound is split into those.

const (
	// minPart is the shortest part a compound is split into.
	minPart = 3
	// minHead is the shortest final part added on its own. Swedish compounds
	// end with what the thing is (a falukorv is a korv), so the last part is
	// worth finding even when what comes before it is no word.
	minHead = 4
	// minCompound is the shortest word worth splitting.
	minCompound = 6
	// minNames is in how many names a word must stand alone to be known.
	minNames = 2
)

// Vocabulary is the set of known words, lower case.
type Vocabulary map[string]bool

// NewVocabulary learns the words that stand alone in at least two names.
func NewVocabulary(names []string) Vocabulary {
	seen := map[string]int{}
	for _, name := range names {
		unique := map[string]bool{}
		for _, word := range Words(name) {
			unique[word] = true
		}
		for word := range unique {
			seen[word]++
		}
	}
	v := Vocabulary{}
	for word, names := range seen {
		if names >= minNames && len([]rune(word)) >= minPart {
			v[word] = true
		}
	}
	return v
}

// VocabularyOf rebuilds a vocabulary as stored.
func VocabularyOf(words []string) Vocabulary {
	v := make(Vocabulary, len(words))
	for _, word := range words {
		v[word] = true
	}
	return v
}

// List returns the words sorted, for storing.
func (v Vocabulary) List() []string {
	out := make([]string, 0, len(v))
	for word := range v {
		out = append(out, word)
	}
	sort.Strings(out)
	return out
}

// Words splits text into lower case words of letters.
func Words(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) })
}

// Split breaks a compound into known words, allowing the linking s of
// havregrynsgröt and at most three parts. Nil when it is no such compound.
func (v Vocabulary) Split(word string) []string {
	return v.split([]rune(word), 2)
}

func (v Vocabulary) split(word []rune, depth int) []string {
	for i := minPart; i <= len(word)-minPart; i++ {
		head := string(word[:i])
		if !v[head] {
			continue
		}
		tails := [][]rune{word[i:]}
		if word[i] == 's' && len(word)-i-1 >= minPart {
			tails = append(tails, word[i+1:])
		}
		for _, tail := range tails {
			if v[string(tail)] {
				return []string{head, string(tail)}
			}
			if depth > 1 {
				if rest := v.split(tail, depth-1); rest != nil {
					return append([]string{head}, rest...)
				}
			}
		}
	}
	return nil
}

// Head returns the longest known word a compound ends with, or "".
func (v Vocabulary) Head(word string) string {
	runes := []rune(word)
	for i := minPart; i <= len(runes)-minHead; i++ {
		if tail := string(runes[i:]); v[tail] {
			return tail
		}
	}
	return ""
}

// Terms returns the words a text's compounds hold, those of the text itself
// left out, joined by spaces for search_terms.
func (v Vocabulary) Terms(text string) string {
	words := Words(text)
	have := map[string]bool{}
	for _, word := range words {
		have[word] = true
	}
	var out []string
	add := func(term string) {
		if term != "" && !have[term] {
			have[term] = true
			out = append(out, term)
		}
	}
	for _, word := range words {
		if len([]rune(word)) < minCompound {
			continue
		}
		for _, part := range v.Split(word) {
			add(part)
		}
		add(v.Head(word))
	}
	return strings.Join(out, " ")
}

// Query builds a to_tsquery expression matching every word of a search.
// Each word may also match as the parts it splits into, so "kycklingfilé"
// finds "Kyckling bröstfilé". The last word matches as a prefix, for search as
// you type. Empty when the search has no words. Words hold letters only, so
// nothing a user types can become an operator.
func (v Vocabulary) Query(search string) string {
	words := Words(search)
	clauses := make([]string, len(words))
	for i, word := range words {
		term := word
		if i == len(words)-1 {
			term += ":*"
		}
		if parts := v.Split(word); parts != nil {
			term = "(" + term + " | (" + strings.Join(parts, " & ") + "))"
		}
		clauses[i] = term
	}
	return strings.Join(clauses, " & ")
}
