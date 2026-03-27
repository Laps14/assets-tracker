package stocks

import (
	"bytes"
	"errors"
	"fmt"
	"hash/maphash"
	"io"
	"math"
	"os"
)

type Stocks struct {
	lfactor float32
	stocks [][]Stock
	hash maphash.Hash
}

func (s *Stocks) init() {
	s.lfactor = 0.6
	s.stocks = make([][]Stock, 41)
}

func (s *Stocks) increaseSize() {
	temp := make([][]Stock, 0)
	temp_length := len(s.stocks)

	for !s.newPNSize(&temp_length, &temp) {}

	for i, stocks := 0, len(s.stocks); i < m; i++ {
		for _, j := range s.stocks[i] {
			s.hash.WriteString(j)
			temp[s.hash.Sum64() % uint64(len(temp))] = append(temp[s.hash.Sum64() % uint64(len(temp))], j)
			s.hash.Reset()
		}
	}

	s.stocks = temp
}

func (s *Stocks) decreaseSize() {}

func (s *Stocks) insert(stock *Stock) {

	if len(nitem) < 3{
		s.hash.WriteString(nitem)
	} else {
		s.hash.WriteString(nitem[:3])
	}

	n := s.stocks[s.hash.Sum64() % uint64(len(s.stocks))]

	if (float32(len(n)) / float32(len(s.stocks))) > s.lfactor {
		s.hash.Reset()
		s.increaseSize()
		s.hash.WriteString(nitem)
		n = s.stocks[s.hash.Sum64() % uint64(len(s.stocks))]
	}

	s.stocks[s.hash.Sum64() % uint64(len(s.stocks))] = append(n, nitem)

	s.hash.Reset()
}

func (s *Stocks) remove(nitestocks string) {}

func (s *Stocks) search(nitestocks string) (uint64, error) {

	if len(nitem) < 3{
		s.hash.WriteString(nitem)
	} else {
		s.hash.WriteString(nitem[:3])
	}

	n := s.hash.Sum64() % uint64(len(s.stocks))

	for _, str := range s.stocks[n]{
		if str != nitestocks { continue }
		s.hash.Reset()
		return n, nil
	}

	s.hash.Reset()
	return 0, errors.New("Not found")
}

func (s *Stocks) newPNSize(n *int, temp *[][]string) bool {
	*n = (*n+1) * 2

	if *n % 2 == 0 { *n += 1 }

	sqrt_stocks := int(math.Sqrt(float64(*n)))

	for i := 2; i <= sqrt_m; i++ {
		if *n % i == 0 { return false }
	}

	*temp = make([][]string, *n)

	return true
}

func (s *Stocks) printAll() {
	for i, stocks := 0, len(s.stocks); i < m; i++ {
		if len(s.stocks[i]) != 0 { fmt.Printf("%d -> ", i) }
		for _, j := range s.stocks[i] {
			fmt.Printf("[%s]\t", j)
		}
		if len(s.stocks[i]) != 0 { fmt.Printf("\n") }
	}
}
