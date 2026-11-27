package tts

import (
	"encoding/xml"
	"strings"
	"testing"

	"biblio-audiobook-builder-tts/internal/ssml"
)

func TestCommaBreaksSurviveChunking(t *testing.T) {
	text := strings.Repeat("alpha, beta, gamma, delta, ", 40) + "omega."
	marked := ssml.AddSSMLBreaks(text, (&ConversionOptions{CommaBreakMs: 120}).SSMLOptions())
	want := strings.Count(marked, `<break time="120ms"/>`)
	if want != 160 {
		t.Fatalf("breaks before chunking = %d", want)
	}

	for _, size := range []int{60, 97, 150, 900} {
		chunker := NewChunker(&ChunkerConfig{Mode: ChunkBySentence, MaxChunkSize: size, PreserveWords: true})
		got := 0
		for _, chunk := range chunker.MergeSmallChunks(chunker.Chunk(marked), 100) {
			var doc struct{}
			if err := xml.Unmarshal([]byte("<speak>"+chunk+"</speak>"), &doc); err != nil {
				t.Fatalf("size %d: malformed chunk %q: %v", size, chunk, err)
			}
			got += strings.Count(chunk, `<break time="120ms"/>`)
		}
		if got != want {
			t.Errorf("size %d: breaks after chunking = %d, want %d", size, got, want)
		}
	}
}
