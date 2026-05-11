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
	ID uint64 // Is it necessary?
}

func (s *Stock) String() {
	fmt.Printf("\nINDICADOR: %s\n" +
		"DESCRIÇÃO: %s\n" +
		"VALOR COTA: %.2f\n"+
		"COMPANY LOGO: %s\n",
		s.Title, s.Description, s.StockVal, s.CompanyLogo)
}
