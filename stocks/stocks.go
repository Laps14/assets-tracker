package stocks

import (
	"fmt"
	"hash/maphash"
	"math"
)

// A struct to hold a hash table of Stock
// Could be implemented using a map, but
// I wanted to make it as a matrix just
// for learning purposes.
type Stocks struct {
	lfactor float32
	stocks [][]Stock
	hash maphash.Hash
}

// A factory function to make a new Stocks
// hash table.
func NewStocks() *Stocks {
	return &Stocks{
		lfactor: 0.6,
		stocks: make([][]Stock, 41),
	}
}

// Stocks hash table's function to increase
// the size of the table to accommodate new
// Stock.
func (s *Stocks) increaseSize() {
	var (
		temp = make([][]Stock, 0)
		temp_length = len(s.stocks)
	)

	for !s.newPNSize(&temp_length, &temp) {} // while a valid prime number isn't assigned to temp's length

	for i, stocks := 0, len(s.stocks); i < stocks; i++ {
		for _, j := range s.stocks[i] {
			s.hash.WriteString(j.Title)
			temp[s.hash.Sum64() % uint64(len(temp))] = append(temp[s.hash.Sum64() % uint64(len(temp))], j)
			s.hash.Reset()
		}
	}

	s.stocks = temp
}

func (s *Stocks) insert(stock *Stock) {

	if len(stock.Title) < 3{
		s.hash.WriteString(stock.Title)
	} else {
		s.hash.WriteString(stock.Title[:3])
	}

	n := s.stocks[s.hash.Sum64() % uint64(len(s.stocks))]

	if (float32(len(n)) / float32(len(s.stocks))) > s.lfactor {
		s.hash.Reset()
		s.increaseSize()
		s.hash.WriteString(stock.Title)
		n = s.stocks[s.hash.Sum64() % uint64(len(s.stocks))]
	}

	s.stocks[s.hash.Sum64() % uint64(len(s.stocks))] = append(n, *stock)

	s.hash.Reset()
}

func (s *Stocks) search(stock_title string) (*Stock, error) {

	if len(stock_title) < 3{
		s.hash.WriteString(stock_title)
	} else {
		s.hash.WriteString(stock_title[:3])
	}

	n := s.hash.Sum64() % uint64(len(s.stocks))

	for _, stock := range s.stocks[n]{
		if stock.Title != stock_title { continue }
		s.hash.Reset()
		return &stock, nil
	}

	s.hash.Reset()
	return nil, fmt.Errorf("'%s' Not found", stock_title)
}

func (s *Stocks) newPNSize(n *int, temp *[][]Stock) bool {
	*n = (*n+1) * 2

	if *n % 2 == 0 { *n += 1 }

	sqrt_stocks := int(math.Sqrt(float64(*n)))

	for i := 2; i <= sqrt_stocks; i++ {
		if *n % i == 0 { return false }
	}

	*temp = make([][]Stock, *n)

	return true
}

func (s *Stocks) printAll() {
	for i, stocks := 0, len(s.stocks); i < stocks; i++ {
		if len(s.stocks[i]) != 0 { fmt.Printf("%d -> ", i) }
		for _, j := range s.stocks[i] {
			fmt.Printf("[%s]\t", j.Title)
		}
		if len(s.stocks[i]) != 0 { fmt.Printf("\n") }
	}
}
