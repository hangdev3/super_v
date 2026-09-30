package core

import (
	"fmt"
	"testing"
)

func TestHistoryUsesTenItemDefaultAndEvictsTheOldestEntry(t *testing.T) {
	// Arrange
	history := NewHistory(0)
	for number := 1; number <= 11; number++ {
		history.Add(fmt.Sprintf("item-%02d", number))
	}

	// Act
	items := history.Items()

	// Assert
	if len(items) != 10 {
		t.Fatalf("expected the default history limit to be 10, got %d", len(items))
	}
	if items[0].Text != "item-11" || items[len(items)-1].Text != "item-02" {
		t.Fatalf("expected newest-first eviction of the oldest item, got %#v", items)
	}
}

func TestHistoryRecordsNewestTextAndSuppressesConsecutiveDuplicate(t *testing.T) {
	// Arrange
	history := NewHistory(5)

	// Act
	firstAdded := history.Add("primeiro")
	duplicateAdded := history.Add("primeiro")
	secondAdded := history.Add("segundo")

	// Assert
	if !firstAdded || !secondAdded {
		t.Fatalf("expected distinct clipboard entries to be added")
	}
	if duplicateAdded {
		t.Fatalf("expected a consecutive duplicate not to be added")
	}

	items := history.Items()
	if len(items) != 2 {
		t.Fatalf("expected 2 history items, got %d", len(items))
	}
	if items[0].Text != "segundo" || items[1].Text != "primeiro" {
		t.Fatalf("expected newest item first, got %#v", items)
	}
}

func TestHistorySuppressesDuplicateAlreadyPresentAfterAnotherItem(t *testing.T) {
	// Arrange
	history := NewHistory(5)
	history.Add("go run -buildvcs=false ./cmd/super-v --show-on-start")
	history.Add("texto diferente")

	// Act
	duplicateAdded := history.Add("go run -buildvcs=false ./cmd/super-v --show-on-start")

	// Assert
	if duplicateAdded {
		t.Fatalf("expected an existing history text not to be added again")
	}
	items := history.Items()
	if len(items) != 2 {
		t.Fatalf("expected duplicate content not to create a new row, got %d items", len(items))
	}
}

func TestHistoryRemovesOldestItemsWhenLimitIsExceeded(t *testing.T) {
	// Arrange
	history := NewHistory(2)

	// Act
	history.Add("um")
	history.Add("dois")
	history.Add("tres")

	// Assert
	items := history.Items()
	if len(items) != 2 {
		t.Fatalf("expected history to respect its limit, got %d items", len(items))
	}
	if items[0].Text != "tres" || items[1].Text != "dois" {
		t.Fatalf("expected oldest item to be removed, got %#v", items)
	}
}

func TestHistorySelectsAnItemByIDWithoutExposingInternalState(t *testing.T) {
	// Arrange
	history := NewHistory(5)
	history.Add("copiar")
	item := history.Items()[0]

	// Act
	selected, found := history.Select(item.ID)
	selected.Text = "alterado fora do historico"

	// Assert
	if !found {
		t.Fatalf("expected item %q to be found", item.ID)
	}
	if history.Items()[0].Text != "copiar" {
		t.Fatalf("expected history state not to be changed through selected copy")
	}
}

func TestHistoryIgnoresEmptyTextAndCanBeCleared(t *testing.T) {
	// Arrange
	history := NewHistory(5)

	// Act
	emptyAdded := history.Add("   ")
	history.Add("conteudo")
	history.Clear()

	// Assert
	if emptyAdded {
		t.Fatalf("expected whitespace-only clipboard content to be ignored")
	}
	if len(history.Items()) != 0 {
		t.Fatalf("expected clear to remove all history items")
	}
}

func TestHistoryStoresTextAndImageItemsWithTheirKind(t *testing.T) {
	// Arrange
	history := NewHistory(5)

	// Act
	textAdded := history.AddText("texto")
	imageAdded := history.AddImage("/home/look/Imagens/Captures/capture.png")

	// Assert
	if !textAdded || !imageAdded {
		t.Fatalf("expected text and image entries to be added")
	}
	items := history.Items()
	if items[0].Kind != ClipKindImage || items[0].ImagePath == "" {
		t.Fatalf("expected newest item to be an image, got %#v", items[0])
	}
	if items[1].Kind != ClipKindText || items[1].Text != "texto" {
		t.Fatalf("expected older item to be text, got %#v", items[1])
	}
}

func TestHistorySuppressesDuplicateImagePath(t *testing.T) {
	// Arrange
	history := NewHistory(5)
	history.AddImage("/tmp/capture.png")

	// Act
	duplicateAdded := history.AddImage("/tmp/capture.png")

	// Assert
	if duplicateAdded {
		t.Fatalf("expected the same image path not to be duplicated")
	}
}
