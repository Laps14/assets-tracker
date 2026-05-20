package stocks

import "fmt"

type Stock struct {
	Title string
	Description string
	Sect Sector
	TotalSupply float64
	StockVal float64
	TargetVals []float64
	CompanyLogo string
	ID int64
}

func (s *Stock) String() {
	fmt.Printf("\nINDICADOR: %s\n" +
		"DESCRIÇÃO: %s\n" +
		"VALOR COTA: %.2f\n" +
		"TARGET VALS: %v\n" +
		"ID: %v\n\n",
		s.Title, s.Description, s.StockVal, s.TargetVals, s.ID)
}
