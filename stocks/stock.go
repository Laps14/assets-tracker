package stocks

import "fmt"

type Stock struct {
	Title string
	Description string
	Sect Sector
	TotalSupply float64
	StockVal float64
	ID uint64 // Is it necessary?
}

func (s *Stock) String() {
	fmt.Printf("\nINDICADOR: %s\n" +
		"DESCRIÇÃO: %s\n" +
		"VALOR COTA: %f\n\n",
		s.Title, s.Description, s.StockVal)
}
