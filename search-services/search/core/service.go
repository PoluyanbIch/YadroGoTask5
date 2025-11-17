package core

import (
	"context"
	"log/slog"
	"math"
	"sort"
	"strings"
)

type Service struct {
	log    *slog.Logger
	db     DB
	words  Words
	update Update
}

func NewService(log *slog.Logger, db DB, words Words, update Update) *Service {
	s := &Service{
		log:    log,
		db:     db,
		words:  words,
		update: update,
	}
	return s
}

func (s *Service) Search(ctx context.Context, phrase string, limit int) ([]Comic, error) {
	if err := s.update.Update(ctx); err != nil {
		s.log.Error("Search/update error", "error", err)
		return nil, err
	}
	normPhrase, err := s.words.Norm(ctx, phrase)
	if err != nil {
		s.log.Error("Search/words error", "error", err)
		return nil, err
	}
	comics, err := s.db.Read(ctx)
	if err != nil {
		s.log.Error("Search/db error", "error", err)
		return nil, err
	}
	comicsMap := make(map[int]DBComic)
	for _, comic := range comics {
		comicsMap[comic.ID] = comic
	}
	res := search(normPhrase, comics)

	type result struct {
		ID    int
		Score float64
	}
	var sortedResult []result
	for id, score := range res {
		sortedResult = append(sortedResult, result{ID: id, Score: score})
	}
	sort.Slice(sortedResult, func(i, j int) bool { return sortedResult[i].Score > sortedResult[j].Score })
	var searchResult []Comic
	for _, i := range sortedResult {
		searchResult = append(searchResult, Comic{ID: i.ID, URL: comicsMap[i.ID].URL})
	}
	if len(searchResult) < limit {
		limit = len(searchResult)
	}
	return searchResult[:limit], nil
}

func search(phrase []string, comics []DBComic) map[int]float64 {
	res := make(map[int]float64)

	comicsMap := make(map[int]DBComic)
	for _, comic := range comics {
		comicsMap[comic.ID] = comic
	}

	var comicsWithWords []int
	for _, c := range comics {
		for _, word := range phrase {
			if countSubstringInField(word, c.Description) > 0 {
				comicsWithWords = append(comicsWithWords, c.ID)
				break
			}
			if countSubstringInField(word, c.Alt) > 0 {
				comicsWithWords = append(comicsWithWords, c.ID)
				break
			}
			if countSubstringInField(word, c.Title) > 0 {
				comicsWithWords = append(comicsWithWords, c.ID)
				break
			}
		}
	}

	idfCache := make(map[string]float64)
	for _, word := range phrase {
		idfCache[word] = calculateIDF(word, comics)
	}
	const (
		titleWeight       = 3.0
		altWeight         = 2.0
		descriptionWeight = 1.0
		fullMatchBonus    = 100.0
	)

	for _, id := range comicsWithWords {

		res[id] = 0
		countMatchWords := 0
		for _, word := range phrase {
			res[id] += idfCache[word] * calculateTF(word, comicsMap[id].Title) * titleWeight
			res[id] += idfCache[word] * calculateTF(word, comicsMap[id].Alt) * altWeight
			res[id] += idfCache[word] * calculateTF(word, comicsMap[id].Description) * descriptionWeight
			if countSubstringInField(word, comicsMap[id].Title) > 0 {
				countMatchWords++
			} else if countSubstringInField(word, comicsMap[id].Alt) > 0 {
				countMatchWords++
			} else if countSubstringInField(word, comicsMap[id].Description) > 0 {
				countMatchWords++
			}
		}
		if countMatchWords == len(phrase) {
			res[id] *= fullMatchBonus
		}
	}
	return res
}

func calculateTF(word string, field map[string]int) float64 {
	if field == nil {
		return 0
	}
	num := countSubstringInField(word, field)
	sum := 0
	for _, val := range field {
		sum += val
	}
	if sum == 0 {
		return 0
	}
	return float64(num) / float64(sum)
}

func calculateIDF(word string, comics []DBComic) float64 {
	num := 0
	for _, comic := range comics {
		title := comic.Title
		alt := comic.Alt
		description := comic.Description
		if countSubstringInField(word, title) > 0 {
			num++
		} else if countSubstringInField(word, alt) > 0 {
			num++
		} else if countSubstringInField(word, description) > 0 {
			num++
		}
	}
	if num == 0 {
		return 0
	}
	sum := len(comics)
	return math.Log(float64(sum) / float64(num))
}

func countSubstringInField(word string, field map[string]int) int {
	num := 0
	for w := range field {
		if strings.Contains(word, w) || strings.Contains(w, word) {
			num += field[w]
		}
	}
	return num
}
